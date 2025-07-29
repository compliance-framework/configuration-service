//go:build integration

package oscal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/api"
	"github.com/compliance-framework/api/internal/api/handler"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/tests"
	oscaltypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

func TestAssessmentResultsApi(t *testing.T) {
	fmt.Println("Starting Assessment Results API tests")
	suite.Run(t, new(AssessmentResultsApiIntegrationSuite))
}

type AssessmentResultsApiIntegrationSuite struct {
	tests.IntegrationTestSuite
	server *api.Server
	logger *zap.SugaredLogger
}

func (suite *AssessmentResultsApiIntegrationSuite) SetupSuite() {
	fmt.Println("Setting up Assessment Results API test suite")
	suite.IntegrationTestSuite.SetupSuite()

	logConf := zap.NewDevelopmentConfig()
	logConf.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	logger, _ := logConf.Build()
	suite.logger = logger.Sugar()
	suite.server = api.NewServer(context.Background(), suite.logger, suite.Config)
	RegisterHandlers(suite.server, suite.logger, suite.DB, suite.Config)
	fmt.Println("Server initialized")
}

func (suite *AssessmentResultsApiIntegrationSuite) SetupTest() {
	err := suite.Migrator.Refresh()
	suite.Require().NoError(err)
}

// Helper method to create a test request with Bearer token authentication
func (suite *AssessmentResultsApiIntegrationSuite) createRequest(method, path string, body any) (*httptest.ResponseRecorder, *http.Request) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		suite.Require().NoError(err, "Failed to marshal request body")
	}

	token, err := suite.GetAuthToken()
	suite.Require().NoError(err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, fmt.Sprintf("Bearer %s", *token))

	return rec, req
}

// Helper function to create a basic Assessment Results and return its UUID
func (suite *AssessmentResultsApiIntegrationSuite) createBasicAssessmentResults() string {
	arUUID := uuid.New().String()
	
	createAR := oscaltypes.AssessmentResults{
		UUID: arUUID,
		Metadata: oscaltypes.Metadata{
			Title:   "Test Assessment Results",
			Version: "1.0.0",
		},
		ImportAp: oscaltypes.ImportAp{
			Href: "https://example.com/assessment-plan.json",
		},
		Results: []oscaltypes.Result{
			{
				UUID:        uuid.New().String(),
				Title:       "Test Result",
				Description: "Test result description",
				Start:       time.Now(),
				ReviewedControls: oscaltypes.ReviewedControls{
					Description: "Controls reviewed",
				},
			},
		},
	}

	rec, req := suite.createRequest(http.MethodPost, "/api/oscal/assessment-results", createAR)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusCreated, rec.Code)

	return arUUID
}

// Test Create endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestCreateAssessmentResults() {
	testAR := oscaltypes.AssessmentResults{
		UUID: uuid.New().String(),
		Metadata: oscaltypes.Metadata{
			Title:   "Test Assessment Results",
			Version: "1.0.0",
		},
		ImportAp: oscaltypes.ImportAp{
			Href: "https://example.com/assessment-plan.json",
		},
		Results: []oscaltypes.Result{
			{
				UUID:        uuid.New().String(),
				Title:       "Test Result",
				Description: "Test result description",
				Start:       time.Now(),
				ReviewedControls: oscaltypes.ReviewedControls{
					Description: "Controls reviewed",
				},
			},
		},
	}

	rec, req := suite.createRequest(http.MethodPost, "/api/oscal/assessment-results", testAR)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusCreated, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.AssessmentResults]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(testAR.UUID, response.Data.UUID)
	suite.Equal(testAR.Metadata.Title, response.Data.Metadata.Title)
}

// Test Get endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestGetAssessmentResults() {
	arUUID := suite.createBasicAssessmentResults()

	rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s", arUUID), nil)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.AssessmentResults]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(arUUID, response.Data.UUID)
}

