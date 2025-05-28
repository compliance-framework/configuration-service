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
	server *api.Server
	logger *zap.SugaredLogger
}

func (suite *ComponentDefinitionApiIntegrationSuite) SetupSuite() {
	fmt.Println("Setting up Component Definition API test suite")
	suite.IntegrationTestSuite.SetupSuite()

	// Setup logger and server once for all tests
	logger, _ := zap.NewDevelopment()
	suite.logger = logger.Sugar()
	suite.server = api.NewServer(context.Background(), suite.logger)
	RegisterHandlers(suite.server, suite.logger, suite.DB)
	fmt.Println("Server initialized")
}

func (suite *ComponentDefinitionApiIntegrationSuite) SetupTest() {
	// Reset database before each test
	err := suite.Migrator.Refresh()
	suite.Require().NoError(err, "Failed to refresh database")
	fmt.Println("Database refreshed successfully")
}

// Helper method to create a test request
func (suite *ComponentDefinitionApiIntegrationSuite) createRequest(method, path string, body interface{}) (*httptest.ResponseRecorder, *http.Request) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		suite.Require().NoError(err, "Failed to marshal request body")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	return rec, req
}

// Helper method to create a base component definition
func (suite *ComponentDefinitionApiIntegrationSuite) createBaseComponentDefinition() string {
	baseCompDef := createValidComponentDefinition()
	rec, req := suite.createRequest(http.MethodPost, "/api/oscal/component-definitions", baseCompDef)

	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusCreated, rec.Code, "Failed to create base component definition")

	response := &handler.GenericDataResponse[oscaltypes.ComponentDefinition]{}
	err := json.Unmarshal(rec.Body.Bytes(), response)
	suite.Require().NoError(err, "Failed to unmarshal creation response")

	return response.Data.UUID
}

func (suite *ComponentDefinitionApiIntegrationSuite) TestCreateCompleteComponentDefinition() {
	fmt.Println("Running TestCreateCompleteComponentDefinition")

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

		// Send POST request
		rec, req := suite.createRequest(http.MethodPost, "/api/oscal/component-definitions", completeCompDef)
		suite.server.E().ServeHTTP(rec, req)

		// Check response
		suite.Equal(http.StatusCreated, rec.Code, "Failed to create complete component definition")

		// Unmarshal and verify response
		response := &handler.GenericDataResponse[oscaltypes.ComponentDefinition]{}
		err := json.Unmarshal(rec.Body.Bytes(), response)
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
		rec, req = suite.createRequest(http.MethodGet, "/api/oscal/component-definitions/"+response.Data.UUID, nil)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to get created component definition")

		getResponse := &handler.GenericDataResponse[oscaltypes.ComponentDefinition]{}
		err = json.Unmarshal(rec.Body.Bytes(), getResponse)
		suite.Require().NoError(err, "Failed to unmarshal GET response")
		suite.Equal(response.Data.UUID, getResponse.Data.UUID)
	})
}

