package oscal

import (
	"errors"
	"net/http"
	"time"

	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/defenseunicorns/go-oscal/src/pkg/versioning"
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
	api.POST("", h.Create)
	api.GET("/:id", h.Get)
	api.PUT("/:id", h.Update)
	api.GET("/:id/full", h.Full)
	api.GET("/:id/back-matter", h.GetBackMatter)
	api.GET("/:id/components", h.GetComponents)
	api.GET("/:id/capabilities", h.GetCapabilities)
	api.GET("/:id/import-component-definitions", h.GetImportComponentDefinitions)
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

// Create godoc
//
//	@Summary		Create a component definition
//	@Description	Creates a new component definition.
//	@Tags			Oscal
//	@Accept			json
//	@Produce		json
//	@Param			componentDefinition	body		oscalTypes_1_1_3.ComponentDefinition	true	"Component Definition"
//	@Success		201					{object}	handler.GenericDataResponse[oscalTypes_1_1_3.ComponentDefinition]
//	@Failure		400					{object}	api.Error
//	@Failure		500					{object}	api.Error
//	@Router			/oscal/component-definitions [post]
func (h *ComponentDefinitionHandler) Create(ctx echo.Context) error {
	now := time.Now()

	var oscalCat oscalTypes_1_1_3.ComponentDefinition
	if err := ctx.Bind(&oscalCat); err != nil {
		h.sugar.Warnw("Invalid create component definition request", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	relCat := &relational.ComponentDefinition{}
	relCat.UnmarshalOscal(oscalCat)
	relCat.Metadata.LastModified = &now
	relCat.Metadata.OscalVersion = versioning.GetLatestSupportedVersion()
	if err := h.db.Create(relCat).Error; err != nil {
		h.sugar.Errorf("Failed to create component definition: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return ctx.JSON(http.StatusCreated, handler.GenericDataResponse[oscalTypes_1_1_3.ComponentDefinition]{Data: *relCat.MarshalOscal()})
}

// Update godoc
//
//	@Summary		Update a component definition
//	@Description	Updates an existing component definition.
//	@Tags			Oscal
//	@Accept			json
//	@Produce		json
//	@Param			id					path		string									true	"Component Definition ID"
//	@Param			componentDefinition	body		oscalTypes_1_1_3.ComponentDefinition	true	"Updated Component Definition object"
//	@Success		200					{object}	handler.GenericDataResponse[oscalTypes_1_1_3.ComponentDefinition]
//	@Failure		400					{object}	api.Error
//	@Failure		404					{object}	api.Error
//	@Failure		500					{object}	api.Error
//	@Router			/oscal/component-definitions/{id} [put]
func (h *ComponentDefinitionHandler) Update(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var oscalCat oscalTypes_1_1_3.ComponentDefinition
	if err := ctx.Bind(&oscalCat); err != nil {
		h.sugar.Warnw("Invalid update component definition request", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	now := time.Now()
	relCat := &relational.ComponentDefinition{}
	relCat.UnmarshalOscal(oscalCat)
	relCat.Metadata.LastModified = &now
	relCat.Metadata.OscalVersion = versioning.GetLatestSupportedVersion()
	if err := h.db.Where("id = ?", id).Updates(relCat).Error; err != nil {
		h.sugar.Errorf("Failed to update component definition: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.ComponentDefinition]{Data: *relCat.MarshalOscal()})
}

// GetBackMatter godoc
//
//	@Summary		Get back-matter for a Catalog
//	@Description	Retrieves the back-matter for a given Catalog.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id	path		string	true	"Catalog ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.BackMatter]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/back-matter [get]
func (h *ComponentDefinitionHandler) GetBackMatter(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.
		Preload("BackMatter").
		Preload("BackMatter.Resources").
		First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	//handler.GenericDataResponse[struct {
	//			UUID     uuid.UUID           `json:"uuid"`
	//			Metadata relational.Metadata `json:"metadata"`
	//		}]{}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[*oscalTypes_1_1_3.BackMatter]{Data: componentDefinition.BackMatter.MarshalOscal()})
}

// Full godoc
//
//	@Summary		Get a complete Component Definition
//	@Description	Retrieves a complete Component Definition by its ID, including all metadata and revisions.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id	path		string	true	"Component Definition ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.ComponentDefinition]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/full [get]
func (h *ComponentDefinitionHandler) Full(ctx echo.Context) error {
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
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.ComponentDefinition]{Data: *componentDefinition.MarshalOscal()})
}

// GetComponents godoc
//
//	@Summary		Get components for a component definition
//	@Description	Retrieves all components for a given component definition.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id	path		string	true	"Component Definition ID"
//	@Success		200	{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.DefinedComponent]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/components [get]
func (h *ComponentDefinitionHandler) GetComponents(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.
		Preload("Components").
		First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	oscalComponents := make([]oscalTypes_1_1_3.DefinedComponent, len(componentDefinition.Components))
	for i, component := range componentDefinition.Components {
		oscalComponents[i] = *component.MarshalOscal()
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.DefinedComponent]{Data: oscalComponents})
}

// GetCapabilities godoc
//
//	@Summary		Get capabilities for a component definition
//	@Description	Retrieves all capabilities for a given component definition.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id	path		string	true	"Component Definition ID"
//	@Success		200	{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.Capability]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/capabilities [get]
func (h *ComponentDefinitionHandler) GetCapabilities(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.
		Preload("Capabilities").
		First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	oscalCapabilities := make([]oscalTypes_1_1_3.Capability, len(componentDefinition.Capabilities))
	for i, capability := range componentDefinition.Capabilities {
		oscalCapabilities[i] = *capability.MarshalOscal()
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.Capability]{Data: oscalCapabilities})
}

// GetImportComponentDefinitions godoc
//
//	@Summary		Get import component definitions for a component definition
//	@Description	Retrieves all import component definitions for a given component definition.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id	path		string	true	"Component Definition ID"
//	@Success		200	{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.ImportComponentDefinition]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/import-component-definitions [get]
func (h *ComponentDefinitionHandler) GetImportComponentDefinitions(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.
		Preload("ImportComponentDefinitions").
		First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	oscalImportComponentDefinitions := make([]oscalTypes_1_1_3.ImportComponentDefinition, len(componentDefinition.ImportComponentDefinitions))
	for i, importComponentDefinition := range componentDefinition.ImportComponentDefinitions {
		oscalImportComponentDefinitions[i] = *importComponentDefinition.MarshalOscal()
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.ImportComponentDefinition]{Data: oscalImportComponentDefinitions})
}
