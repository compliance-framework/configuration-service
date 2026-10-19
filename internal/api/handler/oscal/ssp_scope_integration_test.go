//go:build integration

package oscal

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/compliance-framework/api/internal/api"
	"github.com/compliance-framework/api/internal/api/handler"
	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/tests"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/datatypes"
)

// SSPScopeApiIntegrationSuite exercises the scope routes through the embedded Cedar PDP.
type SSPScopeApiIntegrationSuite struct {
	tests.IntegrationTestSuite
	server      *api.Server
	contributor string
	viewer      string
	sspID       uuid.UUID
}

func TestSSPScopeApi(t *testing.T) {
	suite.Run(t, new(SSPScopeApiIntegrationSuite))
}

func (s *SSPScopeApiIntegrationSuite) SetupTest() {
	s.Require().NoError(s.Migrator.Refresh())
	s.contributor = createRoledUser(&s.IntegrationTestSuite, "contributor@example.com", "contributor")
	s.viewer = createRoledUser(&s.IntegrationTestSuite, "viewer@example.com", "viewer")
	s.server = newCedarServer(&s.IntegrationTestSuite)
	s.sspID = uuid.New()
	s.Require().NoError(s.DB.Create(&relational.SystemSecurityPlan{UUIDModel: relational.UUIDModel{ID: &s.sspID}}).Error)
}

func (s *SSPScopeApiIntegrationSuite) path(suffix string) string {
	return "/api/oscal/system-security-plans/" + s.sspID.String() + "/scope" + suffix
}

func labelsBody(pairs ...string) map[string]any {
	labels := []map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		labels = append(labels, map[string]string{"key": pairs[i], "value": pairs[i+1]})
	}
	return map[string]any{"labels": labels}
}

func (s *SSPScopeApiIntegrationSuite) getScope(token string) sspScopeResponse {
	rec := authedRequest(&s.IntegrationTestSuite, s.server, http.MethodGet, s.path(""), token, nil)
	s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())
	var out handler.GenericDataResponse[sspScopeResponse]
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &out))
	return out.Data
}

func (s *SSPScopeApiIntegrationSuite) TestContributorManagesScope() {
	empty := s.getScope(s.contributor)
	s.False(empty.AutoFilters)
	s.Empty(empty.Entries)
	s.Empty(empty.Exceptions)

	rec := authedRequest(&s.IntegrationTestSuite, s.server, http.MethodPost, s.path("/entries"), s.contributor, labelsBody("Repo", " a "))
	s.Require().Equal(http.StatusCreated, rec.Code, rec.Body.String())
	var raw map[string]map[string]any
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &raw))
	s.Equal("manual", raw["data"]["origin"])
	s.Contains(raw["data"], "template-id", "kebab-case like the other SSP sub-resources")
	s.Contains(raw["data"], "created-at")
	s.Equal([]any{map[string]any{"key": "repo", "value": "a"}}, raw["data"]["labels"])

	rec = authedRequest(&s.IntegrationTestSuite, s.server, http.MethodPost, s.path("/entries"), s.contributor, labelsBody("aws_account", "111", "region", "eu-west-1"))
	s.Require().Equal(http.StatusCreated, rec.Code, rec.Body.String())

	rec = authedRequest(&s.IntegrationTestSuite, s.server, http.MethodPost, s.path("/exceptions"), s.contributor, labelsBody("repo", "A"))
	s.Require().Equal(http.StatusCreated, rec.Code, rec.Body.String())
	var exception handler.GenericDataResponse[sspScopeExceptionResponse]
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &exception))

	rec = authedRequest(&s.IntegrationTestSuite, s.server, http.MethodPut, s.path(""), s.contributor, map[string]any{"auto-filters": true})
	s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())

	scope := s.getScope(s.viewer)
	s.True(scope.AutoFilters)
	s.Require().Len(scope.Entries, 1)
	s.Equal([]relational.ScopeLabel{{Key: "aws_account", Value: "111"}, {Key: "region", Value: "eu-west-1"}}, scope.Entries[0].Labels)
	s.Require().Len(scope.Exceptions, 1)

	rec = authedRequest(&s.IntegrationTestSuite, s.server, http.MethodDelete, s.path("/exceptions/"+exception.Data.ID.String()), s.contributor, nil)
	s.Require().Equal(http.StatusNoContent, rec.Code, rec.Body.String())
	scope = s.getScope(s.contributor)
	s.Empty(scope.Exceptions)
	s.Len(scope.Entries, 1, "deleting an exception does not bring the entry back")
}

