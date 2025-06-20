//go:build integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/service"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/compliance-framework/configuration-service/internal/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestHeartbeatApi(t *testing.T) {
	suite.Run(t, new(HeartbeatApiIntegrationSuite))
}

type HeartbeatApiIntegrationSuite struct {
	tests.IntegrationTestSuite
}

func (suite *HeartbeatApiIntegrationSuite) TestHeartbeatCreate() {
	suite.T().Run("Heartbeat request validation", func(t *testing.T) {
		err := suite.Migrator.Refresh()
		suite.Require().NoError(err)

		// Create two catalogs with the same group ID structure
		heartbeat := HeartbeatCreateRequest{}
		logger, _ := zap.NewDevelopment()
		server := api.NewServer(context.Background(), logger.Sugar())
		RegisterHandlers(server, logger.Sugar(), suite.DB, suite.Config)
		rec := httptest.NewRecorder()
		reqBody, _ := json.Marshal(heartbeat)
		req := httptest.NewRequest(http.MethodPost, "/api/agent/heartbeat", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		server.E().ServeHTTP(rec, req)
		assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)
	})

	suite.T().Run("Heartbeat create", func(t *testing.T) {
		err := suite.Migrator.Refresh()
		suite.Require().NoError(err)

		// Create two catalogs with the same group ID structure
		heartbeat := HeartbeatCreateRequest{
			UUID:      uuid.New(),
			CreatedAt: time.Now(),
		}

		logger, _ := zap.NewDevelopment()
		server := api.NewServer(context.Background(), logger.Sugar())
		RegisterHandlers(server, logger.Sugar(), suite.DB, suite.Config)
		rec := httptest.NewRecorder()
		reqBody, _ := json.Marshal(heartbeat)
		req := httptest.NewRequest(http.MethodPost, "/api/agent/heartbeat", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		server.E().ServeHTTP(rec, req)
		assert.Equal(suite.T(), http.StatusCreated, rec.Code)

		var count int64
		// Counting users with specific names
		suite.DB.Model(&service.Heartbeat{}).Count(&count)
		suite.Equal(int64(1), count)
	})
}
