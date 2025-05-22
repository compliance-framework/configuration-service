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

	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/api/handler"
	"github.com/compliance-framework/configuration-service/internal/tests"
	oscaltypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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

func (suite *ComponentDefinitionApiIntegrationSuite) TestCreateCompleteComponentDefinition() {
	fmt.Println("Running TestCreateCompleteComponentDefinition")
	logger, _ := zap.NewDevelopment()

	// Reset database
	err := suite.Migrator.Refresh()
	suite.Require().NoError(err, "Failed to refresh database")
	fmt.Println("Database refreshed successfully")

	// Setup server
	server := api.NewServer(context.Background(), logger.Sugar())
	RegisterHandlers(server, logger.Sugar(), suite.DB)
	fmt.Println("Server initialized")

	suite.Run("Successfully creates a complete component definition", func() {
		// Generate UUIDs dynamically
		componentUUID := uuid.New().String()
		backMatterResourceUUID := uuid.New().String()
		componentImplUUID := uuid.New().String()
		implementedReqUUID := uuid.New().String()
		capabilityUUID := uuid.New().String()
		capabilityImplUUID := uuid.New().String()
		partyUUID := uuid.New().String()
		locationUUID := uuid.New().String()

		// Create test data using helper functions
		backMatterResource := createTestBackMatterResource(backMatterResourceUUID)
		component := createTestComponent(componentUUID, componentImplUUID, implementedReqUUID)
		capability := createTestCapability(capabilityUUID, capabilityImplUUID)
		metadata := createTestMetadata(partyUUID, locationUUID)

		// Create the full component definition
		completeCompDef := oscaltypes.ComponentDefinition{
			UUID:     uuid.New().String(),
			Metadata: metadata,
			ImportComponentDefinitions: &[]oscaltypes.ImportComponentDefinition{
				{
					Href: "https://example.com/components/base",
				},
			},
			Components:   &[]oscaltypes.DefinedComponent{component},
			Capabilities: &[]oscaltypes.Capability{capability},
			BackMatter: &oscaltypes.BackMatter{
				Resources: &[]oscaltypes.Resource{backMatterResource},
			},
		}

		// Marshal the component definition to JSON
		reqBody, err := json.Marshal(completeCompDef)
		suite.Require().NoError(err, "Failed to marshal complete component definition")

		// Send POST request
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/oscal/component-definitions", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		server.E().ServeHTTP(rec, req)

		// Check response
		suite.Equal(http.StatusCreated, rec.Code, "Failed to create complete component definition")

		// Unmarshal and verify response
		response := &handler.GenericDataResponse[oscaltypes.ComponentDefinition]{}
		err = json.Unmarshal(rec.Body.Bytes(), response)
		suite.Require().NoError(err, "Failed to unmarshal creation response")

		// Verify specific fields
		suite.Equal(completeCompDef.UUID, response.Data.UUID)
		suite.Equal(completeCompDef.Metadata.Title, response.Data.Metadata.Title)
		suite.Require().NotNil(response.Data.Components)
		suite.Equal(component.Title, (*response.Data.Components)[0].Title)
		suite.Equal(component.Type, (*response.Data.Components)[0].Type)

		// Verify capabilities were created
		suite.Require().NotNil(response.Data.Capabilities)
		suite.Equal(capability.Name, (*response.Data.Capabilities)[0].Name)

		// Verify back-matter resources
		suite.Require().NotNil(response.Data.BackMatter)
		suite.Require().NotNil(response.Data.BackMatter.Resources)
		suite.Equal(backMatterResource.Title, (*response.Data.BackMatter.Resources)[0].Title)

		fmt.Println("Complete component definition created successfully with UUID:", response.Data.UUID)

		// Verify we can retrieve the component definition
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/api/oscal/component-definitions/"+response.Data.UUID, nil)
		server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to get created component definition")

		getResponse := &handler.GenericDataResponse[oscaltypes.ComponentDefinition]{}
		err = json.Unmarshal(rec.Body.Bytes(), getResponse)
		suite.Require().NoError(err, "Failed to unmarshal GET response")
		suite.Equal(response.Data.UUID, getResponse.Data.UUID)
	})
}

