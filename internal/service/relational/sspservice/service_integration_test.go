//go:build integration

package sspservice_test

import (
	"errors"
	"testing"

	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/service/relational/sspservice"
	"github.com/compliance-framework/api/internal/tests"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type SSPServiceIntegrationSuite struct {
	tests.IntegrationTestSuite
	controls  map[uuid.UUID][]string
	svc       *sspservice.Service
	profileID uuid.UUID
}

func TestSSPServiceIntegration(t *testing.T) {
	suite.Run(t, new(SSPServiceIntegrationSuite))
}

func (s *SSPServiceIntegrationSuite) SetupTest() {
	s.Require().NoError(s.Migrator.Refresh())
	s.controls = map[uuid.UUID][]string{}
	s.svc = sspservice.NewService(func(profileID uuid.UUID) ([]string, error) {
		return s.controls[profileID], nil
	})
	s.profileID = uuid.New()
	s.Require().NoError(s.DB.Create(&relational.Profile{UUIDModel: relational.UUIDModel{ID: &s.profileID}}).Error)
}

func (s *SSPServiceIntegrationSuite) createSSP(db *gorm.DB) uuid.UUID {
	ssp, err := s.svc.Create(db, oscalTypes_1_1_3.SystemSecurityPlan{
		UUID:     uuid.NewString(),
		Metadata: oscalTypes_1_1_3.Metadata{Title: "SSP", Version: "1.0"},
	})
	s.Require().NoError(err)
	s.NotNil(ssp.Metadata.LastModified)
	s.NotEmpty(ssp.Metadata.OscalVersion)
	return *ssp.ID
}

func (s *SSPServiceIntegrationSuite) controlIDs(sspID uuid.UUID) []string {
	var ids []string
	s.Require().NoError(s.DB.Model(&relational.ImplementedRequirement{}).
		Joins("JOIN control_implementations ON control_implementations.id = implemented_requirements.control_implementation_id").
		Where("control_implementations.system_security_plan_id = ?", sspID).
		Order("control_id").Pluck("control_id", &ids).Error)
	return ids
}

func (s *SSPServiceIntegrationSuite) TestAttachProfileCreatesMissingRequirements() {
	sspID := s.createSSP(s.DB)
	other := uuid.New()
	s.Require().NoError(s.DB.Create(&relational.Profile{UUIDModel: relational.UUIDModel{ID: &other}}).Error)
	s.controls[other] = []string{"AC-1"}
	s.controls[s.profileID] = []string{"ac-1", "AC-2"}

	s.Require().NoError(s.svc.AttachProfile(s.DB, sspID, other))
	s.Require().NoError(s.svc.AttachProfile(s.DB, sspID, s.profileID))
	s.Equal([]string{"AC-1", "AC-2"}, s.controlIDs(sspID), "ac-1 matches the existing AC-1 case-insensitively")

	s.ErrorIs(s.svc.AttachProfile(s.DB, sspID, s.profileID), sspservice.ErrProfileAlreadyBound)
	s.ErrorIs(s.svc.AttachProfile(s.DB, uuid.New(), s.profileID), sspservice.ErrSSPNotFound)
	s.ErrorIs(s.svc.AttachProfile(s.DB, sspID, uuid.New()), sspservice.ErrProfileNotFound)
}

func (s *SSPServiceIntegrationSuite) TestNoControlsLeavesNoBinding() {
	sspID := s.createSSP(s.DB)
	s.ErrorIs(s.svc.AttachProfile(s.DB, sspID, s.profileID), sspservice.ErrNoResolvedControls)

	var bindings int64
	s.Require().NoError(s.DB.Model(&relational.SSPProfile{}).Where("system_security_plan_id = ?", sspID).Count(&bindings).Error)
	s.Zero(bindings)
}

// TestRunsInCallersTransaction checks that both calls take part in the caller's
// transaction: rolling it back leaves neither the SSP nor its requirements.
func (s *SSPServiceIntegrationSuite) TestRunsInCallersTransaction() {
	s.controls[s.profileID] = []string{"AC-1"}
	var sspID uuid.UUID
	rollback := errors.New("rollback")
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		sspID = s.createSSP(tx)
		s.Require().NoError(s.svc.AttachProfile(tx, sspID, s.profileID))
		return rollback
	})
	s.Require().ErrorIs(err, rollback)

	var ssps int64
	s.Require().NoError(s.DB.Model(&relational.SystemSecurityPlan{}).Where("id = ?", sspID).Count(&ssps).Error)
	s.Zero(ssps)
	s.Empty(s.controlIDs(sspID))
}
