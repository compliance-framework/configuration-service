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

	"github.com/compliance-framework/api/internal/api"
	"github.com/compliance-framework/api/internal/api/handler"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/tests"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

func TestAssessmentPlanApi(t *testing.T) {
	suite.Run(t, new(AssessmentPlanApiIntegrationSuite))
}

type AssessmentPlanApiIntegrationSuite struct {
	tests.IntegrationTestSuite
	server *api.Server
	logger *zap.SugaredLogger
}

func (suite *AssessmentPlanApiIntegrationSuite) SetupSuite() {
	suite.IntegrationTestSuite.SetupSuite()

	logConf := zap.NewDevelopmentConfig()
	logConf.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	logger, _ := logConf.Build()
	suite.logger = logger.Sugar()
	suite.server = api.NewServer(context.Background(), suite.logger, suite.Config)
	RegisterHandlers(suite.server, suite.logger, suite.DB, suite.Config)
}

func (suite *AssessmentPlanApiIntegrationSuite) SetupTest() {
	err := suite.Migrator.Refresh()
	suite.Require().NoError(err)
}

func (suite *AssessmentPlanApiIntegrationSuite) TestAssessmentPlanCreate() {
	id := uuid.New()
	rec, req := suite.createRequest(
		http.MethodPost,
		"/api/oscal/assessment-plans",
		&oscalTypes_1_1_3.AssessmentPlan{
			UUID: id.String(),
			Metadata: oscalTypes_1_1_3.Metadata{
				Title:   "Test Assessment Plan",
				Version: "1.0.0",
			},
			ImportSsp: oscalTypes_1_1_3.ImportSsp{
				Href: "test-ssp-reference",
			},
		},
	)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusCreated, rec.Code)

	// Verify response
	var response handler.GenericDataResponse[*oscalTypes_1_1_3.AssessmentPlan]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Equal(id.String(), response.Data.UUID)
	suite.Equal("Test Assessment Plan", response.Data.Metadata.Title)
}