func (suite *ComponentDefinitionApiIntegrationSuite) TestCreateImportComponentDefinitions() {
	fmt.Println("Running TestCreateImportComponentDefinitions")

	suite.Run("Successfully creates import component definitions", func() {
		// First create a base component definition to add imports to
		componentDefID := suite.createBaseComponentDefinition()

		// Create test import component definitions
		importComponentDefs := []oscaltypes.ImportComponentDefinition{
			{
				Href: "https://example.com/components/base",
			},
			{
				Href: "https://example.com/components/security",
			},
		}

		// Send POST request to create import component definitions
		rec, req := suite.createRequest(
			http.MethodPost,
			fmt.Sprintf("/api/oscal/component-definitions/%s/import-component-definitions", componentDefID),
			importComponentDefs,
		)
		suite.server.E().ServeHTTP(rec, req)

		// Check response
		suite.Equal(http.StatusOK, rec.Code, "Failed to create import component definitions")

		// Unmarshal and verify response
		importResponse := &handler.GenericDataListResponse[oscaltypes.ImportComponentDefinition]{}
		err := json.Unmarshal(rec.Body.Bytes(), importResponse)
		suite.Require().NoError(err, "Failed to unmarshal import component definitions response")

		// Verify the response contains the correct number of import component definitions
		suite.Equal(len(importComponentDefs), len(importResponse.Data), "Number of import component definitions doesn't match")

		// Verify each import component definition
		for i, importDef := range importResponse.Data {
			suite.Equal(importComponentDefs[i].Href, importDef.Href, "Import component definition href doesn't match")
		}

		// Verify we can retrieve the import component definitions
		rec, req = suite.createRequest(
			http.MethodGet,
			fmt.Sprintf("/api/oscal/component-definitions/%s/import-component-definitions", componentDefID),
			nil,
		)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to get import component definitions")

		getResponse := &handler.GenericDataListResponse[oscaltypes.ImportComponentDefinition]{}
		err = json.Unmarshal(rec.Body.Bytes(), getResponse)
		suite.Require().NoError(err, "Failed to unmarshal GET response")
		suite.Equal(len(importComponentDefs), len(getResponse.Data), "Number of retrieved import component definitions doesn't match")
	})
}

func (suite *ComponentDefinitionApiIntegrationSuite) TestCreateComponents() {
	fmt.Println("Running TestCreateComponents")

	suite.Run("Successfully creates components for a component definition", func() {
		// First create a base component definition to add components to
		componentDefID := suite.createBaseComponentDefinition()

		// Create test components
		components := []oscaltypes.DefinedComponent{
			{
				UUID:        uuid.New().String(),
				Type:        "software",
				Title:       "Web Server Component",
				Description: "A web server component for testing",
				Purpose:     "Web serving",
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
			},
			{
				UUID:        uuid.New().String(),
				Type:        "service",
				Title:       "Database Component",
				Description: "A database component for testing",
				Purpose:     "Data storage",
				Protocols: &[]oscaltypes.Protocol{
					{
						UUID:  uuid.New().String(),
						Name:  "postgres",
						Title: "PostgreSQL Protocol",
						PortRanges: &[]oscaltypes.PortRange{
							{
								Start:     5432,
								End:       5432,
								Transport: "TCP",
							},
						},
					},
				},
			},
		}

		// Send POST request to create components
		rec, req := suite.createRequest(
			http.MethodPost,
			fmt.Sprintf("/api/oscal/component-definitions/%s/components", componentDefID),
			components,
		)
		suite.server.E().ServeHTTP(rec, req)

		// Check response
		suite.Equal(http.StatusOK, rec.Code, "Failed to create components")

		// Unmarshal and verify response
		componentsResponse := &handler.GenericDataListResponse[oscaltypes.DefinedComponent]{}
		err := json.Unmarshal(rec.Body.Bytes(), componentsResponse)
		suite.Require().NoError(err, "Failed to unmarshal components response")

		// Verify the response contains the correct number of components
		suite.Equal(len(components), len(componentsResponse.Data), "Number of components doesn't match")

		// Verify each component
		for i, component := range componentsResponse.Data {
			suite.Equal(components[i].UUID, component.UUID, "Component UUID doesn't match")
			suite.Equal(components[i].Type, component.Type, "Component type doesn't match")
			suite.Equal(components[i].Title, component.Title, "Component title doesn't match")
			suite.Equal(components[i].Description, component.Description, "Component description doesn't match")
			suite.Equal(components[i].Purpose, component.Purpose, "Component purpose doesn't match")
		}

		fmt.Printf("Successfully created %d components for component definition %s\n", len(components), componentDefID)

		// Verify we can retrieve the components
		rec, req = suite.createRequest(
			http.MethodGet,
			fmt.Sprintf("/api/oscal/component-definitions/%s/components", componentDefID),
			nil,
		)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to get components")

		getResponse := &handler.GenericDataListResponse[oscaltypes.DefinedComponent]{}
		err = json.Unmarshal(rec.Body.Bytes(), getResponse)
		suite.Require().NoError(err, "Failed to unmarshal GET response")
		suite.Equal(len(components), len(getResponse.Data), "Number of retrieved components doesn't match")
	})

	suite.Run("Fails to create components for non-existent component definition", func() {
		nonExistentID := uuid.New().String()
		components := []oscaltypes.DefinedComponent{
			{
				UUID:        uuid.New().String(),
				Type:        "software",
				Title:       "Test Component",
				Description: "A test component",
				Purpose:     "Testing",
			},
		}

		rec, req := suite.createRequest(
			http.MethodPost,
			fmt.Sprintf("/api/oscal/component-definitions/%s/components", nonExistentID),
			components,
		)
		suite.server.E().ServeHTTP(rec, req)

		suite.Equal(http.StatusNotFound, rec.Code, "Expected 404 for non-existent component definition")
	})

	suite.Run("Fails to create components with invalid data", func() {
		componentDefID := suite.createBaseComponentDefinition()

		// Create invalid component (missing required fields)
		invalidComponents := []oscaltypes.DefinedComponent{
			{
				UUID: uuid.New().String(),
				// Missing required fields like Type, Title, etc.
			},
		}

		rec, req := suite.createRequest(
			http.MethodPost,
			fmt.Sprintf("/api/oscal/component-definitions/%s/components", componentDefID),
			invalidComponents,
		)
		suite.server.E().ServeHTTP(rec, req)

		suite.Equal(http.StatusBadRequest, rec.Code, "Expected 400 for invalid component data")
	})
}

