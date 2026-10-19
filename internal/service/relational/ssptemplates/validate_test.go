package ssptemplates

import (
	"strings"
	"testing"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validInput() Input {
	return Input{Name: "Apps", Keys: []string{"_app"}, TitleTemplate: "SSP for {{ ._app }}"}
}

func TestBuildNormalizes(t *testing.T) {
	profile := uuid.New()
	disabled := false
	in := Input{
		Name:        "  Apps ",
		Enabled:     &disabled,
		Keys:        []string{" Region", "_APP"},
		ScopeLabels: [][]string{{"Repo"}, {"region", "AWS_Account"}},
		Filter: &labelfilter.Filter{Scope: &labelfilter.Scope{Condition: &labelfilter.Condition{
			Label: "auto_create_ssp", Operator: "=", Value: "true",
		}}},
		TitleTemplate: `SSP for {{ index . "_app" }}`,
		Props:         []relational.SSPTemplateProp{{Name: " application ", Value: "{{ ._app }}"}},
		ProfileIDs:    []uuid.UUID{profile, profile},
	}
	tmpl, err := Build(in)
	require.NoError(t, err)
	assert.Equal(t, "Apps", tmpl.Name)
	assert.False(t, tmpl.Enabled)
	assert.Equal(t, []string{"_app", "region"}, []string(tmpl.Keys))
	assert.Equal(t, [][]string{{"repo"}, {"aws_account", "region"}}, tmpl.ScopeLabels.Data())
	assert.Equal(t, 1, tmpl.MinStreams)
	require.NotNil(t, tmpl.Filter)
	assert.Equal(t, "auto_create_ssp", tmpl.Filter.Data().Scope.Label)
	assert.Equal(t, []relational.SSPTemplateProp{{Name: "application", Value: "{{ ._app }}"}}, []relational.SSPTemplateProp(tmpl.Props))
	assert.Equal(t, []uuid.UUID{profile}, []uuid.UUID(tmpl.ProfileIDs))
}

func TestBuildDefaults(t *testing.T) {
	in := validInput()
	in.Filter = &labelfilter.Filter{Scope: &labelfilter.Scope{}}
	tmpl, err := Build(in)
	require.NoError(t, err)
	assert.True(t, tmpl.Enabled)
	assert.Equal(t, 1, tmpl.MinStreams)
	assert.Nil(t, tmpl.Filter, "an empty filter matches everything, so it is not stored")
	assert.Empty(t, tmpl.ScopeLabels.Data())
}

func TestBuildRejects(t *testing.T) {
	cond := func(label, op string) labelfilter.Scope {
		return labelfilter.Scope{Condition: &labelfilter.Condition{Label: label, Operator: op, Value: "v"}}
	}
	query := func(op string, scopes ...labelfilter.Scope) *labelfilter.Filter {
		return &labelfilter.Filter{Scope: &labelfilter.Scope{Query: &labelfilter.Query{Operator: op, Scopes: scopes}}}
	}
	broken := "{{ .x"
	cases := map[string]struct {
		edit func(*Input)
		want string
	}{
		"empty name":           {func(in *Input) { in.Name = " " }, "name is required"},
		"no keys":              {func(in *Input) { in.Keys = nil }, "keys: between 1 and 5 labels"},
		"too many keys":        {func(in *Input) { in.Keys = []string{"a", "b", "c", "d", "e", "f"} }, "keys: between 1 and 5 labels"},
		"empty key":            {func(in *Input) { in.Keys = []string{"_app", " "} }, "keys: label keys and values must not be empty"},
		"duplicate key":        {func(in *Input) { in.Keys = []string{"_app", "_APP"} }, "keys: duplicate key"},
		"key with =":           {func(in *Input) { in.Keys = []string{"a=b"} }, "must not contain '='"},
		"long key":             {func(in *Input) { in.Keys = []string{strings.Repeat("k", 257)} }, "at most 256 characters"},
		"too many groups":      {func(in *Input) { in.ScopeLabels = make([][]string, 11) }, "at most 10 scope-label groups"},
		"empty group":          {func(in *Input) { in.ScopeLabels = [][]string{{}} }, "scope-labels[0]: between 1 and 5 labels"},
		"too many group items": {func(in *Input) { in.ScopeLabels = [][]string{{"a", "b", "c", "d", "e", "f"}} }, "scope-labels[0]: between 1 and 5 labels"},
		"empty label":          {func(in *Input) { in.ScopeLabels = [][]string{{"repo", ""}} }, "scope-labels[0]: label keys"},
		"repeated group":       {func(in *Input) { in.ScopeLabels = [][]string{{"a", "b"}, {"B", "a"}} }, "scope-labels[1] repeats"},
		"min-streams 0":        {func(in *Input) { in.MinStreams = new(int) }, "min-streams must be at least 1"},
		"no title":             {func(in *Input) { in.TitleTemplate = "  " }, "title-template is required"},
		"bad title":            {func(in *Input) { in.TitleTemplate = broken }, "title-template:"},
		"bad description":      {func(in *Input) { in.DescriptionTemplate = &broken }, "description-template:"},
		"prop without name":    {func(in *Input) { in.Props = []relational.SSPTemplateProp{{Name: "", Value: "x"}} }, "props[0]: name is required"},
		"bad prop value":       {func(in *Input) { in.Props = []relational.SSPTemplateProp{{Name: "n", Value: broken}} }, "props[0].value:"},
		"bad operator":         {func(in *Input) { in.Filter = query("AND", cond("a", "~")) }, "unsupported condition operator"},
		"no label":             {func(in *Input) { in.Filter = query("AND", cond("", "=")) }, "a condition needs a label"},
		"bad query":            {func(in *Input) { in.Filter = query("XOR", cond("a", "=")) }, "unsupported query operator"},
		"empty query":          {func(in *Input) { in.Filter = query("or") }, "at least one scope"},
		"empty sub-scope":      {func(in *Input) { in.Filter = query("and", labelfilter.Scope{}) }, "must not be empty"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			in := validInput()
			tc.edit(&in)
			_, err := Build(in)
			require.ErrorIs(t, err, ErrInvalidTemplate)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}
