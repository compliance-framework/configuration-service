//go:build integration

package worker

import (
	"context"
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/service/relational/sspscope"
	"github.com/compliance-framework/api/internal/tests"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/stretchr/testify/suite"
)

type SSPScopeAutoFiltersWorkerIntegrationSuite struct {
	tests.IntegrationTestSuite
}

func TestSSPScopeAutoFiltersWorkerIntegrationSuite(t *testing.T) {
	suite.Run(t, new(SSPScopeAutoFiltersWorkerIntegrationSuite))
}

func (s *SSPScopeAutoFiltersWorkerIntegrationSuite) TestWorkCreatesMissingAutoFilters() {
	s.Require().NoError(s.Migrator.Refresh())
	sspID := uuid.New()
	s.Require().NoError(s.DB.Create(&relational.SystemSecurityPlan{UUIDModel: relational.UUIDModel{ID: &sspID}}).Error)
	scope := sspscope.NewService(s.DB)
	_, err := scope.AddEntry(sspID, []relational.ScopeLabel{{Key: "repo", Value: "a"}}, nil)
	s.Require().NoError(err)
	s.Require().NoError(scope.SetAutoFilters(sspID, true))

	end := time.Now().Add(-time.Hour)
	s.Require().NoError(s.DB.Create(&relational.Evidence{
		UUID: uuid.New(), Title: "e", Start: end.Add(-time.Minute), End: end,
		Labels: []relational.Labels{{Name: "repo", Value: "a"}, {Name: "_policy", Value: "p1"}},
	}).Error)

	worker := NewSSPScopeAutoFiltersWorker(s.DB, nil)
	s.Require().NoError(worker.Work(context.Background(), &river.Job[SSPScopeAutoFiltersArgs]{}))

	var filters []relational.Filter
	s.Require().NoError(s.DB.Where("ssp_id = ? AND origin = ?", sspID, relational.FilterOriginScopePolicy).Find(&filters).Error)
	s.Require().Len(filters, 1)
	s.Equal("p1", filters[0].Name)
}
