package handler

import (
	"net/http"

	"github.com/compliance-framework/configuration-service/api"
	"github.com/compliance-framework/configuration-service/domain"
	"github.com/compliance-framework/configuration-service/service"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// TODO: Publishing the events from the handler is not a good idea. We should
//  publish the events from the domain services, following the business logic.

type ResultHandler struct {
	service *service.ResultService
	sugar   *zap.SugaredLogger
}

func (h *ResultHandler) Register(api *echo.Group) {
	// TODO: Most of the methods require other ops like delete and update
	api.GET("/result", h.QueryResults)
	api.GET("/result/:id", h.GetResult)
}

func NewResultHandler(l *zap.SugaredLogger, s *service.ResultService) *ResultHandler {
	return &ResultHandler{
		sugar:   l,
		service: s,
	}
}

// CreateResult godoc
// @Summary 		Create a result
// @Description 	Creates a new result in the system
// @Accept  		json
// @Produce  		json
// @Param   		result body createResultRequest true "Result to add"
// @Success 		201 {object} resultIdResponse
// @Failure 		401 {object} api.Error
// @Failure 		422 {object} api.Error
// @Failure 		500 {object} api.Error
// @Router 			/api/result [post]
func (h *ResultHandler) CreateResult(ctx echo.Context) error {
	// Initialize a new result object
	p := domain.NewResult()

	// Initialize a new createResultRequest object
	req := createResultRequest{}

	// Bind the incoming request to the result object
	// If there's an error, return a 422 status code with the error message
	if err := req.bind(ctx, p); err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, api.NewError(err))
	}

	// Attempt to create the result in the service
	// If there's an error, return a 500 status code with the error message
	id, err := h.service.Create(p)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// If everything went well, return a 201 status code with the ID of the created result
	return ctx.JSON(http.StatusCreated, resultIdResponse{
		Id: id,
	})
}