func (suite *ComponentDefinitionApiIntegrationSuite) TestCreateImportComponentDefinitions() {
	fmt.Println("Running TestCreateImportComponentDefinitions")
	logger, _ := zap.NewDevelopment()

	// Reset database
	err := suite.Migrator.Refresh()
	suite.Require().NoError(err, "Failed to refresh database")
	fmt.Println("Database refreshed successfully")

	// Setup server
	server := api.NewServer(context.Background(), logger.Sugar())
	RegisterHandlers(server, logger.Sugar(), suite.DB)
	fmt.Println("Server initialized")

	suite.Run("Successfully creates import component definitions", func() {
		// First create a base component definition to add imports to
		baseCompDef := createValidComponentDefinition()
		reqBody, err := json.Marshal(baseCompDef)
		suite.Require().NoError(err, "Failed to marshal base component definition")

		// Create the base component definition
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/oscal/component-definitions", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		server.E().ServeHTTP(rec, req)

		// Check response
		suite.Equal(http.StatusCreated, rec.Code, "Failed to create base component definition")

		// Unmarshal response to get the created component definition ID
		response := &handler.GenericDataResponse[oscaltypes.ComponentDefinition]{}
		err = json.Unmarshal(rec.Body.Bytes(), response)
		suite.Require().NoError(err, "Failed to unmarshal creation response")
		componentDefID := response.Data.UUID

		// Create test import component definitions
		importComponentDefs := []oscaltypes.ImportComponentDefinition{
			{
				Href: "https://example.com/components/base",
			},
			{
				Href: "https://example.com/components/security",
			},
		}

		// Marshal the import component definitions to JSON
		reqBody, err = json.Marshal(importComponentDefs)
		suite.Require().NoError(err, "Failed to marshal import component definitions")

		// Send POST request to create import component definitions
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/oscal/component-definitions/%s/import-component-definitions", componentDefID), bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		server.E().ServeHTTP(rec, req)

		// Check response
		suite.Equal(http.StatusOK, rec.Code, "Failed to create import component definitions")

		// Unmarshal and verify response
		importResponse := &handler.GenericDataListResponse[oscaltypes.ImportComponentDefinition]{}
		err = json.Unmarshal(rec.Body.Bytes(), importResponse)
		suite.Require().NoError(err, "Failed to unmarshal import component definitions response")

		// Verify the response contains the correct number of import component definitions
		suite.Equal(len(importComponentDefs), len(importResponse.Data), "Number of import component definitions doesn't match")

		// Verify each import component definition
		for i, importDef := range importResponse.Data {
			suite.Equal(importComponentDefs[i].Href, importDef.Href, "Import component definition href doesn't match")
		}

		// Verify we can retrieve the import component definitions
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/oscal/component-definitions/%s/import-component-definitions", componentDefID), nil)
		server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to get import component definitions")

		getResponse := &handler.GenericDataListResponse[oscaltypes.ImportComponentDefinition]{}
		err = json.Unmarshal(rec.Body.Bytes(), getResponse)
		suite.Require().NoError(err, "Failed to unmarshal GET response")
		suite.Equal(len(importComponentDefs), len(getResponse.Data), "Number of retrieved import component definitions doesn't match")
	})
}

// Helper functions to create test data
func createTestBackMatterResource(uuid string) oscaltypes.Resource {
	return oscaltypes.Resource{
		UUID:        uuid,
		Title:       "Sample Resource",
		Description: "Sample resource description",
		DocumentIds: &[]oscaltypes.DocumentId{
			{
				Scheme:     "https://example.com/identifiers",
				Identifier: "DOC-ID-1",
			},
		},
		Citation: &oscaltypes.Citation{
			Text: "Sample citation text",
			Links: &[]oscaltypes.Link{
				{
					Href:      "https://example.com/resource",
					Rel:       "related",
					MediaType: "text/html",
					Text:      "Resource Link",
				},
			},
			Props: &[]oscaltypes.Property{
				{
					Name:  "version",
					Value: "1.0",
				},
			},
		},
		Base64: &oscaltypes.Base64{
			Filename:  "sample.txt",
			MediaType: "text/plain",
			Value:     "c2FtcGxlIGJhc2U2NCBlbmNvZGVkIGRhdGE=",
		},
		Remarks: "Sample remarks for the resource",
	}
}