// Test List endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestListAssessmentResults() {
	// Create multiple assessment results
	arUUID1 := suite.createBasicAssessmentResults()
	arUUID2 := suite.createBasicAssessmentResults()

	rec, req := suite.createRequest(http.MethodGet, "/api/oscal/assessment-results", nil)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataListResponse[oscaltypes.AssessmentResults]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.GreaterOrEqual(len(response.Data), 2)

	// Verify our created items are in the list
	uuids := make(map[string]bool)
	for _, ar := range response.Data {
		uuids[ar.UUID] = true
	}
	suite.True(uuids[arUUID1])
	suite.True(uuids[arUUID2])
}

// Test Update endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestUpdateAssessmentResults() {
	arUUID := suite.createBasicAssessmentResults()

	// Get the full assessment results (including Results)
	getRec, getReq := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/full", arUUID), nil)
	suite.server.E().ServeHTTP(getRec, getReq)
	suite.Equal(http.StatusOK, getRec.Code)

	var getResponse handler.GenericDataResponse[oscaltypes.AssessmentResults]
	err := json.Unmarshal(getRec.Body.Bytes(), &getResponse)
	suite.NoError(err)

	// Update the assessment results
	updatedAR := getResponse.Data
	updatedAR.Metadata.Title = "Updated Assessment Results"
	updatedAR.Metadata.Version = "2.0.0"

	rec, req := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-results/%s", arUUID), updatedAR)
	suite.server.E().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		suite.T().Logf("Update response: %d, body: %s", rec.Code, rec.Body.String())
	}
	suite.Equal(http.StatusOK, rec.Code)

	var response handler.GenericDataResponse[oscaltypes.AssessmentResults]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Updated Assessment Results", response.Data.Metadata.Title)
	suite.Equal("2.0.0", response.Data.Metadata.Version)
}

// Test Get Full endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestGetFullAssessmentResults() {
	arUUID := suite.createBasicAssessmentResults()

	rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/full", arUUID), nil)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.AssessmentResults]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(arUUID, response.Data.UUID)
	suite.Len(response.Data.Results, 1)
}

// Test Delete endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestDeleteAssessmentResults() {
	arUUID := suite.createBasicAssessmentResults()

	rec, req := suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-results/%s", arUUID), nil)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusNoContent, rec.Code)

	// Verify deletion
	getRec, getReq := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s", arUUID), nil)
	suite.server.E().ServeHTTP(getRec, getReq)
	suite.Equal(http.StatusNotFound, getRec.Code)
}

// Test Get Metadata endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestGetAssessmentResultsMetadata() {
	arUUID := suite.createBasicAssessmentResults()

	rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/metadata", arUUID), nil)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.Metadata]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Test Assessment Results", response.Data.Title)
}

// Test Update Metadata endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestUpdateAssessmentResultsMetadata() {
	arUUID := suite.createBasicAssessmentResults()

	metadata := oscaltypes.Metadata{
		Title:   "New Metadata Title",
		Version: "2.0.0",
	}
	rec, req := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-results/%s/metadata", arUUID), metadata)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.Metadata]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("New Metadata Title", response.Data.Title)
	suite.Equal("2.0.0", response.Data.Version)
}

// Test Get Import AP endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestGetAssessmentResultsImportAp() {
	arUUID := suite.createBasicAssessmentResults()

	rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/import-ap", arUUID), nil)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.ImportAp]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("https://example.com/assessment-plan.json", response.Data.Href)
}

// Test Update Import AP endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestUpdateAssessmentResultsImportAp() {
	arUUID := suite.createBasicAssessmentResults()

	importAp := oscaltypes.ImportAp{
		Href:    "https://example.com/new-assessment-plan.json",
		Remarks: "Updated assessment plan reference",
	}
	rec, req := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-results/%s/import-ap", arUUID), importAp)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.ImportAp]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(importAp.Href, response.Data.Href)
	suite.Equal(importAp.Remarks, response.Data.Remarks)
}

// Test Create Result endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestCreateResult() {
	arUUID := suite.createBasicAssessmentResults()

	testResult := oscaltypes.Result{
		UUID:        uuid.New().String(),
		Title:       "Test Result",
		Description: "Test result description",
		Start:       time.Now(),
		ReviewedControls: oscaltypes.ReviewedControls{
			Description: "Controls reviewed",
		},
	}

	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results", arUUID), testResult)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusCreated, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.Result]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(testResult.UUID, response.Data.UUID)
}