func (suite *ComponentDefinitionApiIntegrationSuite) TestUpdateComponents() {
	fmt.Println("Running TestUpdateComponents")

	suite.Run("Successfully updates components for a component definition", func() {
		// First create a base component definition to add components to
		componentDefID := suite.createBaseComponentDefinition()

		// Create initial components
		initialComponents := []oscaltypes.DefinedComponent{
			{
				UUID:        uuid.New().String(),
				Type:        "software",
				Title:       "Initial Web Server Component",
				Description: "An initial web server component for testing",
				Purpose:     "Web serving",
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
			},
		}

		// Create initial components
		rec, req := suite.createRequest(
			http.MethodPost,
			fmt.Sprintf("/api/oscal/component-definitions/%s/components", componentDefID),
			initialComponents,
		)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to create initial components")

		// Create updated components with modified data
		updatedComponents := []oscaltypes.DefinedComponent{
			{
				UUID:        initialComponents[0].UUID, // Keep the same UUID
				Type:        "service",                 // Changed type
				Title:       "Updated Web Server Component",
				Description: "An updated web server component for testing",
				Purpose:     "Enhanced web serving",
				Protocols: &[]oscaltypes.Protocol{
					{
						UUID:  uuid.New().String(),
						Name:  "http",
						Title: "HTTP Protocol",
						PortRanges: &[]oscaltypes.PortRange{
							{
								Start:     80,
								End:       80,
								Transport: "TCP",
							},
						},
					},
				},
			},
		}

		// Send PUT request to update components
		rec, req = suite.createRequest(
			http.MethodPut,
			fmt.Sprintf("/api/oscal/component-definitions/%s/components", componentDefID),
			updatedComponents,
		)
		suite.server.E().ServeHTTP(rec, req)

		// Check response
		suite.Equal(http.StatusOK, rec.Code, "Failed to update components")

		// Unmarshal and verify response
		componentsResponse := &handler.GenericDataListResponse[oscaltypes.DefinedComponent]{}
		err := json.Unmarshal(rec.Body.Bytes(), componentsResponse)
		suite.Require().NoError(err, "Failed to unmarshal components response")

		// Verify the response contains the correct number of components
		suite.Equal(len(updatedComponents), len(componentsResponse.Data), "Number of components doesn't match")

		// Verify each component was updated correctly
		for i, component := range componentsResponse.Data {
			suite.Equal(updatedComponents[i].UUID, component.UUID, "Component UUID doesn't match")
			suite.Equal(updatedComponents[i].Type, component.Type, "Component type wasn't updated")
			suite.Equal(updatedComponents[i].Title, component.Title, "Component title wasn't updated")
			suite.Equal(updatedComponents[i].Description, component.Description, "Component description wasn't updated")
			suite.Equal(updatedComponents[i].Purpose, component.Purpose, "Component purpose wasn't updated")
		}

		fmt.Printf("Successfully updated components for component definition %s\n", componentDefID)

		// Verify we can retrieve the updated components
		rec, req = suite.createRequest(
			http.MethodGet,
			fmt.Sprintf("/api/oscal/component-definitions/%s/components", componentDefID),
			nil,
		)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to get updated components")

		getResponse := &handler.GenericDataListResponse[oscaltypes.DefinedComponent]{}
		err = json.Unmarshal(rec.Body.Bytes(), getResponse)
		suite.Require().NoError(err, "Failed to unmarshal GET response")
		suite.Equal(len(updatedComponents), len(getResponse.Data), "Number of retrieved components doesn't match")

		// Verify the retrieved components match the updates
		for i, component := range getResponse.Data {
			suite.Equal(updatedComponents[i].UUID, component.UUID, "Retrieved component UUID doesn't match")
			suite.Equal(updatedComponents[i].Type, component.Type, "Retrieved component type doesn't match")
			suite.Equal(updatedComponents[i].Title, component.Title, "Retrieved component title doesn't match")
			suite.Equal(updatedComponents[i].Description, component.Description, "Retrieved component description doesn't match")
			suite.Equal(updatedComponents[i].Purpose, component.Purpose, "Retrieved component purpose doesn't match")
		}
	})
}

