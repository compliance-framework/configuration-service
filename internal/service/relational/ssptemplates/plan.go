package ssptemplates

import (
	"sort"
	"strings"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/service/relational/sspscope"
	"github.com/compliance-framework/api/internal/service/relational/templates"
	"github.com/google/uuid"
)

// Plan is what discovery would do for one template, given the latest evidence.
type Plan struct {
	SSPs           []PlannedSSP   `json:"ssps"`
	BelowThreshold []GroupStreams `json:"below-threshold"`
	MissingKeys    int            `json:"missing-keys"` // streams that matched the filter but lacked a usable key value
	Invalid        []InvalidGroup `json:"invalid"`
}

// PlannedSSP is one group of streams sharing the template's key values: an SSP to create,
// or the existing instance to update.
type PlannedSSP struct {
	KeyValues map[string]string `json:"key-values"`
	KeyHash   string            `json:"key-hash"`
	Title     string            `json:"title"`
	// Exists is true when an instance records this group. Its SSPID is nil when the SSP
	// was deleted: discovery then leaves the group alone.
	Exists bool       `json:"exists"`
	SSPID  *uuid.UUID `json:"ssp-id"`
	// Streams is the number of streams in the group.
	Streams int `json:"streams"`
	// Learn are the scope entries discovery would add: not in the scope yet and not
	// excepted.
	Learn              [][]relational.ScopeLabel `json:"learn"`
	SkippedByException [][]relational.ScopeLabel `json:"skipped-by-exception"`
	// InvalidEntries are learned entries a scope cannot hold (a value over the length
	// limit, say); discovery skips them.
	InvalidEntries []InvalidEntry `json:"invalid-entries"`
}

// GroupStreams is a group under the template's min-streams.
type GroupStreams struct {
	KeyValues map[string]string `json:"key-values"`
	Streams   int               `json:"streams"`
}

// InvalidGroup is a group whose key values cannot be stored as scope labels, or whose
// title does not render; no SSP is created for it.
type InvalidGroup struct {
	KeyValues map[string]string `json:"key-values"`
	Reason    string            `json:"reason"`
}

// InvalidEntry is a learned entry a scope cannot hold.
type InvalidEntry struct {
	Labels []relational.ScopeLabel `json:"labels"`
	Reason string                  `json:"reason"`
}

// HashesBySSP holds, per SSP, the MatchHashes of its scope entries or exceptions.
type HashesBySSP map[uuid.UUID]map[string]struct{}

// group is the streams sharing one combination of key values.
type group struct {
	keyValues map[string]string
	streams   []map[string][]string
}

// BuildPlan applies tmpl to the latest evidence streams. It is pure: instances are the
// template's instances, and entries and exceptions the scope hashes of their SSPs.
// Evidence label keys and values arrive lower-cased (LoadLatestEvidenceStreams), so key
// values, titles and learned entries are lower-cased too.
func BuildPlan(tmpl relational.SSPTemplate, streams []relational.LatestEvidenceStream, instances []relational.SSPTemplateInstance, entries, exceptions HashesBySSP) Plan {
	plan := Plan{SSPs: []PlannedSSP{}, BelowThreshold: []GroupStreams{}, Invalid: []InvalidGroup{}}
	var filter *labelfilter.Scope
	if tmpl.Filter != nil {
		filter = tmpl.Filter.Data().Scope
	}

	groups := map[string]*group{}
	for _, stream := range streams {
		// MatchLabels only fails on an unknown query operator, which Build rejects.
		if ok, err := labelfilter.MatchLabels(filter, stream.Labels); err != nil || !ok {
			continue
		}
		keyValues, ok := keyValuesOf(tmpl.Keys, stream.Labels)
		if !ok {
			plan.MissingKeys++
			continue
		}
		id := joinPairs(keyValues)
		if groups[id] == nil {
			groups[id] = &group{keyValues: keyValues}
		}
		groups[id].streams = append(groups[id].streams, stream.Labels)
	}

	byHash := make(map[string]relational.SSPTemplateInstance, len(instances))
	for _, instance := range instances {
		byHash[instance.KeyHash] = instance
	}
	ids := make([]string, 0, len(groups))
	for id := range groups {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		g := groups[id]
		if len(g.streams) < tmpl.MinStreams {
			plan.BelowThreshold = append(plan.BelowThreshold, GroupStreams{KeyValues: g.keyValues, Streams: len(g.streams)})
			continue
		}
		keyEntry, keyHash, err := sspscope.Normalize(toLabels(g.keyValues))
		if err != nil {
			plan.Invalid = append(plan.Invalid, InvalidGroup{KeyValues: g.keyValues, Reason: err.Error()})
			continue
		}
		title, err := templates.RenderTemplate(tmpl.TitleTemplate, g.keyValues)
		if err != nil {
			plan.Invalid = append(plan.Invalid, InvalidGroup{KeyValues: g.keyValues, Reason: err.Error()})
			continue
		}
		planned := PlannedSSP{
			KeyValues: g.keyValues, KeyHash: keyHash, Title: title, Streams: len(g.streams),
			Learn: [][]relational.ScopeLabel{}, SkippedByException: [][]relational.ScopeLabel{}, InvalidEntries: []InvalidEntry{},
		}
		var known, excepted map[string]struct{}
		if instance, ok := byHash[keyHash]; ok {
			planned.Exists, planned.SSPID = true, instance.SSPID
			if instance.SSPID != nil {
				known, excepted = entries[*instance.SSPID], exceptions[*instance.SSPID]
			}
		}
		if !planned.Exists || planned.SSPID != nil {
			candidates := append([][]relational.ScopeLabel{keyEntry}, scopeLabelEntries(tmpl.ScopeLabels.Data(), g.streams, tmpl.MinStreams)...)
			classify(&planned, candidates, known, excepted)
		}
		plan.SSPs = append(plan.SSPs, planned)
	}
	return plan
}

