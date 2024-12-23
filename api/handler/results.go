package handler

import (
	"encoding/json"
	"fmt"
	"github.com/compliance-framework/configuration-service/domain"
	"net/http"

	"github.com/compliance-framework/configuration-service/api"
	//"github.com/compliance-framework/configuration-service/domain"
	"github.com/compliance-framework/configuration-service/service"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ResultsHandler struct {
	service *service.ResultsService
	sugar   *zap.SugaredLogger
}

func (h *ResultsHandler) Register(api *echo.Group) {
	api.GET("", h.All)
	api.POST("", h.Create)
}

func NewResultsHandler(l *zap.SugaredLogger, s *service.ResultsService) *ResultsHandler {
	return &ResultsHandler{
		sugar:   l,
		service: s,
	}
}

func (h *ResultsHandler) Create(c echo.Context) error {
	result := []byte{}
	_, err := c.Request().Body.Read(result)
	if err != nil {
		h.sugar.Error(err)
	}

	res := &domain.Result{}
	err = json.NewDecoder(c.Request().Body).Decode(res)
	if err != nil {
		h.sugar.Error(err)
		return c.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	savedResult, err := h.service.Create(c.Request().Context(), res)
	if err != nil {
		h.sugar.Error(err)
		return c.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	fmt.Println(savedResult)

	return c.JSON(http.StatusOK, savedResult)
}

func (h *ResultsHandler) All(c echo.Context) error {
	savedResults, err := h.service.GetAll(c.Request().Context())
	if err != nil {
		h.sugar.Error(err)
	}

	fmt.Println(savedResults)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return c.JSON(http.StatusOK, savedResults)
}
