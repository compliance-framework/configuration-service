// Package ssptemplates stores SSP templates: rules that create one SSP per distinct
// combination of key label values in the latest evidence and keep each SSP's scope up to
// date.
package ssptemplates

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/service/relational/sspscope"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Limits on a template, enforced on every write.
const (
	MaxKeys             = 5
	MaxScopeLabelGroups = 10
	MaxLabelsPerGroup   = 5
)

// ErrInvalidTemplate is returned when a template fails validation; callers answer 400.
var ErrInvalidTemplate = errors.New("invalid SSP template")

// Input is a template as submitted. Nil Enabled means true and nil MinStreams means 1.
type Input struct {
	Name                string
	Enabled             *bool
	Keys                []string
	Filter              *labelfilter.Filter
	ScopeLabels         [][]string
	MinStreams          *int
	TitleTemplate       string
	DescriptionTemplate *string
	Props               []relational.SSPTemplateProp
	ProfileIDs          []uuid.UUID
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidTemplate, fmt.Sprintf(format, args...))
}

// Build validates in and returns it as a template, normalized: keys and each scope-label
// group trimmed, lower-cased and sorted, an empty filter dropped, profile ids
// de-duplicated. It does not check that the profiles exist (the service does).
func Build(in Input) (relational.SSPTemplate, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return relational.SSPTemplate{}, invalid("name is required")
	}
	keys, err := normalizeLabelKeys(in.Keys, "keys", MaxKeys)
	if err != nil {
		return relational.SSPTemplate{}, err
	}

	if len(in.ScopeLabels) > MaxScopeLabelGroups {
		return relational.SSPTemplate{}, invalid("at most %d scope-label groups", MaxScopeLabelGroups)
	}
	groups := make([][]string, 0, len(in.ScopeLabels))
	seen := map[string]struct{}{}
	for i, group := range in.ScopeLabels {
		normalized, err := normalizeLabelKeys(group, fmt.Sprintf("scope-labels[%d]", i), MaxLabelsPerGroup)
		if err != nil {
			return relational.SSPTemplate{}, err
		}
		id := strings.Join(normalized, "\n")
		if _, dup := seen[id]; dup {
			return relational.SSPTemplate{}, invalid("scope-labels[%d] repeats another group", i)
		}
		seen[id] = struct{}{}
		groups = append(groups, normalized)
	}

	minStreams := 1
	if in.MinStreams != nil {
		minStreams = *in.MinStreams
	}
	if minStreams < 1 {
		return relational.SSPTemplate{}, invalid("min-streams must be at least 1")
	}

	if strings.TrimSpace(in.TitleTemplate) == "" {
		return relational.SSPTemplate{}, invalid("title-template is required")
	}
	if err := parseTemplate("title-template", in.TitleTemplate); err != nil {
		return relational.SSPTemplate{}, err
	}
	if in.DescriptionTemplate != nil {
		if err := parseTemplate("description-template", *in.DescriptionTemplate); err != nil {
			return relational.SSPTemplate{}, err
		}
	}
	props := make([]relational.SSPTemplateProp, 0, len(in.Props))
	for i, prop := range in.Props {
		propName := strings.TrimSpace(prop.Name)
		if propName == "" {
			return relational.SSPTemplate{}, invalid("props[%d]: name is required", i)
		}
		if err := parseTemplate(fmt.Sprintf("props[%d].value", i), prop.Value); err != nil {
			return relational.SSPTemplate{}, err
		}
		props = append(props, relational.SSPTemplateProp{Name: propName, Value: prop.Value})
	}

	var filter *datatypes.JSONType[labelfilter.Filter]
	if in.Filter != nil && in.Filter.Scope != nil && (in.Filter.Scope.Condition != nil || in.Filter.Scope.Query != nil) {
		if err := validateScope(*in.Filter.Scope); err != nil {
			return relational.SSPTemplate{}, err
		}
		f := datatypes.NewJSONType(*in.Filter)
		filter = &f
	}

	profileIDs := make([]uuid.UUID, 0, len(in.ProfileIDs))
	for _, id := range in.ProfileIDs {
		if !slices.Contains(profileIDs, id) {
			profileIDs = append(profileIDs, id)
		}
	}

	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	return relational.SSPTemplate{
		Name:                name,
		Enabled:             enabled,
		Keys:                keys,
		Filter:              filter,
		ScopeLabels:         datatypes.NewJSONType(groups),
		MinStreams:          minStreams,
		TitleTemplate:       in.TitleTemplate,
		DescriptionTemplate: in.DescriptionTemplate,
		Props:               props,
		ProfileIDs:          profileIDs,
	}, nil
}

// normalizeLabelKeys trims, lower-cases and sorts 1 to limit label keys, rejecting keys a
// scope entry could not hold (sspscope.Normalize's rules: empty, over-long, '=' or
// control characters, duplicates).
func normalizeLabelKeys(keys []string, field string, limit int) ([]string, error) {
	if len(keys) == 0 || len(keys) > limit {
		return nil, invalid("%s: between 1 and %d labels are required", field, limit)
	}
	labels := make([]relational.ScopeLabel, 0, len(keys))
	for _, key := range keys {
		// The value only satisfies Normalize; the key rules are what is checked.
		labels = append(labels, relational.ScopeLabel{Key: key, Value: "v"})
	}
	normalized, _, err := sspscope.Normalize(labels)
	if err != nil {
		return nil, invalid("%s: %s", field, strings.TrimPrefix(err.Error(), sspscope.ErrInvalidLabels.Error()+": "))
	}
	out := make([]string, 0, len(normalized))
	for _, label := range normalized {
		out = append(out, label.Key)
	}
	return out, nil
}

// parseTemplate checks that text parses as a Go text/template, the way
// templates.RenderTemplate will execute it.
func parseTemplate(field, text string) error {
	if _, err := template.New(field).Option("missingkey=zero").Parse(text); err != nil {
		return invalid("%s: %v", field, err)
	}
	return nil
}

// validateScope checks a label filter scope: each node is a condition or a query, a
// condition names a label and uses "=" or "!=", and a query is AND or OR over at least
// one scope.
func validateScope(scope labelfilter.Scope) error {
	condition, query := scope.Condition, scope.Query
	switch {
	case condition != nil && query != nil:
		return invalid("filter: a scope is either a condition or a query")
	case condition != nil:
		if strings.TrimSpace(condition.Label) == "" {
			return invalid("filter: a condition needs a label")
		}
		if condition.Operator != "=" && condition.Operator != "!=" {
			return invalid("filter: unsupported condition operator %q", condition.Operator)
		}
		return nil
	case query != nil:
		op := strings.ToUpper(query.Operator)
		if op != "AND" && op != "OR" {
			return invalid("filter: unsupported query operator %q", query.Operator)
		}
		if len(query.Scopes) == 0 {
			return invalid("filter: a query needs at least one scope")
		}
		for _, child := range query.Scopes {
			if err := validateScope(child); err != nil {
				return err
			}
		}
		return nil
	default:
		return invalid("filter: a nested scope must not be empty")
	}
}
