package oscal

import (
	"fmt"
	"github.com/compliance-framework/configuration-service/internal/api/handler"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"net/http"

	"github.com/compliance-framework/configuration-service/internal/service"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// CatalogHandler handles CRUD operations for Catalogs.
type CatalogHandler struct {
	service        *service.CatalogService
	controlService *service.CatalogControlService
	groupService   *service.CatalogGroupService
	sugar          *zap.SugaredLogger
}

// NewCatalogHandler creates a new CatalogHandler.
func NewCatalogHandler(l *zap.SugaredLogger, s *service.CatalogService, g *service.CatalogGroupService, c *service.CatalogControlService) *CatalogHandler {
	return &CatalogHandler{
		sugar:          l,
		service:        s,
		groupService:   g,
		controlService: c,
	}
}

// Register registers the Catalog endpoints.
func (h *CatalogHandler) Register(api *echo.Group) {
	api.POST("", h.Create)
}

func (h *CatalogHandler) Create(ctx echo.Context) error {
	fmt.Println("Trying to create OSCAL catalog")

	//// Return the created catalog wrapped in a GenericDataResponse.
	return ctx.JSON(http.StatusCreated, handler.GenericDataResponse[oscaltypes113.Catalog]{
		Data: oscaltypes113.Catalog{},
	})
}