// Test Get Results endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestGetResults() {
	arUUID := suite.createBasicAssessmentResults()

	// Add an additional result
	testResult := oscaltypes.Result{
		UUID:        uuid.New().String(),
		Title:       "Additional Result",
		Description: "Additional result description",
		Start:       time.Now(),
		ReviewedControls: oscaltypes.ReviewedControls{
			Description: "Additional controls reviewed",
		},
	}
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results", arUUID), testResult)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Equal(http.StatusCreated, createRec.Code)

	rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/results", arUUID), nil)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataListResponse[oscaltypes.Result]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Len(response.Data, 2) // One from creation + one we just added
}

// Test Get single Result endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestGetResult() {
	arUUID := suite.createBasicAssessmentResults()

	// Create a result to get
	testResult := oscaltypes.Result{
		UUID:        uuid.New().String(),
		Title:       "Test Result",
		Description: "Test result description",
		Start:       time.Now(),
		ReviewedControls: oscaltypes.ReviewedControls{
			Description: "Controls reviewed",
		},
	}
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results", arUUID), testResult)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Equal(http.StatusCreated, createRec.Code)

	rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s", arUUID, testResult.UUID), nil)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.Result]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(testResult.UUID, response.Data.UUID)
}

// Test Update Result endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestUpdateResult() {
	arUUID := suite.createBasicAssessmentResults()

	// Create a result to update
	testResult := oscaltypes.Result{
		UUID:        uuid.New().String(),
		Title:       "Test Result",
		Description: "Test result description",
		Start:       time.Now(),
		ReviewedControls: oscaltypes.ReviewedControls{
			Description: "Controls reviewed",
		},
	}
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results", arUUID), testResult)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Equal(http.StatusCreated, createRec.Code)

	// Update the result
	testResult.Title = "Updated Result"
	rec, req := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s", arUUID, testResult.UUID), testResult)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.Result]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Updated Result", response.Data.Title)
}

// Test Delete Result endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestDeleteResult() {
	arUUID := suite.createBasicAssessmentResults()

	// Create a result to delete
	testResult := oscaltypes.Result{
		UUID:        uuid.New().String(),
		Title:       "Test Result",
		Description: "Test result description",
		Start:       time.Now(),
		ReviewedControls: oscaltypes.ReviewedControls{
			Description: "Controls reviewed",
		},
	}
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results", arUUID), testResult)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Equal(http.StatusCreated, createRec.Code)

	rec, req := suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s", arUUID, testResult.UUID), nil)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusNoContent, rec.Code)

	// Verify deletion
	getRec, getReq := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s", arUUID, testResult.UUID), nil)
	suite.server.E().ServeHTTP(getRec, getReq)
	suite.Equal(http.StatusNotFound, getRec.Code)
}

// Test Create with invalid UUID
func (suite *AssessmentResultsApiIntegrationSuite) TestCreateAssessmentResultsWithInvalidUUID() {
	invalidAR := oscaltypes.AssessmentResults{
		UUID: "invalid-uuid",
		Metadata: oscaltypes.Metadata{
			Title:   "Test",
			Version: "1.0.0",
		},
		ImportAp: oscaltypes.ImportAp{
			Href: "https://example.com/ap.json",
		},
		Results: []oscaltypes.Result{},
	}
	rec, req := suite.createRequest(http.MethodPost, "/api/oscal/assessment-results", invalidAR)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusBadRequest, rec.Code)
}

// Test Create without required fields
func (suite *AssessmentResultsApiIntegrationSuite) TestCreateAssessmentResultsWithoutRequiredFields() {
	invalidAR := oscaltypes.AssessmentResults{
		UUID: uuid.New().String(),
		// Missing metadata
	}
	rec, req := suite.createRequest(http.MethodPost, "/api/oscal/assessment-results", invalidAR)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusBadRequest, rec.Code)
}

// Test Get non-existent assessment results
func (suite *AssessmentResultsApiIntegrationSuite) TestGetNonExistentAssessmentResults() {
	nonExistentID := uuid.New().String()
	rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s", nonExistentID), nil)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusNotFound, rec.Code)
}