func (suite *ComponentDefinitionApiIntegrationSuite) TestUpdateDefinedComponent() {
	fmt.Println("Running TestUpdateDefinedComponent")

	suite.Run("Successfully updates a defined component", func() {
		// First create a base component definition
		componentDefID := suite.createBaseComponentDefinition()

		// Create initial component
		initialComponent := oscaltypes.DefinedComponent{
			UUID:        uuid.New().String(),
			Type:        "software",
			Title:       "Initial Web Server Component",
			Description: "An initial web server component for testing",
			Purpose:     "Web serving",
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
		}

		// Create initial component
		rec, req := suite.createRequest(
			http.MethodPost,
			fmt.Sprintf("/api/oscal/component-definitions/%s/components", componentDefID),
			[]oscaltypes.DefinedComponent{initialComponent},
		)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to create initial component")

		// Create updated component with modified data
		updatedComponent := oscaltypes.DefinedComponent{
			UUID:        initialComponent.UUID, // Keep the same UUID
			Type:        "service",             // Changed type
			Title:       "Updated Web Server Component",
			Description: "An updated web server component for testing",
			Purpose:     "Enhanced web serving",
			Protocols: &[]oscaltypes.Protocol{
				{
					UUID:  uuid.New().String(),
					Name:  "http",
					Title: "HTTP Protocol",
					PortRanges: &[]oscaltypes.PortRange{
						{
							Start:     80,
							End:       80,
							Transport: "TCP",
						},
					},
				},
			},
		}

		// Send PUT request to update the defined component
		rec, req = suite.createRequest(
			http.MethodPut,
			fmt.Sprintf("/api/oscal/component-definitions/%s/components", componentDefID),
			[]oscaltypes.DefinedComponent{updatedComponent},
		)
		suite.server.E().ServeHTTP(rec, req)

		// Check response
		suite.Equal(http.StatusOK, rec.Code, "Failed to update defined component")

		// Unmarshal and verify response
		componentsResponse := &handler.GenericDataListResponse[oscaltypes.DefinedComponent]{}
		err := json.Unmarshal(rec.Body.Bytes(), componentsResponse)
		suite.Require().NoError(err, "Failed to unmarshal components response")

		// Verify the response contains the correct number of components
		suite.Equal(1, len(componentsResponse.Data), "Number of components doesn't match")

		// Verify the component was updated correctly
		component := componentsResponse.Data[0]
		suite.Equal(updatedComponent.UUID, component.UUID, "Component UUID doesn't match")
		suite.Equal(updatedComponent.Type, component.Type, "Component type wasn't updated")
		suite.Equal(updatedComponent.Title, component.Title, "Component title wasn't updated")
		suite.Equal(updatedComponent.Description, component.Description, "Component description wasn't updated")
		suite.Equal(updatedComponent.Purpose, component.Purpose, "Component purpose wasn't updated")

		fmt.Printf("Successfully updated defined component %s for component definition %s\n", initialComponent.UUID, componentDefID)

		// Verify we can retrieve the updated component
		rec, req = suite.createRequest(
			http.MethodGet,
			fmt.Sprintf("/api/oscal/component-definitions/%s/components/%s", componentDefID, initialComponent.UUID),
			nil,
		)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to get updated component")

		getResponse := &handler.GenericDataResponse[oscaltypes.DefinedComponent]{}
		err = json.Unmarshal(rec.Body.Bytes(), getResponse)
		suite.Require().NoError(err, "Failed to unmarshal GET response")

		// Verify the retrieved component matches the updates
		suite.Equal(updatedComponent.UUID, getResponse.Data.UUID, "Retrieved component UUID doesn't match")
		suite.Equal(updatedComponent.Type, getResponse.Data.Type, "Retrieved component type doesn't match")
		suite.Equal(updatedComponent.Title, getResponse.Data.Title, "Retrieved component title doesn't match")
		suite.Equal(updatedComponent.Description, getResponse.Data.Description, "Retrieved component description doesn't match")
		suite.Equal(updatedComponent.Purpose, getResponse.Data.Purpose, "Retrieved component purpose doesn't match")
	})
}