func (suite *AssessmentPlanApiIntegrationSuite) TestAssessmentPlanGet() {
	id := uuid.New()
	suite.DB.Create(&relational.AssessmentPlan{
		UUIDModel: relational.UUIDModel{
			ID: &id,
		},
		Metadata: relational.Metadata{
			Title: "Test Assessment Plan",
		},
	})

	// Test Get
	getRec, getReq := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s", id.String()), nil)
	suite.server.E().ServeHTTP(getRec, getReq)
	suite.Equal(http.StatusOK, getRec.Code)

	// Verify response
	var response handler.GenericDataResponse[*oscalTypes_1_1_3.AssessmentPlan]
	err := json.Unmarshal(getRec.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Equal(id.String(), response.Data.UUID)
	suite.Equal("Test Assessment Plan", response.Data.Metadata.Title)
}

func (suite *AssessmentPlanApiIntegrationSuite) TestAssessmentPlanList() {
	// Create multiple test plans
	for i := 0; i < 3; i++ {
		planID := uuid.New()
		testPlan := &oscalTypes_1_1_3.AssessmentPlan{
			UUID: planID.String(),
			Metadata: oscalTypes_1_1_3.Metadata{
				Title:   fmt.Sprintf("Test Plan %d", i+1),
				Version: "1.0.0",
			},
			ImportSsp: oscalTypes_1_1_3.ImportSsp{
				Href: "test-ssp-reference",
			},
		}

		createRec, createReq := suite.createRequest(http.MethodPost, "/api/oscal/assessment-plans", testPlan)
		suite.server.E().ServeHTTP(createRec, createReq)
		suite.Equal(http.StatusCreated, createRec.Code)
	}

	// Test List
	listRec, listReq := suite.createRequest(http.MethodGet, "/api/oscal/assessment-plans", nil)
	suite.server.E().ServeHTTP(listRec, listReq)
	suite.Equal(http.StatusOK, listRec.Code)

	// Verify response structure
	var response handler.GenericDataListResponse[oscalTypes_1_1_3.AssessmentPlan]
	err := json.Unmarshal(listRec.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Len(response.Data, 3)
}

func (suite *AssessmentPlanApiIntegrationSuite) TestAssessmentPlanUpdate() {
	// Create test plan first
	planID := uuid.New()
	testPlan := &oscalTypes_1_1_3.AssessmentPlan{
		UUID: planID.String(),
		Metadata: oscalTypes_1_1_3.Metadata{
			Title:   "Original Title",
			Version: "1.0.0",
		},
		ImportSsp: oscalTypes_1_1_3.ImportSsp{
			Href: "test-ssp-reference",
		},
	}

	// Create the plan
	createRec, createReq := suite.createRequest(http.MethodPost, "/api/oscal/assessment-plans", testPlan)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Equal(http.StatusCreated, createRec.Code)

	// Update the plan
	testPlan.Metadata.Title = "Updated Title"
	updateRec, updateReq := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-plans/%s", planID), testPlan)
	suite.server.E().ServeHTTP(updateRec, updateReq)
	suite.Equal(http.StatusOK, updateRec.Code)

	// Verify update
	var response handler.GenericDataResponse[*oscalTypes_1_1_3.AssessmentPlan]
	err := json.Unmarshal(updateRec.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Equal("Updated Title", response.Data.Metadata.Title)
}

func (suite *AssessmentPlanApiIntegrationSuite) TestAssessmentPlanDelete() {
	// Create test plan first
	planID := uuid.New()
	testPlan := &oscalTypes_1_1_3.AssessmentPlan{
		UUID: planID.String(),
		Metadata: oscalTypes_1_1_3.Metadata{
			Title:   "Test Plan to Delete",
			Version: "1.0.0",
		},
		ImportSsp: oscalTypes_1_1_3.ImportSsp{
			Href: "test-ssp-reference",
		},
	}

	// Create the plan
	createRec, createReq := suite.createRequest(http.MethodPost, "/api/oscal/assessment-plans", testPlan)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Equal(http.StatusCreated, createRec.Code)

	// Delete the plan
	deleteRec, deleteReq := suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-plans/%s", planID), nil)
	suite.server.E().ServeHTTP(deleteRec, deleteReq)
	suite.Equal(http.StatusNoContent, deleteRec.Code)

	// Verify deletion - should return 404
	getRec, getReq := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s", planID), nil)
	suite.server.E().ServeHTTP(getRec, getReq)
	suite.Equal(http.StatusNotFound, getRec.Code)
}

func (suite *AssessmentPlanApiIntegrationSuite) TestAssessmentPlanGetFull() {
	// Create test plan first
	planID := uuid.New()
	testPlan := &oscalTypes_1_1_3.AssessmentPlan{
		UUID: planID.String(),
		Metadata: oscalTypes_1_1_3.Metadata{
			Title:   "Test Plan with Full Data",
			Version: "1.0.0",
		},
		ImportSsp: oscalTypes_1_1_3.ImportSsp{
			Href: "test-ssp-reference",
		},
	}

	// Create the plan
	createRec, createReq := suite.createRequest(http.MethodPost, "/api/oscal/assessment-plans", testPlan)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Equal(http.StatusCreated, createRec.Code)

	// Test Get with full endpoint
	getRec, getReq := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/full", planID), nil)
	suite.server.E().ServeHTTP(getRec, getReq)
	suite.Equal(http.StatusOK, getRec.Code)

	// Verify response
	var response handler.GenericDataResponse[*oscalTypes_1_1_3.AssessmentPlan]
	err := json.Unmarshal(getRec.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Equal(planID.String(), response.Data.UUID)
}

func (suite *AssessmentPlanApiIntegrationSuite) TestAssessmentPlanValidationErrors() {
	suite.Run("CreateWithInvalidUUID", func() {
		invalidPlan := &oscalTypes_1_1_3.AssessmentPlan{
			UUID: "invalid-uuid",
			Metadata: oscalTypes_1_1_3.Metadata{
				Title:   "Test Plan",
				Version: "1.0.0",
			},
			ImportSsp: oscalTypes_1_1_3.ImportSsp{
				Href: "test-ssp-reference",
			},
		}

		rec, req := suite.createRequest(http.MethodPost, "/api/oscal/assessment-plans", invalidPlan)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusBadRequest, rec.Code)
	})

	suite.Run("CreateWithMissingTitle", func() {
		invalidPlan := &oscalTypes_1_1_3.AssessmentPlan{
			UUID: uuid.New().String(),
			Metadata: oscalTypes_1_1_3.Metadata{
				Version: "1.0.0",
			},
			ImportSsp: oscalTypes_1_1_3.ImportSsp{
				Href: "test-ssp-reference",
			},
		}

		rec, req := suite.createRequest(http.MethodPost, "/api/oscal/assessment-plans", invalidPlan)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusBadRequest, rec.Code)
	})

	suite.Run("CreateWithMissingImportSsp", func() {
		invalidPlan := &oscalTypes_1_1_3.AssessmentPlan{
			UUID: uuid.New().String(),
			Metadata: oscalTypes_1_1_3.Metadata{
				Title:   "Test Plan",
				Version: "1.0.0",
			},
			ImportSsp: oscalTypes_1_1_3.ImportSsp{
				Href: "", // Empty href
			},
		}

		rec, req := suite.createRequest(http.MethodPost, "/api/oscal/assessment-plans", invalidPlan)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusBadRequest, rec.Code)
	})
}

func (suite *AssessmentPlanApiIntegrationSuite) TestAssessmentPlanNotFound() {
	nonExistentID := uuid.New()

	suite.Run("GetNonExistent", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s", nonExistentID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusNotFound, rec.Code)
	})

	suite.Run("UpdateNonExistent", func() {
		testPlan := &oscalTypes_1_1_3.AssessmentPlan{
			UUID: nonExistentID.String(),
			Metadata: oscalTypes_1_1_3.Metadata{
				Title:   "Non-existent Plan",
				Version: "1.0.0",
			},
			ImportSsp: oscalTypes_1_1_3.ImportSsp{
				Href: "test-ssp-reference",
			},
		}

		rec, req := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-plans/%s", nonExistentID), testPlan)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusNotFound, rec.Code)
	})

	suite.Run("DeleteNonExistent", func() {
		rec, req := suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-plans/%s", nonExistentID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusNotFound, rec.Code)
	})
}