// Test Update non-existent assessment results
func (suite *AssessmentResultsApiIntegrationSuite) TestUpdateNonExistentAssessmentResults() {
	nonExistentID := uuid.New().String()
	ar := oscaltypes.AssessmentResults{
		UUID: nonExistentID,
		Metadata: oscaltypes.Metadata{
			Title:   "Test",
			Version: "1.0.0",
		},
		ImportAp: oscaltypes.ImportAp{
			Href: "https://example.com/ap.json",
		},
		Results: []oscaltypes.Result{
			{
				UUID:        uuid.New().String(),
				Title:       "Test Result",
				Description: "Test result description",
				Start:       time.Now(),
				ReviewedControls: oscaltypes.ReviewedControls{
					Description: "Controls reviewed",
				},
			},
		},
	}
	rec, req := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-results/%s", nonExistentID), ar)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusNotFound, rec.Code)
}

// Test Back Matter endpoints

// Test Get Back Matter endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestGetBackMatter() {
	arUUID := suite.createBasicAssessmentResults()

	rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter", arUUID), nil)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.BackMatter]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.NotNil(response.Data.Resources)
	suite.Len(*response.Data.Resources, 0) // Should be empty initially
}

// Test Create Back Matter endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestCreateBackMatter() {
	arUUID := suite.createBasicAssessmentResults()

	backMatter := oscaltypes.BackMatter{
		Resources: &[]oscaltypes.Resource{
			{
				UUID:        uuid.New().String(),
				Title:       "Test Resource",
				Description: "Test resource description",
			},
		},
	}

	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter", arUUID), backMatter)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusCreated, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.BackMatter]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.NotNil(response.Data.Resources)
	suite.Len(*response.Data.Resources, 1)
}

// Test Create Back Matter with invalid resource UUID
func (suite *AssessmentResultsApiIntegrationSuite) TestCreateBackMatterWithInvalidResourceUUID() {
	arUUID := suite.createBasicAssessmentResults()

	// Test with missing UUID
	backMatter := oscaltypes.BackMatter{
		Resources: &[]oscaltypes.Resource{
			{
				Title:       "Test Resource",
				Description: "Test resource description",
				// UUID is missing
			},
		},
	}

	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter", arUUID), backMatter)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusBadRequest, rec.Code)

	// Test with invalid UUID format
	backMatter = oscaltypes.BackMatter{
		Resources: &[]oscaltypes.Resource{
			{
				UUID:        "invalid-uuid",
				Title:       "Test Resource",
				Description: "Test resource description",
			},
		},
	}

	rec, req = suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter", arUUID), backMatter)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusBadRequest, rec.Code)
}

// Test Update Back Matter endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestUpdateBackMatter() {
	arUUID := suite.createBasicAssessmentResults()

	// First create back matter
	backMatter := oscaltypes.BackMatter{
		Resources: &[]oscaltypes.Resource{
			{
				UUID:        uuid.New().String(),
				Title:       "Test Resource",
				Description: "Test resource description",
			},
		},
	}
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter", arUUID), backMatter)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Equal(http.StatusCreated, createRec.Code)

	// Update back matter
	updatedBackMatter := oscaltypes.BackMatter{
		Resources: &[]oscaltypes.Resource{
			{
				UUID:        uuid.New().String(),
				Title:       "Updated Resource",
				Description: "Updated resource description",
			},
			{
				UUID:        uuid.New().String(),
				Title:       "Second Resource",
				Description: "Second resource description",
			},
		},
	}

	rec, req := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter", arUUID), updatedBackMatter)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.BackMatter]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.NotNil(response.Data.Resources)
	suite.Len(*response.Data.Resources, 2)
}

