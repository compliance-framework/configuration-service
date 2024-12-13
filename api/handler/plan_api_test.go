//go:build integration

package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/compliance-framework/configuration-service/domain"
	"github.com/compliance-framework/configuration-service/event/bus"
	"github.com/compliance-framework/configuration-service/service"
	"github.com/compliance-framework/configuration-service/tests"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPlanApi(t *testing.T) {
	suite.Run(t, new(PlanIntegrationSuite))
}

type PlanIntegrationSuite struct {
	tests.IntegrationTestSuite
}

func (suite *PlanIntegrationSuite) TestCreatePlan() {
	suite.Run("A plan can be created through the API", func() {
		// Setup
		e := echo.New()
		reqBody, _ := json.Marshal(map[string]interface{}{
			"title": "Some Plan",
		})

		req := httptest.NewRequest(
			http.MethodPost,
			"/plan",
			bytes.NewReader(reqBody),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		logger, _ := zap.NewProduction()
		planHandler := NewPlanHandler(logger.Sugar(), service.NewPlanService(bus.Publish))

		if assert.NoError(suite.T(), planHandler.CreatePlan(c)) {
			assert.Equal(suite.T(), http.StatusCreated, rec.Code)
			assert.NoError(suite.T(), json.Unmarshal(rec.Body.Bytes(), &idResponse{}))
		}
	})
}

func (suite *PlanIntegrationSuite) TestCreateAndGetPlan() {
	suite.Run("A plan can be created, and then fetched through the API", func() {
		logger, _ := zap.NewProduction()
		planHandler := NewPlanHandler(logger.Sugar(), service.NewPlanService(bus.Publish))
		e := echo.New()

		// Setup
		reqBody, _ := json.Marshal(map[string]interface{}{
			"title": "Some Plan",
		})
		req := httptest.NewRequest(
			http.MethodPost,
			"/plan",
			bytes.NewReader(reqBody),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		response := &idResponse{}
		if assert.NoError(suite.T(), planHandler.CreatePlan(c)) {
			assert.Equal(suite.T(), http.StatusCreated, rec.Code)
			assert.NoError(suite.T(), json.Unmarshal(rec.Body.Bytes(), response))
		}

		// Now check if we can fetch that same plan
		req = httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/%s", response.Id),
			nil,
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec = httptest.NewRecorder()
		c = e.NewContext(req, rec)
		c.SetPath("/plan/:id")
		c.SetParamNames("id")
		c.SetParamValues(response.Id)

		getResponse := &domain.Plan{}
		if assert.NoError(suite.T(), planHandler.GetPlan(c)) {
			assert.Equal(suite.T(), http.StatusOK, rec.Code)
			assert.NoError(suite.T(), json.Unmarshal(rec.Body.Bytes(), getResponse))
			assert.Equal(suite.T(), response.Id, getResponse.Id.Hex())
		}
	})
}
