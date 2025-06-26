// go:build integration

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/tests"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

func TestAuthAPI(t *testing.T) {
	suite.Run(t, new(AuthAPIIntegrationSuite))
}

type AuthAPIIntegrationSuite struct {
	tests.IntegrationTestSuite
	logger *zap.SugaredLogger
	server *api.Server
}

type LoginResponse struct {
	Data struct {
		AuthToken string `json:"auth_token"`
	} `json:"data"`
}

type ErrorResponse struct {
	Data struct {
		Email []string `json:"email"`
	} `json:"data"`
}

func (suite *AuthAPIIntegrationSuite) SetupSuite() {
	fmt.Println("Setting up Component Definition API test suite")
	suite.IntegrationTestSuite.SetupSuite()

	// Setup logger and server once for all tests
	logger, _ := zap.NewDevelopment()
	suite.logger = logger.Sugar()
	suite.server = api.NewServer(context.Background(), suite.logger, suite.Config)
	RegisterHandlers(suite.server, suite.logger, suite.DB, suite.Config)
	fmt.Println("Server initialized")
}

func (suite *AuthAPIIntegrationSuite) TestLogin() {
	err := suite.IntegrationTestSuite.Migrator.Refresh()
	suite.Require().NoError(err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte(`{"email":"test@example.com","password":"Pa55w0rd"}`)))
	req.Header.Set("Content-Type", "application/json")
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusOK, rec.Code, "Expected status code 200 OK")

	var resp LoginResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	suite.Require().NoError(err)
	suite.NotEmpty(resp.Data.AuthToken, "Expected non-empty auth token")
}

func (suite *AuthAPIIntegrationSuite) TestLoginInvalidCredentials() {
	for _, testData := range []struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		{Email: "test@example.com", Password: "wrongPassword"},
		{Email: "invalid-email", Password: "Pa55w0rd"},
	} {
		payload, err := json.Marshal(testData)
		suite.Require().NoError(err, "Failed to marshal test data")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusUnauthorized, rec.Code, "Expected status code 401 Unauthorized")

		var response ErrorResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		suite.Require().NoError(err)
		suite.Len(response.Data.Email, 1, "Expected one validation error for email")
	}
}