// Test Delete Back Matter endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestDeleteBackMatter() {
	arUUID := suite.createBasicAssessmentResults()

	// First create back matter
	backMatter := oscaltypes.BackMatter{
		Resources: &[]oscaltypes.Resource{
			{
				UUID:        uuid.New().String(),
				Title:       "Test Resource",
				Description: "Test resource description",
			},
		},
	}
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter", arUUID), backMatter)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Equal(http.StatusCreated, createRec.Code)

	// Delete back matter
	rec, req := suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter", arUUID), nil)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusNoContent, rec.Code)

	// Verify deletion
	getRec, getReq := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter", arUUID), nil)
	suite.server.E().ServeHTTP(getRec, getReq)
	suite.Equal(http.StatusOK, getRec.Code)
	var getResponse handler.GenericDataResponse[oscaltypes.BackMatter]
	err := json.Unmarshal(getRec.Body.Bytes(), &getResponse)
	suite.NoError(err)
	suite.NotNil(getResponse.Data.Resources)
	suite.Len(*getResponse.Data.Resources, 0) // Should be empty after deletion
}

// Test Get Back Matter Resources endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestGetBackMatterResources() {
	arUUID := suite.createBasicAssessmentResults()

	// First create back matter with resources
	backMatter := oscaltypes.BackMatter{
		Resources: &[]oscaltypes.Resource{
			{
				UUID:        uuid.New().String(),
				Title:       "Resource 1",
				Description: "Resource 1 description",
			},
			{
				UUID:        uuid.New().String(),
				Title:       "Resource 2",
				Description: "Resource 2 description",
			},
		},
	}
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter", arUUID), backMatter)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Equal(http.StatusCreated, createRec.Code)
	
	// Verify the back matter was created with resources
	var createResponse handler.GenericDataResponse[oscaltypes.BackMatter]
	err := json.Unmarshal(createRec.Body.Bytes(), &createResponse)
	suite.NoError(err)
	if err != nil || createResponse.Data.Resources == nil {
		suite.T().Logf("Create back matter response body: %s", createRec.Body.String())
	}
	suite.NotNil(createResponse.Data.Resources)
	if createResponse.Data.Resources != nil {
		suite.T().Logf("Created back matter has %d resources", len(*createResponse.Data.Resources))
	}
	suite.Len(*createResponse.Data.Resources, 2)

	// Get resources
	rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter/resources", arUUID), nil)
	suite.server.E().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		suite.T().Logf("Get resources response: %d, body: %s", rec.Code, rec.Body.String())
	}
	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataListResponse[oscaltypes.Resource]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.T().Logf("Get resources returned %d items", len(response.Data))
	suite.Len(response.Data, 2)
}

// Test Create Back Matter Resource endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestCreateBackMatterResource() {
	arUUID := suite.createBasicAssessmentResults()

	resource := oscaltypes.Resource{
		UUID:        uuid.New().String(),
		Title:       "New Resource",
		Description: "New resource description",
	}

	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter/resources", arUUID), resource)
	suite.server.E().ServeHTTP(rec, req)

	suite.Equal(http.StatusCreated, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.Resource]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(resource.UUID, response.Data.UUID)
	suite.Equal(resource.Title, response.Data.Title)
}

// Test Update Back Matter Resource endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestUpdateBackMatterResource() {
	arUUID := suite.createBasicAssessmentResults()

	// First create a resource
	resource := oscaltypes.Resource{
		UUID:        uuid.New().String(),
		Title:       "Original Resource",
		Description: "Original description",
	}
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter/resources", arUUID), resource)
	suite.server.E().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		suite.T().Logf("Create resource response: %d, body: %s", createRec.Code, createRec.Body.String())
	}
	suite.Equal(http.StatusCreated, createRec.Code)

	// Parse the response to get the actual resource ID
	var createResponse handler.GenericDataResponse[oscaltypes.Resource]
	err := json.Unmarshal(createRec.Body.Bytes(), &createResponse)
	suite.NoError(err)
	resourceID := createResponse.Data.UUID
	
	// Update the resource
	resource.Title = "Updated Resource"
	resource.Description = "Updated description"
	rec, req := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter/resources/%s", arUUID, resourceID), resource)
	suite.server.E().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		suite.T().Logf("Update resource response: %d, body: %s", rec.Code, rec.Body.String())
	}
	suite.Equal(http.StatusOK, rec.Code)
	var response handler.GenericDataResponse[oscaltypes.Resource]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Updated Resource", response.Data.Title)
	suite.Equal("Updated description", response.Data.Description)
}

