package ssptemplates

import (
	"strings"
	"testing"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/service/relational/sspscope"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stream builds a stream's labels from key=value pairs; a key may repeat.
func stream(pairs ...string) relational.LatestEvidenceStream {
	labels := map[string][]string{}
	for _, pair := range pairs {
		key, value, _ := strings.Cut(pair, "=")
		labels[key] = append(labels[key], value)
	}
	return relational.LatestEvidenceStream{UUID: uuid.New(), Labels: labels}
}

func planTemplate(t *testing.T, edit func(*Input)) relational.SSPTemplate {
	in := Input{Name: "Apps", Keys: []string{"_app"}, TitleTemplate: "SSP for {{ ._app }}"}
	if edit != nil {
		edit(&in)
	}
	tmpl, err := Build(in)
	require.NoError(t, err)
	return tmpl
}

func labels(pairs ...string) []relational.ScopeLabel {
	out := []relational.ScopeLabel{}
	for _, pair := range pairs {
		key, value, _ := strings.Cut(pair, "=")
		out = append(out, relational.ScopeLabel{Key: key, Value: value})
	}
	return out
}

func hashOf(t *testing.T, pairs ...string) string {
	_, hash, err := sspscope.Normalize(labels(pairs...))
	require.NoError(t, err)
	return hash
}

func TestPlanGroupsByKeyValues(t *testing.T) {
	tmpl := planTemplate(t, func(in *Input) { in.Keys = []string{"_app", "env"} })
	plan := BuildPlan(tmpl, []relational.LatestEvidenceStream{
		stream("_app=payments", "env=prod"),
		stream("_app=payments", "env=prod", "other=x"),
		stream("_app=payments", "env=dev"),
		stream("_app=payments"),                        // missing env
		stream("_app=payments", "env="),                // empty env
		stream("_app=payments", "env=prod", "env=dev"), // multi-valued env
	}, nil, nil, nil)

	assert.Equal(t, 3, plan.MissingKeys)
	require.Len(t, plan.SSPs, 2)
	dev, prod := plan.SSPs[0], plan.SSPs[1]
	assert.Equal(t, map[string]string{"_app": "payments", "env": "dev"}, dev.KeyValues)
	assert.Equal(t, 1, dev.Streams)
	assert.Equal(t, map[string]string{"_app": "payments", "env": "prod"}, prod.KeyValues)
	assert.Equal(t, 2, prod.Streams)
	assert.Equal(t, "SSP for payments", prod.Title)
	assert.False(t, prod.Exists)
	assert.Equal(t, hashOf(t, "_app=payments", "env=prod"), prod.KeyHash)
	assert.Equal(t, [][]relational.ScopeLabel{labels("_app=payments", "env=prod")}, prod.Learn, "the key entry ANDs every key")
}

func TestPlanFilterAndMinStreams(t *testing.T) {
	tmpl := planTemplate(t, func(in *Input) {
		two := 2
		in.MinStreams = &two
		in.Filter = &labelfilter.Filter{Scope: &labelfilter.Scope{Condition: &labelfilter.Condition{Label: "auto", Operator: "=", Value: "true"}}}
	})
	plan := BuildPlan(tmpl, []relational.LatestEvidenceStream{
		stream("_app=payments", "auto=true"),
		stream("_app=payments", "auto=true"),
		stream("_app=test", "auto=true"),
		stream("_app=ignored"), // does not match the filter
		stream("auto=true"),    // matches, lacks the key
	}, nil, nil, nil)

	assert.Equal(t, 1, plan.MissingKeys, "only streams matching the filter count")
	require.Len(t, plan.SSPs, 1)
	assert.Equal(t, "payments", plan.SSPs[0].KeyValues["_app"])
	assert.Equal(t, []GroupStreams{{KeyValues: map[string]string{"_app": "test"}, Streams: 1}}, plan.BelowThreshold)
}

func TestPlanScopeLabelGroups(t *testing.T) {
	tmpl := planTemplate(t, func(in *Input) {
		two := 2
		in.MinStreams = &two
		in.ScopeLabels = [][]string{{"repo"}, {"region", "aws_account"}, {"_app"}}
	})
	plan := BuildPlan(tmpl, []relational.LatestEvidenceStream{
		stream("_app=p", "repo=a", "repo=b", "aws_account=1", "region=eu"),
		stream("_app=p", "repo=a", "aws_account=1", "region=eu", "region=us"),
		stream("_app=p", "repo=c", "aws_account=1"), // no region: no combination
	}, nil, nil, nil)

	require.Len(t, plan.SSPs, 1)
	assert.Equal(t, [][]relational.ScopeLabel{
		labels("_app=p"),
		labels("repo=a"), // b and c are on one stream each, under min-streams
		labels("aws_account=1", "region=eu"),
	}, plan.SSPs[0].Learn)
}

func TestPlanExistingInstances(t *testing.T) {
	tmpl := planTemplate(t, func(in *Input) { in.ScopeLabels = [][]string{{"repo"}} })
	sspID := uuid.New()
	streams := []relational.LatestEvidenceStream{
		stream("_app=kept", "repo=a"),
		stream("_app=kept", "repo=oops"),
		stream("_app=kept", "repo=new"),
		stream("_app=gone", "repo=a"),
	}
	instances := []relational.SSPTemplateInstance{
		{KeyHash: hashOf(t, "_app=kept"), SSPID: &sspID},
		{KeyHash: hashOf(t, "_app=gone")}, // its SSP was deleted
	}
	entries := HashesBySSP{sspID: {hashOf(t, "_app=kept"): {}, hashOf(t, "repo=a"): {}}}
	exceptions := HashesBySSP{sspID: {hashOf(t, "repo=OOPS"): {}}}
	plan := BuildPlan(tmpl, streams, instances, entries, exceptions)

	require.Len(t, plan.SSPs, 2)
	gone, kept := plan.SSPs[0], plan.SSPs[1]
	assert.True(t, gone.Exists)
	assert.Nil(t, gone.SSPID)
	assert.Empty(t, gone.Learn, "discovery leaves a deleted SSP alone")

	assert.True(t, kept.Exists)
	assert.Equal(t, &sspID, kept.SSPID)
	assert.Equal(t, [][]relational.ScopeLabel{labels("repo=new")}, kept.Learn)
	assert.Equal(t, [][]relational.ScopeLabel{labels("repo=oops")}, kept.SkippedByException)
}

func TestPlanInvalidValues(t *testing.T) {
	tmpl := planTemplate(t, func(in *Input) { in.ScopeLabels = [][]string{{"repo"}} })
	long := strings.Repeat("x", 300)
	plan := BuildPlan(tmpl, []relational.LatestEvidenceStream{
		stream("_app=" + long),
		stream("_app=ok", "repo="+long),
	}, nil, nil, nil)

	require.Len(t, plan.Invalid, 1)
	assert.Equal(t, long, plan.Invalid[0].KeyValues["_app"])
	assert.Contains(t, plan.Invalid[0].Reason, "at most 256 characters")
	require.Len(t, plan.SSPs, 1)
	assert.Equal(t, [][]relational.ScopeLabel{labels("_app=ok")}, plan.SSPs[0].Learn)
	require.Len(t, plan.SSPs[0].InvalidEntries, 1)
	assert.Equal(t, labels("repo="+long), plan.SSPs[0].InvalidEntries[0].Labels)
}

func TestPlanEmptyTitle(t *testing.T) {
	tmpl := planTemplate(t, func(in *Input) { in.TitleTemplate = "{{ .APP }}" }) // keys are lower-cased
	plan := BuildPlan(tmpl, []relational.LatestEvidenceStream{stream("_app=payments")}, nil, nil, nil)
	assert.Empty(t, plan.SSPs)
	require.Len(t, plan.Invalid, 1)
	assert.Equal(t, "title-template renders an empty title", plan.Invalid[0].Reason)
}
