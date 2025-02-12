package handler

import (
	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/domain"
	"github.com/compliance-framework/configuration-service/internal/service"
	"github.com/google/uuid"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func NewPlansHandler(l *zap.SugaredLogger, s *service.PlansService) *PlansHandler {
	return &PlansHandler{
		sugar:   l,
		service: s,
	}
}

type PlansHandler struct {
	service *service.PlansService
	sugar   *zap.SugaredLogger
}

func (h *PlansHandler) Register(api *echo.Group) {
	api.GET("", h.GetPlans)
	api.POST("", h.CreatePlan)
	api.GET("/:id", h.GetPlan)
}

// GetPlans godoc
//
//	@Summary		Gets plan summaries
//	@Description	Returns id and title of all the plans in the system
//	@Tags			Plan
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	handler.GenericDataListResponse[domain.Plan]
//	@Failure		500	{object}	api.Error
//	@Router			/plans [get]
func (h *PlansHandler) GetPlans(c echo.Context) error {
	results, err := h.service.GetPlans()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return c.JSON(http.StatusOK, GenericDataListResponse[domain.Plan]{
		Data: *results,
	})
}

// CreatePlan godoc
//
//	@Summary		Create a plan
//	@Description	Creates a new plan in the system
//	@Tags			Plan
//	@Accept			json
//	@Produce		json
//	@Param			plan	body		createPlanRequest	true	"Plan to add"
//	@Success		201		{object}	idResponse
//	@Failure		401		{object}	api.Error
//	@Failure		422		{object}	api.Error
//	@Failure		500		{object}	api.Error
//	@Router			/plan [post]
func (h *PlansHandler) CreatePlan(ctx echo.Context) error {
	// Initialize a new plan object
	p := &domain.Plan{}

	// Initialize a new createPlanRequest object
	req := createPlanRequest{}

	// Bind the incoming request to the plan object
	// If there's an error, return a 422 status code with the error message
	if err := req.bind(ctx, p); err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, api.NewError(err))
	}

	// Attempt to create the plan in the service
	// If there's an error, return a 500 status code with the error message
	_, err := h.service.Create(p)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// If everything went well, return a 201 status code with the ID of the created plan
	return ctx.JSON(http.StatusCreated, GenericDataResponse[PlanResponse]{
		Data: PlanResponse{
			Plan: *p,
		},
	})
}

// GetPlan godoc
//
//	@Summary		Fetches a plan
//	@Description	Fetches a plan in the system
//	@Tags			Plan
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Plan ID"
//	@Success		200	{object}	handler.GenericDataResponse[PlanResponse]
//	@Failure		401	{object}	api.Error
//	@Failure		422	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/plan/:id [get]
func (h *PlansHandler) GetPlan(ctx echo.Context) error {
	plan, err := h.service.GetById(ctx.Request().Context(), uuid.MustParse(ctx.Param("id")))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	} else if plan == nil {
		return ctx.JSON(http.StatusNotFound, api.NotFound())
	}

	return ctx.JSON(http.StatusOK, GenericDataResponse[PlanResponse]{
		Data: PlanResponse{*plan},
	})
}