// Test Delete Back Matter Resource endpoint
func (suite *AssessmentResultsApiIntegrationSuite) TestDeleteBackMatterResource() {
	arUUID := suite.createBasicAssessmentResults()

	// First create a resource
	resource := oscaltypes.Resource{
		UUID:        uuid.New().String(),
		Title:       "Resource to Delete",
		Description: "This will be deleted",
	}
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter/resources", arUUID), resource)
	suite.server.E().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		suite.T().Logf("Create resource response: %d, body: %s", createRec.Code, createRec.Body.String())
	}
	suite.Equal(http.StatusCreated, createRec.Code)

	// Parse the response to get the actual resource ID
	var createResponse handler.GenericDataResponse[oscaltypes.Resource]
	err := json.Unmarshal(createRec.Body.Bytes(), &createResponse)
	suite.NoError(err)
	resourceID := createResponse.Data.UUID
	
	// Delete the resource
	rec, req := suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter/resources/%s", arUUID, resourceID), nil)
	suite.server.E().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		suite.T().Logf("Delete resource response: %d, body: %s", rec.Code, rec.Body.String())
	}
	suite.Equal(http.StatusNoContent, rec.Code)

	// Verify deletion
	getRec, getReq := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter/resources", arUUID), nil)
	suite.server.E().ServeHTTP(getRec, getReq)
	suite.Equal(http.StatusOK, getRec.Code)
	var getResponse handler.GenericDataListResponse[oscaltypes.Resource]
	err = json.Unmarshal(getRec.Body.Bytes(), &getResponse)
	suite.NoError(err)
	suite.Len(getResponse.Data, 0) // Should be empty after deletion
}

// Test Create Resource with invalid UUID
func (suite *AssessmentResultsApiIntegrationSuite) TestCreateResourceWithInvalidUUID() {
	arUUID := suite.createBasicAssessmentResults()

	resource := oscaltypes.Resource{
		UUID:        "invalid-uuid",
		Title:       "Test Resource",
		Description: "Test description",
	}

	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter/resources", arUUID), resource)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusBadRequest, rec.Code)
}

// Test Update non-existent back matter
func (suite *AssessmentResultsApiIntegrationSuite) TestUpdateNonExistentBackMatter() {
	arUUID := suite.createBasicAssessmentResults()

	backMatter := oscaltypes.BackMatter{
		Resources: &[]oscaltypes.Resource{
			{
				UUID:        uuid.New().String(),
				Title:       "Test Resource",
				Description: "Test description",
			},
		},
	}

	rec, req := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter", arUUID), backMatter)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusNotFound, rec.Code)
}

// Test Delete non-existent back matter
func (suite *AssessmentResultsApiIntegrationSuite) TestDeleteNonExistentBackMatter() {
	arUUID := suite.createBasicAssessmentResults()

	rec, req := suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-results/%s/back-matter", arUUID), nil)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusNotFound, rec.Code)
}

// Result sub-endpoints tests

// Helper function to create a result with sub-resources
func (suite *AssessmentResultsApiIntegrationSuite) createResultWithSubResources(arUUID string) string {
	// First create a result
	result := oscaltypes.Result{
		UUID:        uuid.New().String(),
		Title:       "Test Result",
		Description: "Test result description",
		Start:       time.Now(),
		ReviewedControls: oscaltypes.ReviewedControls{
			ControlSelections: []oscaltypes.AssessedControls{
				{
					IncludeControls: &[]oscaltypes.AssessedControlsSelectControlById{
						{ControlId: "ac-1"},
					},
				},
			},
		},
	}

	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results", arUUID), result)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusCreated, rec.Code)

	var resp handler.GenericDataResponse[oscaltypes.Result]
	suite.NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp.Data.UUID
}

