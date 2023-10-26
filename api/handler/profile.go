package handler

import (
	"net/http"

	"github.com/compliance-framework/configuration-service/api"
	"github.com/compliance-framework/configuration-service/domain"
	"github.com/compliance-framework/configuration-service/service"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ProfileHandler struct {
	service *service.ProfileService
	sugar   *zap.SugaredLogger
}

func (h *ProfileHandler) Register(api *echo.Group) {
	api.POST("/profile", h.CreateProfile)
	api.GET("/profile/:id", h.GetById)
	api.GET("/profile/:title", h.GetByTitle)
}

func NewProfileHandler(l *zap.SugaredLogger, s *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		sugar:   l,
		service: s,
	}
}

// CreateProfile godoc
// @Summary 		Create a profile
// @Description 	Creates a new profile in the system
// @Accept  		json
// @Produce  		json
// @Param   		profile body createProfileRequest true "Profile to add"
// @Success 		201 {object} profileIdResponse
// @Failure 		401 {object} apiError
// @Failure 		422 {object} apiError
// @Failure 		500 {object} apiError
// @Router 			/api/profile [post]
func (h *ProfileHandler) CreateProfile(ctx echo.Context) error {
	// Initialize a new profile object
	p := domain.NewProfile()
	req := createProfileRequest{}

	if err := req.bind(ctx, p); err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, api.NewError(err))
	}

	id, err := h.service.Create(p)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusCreated, profileIdResponse{
		Id: id,
	})
}

// GetProfileById godoc
// @Summary 		Get a profile by id
// @Description 	Get a profile by its id
// @Accept  		json
// @Produce  		json
// @Param   		id path string true "Profile id"
// @Success 		200 {object} profile
// @Failure 		401 {object} apiError
// @Failure 		404 {object} apiError
// @Failure 		500 {object} apiError
// @Router 			/api/profile/{id} [get]
func (h *ProfileHandler) GetById(ctx echo.Context) error {
	profile, err := h.service.GetById(ctx.Param("id"))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	if profile == nil {
		return ctx.JSON(http.StatusNotFound, api.NotFound())
	}

	return ctx.JSON(http.StatusOK, profile)
}

// GetByTitle godoc
// @Summary 		Get a profile by title
// @Description 	Get a profile by its title
// @Accept  		json
// @Produce  		json
// @Param   		title path string true "Profile title"
// @Success 		200 {object} profile
// @Failure 		401 {object} apiError
// @Failure 		404 {object} apiError
// @Failure 		500 {object} apiError
// @Router 			/api/profile/{title} [get]
func (h *ProfileHandler) GetByTitle(ctx echo.Context) error {
	profile, err := h.service.GetByTitle(ctx.Param("title"))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	if profile == nil {
		return ctx.JSON(http.StatusNotFound, api.NotFound())
	}

	return ctx.JSON(http.StatusOK, profile)
}