func (suite *ComponentDefinitionApiIntegrationSuite) TestUpdateSingleControlImplementation() {
	fmt.Println("Running TestUpdateSingleControlImplementation")

	// Step 1: Create a base component definition with a component and control implementation
	componentDefID := suite.createBaseComponentDefinition()
	componentUUID := uuid.New().String()
	controlImplUUID := uuid.New().String()
	implementedReqUUID := uuid.New().String()

	component := createTestComponent(componentUUID, controlImplUUID, implementedReqUUID)
	components := []oscaltypes.DefinedComponent{component}

	// Create the component
	rec, req := suite.createRequest(
		http.MethodPost,
		fmt.Sprintf("/api/oscal/component-definitions/%s/components", componentDefID),
		components,
	)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusOK, rec.Code, "Failed to create component")

	// Step 2: Prepare an update for the control implementation
	updatedSource := "https://example.com/updated-source"
	updatedDescription := "Updated control implementation description"
	updatedSetParameters := &[]oscaltypes.SetParameter{
		{
			ParamId: "param-1",
			Values:  []string{"value1"},
		},
	}
	updatedImplementedRequirements := []oscaltypes.ImplementedRequirementControlImplementation{
		{
			UUID:      implementedReqUUID,
			ControlId: "AC-1",
			Remarks:   "Updated remarks",
		},
	}

	updatedControlImpl := oscaltypes.ControlImplementationSet{
		UUID:                    controlImplUUID,
		Source:                  updatedSource,
		Description:             updatedDescription,
		SetParameters:           updatedSetParameters,
		ImplementedRequirements: updatedImplementedRequirements,
	}

	// Step 3: Send PUT request to update the control implementation
	rec, req = suite.createRequest(
		http.MethodPut,
		fmt.Sprintf("/api/oscal/component-definitions/%s/components/%s/control-implementations/%s", componentDefID, componentUUID, controlImplUUID),
		updatedControlImpl,
	)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusOK, rec.Code, "Failed to update control implementation")

	// Step 4: Verify the update in the response
	response := &handler.GenericDataResponse[oscaltypes.ControlImplementationSet]{}
	err := json.Unmarshal(rec.Body.Bytes(), response)
	suite.Require().NoError(err, "Failed to unmarshal update response")
	suite.Equal(updatedSource, response.Data.Source)
	suite.Equal(updatedDescription, response.Data.Description)
	suite.Require().NotNil(response.Data.SetParameters)
	suite.Equal((*updatedSetParameters)[0].ParamId, (*response.Data.SetParameters)[0].ParamId)
	suite.Equal(updatedImplementedRequirements[0].Remarks, response.Data.ImplementedRequirements[0].Remarks)
}

