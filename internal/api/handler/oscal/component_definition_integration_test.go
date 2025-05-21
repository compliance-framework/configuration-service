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

	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/api/handler"
	"github.com/compliance-framework/configuration-service/internal/tests"
	oscaltypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

func TestComponentDefinitionApi(t *testing.T) {
	fmt.Println("Starting Component Definition API tests")
	suite.Run(t, new(ComponentDefinitionApiIntegrationSuite))
}

type ComponentDefinitionApiIntegrationSuite struct {
	tests.IntegrationTestSuite
}

func (suite *ComponentDefinitionApiIntegrationSuite) SetupSuite() {
	fmt.Println("Setting up Component Definition API test suite")
	suite.IntegrationTestSuite.SetupSuite()
}

func (suite *ComponentDefinitionApiIntegrationSuite) TestGetComponentDefinitions() {
	fmt.Println("Running TestGetComponentDefinitions")
	logger, _ := zap.NewDevelopment()

	// Reset database
	err := suite.MongoDatabase.Drop(context.Background())
	suite.Require().NoError(err, "Failed to drop database")
	fmt.Println("Database dropped successfully")

	// Setup server
	server := api.NewServer(context.Background(), logger.Sugar())
	RegisterHandlers(server, logger.Sugar(), suite.MongoDatabase)
	fmt.Println("Server initialized")

	// Test data
	componentDefs := []oscaltypes.ComponentDefinition{
		{
			UUID: "D20DB907-B87D-4D12-8760-D36FDB7A1B31",
			Metadata: oscaltypes.Metadata{
				Title: "Component Definition 1",
			},
			Components: &[]oscaltypes.DefinedComponent{
				{
					UUID:  "COMP-1",
					Title: "Component 1",
					Type:  "service",
				},
			},
		},
		{
			UUID: "D20DB907-B87D-4D12-8760-D36FDB7A1B32",
			Metadata: oscaltypes.Metadata{
				Title: "Component Definition 2",
			},
			Components: &[]oscaltypes.DefinedComponent{
				{
					UUID:  "COMP-2",
					Title: "Component 2",
					Type:  "service",
				},
			},
		},
	}

	// Create test component definitions
	for _, compDef := range componentDefs {
		reqBody, err := json.Marshal(compDef)
		suite.Require().NoError(err, "Failed to marshal component definition")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/oscal/component-definitions", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		server.E().ServeHTTP(rec, req)

		assert.Equal(suite.T(), http.StatusCreated, rec.Code, "Failed to create component definition")
		response := &handler.GenericDataResponse[oscaltypes.ComponentDefinition]{}
		err = json.Unmarshal(rec.Body.Bytes(), response)
		suite.Require().NoError(err, "Failed to unmarshal create response")
		fmt.Printf("Created component definition with UUID: %s\n", compDef.UUID)
	}

	// Test GET requests
	fmt.Println("Testing GET requests")

	// Test successful retrieval
	suite.Run("Successfully retrieves component definitions", func() {
		// Get first component definition
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/oscal/component-definitions/D20DB907-B87D-4D12-8760-D36FDB7A1B31", nil)
		server.E().ServeHTTP(rec, req)
		assert.Equal(suite.T(), http.StatusOK, rec.Code, "Failed to get first component definition")

		response := &handler.GenericDataResponse[oscaltypes.ComponentDefinition]{}
		err = json.Unmarshal(rec.Body.Bytes(), response)
		suite.Require().NoError(err, "Failed to unmarshal first GET response")
		suite.Equal("Component Definition 1", response.Data.Metadata.Title)
		suite.Require().NotNil(response.Data.Components)
		suite.Equal("Component 1", (*response.Data.Components)[0].Title)
		fmt.Println("First component definition retrieved successfully")

		// Get second component definition
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/api/oscal/component-definitions/D20DB907-B87D-4D12-8760-D36FDB7A1B32", nil)
		server.E().ServeHTTP(rec, req)
		assert.Equal(suite.T(), http.StatusOK, rec.Code, "Failed to get second component definition")

		response = &handler.GenericDataResponse[oscaltypes.ComponentDefinition]{}
		err = json.Unmarshal(rec.Body.Bytes(), response)
		suite.Require().NoError(err, "Failed to unmarshal second GET response")
		suite.Equal("Component Definition 2", response.Data.Metadata.Title)
		suite.Require().NotNil(response.Data.Components)
		suite.Equal("Component 2", (*response.Data.Components)[0].Title)
		fmt.Println("Second component definition retrieved successfully")
	})

	// Test not found case
	suite.Run("Returns 404 for non-existent component", func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/oscal/component-definitions/D20DB907-B87D-4D12-8760-D36FDB7A1B33", nil)
		server.E().ServeHTTP(rec, req)
		assert.Equal(suite.T(), http.StatusNotFound, rec.Code, "Expected 404 for non-existent component")
	})

	// Test invalid UUID
	suite.Run("Returns 400 for invalid UUID", func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/oscal/component-definitions/invalid-uuid", nil)
		server.E().ServeHTTP(rec, req)
		assert.Equal(suite.T(), http.StatusBadRequest, rec.Code, "Expected 400 for invalid UUID")
	})

	fmt.Println("TestGetComponentDefinitions completed successfully")
}
