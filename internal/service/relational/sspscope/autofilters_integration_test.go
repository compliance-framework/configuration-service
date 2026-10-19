//go:build integration

package sspscope_test

import (
	"context"
	"time"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/google/uuid"
)

func (s *ScopeServiceIntegrationSuite) createEvidence(labels map[string]string) {
	rows := make([]relational.Labels, 0, len(labels))
	for name, value := range labels {
		rows = append(rows, relational.Labels{Name: name, Value: value})
	}
	end := time.Now().Add(-time.Hour)
	s.Require().NoError(s.DB.Create(&relational.Evidence{
		UUID: uuid.New(), Title: "e", Start: end.Add(-time.Minute), End: end, Labels: rows,
	}).Error)
}

func (s *ScopeServiceIntegrationSuite) autoFilters() map[string]relational.Filter {
	var filters []relational.Filter
	s.Require().NoError(s.DB.Preload("Controls").Preload("Components").
		Where("ssp_id = ? AND origin = ?", s.sspID, relational.FilterOriginScopePolicy).Find(&filters).Error)
	out := map[string]relational.Filter{}
	for _, filter := range filters {
		out[filter.Name] = filter
	}
	return out
}

func (s *ScopeServiceIntegrationSuite) TestAutoFiltersOffByDefault() {
	s.createEvidence(map[string]string{"repo": "a", "_policy": "p1"})
	_, err := s.svc.AddEntry(s.sspID, labels("repo", "a"), nil)
	s.Require().NoError(err)
	s.Empty(s.autoFilters())

	created, err := s.svc.ReconcileAutoFilters(s.sspID)
	s.Require().NoError(err)
	s.Zero(created)
}

func (s *ScopeServiceIntegrationSuite) TestAutoFiltersCreatedOnceAndNotRecreated() {
	s.createEvidence(map[string]string{"repo": "a", "_policy": "p1"})
	s.createEvidence(map[string]string{"repo": "a", "_policy": "P1"})
	s.createEvidence(map[string]string{"repo": "b", "_policy": "p2"})
	s.createEvidence(map[string]string{"repo": "c", "_policy": "p3"})
	_, err := s.svc.AddEntry(s.sspID, labels("repo", "a"), nil)
	s.Require().NoError(err)
	_, err = s.svc.AddEntry(s.sspID, labels("repo", "b"), nil)
	s.Require().NoError(err)

	// Enabling creates one following filter per in-scope policy.
	s.Require().NoError(s.svc.SetAutoFilters(s.sspID, true))
	filters := s.autoFilters()
	s.Require().Len(filters, 2, "p3 is out of scope; p1 and P1 match the same evidence")
	p1, ok := filters["P1"]
	if !ok {
		p1 = filters["p1"]
	}
	s.Require().NotNil(p1.ID)
	s.Require().NotNil(p1.ScopeBase)
	s.Equal(relational.EvidencePolicyLabel, p1.ScopeBase.Data().Scope.Condition.Label)
	s.Empty(p1.Controls)
	s.Empty(p1.Components)
	s.Contains(filters, "p2")

	// The filter follows the scope: it matches policy p1 only inside the scope.
	match := func(filter relational.Filter, sets ...map[string][]string) []int {
		var stored relational.Filter
		s.Require().NoError(s.DB.First(&stored, "id = ?", filter.ID).Error)
		got, err := labelfilter.NewIndex(sets).Match(stored.Filter.Data().Scope)
		s.Require().NoError(err)
		return got
	}
	inA := map[string][]string{"repo": {"a"}, "_policy": {"p1"}}
	inC := map[string][]string{"repo": {"c"}, "_policy": {"p1"}}
	s.Equal([]int{0}, match(p1, inA, inC))

	// A scope change while on rewrites existing filters and adds the new policies.
	_, err = s.svc.AddEntry(s.sspID, labels("repo", "c"), nil)
	s.Require().NoError(err)
	s.Equal([]int{0, 1}, match(p1, inA, inC))
	s.Len(s.autoFilters(), 3)

	// A deleted automatic filter is not recreated.
	s.Require().NoError(s.DB.Delete(&relational.Filter{}, "id = ?", p1.ID).Error)
	var row relational.SSPScopePolicyFilter
	s.Require().NoError(s.DB.Where("ssp_id = ? AND filter_id IS NULL", s.sspID).First(&row).Error)
	created, err := s.svc.ReconcileAutoFilters(s.sspID)
	s.Require().NoError(err)
	s.Zero(created)
	s.Len(s.autoFilters(), 2)

	// Policies never disappear: the filter of a policy no longer in scope stays.
	_, err = s.svc.AddException(s.sspID, labels("repo", "b"), nil)
	s.Require().NoError(err)
	s.Contains(s.autoFilters(), "p2")
}

func (s *ScopeServiceIntegrationSuite) TestReconcileAllPicksUpNewPolicies() {
	other := s.createSSP()
	for _, sspID := range []uuid.UUID{s.sspID, other} {
		_, err := s.svc.AddEntry(sspID, labels("repo", "a"), nil)
		s.Require().NoError(err)
	}
	s.Require().NoError(s.svc.SetAutoFilters(s.sspID, true))
	s.Empty(s.autoFilters(), "no policies yet")

	// New evidence alone does not change the scope; the periodic reconcile picks it up,
	// only for SSPs with automatic filters on.
	s.createEvidence(map[string]string{"repo": "a", "_policy": "late"})
	created, err := s.svc.ReconcileAllAutoFilters(context.Background())
	s.Require().NoError(err)
	s.Equal(1, created)
	s.Contains(s.autoFilters(), "late")

	var otherCount int64
	s.Require().NoError(s.DB.Model(&relational.Filter{}).Where("ssp_id = ?", other).Count(&otherCount).Error)
	s.Zero(otherCount)
}