func (suite *ComponentDefinitionApiIntegrationSuite) TestUpdateCapability() {
	fmt.Println("Running TestUpdateCapability")

	suite.Run("Successfully updates a capability for a component definition", func() {
		// Step 1: Create a base component definition
		componentDefID := suite.createBaseComponentDefinition()

		// Step 2: Create an initial capability
		capabilityUUID := uuid.New().String()
		capabilityImplUUID := uuid.New().String()
		initialCapability := createTestCapability(capabilityUUID, capabilityImplUUID)

		// POST the capability
		rec, req := suite.createRequest(
			http.MethodPost,
			fmt.Sprintf("/api/oscal/component-definitions/%s/capabilities", componentDefID),
			[]oscaltypes.Capability{initialCapability},
		)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to create capability")

		// Step 3: Prepare an updated capability (change name, description, add remarks)
		updatedCapability := initialCapability
		updatedCapability.Name = "Updated Security Monitoring"
		updatedCapability.Description = "Updated description for security monitoring capability"
		updatedCapability.Remarks = "Updated remarks for capability"

		// PUT the update
		rec, req = suite.createRequest(
			http.MethodPut,
			fmt.Sprintf("/api/oscal/component-definitions/%s/capabilities/%s", componentDefID, capabilityUUID),
			updatedCapability,
		)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to update capability")

		// Unmarshal and verify response
		updateResponse := &handler.GenericDataResponse[oscaltypes.Capability]{}
		err := json.Unmarshal(rec.Body.Bytes(), updateResponse)
		suite.Require().NoError(err, "Failed to unmarshal update response")
		suite.Equal(updatedCapability.Name, updateResponse.Data.Name, "Capability name wasn't updated")
		suite.Equal(updatedCapability.Description, updateResponse.Data.Description, "Capability description wasn't updated")
		suite.Equal(updatedCapability.Remarks, updateResponse.Data.Remarks, "Capability remarks weren't updated")

		// Step 4: GET the capabilities and verify the update is reflected
		rec, req = suite.createRequest(
			http.MethodGet,
			fmt.Sprintf("/api/oscal/component-definitions/%s/capabilities", componentDefID),
			nil,
		)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to get capabilities")

		getResponse := &handler.GenericDataListResponse[oscaltypes.Capability]{}
		err = json.Unmarshal(rec.Body.Bytes(), getResponse)
		suite.Require().NoError(err, "Failed to unmarshal GET response")
		var found bool
		for _, cap := range getResponse.Data {
			if cap.UUID == capabilityUUID {
				found = true
				suite.Equal(updatedCapability.Name, cap.Name, "GET: Capability name wasn't updated")
				suite.Equal(updatedCapability.Description, cap.Description, "GET: Capability description wasn't updated")
				suite.Equal(updatedCapability.Remarks, cap.Remarks, "GET: Capability remarks weren't updated")
			}
		}
		suite.True(found, "Updated capability not found in GET response")
	})
}