// TestAssessmentPlanSubResourceEndpoints tests all sub-resource endpoints
func (suite *AssessmentPlanApiIntegrationSuite) TestAssessmentPlanSubResourceEndpoints() {
	// Create a test plan first
	planID := uuid.New()
	testPlan := &oscalTypes_1_1_3.AssessmentPlan{
		UUID: planID.String(),
		Metadata: oscalTypes_1_1_3.Metadata{
			Title:   "Test Plan for Sub-Resources",
			Version: "1.0.0",
		},
		ImportSsp: oscalTypes_1_1_3.ImportSsp{
			Href: "test-ssp-reference",
		},
	}

	// Create the plan
	createRec, createReq := suite.createRequest(http.MethodPost, "/api/oscal/assessment-plans", testPlan)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Equal(http.StatusCreated, createRec.Code)

	suite.Run("GetMetadata", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/metadata", planID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code)

		var response handler.GenericDataResponse[*oscalTypes_1_1_3.Metadata]
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		suite.Require().NoError(err)
		suite.Equal("Test Plan for Sub-Resources", response.Data.Title)
		suite.Equal("1.0.0", response.Data.Version)
	})

	suite.Run("GetImportSsp", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/import-ssp", planID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code)

		var response handler.GenericDataResponse[*oscalTypes_1_1_3.ImportSsp]
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		suite.Require().NoError(err)
		suite.Equal("test-ssp-reference", response.Data.Href)
	})

	suite.Run("GetLocalDefinitions_NotFound", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/local-definitions", planID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusNotFound, rec.Code)
	})

	suite.Run("GetBackMatter_NotFound", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/back-matter", planID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusNotFound, rec.Code)
	})
}

// TestAssessmentPlanSubResourceEndpointsInvalidUUID tests sub-resource endpoints with invalid UUIDs
func (suite *AssessmentPlanApiIntegrationSuite) TestAssessmentPlanSubResourceEndpointsInvalidUUID() {
	invalidUUID := "invalid-uuid"

	suite.Run("GetMetadata_InvalidUUID", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/metadata", invalidUUID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusBadRequest, rec.Code)
	})

	suite.Run("GetImportSsp_InvalidUUID", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/import-ssp", invalidUUID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusBadRequest, rec.Code)
	})

	suite.Run("GetLocalDefinitions_InvalidUUID", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/local-definitions", invalidUUID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusBadRequest, rec.Code)
	})

	suite.Run("GetTermsAndConditions_InvalidUUID", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/terms-and-conditions", invalidUUID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusBadRequest, rec.Code)
	})

	suite.Run("GetBackMatter_InvalidUUID", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/back-matter", invalidUUID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusBadRequest, rec.Code)
	})
}

// TestAssessmentPlanSubResourceEndpointsNotFound tests sub-resource endpoints with non-existent assessment plans
func (suite *AssessmentPlanApiIntegrationSuite) TestAssessmentPlanSubResourceEndpointsNotFound() {
	nonExistentID := uuid.New()

	suite.Run("GetMetadata_PlanNotFound", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/metadata", nonExistentID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusNotFound, rec.Code)
	})

	suite.Run("GetImportSsp_PlanNotFound", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/import-ssp", nonExistentID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusNotFound, rec.Code)
	})

	suite.Run("GetLocalDefinitions_PlanNotFound", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/local-definitions", nonExistentID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusNotFound, rec.Code)
	})

	suite.Run("GetTermsAndConditions_PlanNotFound", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/terms-and-conditions", nonExistentID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusNotFound, rec.Code)
	})

	suite.Run("GetBackMatter_PlanNotFound", func() {
		rec, req := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/back-matter", nonExistentID), nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusNotFound, rec.Code)
	})
}

// Helper method to create a test request with Bearer token authentication
func (suite *AssessmentPlanApiIntegrationSuite) createRequest(method, path string, body any) (*httptest.ResponseRecorder, *http.Request) {
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
