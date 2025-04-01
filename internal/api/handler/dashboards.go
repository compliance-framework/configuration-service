package handler

import (
	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/service"
	"github.com/google/uuid"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func NewDashboardHandler(l *zap.SugaredLogger, s *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		sugar:   l,
		service: s,
	}
}

type DashboardHandler struct {
	service *service.DashboardService
	sugar   *zap.SugaredLogger
}

func (h *DashboardHandler) Register(api *echo.Group) {
	api.GET("", h.List)
	api.GET("/:id", h.Get)
	api.POST("", h.Create)
}

// Get godoc
//
//	@Tags		Assessment Plans
//	@Summary	Fetch a single assessment plan
//	@Param		id	path		string	true	"Plan ID"
//	@Success	200	{object}	handler.GenericDataResponse[service.Dashboard]
//	@Failure	401	{object}	api.Error
//	@Failure	404	{object}	api.Error
//	@Failure	500	{object}	api.Error
//	@Router		/dashboard/:id [get]
func (h *DashboardHandler) Get(ctx echo.Context) error {
	dashboard, err := h.service.Get(ctx.Request().Context(), uuid.MustParse(ctx.Param("id")))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	} else if dashboard == nil {
		return ctx.JSON(http.StatusNotFound, api.NotFound())
	}

	return ctx.JSON(http.StatusOK, GenericDataResponse[service.Dashboard]{
		Data: *dashboard,
	})
}

// List godoc
//
//	@Tags		Assessment Plans
//	@Summary	Fetch all assessment plans
//	@Success	200	{object}	handler.GenericDataListResponse[service.Dashboard]
//	@Failure	401	{object}	api.Error
//	@Failure	500	{object}	api.Error
//	@Router		/dashboard [get]
func (h *DashboardHandler) List(c echo.Context) error {
	results, err := h.service.List(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return c.JSON(http.StatusOK, GenericDataListResponse[service.Dashboard]{
		Data: *results,
	})
}

// Create godoc
//
//	@Summary	Create a new assessment plan
//	@Tags		Assessment Plans
//	@Param		plan	body		createPlanRequest	true	"Plan to add"
//	@Success	201		{object}	handler.GenericDataResponse[service.Dashboard]
//	@Failure	401		{object}	api.Error
//	@Failure	422		{object}	api.Error
//	@Failure	500		{object}	api.Error
//	@Router		/dashboard [post]
func (h *DashboardHandler) Create(ctx echo.Context) error {
	// Initialize a new plan object
	p := &service.Dashboard{}

	// Initialize a new createPlanRequest object
	req := createDashboardRequest{}

	// Bind the incoming request to the plan object
	// If there's an error, return a 422 status code with the error message
	if err := req.bind(ctx, p); err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, api.NewError(err))
	}

	// Attempt to create the plan in the resultService
	// If there's an error, return a 500 status code with the error message
	_, err := h.service.Create(ctx.Request().Context(), p)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// If everything went well, return a 201 status code with the ID of the created plan
	return ctx.JSON(http.StatusCreated, GenericDataResponse[service.Dashboard]{
		Data: *p,
	})
}
