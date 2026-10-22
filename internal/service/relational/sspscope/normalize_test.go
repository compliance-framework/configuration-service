package sspscope

import (
	"strings"
	"testing"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/stretchr/testify/suite"
)

type NormalizeSuite struct {
	suite.Suite
}

func TestNormalize(t *testing.T) {
	suite.Run(t, new(NormalizeSuite))
}

func labels(pairs ...string) []relational.ScopeLabel {
	out := make([]relational.ScopeLabel, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, relational.ScopeLabel{Key: pairs[i], Value: pairs[i+1]})
	}
	return out
}

func (s *NormalizeSuite) TestTrimsLowerCasesKeysAndSorts() {
	got, _, err := Normalize(labels(" Region ", " eu-West-1 ", "AWS_Account", "111"))
	s.Require().NoError(err)
	s.Equal(labels("aws_account", "111", "region", "eu-West-1"), got)
}

func (s *NormalizeSuite) TestRejectsInvalidInput() {
	long := strings.Repeat("é", MaxLabelPartLength+1)
	tooMany := []relational.ScopeLabel{}
	for i := 0; i <= MaxLabelsPerEntry; i++ {
		tooMany = append(tooMany, relational.ScopeLabel{Key: "k" + strings.Repeat("x", i), Value: "v"})
	}
	cases := map[string][]relational.ScopeLabel{
		"no labels":          nil,
		"empty key":          labels(" ", "a"),
		"empty value":        labels("repo", "  "),
		"key-only":           labels("foo", ""),
		"duplicate key":      labels("repo", "a", "REPO", "b"),
		"too many labels":    tooMany,
		"key too long":       labels(long, "a"),
		"value too long":     labels("repo", long),
		"key with =":         labels("a=b", "c"),
		"value with newline": labels("a", "b\nc=d"),
		"key with tab":       labels("a\tb", "c"),
	}
	for name, input := range cases {
		_, _, err := Normalize(input)
		s.ErrorIs(err, ErrInvalidLabels, name)
	}
}

func (s *NormalizeSuite) TestAcceptsLimits() {
	exact := strings.Repeat("é", MaxLabelPartLength)
	_, _, err := Normalize(labels(exact, exact))
	s.NoError(err)
}

func (s *NormalizeSuite) TestMatchHashCollidesForSameEvidence() {
	_, a, err := Normalize(labels("repo", "API", "Env", "prod"))
	s.Require().NoError(err)
	_, b, err := Normalize(labels(" env", "PROD ", "REPO", "api"))
	s.Require().NoError(err)
	s.Equal(a, b, "order, case and whitespace must not change the hash")

	_, c, err := Normalize(labels("repo", "api"))
	s.Require().NoError(err)
	s.NotEqual(a, c)
	s.Len(a, 64)
}

func entry(pairs ...string) relational.SSPScopeEntry {
	normalized, hash, err := Normalize(labels(pairs...))
	if err != nil {
		panic(err)
	}
	return relational.SSPScopeEntry{Labels: normalized, MatchHash: hash}
}

func eq(label, value string) labelfilter.Scope {
	return labelfilter.Scope{Condition: &labelfilter.Condition{Label: label, Operator: "=", Value: value}}
}

func (s *NormalizeSuite) TestToFilterEmpty() {
	f, ok := ToFilter(nil)
	s.False(ok)
	s.Nil(f.Scope)
}

func (s *NormalizeSuite) TestToFilterSingleLabelCollapses() {
	f, ok := ToFilter([]relational.SSPScopeEntry{entry("repo", "a")})
	s.True(ok)
	want := eq("repo", "a")
	s.Equal(&want, f.Scope)
}

func (s *NormalizeSuite) TestToFilterShape() {
	a := entry("repo", "a")
	multi := entry("region", "eu-west-1", "aws_account", "111")
	f, ok := ToFilter([]relational.SSPScopeEntry{a, multi})
	s.Require().True(ok)
	s.Require().NotNil(f.Scope.Query)
	s.Equal("OR", f.Scope.Query.Operator)
	s.Require().Len(f.Scope.Scopes, 2)

	and := labelfilter.Scope{Query: &labelfilter.Query{Operator: "AND", Scopes: []labelfilter.Scope{
		eq("aws_account", "111"), eq("region", "eu-west-1"),
	}}}
	s.ElementsMatch([]labelfilter.Scope{eq("repo", "a"), and}, f.Scope.Scopes)

	// Deterministic: the order of the input entries does not change the filter.
	g, _ := ToFilter([]relational.SSPScopeEntry{multi, a})
	s.Equal(f, g)
}

func (s *NormalizeSuite) TestToFilterMatchesEvidence() {
	f, _ := ToFilter([]relational.SSPScopeEntry{entry("repo", "a"), entry("aws_account", "111", "region", "eu-west-1")})
	idx := labelfilter.NewIndex([]map[string][]string{
		{"repo": {"a"}},
		{"repo": {"b"}},
		{"aws_account": {"111"}, "region": {"eu-west-1"}},
		{"aws_account": {"111"}, "region": {"us-east-1"}},
	})
	got, err := idx.Match(f.Scope)
	s.Require().NoError(err)
	s.Equal([]int{0, 2}, got)
}