// classify sorts candidate entries into planned's Learn, SkippedByException and
// InvalidEntries, dropping those already in the scope and repeats (a scope-label group
// holding a key repeats the key entry).
func classify(planned *PlannedSSP, candidates [][]relational.ScopeLabel, known, excepted map[string]struct{}) {
	seen := map[string]struct{}{}
	for _, labels := range candidates {
		normalized, hash, err := sspscope.Normalize(labels)
		if err == nil && contains(seen, hash) {
			continue
		}
		seen[hash] = struct{}{}
		switch {
		case err != nil:
			planned.InvalidEntries = append(planned.InvalidEntries, InvalidEntry{Labels: labels, Reason: err.Error()})
		case contains(excepted, hash):
			planned.SkippedByException = append(planned.SkippedByException, normalized)
		case !contains(known, hash):
			planned.Learn = append(planned.Learn, normalized)
		}
	}
}

// scopeLabelEntries returns, for each scope-label group, every combination of the
// group's label values seen on at least minStreams streams, in a stable order.
func scopeLabelEntries(scopeLabels [][]string, streams []map[string][]string, minStreams int) [][]relational.ScopeLabel {
	var out [][]relational.ScopeLabel
	for _, labelGroup := range scopeLabels {
		counts := map[string]int{}
		combos := map[string][]relational.ScopeLabel{}
		for _, labels := range streams {
			// A stream counts once per combination, however many times it carries it.
			seen := map[string]struct{}{}
			for _, combo := range combinations(labelGroup, labels) {
				id := joinLabels(combo)
				if _, dup := seen[id]; dup {
					continue
				}
				seen[id] = struct{}{}
				counts[id]++
				combos[id] = combo
			}
		}
		ids := make([]string, 0, len(combos))
		for id, count := range counts {
			if count >= minStreams {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		for _, id := range ids {
			out = append(out, combos[id])
		}
	}
	return out
}

// combinations returns every combination of one non-empty value per label of
// labelGroup that labels carries (a label can have several values); none when a label is
// missing.
func combinations(labelGroup []string, labels map[string][]string) [][]relational.ScopeLabel {
	combos := [][]relational.ScopeLabel{{}}
	for _, key := range labelGroup {
		var next [][]relational.ScopeLabel
		for _, value := range labels[key] {
			if value == "" {
				continue
			}
			for _, combo := range combos {
				extended := append(append([]relational.ScopeLabel{}, combo...), relational.ScopeLabel{Key: key, Value: value})
				next = append(next, extended)
			}
		}
		combos = next
	}
	return combos
}

// keyValuesOf returns the stream's value for each key; ok is false when a key is
// missing, empty or carries several values.
func keyValuesOf(keys []string, labels map[string][]string) (map[string]string, bool) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		values := labels[key]
		if len(values) != 1 || values[0] == "" {
			return nil, false
		}
		out[key] = values[0]
	}
	return out, true
}

func toLabels(keyValues map[string]string) []relational.ScopeLabel {
	labels := make([]relational.ScopeLabel, 0, len(keyValues))
	for key, value := range keyValues {
		labels = append(labels, relational.ScopeLabel{Key: key, Value: value})
	}
	return labels
}

// joinPairs is a stable identity for key values (keys sorted).
func joinPairs(keyValues map[string]string) string {
	labels := toLabels(keyValues)
	sort.Slice(labels, func(i, j int) bool { return labels[i].Key < labels[j].Key })
	return joinLabels(labels)
}

// joinLabels joins labels as key=value lines: the grouping identity of a combination.
func joinLabels(labels []relational.ScopeLabel) string {
	parts := make([]string, len(labels))
	for i, label := range labels {
		parts[i] = label.Key + "=" + label.Value
	}
	return strings.Join(parts, "\n")
}

func contains(set map[string]struct{}, key string) bool {
	_, ok := set[key]
	return ok
}
