//go:build integration

package ssptemplates_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/service/relational/sspscope"
	"github.com/compliance-framework/api/internal/service/relational/ssptemplates"
	"github.com/compliance-framework/api/internal/tests"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type DiscoveryIntegrationSuite struct {
	tests.IntegrationTestSuite
	controls  []string // what the stub resolver returns for any profile
	discovery *ssptemplates.Discovery
	templates *ssptemplates.Service
}

func TestDiscoveryIntegration(t *testing.T) {
	suite.Run(t, new(DiscoveryIntegrationSuite))
}

func (s *DiscoveryIntegrationSuite) SetupTest() {
	s.Require().NoError(s.Migrator.Refresh())
	s.controls = []string{"AC-1", "AC-2"}
	s.discovery = ssptemplates.NewDiscovery(s.DB, func(uuid.UUID) ([]string, error) { return s.controls, nil })
	s.templates = ssptemplates.NewService(s.DB)
}

func (s *DiscoveryIntegrationSuite) evidence(pairs ...string) {
	id := uuid.New()
	labels := []relational.Labels{}
	for i := 0; i+1 < len(pairs); i += 2 {
		labels = append(labels, relational.Labels{Name: pairs[i], Value: pairs[i+1]})
	}
	s.Require().NoError(s.DB.Create(&relational.Evidence{
		UUIDModel: relational.UUIDModel{ID: &id}, UUID: uuid.New(), Title: "e",
		Start: time.Now().Add(-time.Minute), End: time.Now(), Labels: labels,
	}).Error)
}

func (s *DiscoveryIntegrationSuite) template(edit func(*ssptemplates.Input)) uuid.UUID {
	description := "Created from _app={{ ._app }}"
	in := ssptemplates.Input{
		Name: "Apps", Keys: []string{"_app"}, ScopeLabels: [][]string{{"repo"}},
		TitleTemplate: "SSP for {{ ._app }}", DescriptionTemplate: &description,
		Props: []relational.SSPTemplateProp{{Name: "application", Value: "{{ ._app }}"}},
	}
	if edit != nil {
		edit(&in)
	}
	tmpl, err := s.templates.Create(in)
	s.Require().NoError(err)
	return *tmpl.ID
}

func (s *DiscoveryIntegrationSuite) run(templateID uuid.UUID) ssptemplates.RunResult {
	result, err := s.discovery.Run(context.Background(), templateID)
	s.Require().NoError(err)
	return result
}

func (s *DiscoveryIntegrationSuite) instance(templateID uuid.UUID) relational.SSPTemplateInstance {
	var instance relational.SSPTemplateInstance
	s.Require().NoError(s.DB.Where("template_id = ?", templateID).First(&instance).Error)
	return instance
}

func (s *DiscoveryIntegrationSuite) entries(sspID uuid.UUID) []string {
	var rows []relational.SSPScopeEntry
	s.Require().NoError(s.DB.Where("ssp_id = ?", sspID).Order("match_hash").Find(&rows).Error)
	out := []string{}
	for _, row := range rows {
		s.Equal(relational.ScopeEntryOriginTemplate, row.Origin)
		s.Require().NotNil(row.TemplateID)
		out = append(out, fmt.Sprint(row.Labels))
	}
	sort.Strings(out)
	return out
}

func (s *DiscoveryIntegrationSuite) TestCreatesSSPWithProfile() {
	profileID := uuid.New()
	s.Require().NoError(s.DB.Create(&relational.Profile{UUIDModel: relational.UUIDModel{ID: &profileID}}).Error)
	templateID := s.template(func(in *ssptemplates.Input) { in.ProfileIDs = []uuid.UUID{profileID} })
	s.evidence("_app", "Payments", "repo", "A")

	s.Equal(ssptemplates.RunResult{Created: 1}, s.run(templateID))

	instance := s.instance(templateID)
	s.Require().NotNil(instance.SSPID)
	s.Equal(ssptemplates.SSPID(templateID, instance.KeyHash), *instance.SSPID, "deterministic id")
	s.False(instance.ProfilesPending)
	s.Nil(instance.LastError)
	var ssp relational.SystemSecurityPlan
	s.Require().NoError(s.DB.Preload("Metadata").Preload("SystemCharacteristics").Preload("Profiles").
		Preload("ControlImplementation.ImplementedRequirements").First(&ssp, "id = ?", instance.SSPID).Error)
	oscal := ssp.MarshalOscal()
	s.Equal("SSP for Payments", oscal.Metadata.Title)
	s.Equal("1.0", oscal.Metadata.Version)
	s.Equal("SSP for Payments", oscal.SystemCharacteristics.SystemName)
	s.Equal("Created from _app=Payments", oscal.SystemCharacteristics.Description)
	s.Equal("under-development", oscal.SystemCharacteristics.Status.State)
	s.Require().NotNil(oscal.Metadata.Props)
	props := map[string]string{}
	for _, prop := range *oscal.Metadata.Props {
		props[prop.Name] = prop.Value
	}
	s.Equal(map[string]string{"application": "Payments", "ssp-template": templateID.String()}, props)
	s.Require().Len(ssp.Profiles, 1)
	s.Len(ssp.ControlImplementation.ImplementedRequirements, 2)
	s.Equal([]string{"[{_app Payments}]", "[{repo A}]"}, s.entries(*instance.SSPID))

	var tmpl relational.SSPTemplate
	s.Require().NoError(s.DB.First(&tmpl, "id = ?", templateID).Error)
	s.NotNil(tmpl.LastRunAt)
}

