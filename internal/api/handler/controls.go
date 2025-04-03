package handler

import (
	"net/http"

	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/service"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// DashboardHandler handles CRUD operations for dashboards.
type CatalogControlHandler struct {
	service *service.CatalogControlService
	sugar   *zap.SugaredLogger
}

// NewDashboardHandler creates a new DashboardHandler.
func NewCatalogControlHandler(l *zap.SugaredLogger, s *service.CatalogControlService) *CatalogControlHandler {
	return &CatalogControlHandler{
		sugar:   l,
		service: s,
	}
}

// Register registers the dashboard endpoints.
func (h *CatalogControlHandler) Register(api *echo.Group) {
	api.GET("/:class/:id", h.Get)
	api.GET("/:type/:class/:id", h.List)
}

// Get godoc
//
//	@Summary		Get a dashboard
//	@Description	Retrieves a single dashboard by its unique ID.
//	@Tags			Dashboards
//	@Produce		json
//	@Param			id	path		string	true	"Dashboard ID"
//	@Success		200	{object}	GenericDataResponse[service.Dashboard]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/dashboard/{id} [get]
func (h *CatalogControlHandler) Get(ctx echo.Context) error {
	control, err := h.service.Get(ctx.Request().Context(), ctx.Param("class"), ctx.Param("id"))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	} else if control == nil {
		return ctx.JSON(http.StatusNotFound, api.NotFound())
	}

	return ctx.JSON(http.StatusOK, GenericDataResponse[service.CatalogControl]{
		Data: *control,
	})
}

// List godoc
//
//	@Summary		List dashboards
//	@Description	Retrieves all dashboards.
//	@Tags			Dashboards
//	@Produce		json
//	@Success		200	{object}	GenericDataListResponse[service.Dashboard]
//	@Failure		400	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/dashboard [get]
func (h *CatalogControlHandler) List(c echo.Context) error {
	results, err := h.service.FindFor(c.Request().Context(), service.CatalogItemParentIdentifier{
		ID:    c.Param("id"),
		Class: c.Param("class"),
		Type:  service.CatalogItemParentType(c.Param("type")),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return c.JSON(http.StatusOK, GenericDataListResponse[service.CatalogControl]{
		Data: results,
	})
}
