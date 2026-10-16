package relational

import (
	"fmt"
	"math/rand"
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

// BenchmarkCountStatusesForFilterSets matches 1,000 distinct filters, shared across
// 1,000 sets as controls share filters, against 10k synthetic latest streams.
func BenchmarkCountStatusesForFilterSets(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	value := func(prefix string, n int) string { return fmt.Sprintf("%s-%d", prefix, rng.Intn(n)) }
	states := []string{"satisfied", "not-satisfied", ""}

	streams := make([]LatestEvidenceStream, 10_000)
	for i := range streams {
		streams[i] = LatestEvidenceStream{State: states[rng.Intn(len(states))], Labels: map[string][]string{
			"svc": {value("svc", 50)}, "env": {value("env", 4)}, "team": {value("team", 20)}, "region": {value("region", 10)},
		}}
	}

	filters := make([]labelfilter.Filter, 1_000)
	for i := range filters {
		svc, env := labelCondition("svc", "=", value("svc", 50)), labelCondition("env", "!=", value("env", 4))
		switch i % 3 {
		case 0:
			filters[i] = labelfilter.Filter{Scope: svc}
		case 1:
			filters[i] = labelfilter.Filter{Scope: &labelfilter.Scope{Query: &labelfilter.Query{Operator: "and", Scopes: []labelfilter.Scope{*svc, *env}}}}
		default:
			team := labelCondition("team", "=", value("team", 20))
			filters[i] = labelfilter.Filter{Scope: &labelfilter.Scope{Query: &labelfilter.Query{Operator: "or", Scopes: []labelfilter.Scope{*team, *svc}}}}
		}
	}
	sets := make(map[int][]labelfilter.Filter, len(filters))
	for i := range filters {
		sets[i] = []labelfilter.Filter{filters[i], filters[rng.Intn(len(filters))]}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := CountStatusesForFilterSets(streams, sets); err != nil {
			b.Fatal(err)
		}
	}
}
