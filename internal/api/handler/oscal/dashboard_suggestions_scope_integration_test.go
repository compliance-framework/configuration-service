//go:build integration

package oscal

import (
	"encoding/json"
	"fmt"
	"net/http"

	apihandler "github.com/compliance-framework/api/internal/api/handler"
	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/service/relational/sspscope"
	suggestionrel "github.com/compliance-framework/api/internal/service/relational/suggestions"
	"github.com/google/uuid"
)

// seedScopedSSP seeds one control and three evidence label sets (repo a, b, c), and
// returns the SSP id. The SSP has no scope entries yet.
func (suite *DashboardSuggestionsHTTPSuite) seedScopedSSP() uuid.UUID {
	sspID, _, _ := suite.seedScope([]string{"AC-1"}, []map[string]string{
		{"env": "prod", "repo": "a"},
		{"env": "stage", "repo": "b"},
		{"env": "dev", "repo": "c"},
	})
	return sspID
}

func (suite *DashboardSuggestionsHTTPSuite) addScopeEntry(sspID uuid.UUID, key, value string) {
	_, err := sspscope.NewService(suite.DB).AddEntry(sspID, []relational.ScopeLabel{{Key: key, Value: value}}, nil)
	suite.Require().NoError(err)
}

func (suite *DashboardSuggestionsHTTPSuite) get(path string, out any) int {
	rec, req := suite.req(http.MethodGet, path, nil)
	suite.server.E().ServeHTTP(rec, req)
	if rec.Code == http.StatusOK && out != nil {
		suite.Require().NoError(json.Unmarshal(rec.Body.Bytes(), out))
	}
	return rec.Code
}

func (suite *DashboardSuggestionsHTTPSuite) repoValues(sspID uuid.UUID, query string) []string {
	var keys apihandler.GenericDataListResponse[suggestionrel.LabelKeyInput]
	suite.Require().Equal(http.StatusOK, suite.get(fmt.Sprintf("/api/oscal/system-security-plans/%s/dashboard-suggestions/label-keys%s", sspID, query), &keys))
	for _, key := range keys.Data {
		if key.Key == "repo" {
			return key.Values
		}
	}
	return nil
}

func (suite *DashboardSuggestionsHTTPSuite) TestLabelPickersFollowScopeOnlyWhenItHasEntries() {
	sspID := suite.seedScopedSSP()
	base := fmt.Sprintf("/api/oscal/system-security-plans/%s/dashboard-suggestions", sspID)

	suite.ElementsMatch([]string{"a", "b", "c"}, suite.repoValues(sspID, ""), "no scope entries: all evidence")

	suite.addScopeEntry(sspID, "repo", "a")
	suite.addScopeEntry(sspID, "repo", "b")

	suite.ElementsMatch([]string{"a", "b"}, suite.repoValues(sspID, ""))
	suite.ElementsMatch([]string{"a", "b", "c"}, suite.repoValues(sspID, "?ignoreScope=true"))

	var values apihandler.GenericDataListResponse[string]
	suite.Require().Equal(http.StatusOK, suite.get(base+"/label-values?key=repo", &values))
	suite.ElementsMatch([]string{"a", "b"}, values.Data)
	suite.Require().Equal(http.StatusOK, suite.get(base+"/label-values?key=repo&ignoreScope=true", &values))
	suite.ElementsMatch([]string{"a", "b", "c"}, values.Data)

	var sets apihandler.GenericDataListResponse[suggestionrel.LabelSetInput]
	suite.Require().Equal(http.StatusOK, suite.get(base+"/label-sets", &sets))
	suite.Len(sets.Data, 2)
	for _, set := range sets.Data {
		suite.Contains([]string{"a", "b"}, set.Labels["repo"])
	}

	suite.Equal(http.StatusBadRequest, suite.get(base+"/label-keys?ignoreScope=maybe", nil))
	missing := fmt.Sprintf("/api/oscal/system-security-plans/%s/dashboard-suggestions", uuid.New())
	suite.Equal(http.StatusNotFound, suite.get(missing+"/label-keys", nil))
	suite.Equal(http.StatusNotFound, suite.get(missing+"/label-sets", nil))
	suite.Equal(http.StatusOK, suite.get(missing+"/label-keys?ignoreScope=true", nil), "ignoreScope never reads the scope")
}

func (suite *DashboardSuggestionsHTTPSuite) preview(sspID uuid.UUID, body any) (int, dashboardSuggestionPreviewResponse) {
	rec, req := suite.req(http.MethodPost, fmt.Sprintf("/api/oscal/system-security-plans/%s/dashboard-suggestions/preview", sspID), body)
	suite.server.E().ServeHTTP(rec, req)
	var out apihandler.GenericDataResponse[dashboardSuggestionPreviewResponse]
	if rec.Code == http.StatusOK {
		suite.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &out))
	}
	return rec.Code, out.Data
}

func (suite *DashboardSuggestionsHTTPSuite) TestPreviewAndGenerateUseScopedLabelSets() {
	sspID := suite.seedScopedSSP()
	prodOrStage := &labelfilter.Filter{Scope: &labelfilter.Scope{Query: &labelfilter.Query{Operator: "OR", Scopes: []labelfilter.Scope{
		{Condition: &labelfilter.Condition{Label: "env", Operator: "=", Value: "prod"}},
		{Condition: &labelfilter.Condition{Label: "env", Operator: "=", Value: "dev"}},
	}}}}
	withFilter := generateDashboardSuggestionsRequest{Scope: &dashboardSuggestionScopeRequest{LabelFilter: prodOrStage}}

	code, plan := suite.preview(sspID, nil)
	suite.Require().Equal(http.StatusOK, code)
	suite.Equal(3, plan.LabelSetCount, "no scope entries: all evidence")

	suite.addScopeEntry(sspID, "repo", "a")
	suite.addScopeEntry(sspID, "repo", "b")

	code, plan = suite.preview(sspID, nil)
	suite.Require().Equal(http.StatusOK, code)
	suite.Equal(2, plan.LabelSetCount)

	code, plan = suite.preview(sspID, withFilter)
	suite.Require().Equal(http.StatusOK, code)
	suite.Equal(1, plan.LabelSetCount, "the request filter is ANDed with the scope")

	code, _ = suite.preview(uuid.New(), nil)
	suite.Equal(http.StatusNotFound, code)

	// The run stores the effective filter, so the worker's gather sees the same scoped
	// label sets the preview counted.
	rec, req := suite.req(http.MethodPost, fmt.Sprintf("/api/oscal/system-security-plans/%s/dashboard-suggestions/generate", sspID), withFilter)
	suite.server.E().ServeHTTP(rec, req)
	suite.Require().Equal(http.StatusAccepted, rec.Code, rec.Body.String())
	var run suggestionrel.DashboardSuggestionRun
	suite.Require().NoError(suite.DB.Where("ssp_id = ?", sspID).First(&run).Error)
	stored := suggestionrel.LabelFilterFromJSONMap(run.LabelFilter)
	suite.Require().NotNil(stored)
	sets, err := suggestionrel.NewSuggestionService(suite.DB).GatherLabelSets(nil, stored)
	suite.Require().NoError(err)
	suite.Require().Len(sets, 1)
	suite.Equal("a", sets[0].Labels["repo"])
}
