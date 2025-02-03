package handler

import (
	"net/http"

	"github.com/compliance-framework/configuration-service/api"
	"github.com/compliance-framework/configuration-service/domain"
	"github.com/compliance-framework/configuration-service/service"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type PlanHandler struct {
	service *service.PlanService
	sugar   *zap.SugaredLogger
}

func (h *PlanHandler) Register(api *echo.Group) {
	api.POST("", h.CreatePlan)
	api.GET("/:id", h.GetPlan)
}

func NewPlanHandler(l *zap.SugaredLogger, s *service.PlanService) *PlanHandler {
	return &PlanHandler{
		sugar:   l,
		service: s,
	}
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
func (h *PlanHandler) CreatePlan(ctx echo.Context) error {
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
func (h *PlanHandler) GetPlan(ctx echo.Context) error {
	plan, err := h.service.GetById(ctx.Request().Context(), ctx.Param("id"))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	} else if plan == nil {
		return ctx.JSON(http.StatusNotFound, api.NotFound())
	}

	return ctx.JSON(http.StatusOK, GenericDataResponse[PlanResponse]{
		Data: PlanResponse{*plan},
	})
}

// Risks Returns the risks of the result with the given ID.
//
//	@Summary		Return the risks
//	@Description	Return the risks of the result with the given ID.
//	@Tags			Plan
//	@Produce		json
//	@Param			id			path		string	true	"Plan ID"
//	@Param			resultId	path		string	true	"Result ID"
//	@Success		200			{object}	[]domain.Risk
//	@Failure		500			{object}	api.Error	"Internal server error."
//	@Router			/plan/{id}/results/{resultId}/risks [get]
func (h *PlanHandler) Risks(c echo.Context) error {
	risks, err := h.service.Risks(c.Param("id"), c.Param("resultId"))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return c.JSON(http.StatusOK, risks)
}