// Test Observations CRUD
func (suite *AssessmentResultsApiIntegrationSuite) TestResultObservationsCRUD() {
	arUUID := suite.createBasicAssessmentResults()
	resultUUID := suite.createResultWithSubResources(arUUID)

	// Create observation
	observation := oscaltypes.Observation{
		UUID:        uuid.New().String(),
		Description: "Test observation",
		Methods:     []string{"TEST", "INTERVIEW"},
		Types:       &[]string{"finding"},
		Subjects: &[]oscaltypes.SubjectReference{
			{
				SubjectUuid: uuid.New().String(),
				Type:        "component",
			},
		},
		Collected: time.Now(),
	}

	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/observations", arUUID, resultUUID), observation)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusCreated, rec.Code)

	var createResp handler.GenericDataResponse[oscaltypes.Observation]
	suite.NoError(json.Unmarshal(rec.Body.Bytes(), &createResp))
	obsUUID := createResp.Data.UUID

	// Get observations
	rec, req = suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/observations", arUUID, resultUUID), nil)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusOK, rec.Code)

	var listResp handler.GenericDataListResponse[oscaltypes.Observation]
	suite.NoError(json.Unmarshal(rec.Body.Bytes(), &listResp))
	suite.Len(listResp.Data, 1)

	// Update observation
	observation.Description = "Updated observation"
	rec, req = suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/observations/%s", arUUID, resultUUID, obsUUID), observation)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusOK, rec.Code)

	// Delete observation
	rec, req = suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/observations/%s", arUUID, resultUUID, obsUUID), nil)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusNoContent, rec.Code)
}

// Test Risks CRUD
func (suite *AssessmentResultsApiIntegrationSuite) TestResultRisksCRUD() {
	arUUID := suite.createBasicAssessmentResults()
	resultUUID := suite.createResultWithSubResources(arUUID)

	// Create risk
	risk := oscaltypes.Risk{
		UUID:        uuid.New().String(),
		Title:       "Test Risk",
		Description: "Test risk description",
		Statement:   "Risk statement",
		Status:      "open",
		RiskLog: &oscaltypes.RiskLog{
			Entries: []oscaltypes.RiskLogEntry{
				{
					UUID:        uuid.New().String(),
					Title:       "Initial Entry",
					Start:       time.Now(),
					Description: "Initial risk log entry",
				},
			},
		},
	}

	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/risks", arUUID, resultUUID), risk)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusCreated, rec.Code)

	var createResp handler.GenericDataResponse[oscaltypes.Risk]
	suite.NoError(json.Unmarshal(rec.Body.Bytes(), &createResp))
	riskUUID := createResp.Data.UUID

	// Get risks
	rec, req = suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/risks", arUUID, resultUUID), nil)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusOK, rec.Code)

	var listResp handler.GenericDataListResponse[oscaltypes.Risk]
	suite.NoError(json.Unmarshal(rec.Body.Bytes(), &listResp))
	suite.Len(listResp.Data, 1)

	// Update risk
	risk.Title = "Updated Risk"
	rec, req = suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/risks/%s", arUUID, resultUUID, riskUUID), risk)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusOK, rec.Code)

	// Delete risk
	rec, req = suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/risks/%s", arUUID, resultUUID, riskUUID), nil)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusNoContent, rec.Code)
}

// Test Findings CRUD
func (suite *AssessmentResultsApiIntegrationSuite) TestResultFindingsCRUD() {
	arUUID := suite.createBasicAssessmentResults()
	resultUUID := suite.createResultWithSubResources(arUUID)

	// Create finding
	finding := oscaltypes.Finding{
		UUID:        uuid.New().String(),
		Title:       "Test Finding",
		Description: "Test finding description",
		Target: oscaltypes.FindingTarget{
			Type:               "objective-id",
			TargetId:           "ac-1_obj.1",
			Status: oscaltypes.ObjectiveStatus{
				State:  "not-satisfied",
				Reason: "Test reason",
			},
		},
	}

	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/findings", arUUID, resultUUID), finding)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusCreated, rec.Code)

	var createResp handler.GenericDataResponse[oscaltypes.Finding]
	suite.NoError(json.Unmarshal(rec.Body.Bytes(), &createResp))
	findingUUID := createResp.Data.UUID

	// Get findings
	rec, req = suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/findings", arUUID, resultUUID), nil)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusOK, rec.Code)

	var listResp handler.GenericDataListResponse[oscaltypes.Finding]
	suite.NoError(json.Unmarshal(rec.Body.Bytes(), &listResp))
	suite.Len(listResp.Data, 1)

	// Update finding
	finding.Title = "Updated Finding"
	rec, req = suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/findings/%s", arUUID, resultUUID, findingUUID), finding)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusOK, rec.Code)

	// Delete finding
	rec, req = suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/findings/%s", arUUID, resultUUID, findingUUID), nil)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusNoContent, rec.Code)
}

