package oscal

import (
	"errors"
	"net/http"

	"github.com/compliance-framework/configuration-service/internal/api"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/compliance-framework/configuration-service/internal/api/handler"
	"github.com/compliance-framework/configuration-service/internal/service/relational"
)

type ComponentDefinitionHandler struct {
	sugar *zap.SugaredLogger
	db    *gorm.DB
}

func NewComponentDefinitionHandler(sugar *zap.SugaredLogger, db *gorm.DB) *ComponentDefinitionHandler {
	return &ComponentDefinitionHandler{
		sugar: sugar,
		db:    db,
	}
}

func (h *ComponentDefinitionHandler) Register(api *echo.Group) {
	api.GET("", h.List)
	// api.POST("", h.Create)
	api.GET("/:id", h.Get)
}

// List godoc
//
//	@Summary		List component definitions
//	@Description	Retrieves all component definitions.
//	@Tags			Oscal
//	@Produce		json
//	@Success		200	{object}	handler.GenericDataListResponse[oscal.List.responseComponentDefinition]
//	@Failure		400	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/oscal/component-definitions [get]
func (h *ComponentDefinitionHandler) List(ctx echo.Context) error {
	type responseComponentDefinition struct {
		UUID     uuid.UUID                 `json:"uuid"`
		Metadata oscalTypes_1_1_3.Metadata `json:"metadata"`
	}

	var componentDefinitions []relational.ComponentDefinition
	if err := h.db.
		Preload("Metadata").
		Preload("Metadata.Revisions").
		Find(&componentDefinitions).Error; err != nil {
		h.sugar.Warnw("Failed to load component definitions", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	oscalComponentDefinitions := []oscalTypes_1_1_3.ComponentDefinition{}
	for _, componentDefinition := range componentDefinitions {
		oscalComponentDefinitions = append(oscalComponentDefinitions, *componentDefinition.MarshalOscal())
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.ComponentDefinition]{Data: oscalComponentDefinitions})
}

// Get godoc
//
//	@Summary		Get a component definition
//	@Description	Retrieves a single component definition by its unique ID.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id	path		string	true	"Component Definition ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscal.Get.responseComponentDefinition]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/oscal/component-definitions/{id} [get]
func (h *ComponentDefinitionHandler) Get(ctx echo.Context) error {
	type responseComponentDefinition struct {
		UUID     uuid.UUID                 `json:"uuid"`
		Metadata oscalTypes_1_1_3.Metadata `json:"metadata"`
	}

	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.
		Preload("Metadata").
		Preload("Metadata.Revisions").
		First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.sugar.Warnw("Component definition not found", "id", idParam)
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.ComponentDefinition]{Data: *componentDefinition.MarshalOscal()})
}
