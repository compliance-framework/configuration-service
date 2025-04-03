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
	api.GET("/group/:class/:id", h.GetForGroup)
	api.GET("/children/:class/:id", h.GetForControl)
}

// GetForGroup godoc
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
func (h *CatalogControlHandler) GetForGroup(c echo.Context) error {
	results, err := h.service.FindFor(c.Request().Context(), service.CatalogItemParentIdentifier{
		ID:    c.Param("id"),
		Class: c.Param("class"),
		Type:  service.CatalogItemParentTypeGroup,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return c.JSON(http.StatusOK, GenericDataListResponse[service.CatalogControl]{
		Data: results,
	})
}

// GetForControl godoc
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
func (h *CatalogControlHandler) GetForControl(c echo.Context) error {
	results, err := h.service.FindFor(c.Request().Context(), service.CatalogItemParentIdentifier{
		ID:    c.Param("id"),
		Class: c.Param("class"),
		Type:  service.CatalogItemParentTypeControl,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return c.JSON(http.StatusOK, GenericDataListResponse[service.CatalogControl]{
		Data: results,
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
