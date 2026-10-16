package labelfilter

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cond(label, op, value string) Scope {
	return Scope{Condition: &Condition{Label: label, Operator: op, Value: value}}
}

func query(op string, scopes ...Scope) Scope {
	return Scope{Query: &Query{Operator: op, Scopes: scopes}}
}

func TestIndexMatch(t *testing.T) {
	t.Parallel()

	labels := []map[string][]string{
		{"env": {"prod"}, "app": {"web"}},
		{"env": {"staging"}, "app": {"web", "api"}},
		{"app": {"api"}},
		{},
	}
	idx := NewIndex(labels)

	tests := []struct {
		name  string
		scope *Scope
		want  []int
	}{
		{"nil scope matches all", nil, []int{0, 1, 2, 3}},
		{"empty scope matches all", &Scope{}, []int{0, 1, 2, 3}},
		{"equals", new(cond("env", "=", "prod")), []int{0}},
		{"equals is case and space insensitive", new(cond(" ENV ", "=", "Prod ")), []int{0}},
		{"equals any of several values", new(cond("app", "=", "api")), []int{1, 2}},
		{"equals unknown label", new(cond("team", "=", "x")), []int{}},
		{"not equals includes missing label", new(cond("env", "!=", "prod")), []int{1, 2, 3}},
		{"and", new(query("AND", cond("app", "=", "web"), cond("env", "!=", "prod"))), []int{1}},
		{"or", new(query("or", cond("env", "=", "prod"), cond("app", "=", "api"))), []int{0, 1, 2}},
		{"empty and matches all", new(query("and")), []int{0, 1, 2, 3}},
		{"empty or matches all", new(query("or")), []int{0, 1, 2, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := idx.Match(tt.scope)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIndexMatchUnknownOperator(t *testing.T) {
	t.Parallel()

	idx := NewIndex([]map[string][]string{{"env": {"prod"}}})
	_, err := idx.Match(new(query("xor", cond("env", "=", "prod"))))
	assert.ErrorContains(t, err, "unrecognised query operator: xor")

	// Unlike MatchLabels, a bad operator behind a short-circuit still errors.
	_, err = idx.Match(new(query("and", cond("env", "=", "dev"), query("xor"))))
	assert.Error(t, err)
}

func TestIndexMatchEmpty(t *testing.T) {
	t.Parallel()

	got, err := NewIndex(nil).Match(new(cond("env", "!=", "prod")))
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestIndexAgreesWithMatchLabels checks random scopes over random label sets: the
// index must return exactly the sets MatchLabels accepts. 130 sets cover a partial
// last bitset word, which complement must mask.
func TestIndexAgreesWithMatchLabels(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(1))
	keys := []string{"env", "app", "team"}
	values := []string{"a", "b", "c"}

	labels := make([]map[string][]string, 130)
	for i := range labels {
		set := map[string][]string{}
		for _, k := range keys {
			for _, v := range values {
				if rng.Intn(3) == 0 {
					set[k] = append(set[k], v)
				}
			}
		}
		labels[i] = set
	}
	idx := NewIndex(labels)

	var randomScope func(depth int) Scope
	randomScope = func(depth int) Scope {
		if depth == 0 || rng.Intn(3) == 0 {
			if rng.Intn(20) == 0 {
				return Scope{}
			}
			op := []string{"=", "!="}[rng.Intn(2)]
			return cond(keys[rng.Intn(len(keys))], op, values[rng.Intn(len(values))])
		}
		subs := make([]Scope, rng.Intn(4))
		for i := range subs {
			subs[i] = randomScope(depth - 1)
		}
		return query([]string{"and", "AND", "or", "Or"}[rng.Intn(4)], subs...)
	}

	for n := 0; n < 2000; n++ {
		scope := randomScope(3)
		want := []int{}
		for i, set := range labels {
			if mustMatch(t, &scope, set) {
				want = append(want, i)
			}
		}
		got, err := idx.Match(&scope)
		require.NoError(t, err)
		require.Equal(t, want, got, "scope %d: %s", n, fmt.Sprintf("%+v", scope))
	}
}
