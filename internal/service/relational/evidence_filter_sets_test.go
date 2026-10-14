package relational

import (
	"testing"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/stretchr/testify/require"
)

func labelCondition(label, operator, value string) *labelfilter.Scope {
	return &labelfilter.Scope{Condition: &labelfilter.Condition{Label: label, Operator: operator, Value: value}}
}

func filterSetsTestStreams() []LatestEvidenceStream {
	return []LatestEvidenceStream{
		{State: "satisfied", Labels: map[string][]string{"svc": {"ec2"}, "env": {"prod"}}},
		{State: "not-satisfied", Labels: map[string][]string{"svc": {"ec2"}, "env": {"dev"}}},
		{State: "satisfied", Labels: map[string][]string{"svc": {"s3"}, "env": {"prod"}}},
		{State: "", Labels: map[string][]string{}},
	}
}

func TestCountStatusesForFilterSets(t *testing.T) {
	streams := filterSetsTestStreams()
	sets := map[string][]labelfilter.Filter{
		"none":     {},
		"ec2":      {{Scope: labelCondition("svc", "=", "ec2")}},
		"ec2-case": {{Scope: labelCondition("SVC", "=", "EC2")}},
		"not-ec2":  {{Scope: labelCondition("svc", "!=", "ec2")}},
		// A stream matched by two filters of a set is counted once.
		"ec2-or-prod": {{Scope: labelCondition("svc", "=", "ec2")}, {Scope: labelCondition("env", "=", "prod")}},
		"ec2-and-prod": {{Scope: &labelfilter.Scope{Query: &labelfilter.Query{
			Operator: "and",
			Scopes:   []labelfilter.Scope{*labelCondition("svc", "=", "ec2"), *labelCondition("env", "=", "prod")},
		}}}},
		// Unscoped filters match everything only when no filter of the set is scoped.
		"unscoped":       {{}},
		"unscoped-mixed": {{}, {Scope: labelCondition("svc", "=", "s3")}},
		"no-match":       {{Scope: labelCondition("svc", "=", "rds")}},
	}

	got, err := CountStatusesForFilterSets(streams, sets)
	require.NoError(t, err)

	require.Equal(t, map[string][]StatusCount{
		"none":           {},
		"ec2":            {{Count: 1, Status: "not-satisfied"}, {Count: 1, Status: "satisfied"}},
		"ec2-case":       {{Count: 1, Status: "not-satisfied"}, {Count: 1, Status: "satisfied"}},
		"not-ec2":        {{Count: 1, Status: ""}, {Count: 1, Status: "satisfied"}},
		"ec2-or-prod":    {{Count: 1, Status: "not-satisfied"}, {Count: 2, Status: "satisfied"}},
		"ec2-and-prod":   {{Count: 1, Status: "satisfied"}},
		"unscoped":       {{Count: 1, Status: ""}, {Count: 1, Status: "not-satisfied"}, {Count: 2, Status: "satisfied"}},
		"unscoped-mixed": {{Count: 1, Status: "satisfied"}},
		"no-match":       {},
	}, got)
}

func TestCountStatusesForFilterSetsSharedFilterAcrossSets(t *testing.T) {
	streams := filterSetsTestStreams()
	shared := labelfilter.Filter{Scope: labelCondition("svc", "=", "ec2")}
	sets := map[int][]labelfilter.Filter{}
	for i := 0; i < 50; i++ {
		sets[i] = []labelfilter.Filter{shared}
	}

	got, err := CountStatusesForFilterSets(streams, sets)
	require.NoError(t, err)
	require.Len(t, got, 50)
	for i := 0; i < 50; i++ {
		require.Equal(t, []StatusCount{{Count: 1, Status: "not-satisfied"}, {Count: 1, Status: "satisfied"}}, got[i])
	}
}

func TestCountStatusesForFilterSetsNoStreams(t *testing.T) {
	got, err := CountStatusesForFilterSets(nil, map[string][]labelfilter.Filter{
		"ec2":      {{Scope: labelCondition("svc", "=", "ec2")}},
		"unscoped": {{}},
	})
	require.NoError(t, err)
	require.Equal(t, map[string][]StatusCount{"ec2": {}, "unscoped": {}}, got)
}

func TestCountStatusesForFilterSetsInvalidOperator(t *testing.T) {
	_, err := CountStatusesForFilterSets(filterSetsTestStreams(), map[string][]labelfilter.Filter{
		"bad": {{Scope: &labelfilter.Scope{Query: &labelfilter.Query{Operator: "xor", Scopes: []labelfilter.Scope{*labelCondition("svc", "=", "ec2")}}}}},
	})
	require.Error(t, err)
}