func (s *SSPScopeApiIntegrationSuite) TestViewerCannotWrite() {
	s.getScope(s.viewer)

	for _, req := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, s.path("/entries"), labelsBody("repo", "a")},
		{http.MethodPost, s.path("/exceptions"), labelsBody("repo", "a")},
		{http.MethodPut, s.path(""), map[string]any{"auto-filters": true}},
		{http.MethodDelete, s.path("/exceptions/" + uuid.NewString()), nil},
	} {
		rec := authedRequest(&s.IntegrationTestSuite, s.server, req.method, req.path, s.viewer, req.body)
		s.Equal(http.StatusForbidden, rec.Code, "%s %s", req.method, req.path)
	}
	s.Empty(s.getScope(s.contributor).Entries)
}

func (s *SSPScopeApiIntegrationSuite) TestErrors() {
	cases := []struct {
		name, method, path string
		body               any
		want               int
	}{
		{"bad ssp id", http.MethodGet, "/api/oscal/system-security-plans/not-a-uuid/scope", nil, http.StatusBadRequest},
		{"unknown ssp", http.MethodGet, "/api/oscal/system-security-plans/" + uuid.NewString() + "/scope", nil, http.StatusNotFound},
		{"unknown ssp write", http.MethodPost, "/api/oscal/system-security-plans/" + uuid.NewString() + "/scope/entries", labelsBody("repo", "a"), http.StatusNotFound},
		{"key-only entry", http.MethodPost, s.path("/entries"), labelsBody("repo", ""), http.StatusBadRequest},
		{"no labels", http.MethodPost, s.path("/entries"), map[string]any{}, http.StatusBadRequest},
		{"missing auto-filters", http.MethodPut, s.path(""), map[string]any{}, http.StatusBadRequest},
		{"unknown exception", http.MethodDelete, s.path("/exceptions/" + uuid.NewString()), nil, http.StatusNotFound},
		{"bad exception id", http.MethodDelete, s.path("/exceptions/nope"), nil, http.StatusBadRequest},
	}
	for _, tc := range cases {
		rec := authedRequest(&s.IntegrationTestSuite, s.server, tc.method, tc.path, s.contributor, tc.body)
		s.Equal(tc.want, rec.Code, "%s: %s", tc.name, rec.Body.String())
	}

	rec := authedRequest(&s.IntegrationTestSuite, s.server, http.MethodPost, s.path("/entries"), s.contributor, labelsBody("repo", "a"))
	s.Require().Equal(http.StatusCreated, rec.Code)
	rec = authedRequest(&s.IntegrationTestSuite, s.server, http.MethodPost, s.path("/entries"), s.contributor, labelsBody("REPO", "A"))
	s.Equal(http.StatusConflict, rec.Code, rec.Body.String())

	rec = authedRequest(&s.IntegrationTestSuite, s.server, http.MethodPost, s.path("/exceptions"), s.contributor, labelsBody("repo", "b"))
	s.Require().Equal(http.StatusCreated, rec.Code)
	rec = authedRequest(&s.IntegrationTestSuite, s.server, http.MethodPost, s.path("/exceptions"), s.contributor, labelsBody("repo", "b"))
	s.Equal(http.StatusConflict, rec.Code, rec.Body.String())
}

func (s *SSPScopeApiIntegrationSuite) TestRecordsActor() {
	rec := authedRequest(&s.IntegrationTestSuite, s.server, http.MethodPost, s.path("/entries"), s.contributor, labelsBody("repo", "a"))
	s.Require().Equal(http.StatusCreated, rec.Code)

	var user relational.User
	s.Require().NoError(s.DB.Where("email = ?", "contributor@example.com").First(&user).Error)
	var entry relational.SSPScopeEntry
	s.Require().NoError(s.DB.Where("ssp_id = ?", s.sspID).First(&entry).Error)
	s.Require().NotNil(entry.CreatedBy)
	s.Equal(*user.ID, *entry.CreatedBy)
}

func (s *SSPScopeApiIntegrationSuite) TestExceptionThatWouldEmptyAFollowedScopeIsRefused() {
	rec := authedRequest(&s.IntegrationTestSuite, s.server, http.MethodPost, s.path("/entries"), s.contributor, labelsBody("repo", "a"))
	s.Require().Equal(http.StatusCreated, rec.Code, rec.Body.String())
	scopeBase := datatypes.NewJSONType(labelfilter.Filter{})
	s.Require().NoError(s.DB.Create(&relational.Filter{Name: "follows", SSPID: &s.sspID, ScopeBase: &scopeBase}).Error)

	rec = authedRequest(&s.IntegrationTestSuite, s.server, http.MethodPost, s.path("/exceptions"), s.contributor, labelsBody("repo", "a"))
	s.Equal(http.StatusConflict, rec.Code, rec.Body.String())
	s.Contains(rec.Body.String(), "1 filter(s) follow it")
	s.Len(s.getScope(s.contributor).Entries, 1)
}
