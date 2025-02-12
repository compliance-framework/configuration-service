//go:build integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/service"
	"github.com/compliance-framework/configuration-service/internal/tests"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

func TestPlanApi(t *testing.T) {
	suite.Run(t, new(PlanIntegrationSuite))
}

type PlanIntegrationSuite struct {
	tests.IntegrationTestSuite
}

func (suite *PlanIntegrationSuite) TestCreatePlan() {
	suite.Run("A plan can be created through the API", func() {
		logger, _ := zap.NewProduction()
		planHandler := NewPlansHandler(logger.Sugar(), service.NewPlansService(suite.MongoDatabase))

		server := api.NewServer(context.Background(), logger.Sugar())
		planHandler.Register(server.API().Group("/plan"))

		reqBody, _ := json.Marshal(map[string]interface{}{
			"title": "Some Plan",
			"filter": map[string]interface{}{
				"scope": map[string]interface{}{
					"condition": map[string]string{
						"label":    "foo",
						"operator": "=",
						"value":    "bar",
					},
				},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/plan", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		rec := httptest.NewRecorder()
		server.E().ServeHTTP(rec, req)

		assert.Equal(suite.T(), http.StatusCreated, rec.Code, "Expected status 201 Created")

		response := &GenericDataResponse[PlanResponse]{}
		assert.NoError(suite.T(), json.Unmarshal(rec.Body.Bytes(), response), "Failed to parse response from CreatePlan")
		expectedJson, err := json.Marshal(map[string]interface{}{
			"data": map[string]interface{}{
				"uuid":  response.Data.UUID,
				"title": "Some Plan",
				"filter": map[string]interface{}{
					"scope": map[string]interface{}{
						"condition": map[string]string{
							"label":    "foo",
							"operator": "=",
							"value":    "bar",
						},
					},
				},
			},
		})
		if assert.NoError(suite.T(), err) {
			assert.JSONEq(suite.T(), string(expectedJson), rec.Body.String())
		}
	})
}

func (suite *PlanIntegrationSuite) TestCreateAndGetPlan() {
	suite.Run("A plan can be created, and then fetched through the API", func() {
		logger, _ := zap.NewProduction()
		planHandler := NewPlansHandler(logger.Sugar(), service.NewPlansService(suite.MongoDatabase))

		server := api.NewServer(context.Background(), logger.Sugar())
		planHandler.Register(server.API().Group("/plan"))

		// Create a plan
		rec := httptest.NewRecorder()
		reqBody, _ := json.Marshal(map[string]interface{}{
			"title": "Some Plan",
			"filter": map[string]interface{}{
				"scope": map[string]interface{}{
					"condition": map[string]string{
						"label":    "foo",
						"operator": "=",
						"value":    "bar",
					},
				},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/plan", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		server.E().ServeHTTP(rec, req)

		// Assert that the plan was created successfully
		assert.Equal(suite.T(), http.StatusCreated, rec.Code)

		response := &GenericDataResponse[PlanResponse]{}
		assert.NoError(suite.T(), json.Unmarshal(rec.Body.Bytes(), response), "Failed to parse response from CreatePlan")
		assert.NotEmpty(suite.T(), response.Data.UUID, "Response ID should not be empty")

		// Fetch the created plan
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/api/plan/%s", response.Data.UUID),
			nil,
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		server.E().ServeHTTP(rec, req)

		// Assert that the plan was fetched successfully
		assert.Equal(suite.T(), http.StatusOK, rec.Code)

		getResponse := &GenericDataResponse[PlanResponse]{}
		assert.NoError(suite.T(), json.Unmarshal(rec.Body.Bytes(), getResponse), "Failed to parse response from GetPlan")
		expectedJson, err := json.Marshal(map[string]interface{}{
			"data": map[string]interface{}{
				"uuid":  getResponse.Data.UUID,
				"title": "Some Plan",
				"filter": map[string]interface{}{
					"scope": map[string]interface{}{
						"condition": map[string]string{
							"label":    "foo",
							"operator": "=",
							"value":    "bar",
						},
					},
				},
			},
		})
		if assert.NoError(suite.T(), err) {
			assert.JSONEq(suite.T(), string(expectedJson), rec.Body.String())
		}

	})
}