// Test Attestations CRUD
func (suite *AssessmentResultsApiIntegrationSuite) TestResultAttestationsCRUD() {
	arUUID := suite.createBasicAssessmentResults()
	resultUUID := suite.createResultWithSubResources(arUUID)

	// Create attestation
	attestation := oscaltypes.AttestationStatements{
		Parts: []oscaltypes.AssessmentPart{
			{
				UUID:  uuid.New().String(),
				Name:  "attestation",
				Ns:    "https://fedramp.gov/ns/oscal",
				Class: "fedramp",
				Title: "Attestation Statement",
				Prose: "I attest to the accuracy of this assessment.",
			},
		},
		ResponsibleParties: &[]oscaltypes.ResponsibleParty{
			{
				RoleId: "prepared-by",
				PartyUuids: []string{
					uuid.New().String(),
				},
			},
		},
	}

	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/attestations", arUUID, resultUUID), attestation)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusCreated, rec.Code)
	
	// Log the response
	if rec.Code != http.StatusCreated {
		suite.T().Logf("Create attestation response: %s", rec.Body.String())
	}

	// Get the created attestation ID from the database since AttestationStatements doesn't have a UUID field
	var createdAttestation relational.Attestation
	err := suite.DB.Where("result_id = ?", resultUUID).First(&createdAttestation).Error
	suite.NoError(err)
	suite.NotNil(createdAttestation.ID)
	suite.T().Logf("Created attestation ID: %v", createdAttestation.ID)
	
	var attestationUUID string
	if createdAttestation.ID != nil {
		attestationUUID = (*createdAttestation.ID).String()
	}
	suite.NotEmpty(attestationUUID, "Attestation UUID should not be empty")
	suite.T().Logf("Attestation UUID to use for update/delete: %s", attestationUUID)

	// Get attestations
	rec, req = suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/attestations", arUUID, resultUUID), nil)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusOK, rec.Code)

	var listResp handler.GenericDataListResponse[oscaltypes.AttestationStatements]
	suite.NoError(json.Unmarshal(rec.Body.Bytes(), &listResp))
	suite.Len(listResp.Data, 1)

	// Update attestation
	attestation.Parts[0].Prose = "Updated attestation statement"
	rec, req = suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/attestations/%s", arUUID, resultUUID, attestationUUID), attestation)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusOK, rec.Code)

	// Delete attestation
	rec, req = suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/attestations/%s", arUUID, resultUUID, attestationUUID), nil)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusNoContent, rec.Code)
}

// Test validation errors
func (suite *AssessmentResultsApiIntegrationSuite) TestResultSubResourcesValidation() {
	arUUID := suite.createBasicAssessmentResults()
	resultUUID := suite.createResultWithSubResources(arUUID)

	// Test invalid observation (missing required fields)
	invalidObs := oscaltypes.Observation{
		UUID: "invalid-uuid",
		// Missing required description and methods
	}
	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/observations", arUUID, resultUUID), invalidObs)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusBadRequest, rec.Code)

	// Test invalid risk (missing required fields)
	invalidRisk := oscaltypes.Risk{
		UUID: uuid.New().String(),
		// Missing required title, description, statement, status
	}
	rec, req = suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/risks", arUUID, resultUUID), invalidRisk)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusBadRequest, rec.Code)

	// Test invalid finding (missing required fields)
	invalidFinding := oscaltypes.Finding{
		UUID:  uuid.New().String(),
		Title: "Test Finding",
		// Missing required description and target
		Target: oscaltypes.FindingTarget{},
	}
	rec, req = suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/findings", arUUID, resultUUID), invalidFinding)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusBadRequest, rec.Code)

	// Test invalid attestation (missing required parts)
	invalidAttestation := oscaltypes.AttestationStatements{
		// Missing required parts
	}
	rec, req = suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-results/%s/results/%s/attestations", arUUID, resultUUID), invalidAttestation)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusBadRequest, rec.Code)
}