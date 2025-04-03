package handler

import (
	"net/http"

	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/service"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// DashboardHandler handles CRUD operations for dashboards.
type CatalogGroupHandler struct {
	service *service.CatalogGroupService
	sugar   *zap.SugaredLogger
}

// NewDashboardHandler creates a new DashboardHandler.
func NewCatalogGroupHandler(l *zap.SugaredLogger, s *service.CatalogGroupService) *CatalogGroupHandler {
	return &CatalogGroupHandler{
		sugar:   l,
		service: s,
	}
}

// Register registers the dashboard endpoints.
func (h *CatalogGroupHandler) Register(api *echo.Group) {
	api.GET("/catalog/:id", h.Get)
	api.GET("/children/:class/:id", h.GetForGroup)
	//api.GET("/:type/:class/:id", h.List)
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
//	@Router			/groups/{id} [get]
func (h *CatalogGroupHandler) Get(ctx echo.Context) error {
	groups, err := h.service.FindFor(ctx.Request().Context(), service.CatalogItemParentIdentifier{
		ID:    ctx.Param("id"),
		Class: "",
		Type:  service.CatalogItemParentTypeCatalog,
	})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, GenericDataListResponse[service.CatalogGroup]{
		Data: groups,
	})
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
//	@Router			/groups/{id} [get]
func (h *CatalogGroupHandler) GetForGroup(ctx echo.Context) error {
	groups, err := h.service.FindFor(ctx.Request().Context(), service.CatalogItemParentIdentifier{
		ID:    ctx.Param("id"),
		Class: ctx.Param("class"),
		Type:  service.CatalogItemParentTypeGroup,
	})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, GenericDataListResponse[service.CatalogGroup]{
		Data: groups,
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
func (h *CatalogGroupHandler) List(c echo.Context) error {
	results, err := h.service.FindFor(c.Request().Context(), service.CatalogItemParentIdentifier{
		ID:    c.Param("id"),
		Class: c.Param("class"),
		Type:  service.CatalogItemParentType(c.Param("type")),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return c.JSON(http.StatusOK, GenericDataListResponse[service.CatalogGroup]{
		Data: results,
	})
}
