package oscal

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/api/handler"
	"github.com/compliance-framework/configuration-service/internal/tests"
	oscaltypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestOscalCatalogApi(t *testing.T) {
	suite.Run(t, new(CatalogApiIntegrationSuite))
}

type CatalogApiIntegrationSuite struct {
	tests.IntegrationTestSuite
}

// TestDuplicateCatalogGroupID ensures that when multiple catalogs have group children with the same ID,
// their children endpoints only returned the relevant groups.
// This is to prevent a future regression where searching for child groups in a catalog, would return all the groups
// with a matching ID, rather than only the ones which belong to a catalog.
func (suite *CatalogApiIntegrationSuite) TestDuplicateCatalogGroupID() {
	// Create 2 catalogs with the same groups and one sub group.
	// When querying the children on one catalog it would return from the other catalog as well.

	logger, _ := zap.NewDevelopment()

	err := suite.Migrator.Refresh()
	suite.Require().NoError(err)

	server := api.NewServer(context.Background(), logger.Sugar())
	RegisterHandlers(server, logger.Sugar(), suite.DB)

	// Create two catalogs with the same group ID structure
	catalogs := []oscaltypes.Catalog{
		{
			UUID: "D20DB907-B87D-4D12-8760-D36FDB7A1B31",
			Metadata: oscaltypes.Metadata{
				Title: "Catalog 1",
			},
			Groups: &[]oscaltypes.Group{
				{
					ID:    "G-1",
					Title: "Group 1",
					Groups: &[]oscaltypes.Group{
						{
							ID:    "G-1.1",
							Title: "Group 1.1",
						},
					},
				},
			},
		},
		{
			UUID: "D20DB907-B87D-4D12-8760-D36FDB7A1B32",
			Metadata: oscaltypes.Metadata{
				Title: "Catalog 2",
			},
			Groups: &[]oscaltypes.Group{
				{
					ID:    "G-1",
					Title: "Group 2",
					Groups: &[]oscaltypes.Group{
						{
							ID:    "G-1.1",
							Title: "Group 2.1",
						},
					},
				},
			},
		},
	}
	for _, catalog := range catalogs {
		rec := httptest.NewRecorder()
		reqBody, _ := json.Marshal(catalog)
		req := httptest.NewRequest(http.MethodPost, "/api/oscal/catalogs", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		server.E().ServeHTTP(rec, req)
		assert.Equal(suite.T(), http.StatusCreated, rec.Code)
		response := &handler.GenericDataResponse[oscaltypes.Catalog]{}
		err = json.Unmarshal(rec.Body.Bytes(), response)
		suite.Require().NoError(err)
	}

	// Now if we call to check the children for each catalogs' first group, we should only see 1 item

	// The first catalog's group should have the Title Group 1
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/oscal/catalogs/D20DB907-B87D-4D12-8760-D36FDB7A1B31/groups/G-1/groups", bytes.NewReader([]byte{}))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	server.E().ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	response := &handler.GenericDataListResponse[oscaltypes.Group]{}
	err = json.Unmarshal(rec.Body.Bytes(), response)
	suite.Require().NoError(err)
	suite.Len(response.Data, 1)
}
