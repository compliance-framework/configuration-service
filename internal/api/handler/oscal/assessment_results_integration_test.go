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

	suite.Equal(http.StatusOK, rec.Code)
	var response struct {
		Message string `json:"message"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Assessment results deleted successfully", response.Message)

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

	suite.Equal(http.StatusOK, rec.Code)
	var response struct {
		Message string `json:"message"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("Result deleted successfully", response.Message)

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