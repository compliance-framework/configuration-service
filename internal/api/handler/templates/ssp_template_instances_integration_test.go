//go:build integration

package templates_test

import (
	"context"
	"net/http"
	"time"

	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// recordingEnqueuer records the templates whose discovery run was queued.
type recordingEnqueuer struct{ queued []uuid.UUID }

func (e *recordingEnqueuer) EnqueueSSPTemplateDiscovery(_ context.Context, templateID uuid.UUID) error {
	e.queued = append(e.queued, templateID)
	return nil
}

func (s *SSPTemplateApiIntegrationSuite) TestRun() {
	id := s.create(templateBody("Applications", "_app"))["id"].(string)
	disabled := templateBody("Disabled", "_app")
	disabled["enabled"] = false
	disabledID := s.create(disabled)["id"].(string)

	s.Equal(http.StatusAccepted, s.do(s.admin, http.MethodPost, "/"+id+"/run", nil).Code)
	s.Equal([]uuid.UUID{uuid.MustParse(id)}, s.runs.queued)
	s.Equal(http.StatusConflict, s.do(s.admin, http.MethodPost, "/"+disabledID+"/run", nil).Code)
	s.Equal(http.StatusNotFound, s.do(s.admin, http.MethodPost, "/"+uuid.NewString()+"/run", nil).Code)
	s.Equal(http.StatusForbidden, s.do(s.contributor, http.MethodPost, "/"+id+"/run", nil).Code)
	s.Len(s.runs.queued, 1)

	s.server = s.newServer(nil)
	s.Equal(http.StatusServiceUnavailable, s.do(s.admin, http.MethodPost, "/"+id+"/run", nil).Code, "no worker")
}

func (s *SSPTemplateApiIntegrationSuite) TestInstancesAndForget() {
	templateID := uuid.MustParse(s.create(templateBody("Applications", "_app"))["id"].(string))
	sspID := uuid.New()
	s.Require().NoError(s.DB.Create(&relational.SystemSecurityPlan{
		UUIDModel: relational.UUIDModel{ID: &sspID}, Metadata: relational.Metadata{Title: "SSP for payments"},
	}).Error)
	lastError := "attach profile x: profile not found"
	for _, instance := range []relational.SSPTemplateInstance{
		{KeyHash: "live", KeyValues: datatypes.JSONMap{"_app": "payments"}, SSPID: &sspID, LastSeenAt: time.Now(), LastError: &lastError, SkippedEntries: 2,
			SkippedSample: []relational.SkippedScopeEntry{{Labels: []relational.ScopeLabel{{Key: "repo", Value: "x"}}, Reason: "too long"}}},
		{KeyHash: "stale", KeyValues: datatypes.JSONMap{"_app": "old"}, SSPID: &sspID, LastSeenAt: time.Now().Add(-8 * 24 * time.Hour)},
		{KeyHash: "deleted", KeyValues: datatypes.JSONMap{"_app": "gone"}, LastSeenAt: time.Now().Add(-30 * 24 * time.Hour)},
	} {
		instance.TemplateID = templateID
		s.Require().NoError(s.DB.Create(&instance).Error)
	}

	path := "/" + templateID.String() + "/ssps"
	list := s.decode(s.do(s.admin, http.MethodGet, path, nil), http.StatusOK)
	s.Equal(float64(3), list["total"])
	byHash := map[string]map[string]any{}
	for _, item := range list["data"].([]any) {
		byHash[item.(map[string]any)["key-hash"].(string)] = item.(map[string]any)
	}
	live := byHash["live"]
	s.Equal(map[string]any{"_app": "payments"}, live["key-values"])
	s.Equal(sspID.String(), live["ssp-id"])
	s.Equal("SSP for payments", live["title"])
	s.Equal(false, live["stale"])
	s.Equal(lastError, live["last-error"])
	s.Equal(float64(2), live["skipped-entries"])
	s.Len(live["skipped-sample"], 1)
	s.Equal(true, byHash["stale"]["stale"])
	s.Nil(byHash["deleted"]["ssp-id"])
	s.Nil(byHash["deleted"]["title"])
	s.Equal(false, byHash["deleted"]["stale"], "a deleted SSP is not stale")

	s.Equal(http.StatusConflict, s.do(s.admin, http.MethodDelete, path+"/live", nil).Code, "the SSP still exists")
	s.Equal(http.StatusNotFound, s.do(s.admin, http.MethodDelete, path+"/nope", nil).Code)
	s.Equal(http.StatusForbidden, s.do(s.contributor, http.MethodDelete, path+"/deleted", nil).Code)
	s.Equal(http.StatusForbidden, s.do(s.contributor, http.MethodGet, path, nil).Code)
	s.Equal(http.StatusNoContent, s.do(s.admin, http.MethodDelete, path+"/deleted", nil).Code)
	s.Equal(float64(2), s.decode(s.do(s.admin, http.MethodGet, path, nil), http.StatusOK)["total"])
	s.Equal(http.StatusNotFound, s.do(s.admin, http.MethodGet, "/"+uuid.NewString()+"/ssps", nil).Code)
}