func (suite *ComponentDefinitionApiIntegrationSuite) TestUpdateImportComponentDefinitions() {
	fmt.Println("Running TestUpdateImportComponentDefinitions")

	suite.Run("Successfully updates import component definitions", func() {
		// Step 1: Create a base component definition
		componentDefID := suite.createBaseComponentDefinition()

		// Step 2: Create initial import component definitions
		initialImportDefs := []oscaltypes.ImportComponentDefinition{
			{
				Href: "https://example.com/components/base",
			},
			{
				Href: "https://example.com/components/security",
			},
		}

		// Create initial import component definitions
		rec, req := suite.createRequest(
			http.MethodPost,
			fmt.Sprintf("/api/oscal/component-definitions/%s/import-component-definitions", componentDefID),
			initialImportDefs,
		)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to create initial import component definitions")

		// Step 3: Prepare updated import component definitions
		updatedImportDefs := []oscaltypes.ImportComponentDefinition{
			{
				Href: "https://example.com/components/updated-base",
			},
			{
				Href: "https://example.com/components/updated-security",
			},
			{
				Href: "https://example.com/components/new-component",
			},
		}

		// Step 4: Send PUT request to update import component definitions
		rec, req = suite.createRequest(
			http.MethodPut,
			fmt.Sprintf("/api/oscal/component-definitions/%s/import-component-definitions", componentDefID),
			updatedImportDefs,
		)
		suite.server.E().ServeHTTP(rec, req)

		// Check response
		suite.Equal(http.StatusOK, rec.Code, "Failed to update import component definitions")

		// Unmarshal and verify response
		importResponse := &handler.GenericDataListResponse[oscaltypes.ImportComponentDefinition]{}
		err := json.Unmarshal(rec.Body.Bytes(), importResponse)
		suite.Require().NoError(err, "Failed to unmarshal import component definitions response")

		// Verify the response contains the correct number of import component definitions
		suite.Equal(len(updatedImportDefs), len(importResponse.Data), "Number of import component definitions doesn't match")

		// Verify each import component definition
		for i, importDef := range importResponse.Data {
			suite.Equal(updatedImportDefs[i].Href, importDef.Href, "Import component definition href doesn't match")
		}

		fmt.Printf("Successfully updated import component definitions for component definition %s\n", componentDefID)

		// Step 5: Verify we can retrieve the updated import component definitions
		rec, req = suite.createRequest(
			http.MethodGet,
			fmt.Sprintf("/api/oscal/component-definitions/%s/import-component-definitions", componentDefID),
			nil,
		)
		suite.server.E().ServeHTTP(rec, req)
		suite.Equal(http.StatusOK, rec.Code, "Failed to get updated import component definitions")

		getResponse := &handler.GenericDataListResponse[oscaltypes.ImportComponentDefinition]{}
		err = json.Unmarshal(rec.Body.Bytes(), getResponse)
		suite.Require().NoError(err, "Failed to unmarshal GET response")

		// Verify the retrieved import component definitions match the updates
		suite.Equal(len(updatedImportDefs), len(getResponse.Data), "Number of retrieved import component definitions doesn't match")
		for i, importDef := range getResponse.Data {
			suite.Equal(updatedImportDefs[i].Href, importDef.Href, "Retrieved import component definition href doesn't match")
		}
	})

	suite.Run("Fails to update import component definitions for non-existent component definition", func() {
		nonExistentID := uuid.New().String()
		importDefs := []oscaltypes.ImportComponentDefinition{
			{
				Href: "https://example.com/components/test",
			},
		}

		rec, req := suite.createRequest(
			http.MethodPut,
			fmt.Sprintf("/api/oscal/component-definitions/%s/import-component-definitions", nonExistentID),
			importDefs,
		)
		suite.server.E().ServeHTTP(rec, req)

		suite.Equal(http.StatusNotFound, rec.Code, "Expected 404 for non-existent component definition")
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