func (s *DiscoveryIntegrationSuite) TestIdempotentAndLearnsLater() {
	templateID := s.template(nil)
	s.evidence("_app", "payments", "repo", "a")
	s.Equal(ssptemplates.RunResult{Created: 1}, s.run(templateID))
	sspID := *s.instance(templateID).SSPID
	firstSeen := s.instance(templateID).LastSeenAt

	s.Equal(ssptemplates.RunResult{Updated: 1}, s.run(templateID))
	var ssps int64
	s.Require().NoError(s.DB.Model(&relational.SystemSecurityPlan{}).Count(&ssps).Error)
	s.Equal(int64(1), ssps)
	s.Len(s.entries(sspID), 2)
	s.True(s.instance(templateID).LastSeenAt.After(firstSeen))

	// A new value is learned; an excepted one is not.
	s.Require().NoError(s.DB.Create(&relational.SSPScopeException{
		SSPID: sspID, Labels: []relational.ScopeLabel{{Key: "repo", Value: "x"}},
		MatchHash: sspscope.MatchHash([]relational.ScopeLabel{{Key: "repo", Value: "x"}}),
	}).Error)
	s.evidence("_app", "PAYMENTS", "repo", "b")
	s.evidence("_app", "payments", "repo", "x")
	s.run(templateID)
	s.Equal([]string{"[{_app payments}]", "[{repo a}]", "[{repo b}]"}, s.entries(sspID))
	s.Require().NoError(s.DB.Model(&relational.SystemSecurityPlan{}).Count(&ssps).Error)
	s.Equal(int64(1), ssps, "PAYMENTS and payments are one group")
}

func (s *DiscoveryIntegrationSuite) TestNeverRecreatesDeletedSSP() {
	templateID := s.template(nil)
	s.evidence("_app", "payments")
	s.run(templateID)
	s.Require().NoError(s.DB.Delete(&relational.SystemSecurityPlan{}, "id = ?", s.instance(templateID).SSPID).Error)

	s.Equal(ssptemplates.RunResult{}, s.run(templateID))
	s.Nil(s.instance(templateID).SSPID)
	var ssps int64
	s.Require().NoError(s.DB.Model(&relational.SystemSecurityPlan{}).Count(&ssps).Error)
	s.Zero(ssps)
}

func (s *DiscoveryIntegrationSuite) TestCapsNewSSPsPerRun() {
	templateID := s.template(nil)
	for i := 0; i <= ssptemplates.MaxNewSSPsPerRun; i++ {
		s.evidence("_app", fmt.Sprintf("app-%03d", i))
	}
	s.Equal(ssptemplates.RunResult{Created: ssptemplates.MaxNewSSPsPerRun}, s.run(templateID))
	s.Equal(ssptemplates.RunResult{Created: 1, Updated: ssptemplates.MaxNewSSPsPerRun}, s.run(templateID))
}

func (s *DiscoveryIntegrationSuite) TestRetriesFailedProfileAttach() {
	profileID := uuid.New()
	s.Require().NoError(s.DB.Create(&relational.Profile{UUIDModel: relational.UUIDModel{ID: &profileID}}).Error)
	templateID := s.template(func(in *ssptemplates.Input) { in.ProfileIDs = []uuid.UUID{profileID} })
	s.evidence("_app", "payments")
	s.controls = nil

	s.Equal(ssptemplates.RunResult{Created: 1}, s.run(templateID), "a failed attach does not undo the SSP")
	instance := s.instance(templateID)
	s.True(instance.ProfilesPending)
	s.Require().NotNil(instance.LastError)
	s.Contains(*instance.LastError, "no controls were resolved")

	s.controls = []string{"AC-1"}
	s.run(templateID)
	instance = s.instance(templateID)
	s.False(instance.ProfilesPending)
	s.Nil(instance.LastError)
}

func (s *DiscoveryIntegrationSuite) TestSkipsDisabledAndRecordsInvalidEntries() {
	disabled := false
	off := s.template(func(in *ssptemplates.Input) { in.Enabled = &disabled })
	on := s.template(nil)
	s.evidence("_app", "payments", "repo", strings.Repeat("x", 300))

	s.Require().NoError(s.discovery.RunAll(context.Background()))
	var count int64
	s.Require().NoError(s.DB.Model(&relational.SSPTemplateInstance{}).Where("template_id = ?", off).Count(&count).Error)
	s.Zero(count, "a disabled template does nothing")
	instance := s.instance(on)
	s.Equal(1, instance.SkippedEntries)
	s.Len(instance.SkippedSample, 1)
}