func createTestComponent(componentUUID, implUUID, reqUUID string) oscaltypes.DefinedComponent {
	return oscaltypes.DefinedComponent{
		UUID:        componentUUID,
		Type:        "software",
		Title:       "Sample Component",
		Description: "A sample component for testing",
		Purpose:     "Demonstration",
		Protocols: &[]oscaltypes.Protocol{
			{
				UUID:  uuid.New().String(),
				Name:  "https",
				Title: "HTTPS Protocol",
				PortRanges: &[]oscaltypes.PortRange{
					{
						Start:     443,
						End:       443,
						Transport: "TCP",
					},
				},
			},
		},
		ControlImplementations: &[]oscaltypes.ControlImplementationSet{
			{
				UUID:        implUUID,
				Description: "Sample control implementation",
				ImplementedRequirements: []oscaltypes.ImplementedRequirementControlImplementation{
					{
						UUID:      reqUUID,
						ControlId: "AC-1",
						Remarks:   "Access control policy and procedures",
					},
				},
			},
		},
		ResponsibleRoles: &[]oscaltypes.ResponsibleRole{
			{
				RoleId:     "owner",
				PartyUuids: &[]string{uuid.New().String()},
				Remarks:    "Primary system owner",
			},
		},
	}
}

func createTestCapability(capabilityUUID, implUUID string) oscaltypes.Capability {
	return oscaltypes.Capability{
		UUID:        capabilityUUID,
		Name:        "Security Monitoring",
		Description: "Security monitoring capability",
		ControlImplementations: &[]oscaltypes.ControlImplementationSet{
			{
				UUID:        implUUID,
				Description: "Capability control implementation",
				ImplementedRequirements: []oscaltypes.ImplementedRequirementControlImplementation{
					{
						UUID:      uuid.New().String(),
						ControlId: "SI-4",
						Remarks:   "Information system monitoring",
					},
				},
			},
		},
	}
}

func createTestMetadata(partyUUID, locationUUID string) oscaltypes.Metadata {
	now := time.Now()
	return oscaltypes.Metadata{
		Title:        "Complete Component Definition",
		Version:      "1.0.0",
		OscalVersion: "1.1.3",
		Published:    &now,
		LastModified: now,
		Parties: &[]oscaltypes.Party{
			{
				UUID:           partyUUID,
				Type:           "organization",
				Name:           "Example Organization",
				ShortName:      "ExOrg",
				EmailAddresses: &[]string{"contact@example.com"},
				Addresses: &[]oscaltypes.Address{
					{
						Type:       "work",
						AddrLines:  &[]string{"123 Example Street"},
						City:       "Example City",
						State:      "EX",
						PostalCode: "12345",
						Country:    "US",
					},
				},
			},
		},
		Roles: &[]oscaltypes.Role{
			{
				ID:          "owner",
				Title:       "System Owner",
				Description: "Person or organization responsible for the system",
			},
		},
		Locations: &[]oscaltypes.Location{
			{
				UUID:  locationUUID,
				Title: "Primary Data Center",
				Address: &oscaltypes.Address{
					Type:       "work",
					AddrLines:  &[]string{"456 Data Center Avenue"},
					City:       "Server City",
					State:      "SC",
					PostalCode: "67890",
					Country:    "US",
				},
			},
		},
	}
}

func createValidComponentDefinition() oscaltypes.ComponentDefinition {
	now := time.Now()
	return oscaltypes.ComponentDefinition{
		UUID: uuid.New().String(),
		Metadata: oscaltypes.Metadata{
			Title:        "Valid Component Definition",
			Version:      "1.0.0",
			OscalVersion: "1.1.3",
			Published:    &now,
			LastModified: now,
		},
	}
}
