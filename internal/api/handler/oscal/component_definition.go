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
	api.GET("/:id/import-component-definitions", h.GetImportComponentDefinitions)
	api.POST("/:id/import-component-definitions", h.CreateImportComponentDefinitions)
	api.PUT("/:id/import-component-definitions", h.UpdateImportComponentDefinitions)
	api.GET("/:id/components", h.GetComponents)
	api.POST("/:id/components", h.CreateComponents)
	api.PUT("/:id/components", h.UpdateComponents)
	api.GET("/:id/components/:defined-component", h.GetDefinedComponent)
	api.PUT("/:id/components/:defined-component", h.UpdateDefinedComponent)
	api.GET("/:id/components/:defined-component/control-implementations", h.GetControlImplementations)
	api.GET("/:id/components/:defined-component/control-implementations/implemented-requirements", h.GetImplementedRequirements)
	api.GET("/:id/components/:defined-component/control-implementations/statements", h.GetStatements)
	api.GET("/:id/capabilities", h.GetCapabilities)
	api.GET("/:id/capabilities/incorporates-components", h.GetIncorporatesComponents)
	api.GET("/:id/back-matter", h.GetBackMatter)

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

	// Validate required fields
	if oscalCat.UUID == "" {
		h.sugar.Warnw("Missing required field: UUID")
		return ctx.JSON(http.StatusBadRequest, api.NewError(errors.New("UUID is required")))
	}

	// Begin a transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		h.sugar.Errorf("Failed to begin transaction: %v", tx.Error)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(tx.Error))
	}

	// Check if component definition exists
	var existingComponent relational.ComponentDefinition
	if err := tx.First(&existingComponent, "id = ?", id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Errorf("Failed to find component definition: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update component definition
	now := time.Now()
	relCat := &relational.ComponentDefinition{}
	relCat.UnmarshalOscal(oscalCat)
	relCat.ID = &id // Ensure ID is set correctly

	// Validate the unmarshaled data
	if relCat.Metadata.Title == "" {
		tx.Rollback()
		h.sugar.Warnw("Missing required field: Metadata.Title")
		return ctx.JSON(http.StatusBadRequest, api.NewError(errors.New("Metadata.Title is required")))
	}

	// Update component definition with import_component_definitions
	if err := tx.Model(&existingComponent).Where("id = ?", id).Update("import_component_definitions", relCat.ImportComponentDefinitions).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update component definition: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update metadata
	metadataUpdates := map[string]interface{}{
		"last_modified": now,
		"oscal_version": versioning.GetLatestSupportedVersion(),
	}
	if err := tx.Model(&relational.Metadata{}).Where("id = ?", existingComponent.Metadata.ID).Updates(metadataUpdates).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update metadata: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		h.sugar.Errorf("Failed to commit transaction: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.ComponentDefinition]{Data: *relCat.MarshalOscal()})
}

// GetBackMatter godoc
//
//	@Summary		Get back-matter for a Component Definition
//	@Description	Retrieves the back-matter for a given Component Definition.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id	path		string	true	"Component Definition ID"
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

// CreateComponents godoc
//
//	@Summary		Create components for a component definition
//	@Description	Creates new components for a given component definition.
//	@Tags			Oscal
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string								true	"Component Definition ID"
//	@Param			components	body		[]oscalTypes_1_1_3.DefinedComponent	true	"Components to create"
//	@Success		200			{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.DefinedComponent]
//	@Failure		400			{object}	api.Error
//	@Failure		404			{object}	api.Error
//	@Failure		500			{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/components [post]
func (h *ComponentDefinitionHandler) CreateComponents(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var components []oscalTypes_1_1_3.DefinedComponent
	if err := ctx.Bind(&components); err != nil {
		h.sugar.Warnw("Failed to bind components", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Begin a transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		h.sugar.Errorf("Failed to begin transaction: %v", tx.Error)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(tx.Error))
	}

	// Convert to relational model
	var newComponents []relational.DefinedComponent
	for _, component := range components {
		relationalComponent := relational.DefinedComponent{}
		relationalComponent.UnmarshalOscal(component)
		newComponents = append(newComponents, relationalComponent)
	}

	// Append new components to existing ones
	existingComponents := componentDefinition.Components
	updatedComponents := append(existingComponents, newComponents...)

	// Update the component definition with the new components
	if err := tx.Model(&componentDefinition).Update("components", updatedComponents).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update components: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update metadata
	now := time.Now()
	metadataUpdates := map[string]interface{}{
		"last_modified": now,
		"oscal_version": versioning.GetLatestSupportedVersion(),
	}
	if err := tx.Model(&componentDefinition.Metadata).Updates(metadataUpdates).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update metadata: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		h.sugar.Errorf("Failed to commit transaction: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.DefinedComponent]{
		Data: components,
	})
}

// UpdateComponents godoc
//
//	@Summary		Update components for a component definition
//	@Description	Updates the components for a given component definition.
//	@Tags			Oscal
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string								true	"Component Definition ID"
//	@Param			components	body		[]oscalTypes_1_1_3.DefinedComponent	true	"Components to update"
//	@Success		200			{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.DefinedComponent]
//	@Failure		400			{object}	api.Error
//	@Failure		404			{object}	api.Error
//	@Failure		500			{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/components [put]
func (h *ComponentDefinitionHandler) UpdateComponents(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var oscalComponents []oscalTypes_1_1_3.DefinedComponent
	if err := ctx.Bind(&oscalComponents); err != nil {
		h.sugar.Warnw("Failed to bind components", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Begin a transaction to ensure data consistency
	tx := h.db.Begin()
	if tx.Error != nil {
		h.sugar.Errorf("Failed to begin transaction: %v", tx.Error)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(tx.Error))
	}

	// Update each component individually to preserve existing data
	for _, oscalComponent := range oscalComponents {
		relationalComponent := relational.DefinedComponent{}
		relationalComponent.UnmarshalOscal(oscalComponent)
		relationalComponent.ComponentDefinitionID = id

		// Check if the component exists first
		var existingComponent relational.DefinedComponent
		result := tx.First(&existingComponent, "id = ?", relationalComponent.ID)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// Component doesn't exist, create it
				if err := tx.Create(&relationalComponent).Error; err != nil {
					tx.Rollback()
					h.sugar.Errorf("Failed to create new component: %v", err)
					return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
				}
			} else {
				tx.Rollback()
				h.sugar.Errorf("Failed to check if component exists: %v", result.Error)
				return ctx.JSON(http.StatusInternalServerError, api.NewError(result.Error))
			}
		} else {
			// Component exists, update it using a map instead of struct to handle zero values
			updateFields := map[string]interface{}{
				"component_definition_id": id,
				"title":                   relationalComponent.Title,
				"description":             relationalComponent.Description,
				"purpose":                 relationalComponent.Purpose,
				"type":                    relationalComponent.Type,
				"remarks":                 relationalComponent.Remarks,
			}

			// Include related elements if they exist
			if relationalComponent.Props != nil {
				updateFields["props"] = relationalComponent.Props
			}
			if relationalComponent.Links != nil {
				updateFields["links"] = relationalComponent.Links
			}
			if relationalComponent.ResponsibleRoles != nil {
				updateFields["responsible_roles"] = relationalComponent.ResponsibleRoles
			}
			if relationalComponent.Protocols != nil {
				updateFields["protocols"] = relationalComponent.Protocols
			}

			if err := tx.Model(&relational.DefinedComponent{}).Where("id = ?", relationalComponent.ID).Updates(updateFields).Error; err != nil {
				tx.Rollback()
				h.sugar.Errorf("Failed to update component: %v", err)
				return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
			}
		}
	}

	// Update metadata
	now := time.Now()
	metadataUpdates := map[string]interface{}{
		"last_modified": now,
		"oscal_version": versioning.GetLatestSupportedVersion(),
	}
	if err := tx.Model(&relational.Metadata{}).Where("id = ?", componentDefinition.Metadata.ID).Updates(metadataUpdates).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update metadata: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		h.sugar.Errorf("Failed to commit transaction: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.DefinedComponent]{
		Data: oscalComponents,
	})
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

// GetDefinedComponent godoc
//
//	@Summary		Get a defined component for a component definition
//	@Description	Retrieves a defined component for a given component definition.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id					path		string	true	"Component Definition ID"
//	@Param			defined-component	path		string	true	"Defined Component ID"
//	@Success		200					{object}	handler.GenericDataResponse[oscalTypes_1_1_3.DefinedComponent]
//	@Failure		400					{object}	api.Error
//	@Failure		404					{object}	api.Error
//	@Failure		500					{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/components/{defined-component} [get]
func (h *ComponentDefinitionHandler) GetDefinedComponent(ctx echo.Context) error {
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

	definedComponentID := ctx.Param("defined-component")
	var definedComponent relational.DefinedComponent
	if err := h.db.First(&definedComponent, "id = ?", definedComponentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load defined component", "id", definedComponentID, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.DefinedComponent]{Data: *definedComponent.MarshalOscal()})
}

// UpdateDefinedComponent godoc
//
//	@Summary		Update a defined component for a component definition
//	@Description	Updates a defined component for a given component definition.
//	@Tags			Oscal
//	@Accept			json
//	@Produce		json
//	@Param			id					path		string								true	"Component Definition ID"
//	@Param			defined-component	path		string								true	"Defined Component ID"
//	@Param			defined-component	body		oscalTypes_1_1_3.DefinedComponent	true	"Defined Component to update"
//	@Success		200					{object}	handler.GenericDataResponse[oscalTypes_1_1_3.DefinedComponent]
//	@Failure		400					{object}	api.Error
//	@Failure		404					{object}	api.Error
//	@Failure		500					{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/components/{defined-component} [put]
func (h *ComponentDefinitionHandler) UpdateDefinedComponent(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	definedComponentID := ctx.Param("defined-component")
	var definedComponent relational.DefinedComponent
	if err := h.db.First(&definedComponent, "id = ?", definedComponentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load defined component", "id", definedComponentID, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var oscalDefinedComponent oscalTypes_1_1_3.DefinedComponent
	if err := ctx.Bind(&oscalDefinedComponent); err != nil {
		h.sugar.Warnw("Failed to bind defined component", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Begin a transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		h.sugar.Errorf("Failed to begin transaction: %v", tx.Error)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(tx.Error))
	}

	// Update only the fields that are provided in the request
	definedComponent.UnmarshalOscal(oscalDefinedComponent)
	definedComponent.ComponentDefinitionID = id // Ensure proper association

	// Convert struct to map for updates to properly handle zero values
	updateFields := map[string]interface{}{
		"component_definition_id": id,
		"title":                   definedComponent.Title,
		"description":             definedComponent.Description,
		"purpose":                 definedComponent.Purpose,
		"type":                    definedComponent.Type,
		"remarks":                 definedComponent.Remarks,
	}

	// Include related elements if they exist
	if definedComponent.Props != nil {
		updateFields["props"] = definedComponent.Props
	}
	if definedComponent.Links != nil {
		updateFields["links"] = definedComponent.Links
	}
	if definedComponent.ResponsibleRoles != nil {
		updateFields["responsible_roles"] = definedComponent.ResponsibleRoles
	}
	if definedComponent.Protocols != nil {
		updateFields["protocols"] = definedComponent.Protocols
	}

	// Use explicit WHERE clause with the primary key
	if err := tx.Model(&relational.DefinedComponent{}).Where("id = ?", definedComponentID).Updates(updateFields).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update defined component: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update metadata
	now := time.Now()
	metadataUpdates := map[string]interface{}{
		"last_modified": now,
		"oscal_version": versioning.GetLatestSupportedVersion(),
	}
	if err := tx.Model(&componentDefinition.Metadata).Updates(metadataUpdates).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update metadata: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		h.sugar.Errorf("Failed to commit transaction: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.DefinedComponent]{Data: *definedComponent.MarshalOscal()})
}

// GetControlImplementations godoc
//
//	@Summary		Get control implementations for a defined component
//	@Description	Retrieves all control implementations for a given defined component.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id					path		string	true	"Component Definition ID"
//	@Param			defined-component	path		string	true	"Defined Component ID"
//	@Success		200					{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.ControlImplementationSet]
//	@Failure		400					{object}	api.Error
//	@Failure		404					{object}	api.Error
//	@Failure		500					{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/components/{defined-component}/control-implementations [get]
func (h *ComponentDefinitionHandler) GetControlImplementations(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	definedComponentID := ctx.Param("defined-component")
	var definedComponent relational.DefinedComponent
	if err := h.db.First(&definedComponent, "id = ?", definedComponentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load defined component", "id", definedComponentID, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Get the control implementation set IDs from the join table
	var controlImplSetIDs []string
	if err := h.db.Table("defined_components_control_implementation_sets").
		Where("defined_component_id = ?", definedComponentID).
		Pluck("control_implementation_set_id", &controlImplSetIDs).Error; err != nil {
		h.sugar.Warnw("Failed to load control implementation set IDs", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Create a result array for valid control implementation sets
	var oscalControlImplementations []oscalTypes_1_1_3.ControlImplementationSet

	// For each control implementation set ID, try to load and marshal it
	for _, controlImplSetID := range controlImplSetIDs {
		var controlImplSet relational.ControlImplementationSet

		// Try to get the control implementation set and preload its implemented requirements
		if err := h.db.
			Preload("ImplementedRequirements").
			Preload("ImplementedRequirements.Statements").
			First(&controlImplSet, "id = ?", controlImplSetID).Error; err != nil {
			// Log the error but continue with other control implementation sets
			h.sugar.Warnw("Failed to load control implementation set", "id", controlImplSetID, "error", err)
			continue
		}

		// Try to marshal it to OSCAL format
		oscalImpl := controlImplSet.MarshalOscal()
		if oscalImpl != nil {
			oscalControlImplementations = append(oscalControlImplementations, *oscalImpl)
		}
	}

	// Ensure we always return an array, even if empty
	if oscalControlImplementations == nil {
		oscalControlImplementations = []oscalTypes_1_1_3.ControlImplementationSet{}
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.ControlImplementationSet]{
		Data: oscalControlImplementations,
	})
}

// GetImplementedRequirements godoc
//
//	@Summary		Get implemented requirements for a defined component
//	@Description	Retrieves all implemented requirements for a given defined component.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id					path		string	true	"Component Definition ID"
//	@Param			defined-component	path		string	true	"Defined Component ID"
//	@Success		200					{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.ImplementedRequirementControlImplementation]
//	@Failure		400					{object}	api.Error
//	@Failure		404					{object}	api.Error
//	@Failure		500					{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/components/{defined-component}/control-implementations/implemented-requirements [get]
func (h *ComponentDefinitionHandler) GetImplementedRequirements(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	definedComponentID := ctx.Param("defined-component")
	var definedComponent relational.DefinedComponent
	if err := h.db.
		Preload("ControlImplementations").
		Preload("ControlImplementations.ImplementedRequirements").
		First(&definedComponent, "id = ?", definedComponentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load defined component", "id", definedComponentID, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var oscalImplementedRequirements []oscalTypes_1_1_3.ImplementedRequirementControlImplementation
	for _, controlImpl := range definedComponent.ControlImplementations {
		for _, implementedRequirement := range controlImpl.ImplementedRequirements {
			oscalImplementedRequirements = append(oscalImplementedRequirements, *implementedRequirement.MarshalOscal())
		}
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.ImplementedRequirementControlImplementation]{Data: oscalImplementedRequirements})
}

// GetStatements godoc
//
//	@Summary		Get statements for a defined component
//	@Description	Retrieves all statements for a given defined component.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id					path		string	true	"Component Definition ID"
//	@Param			defined-component	path		string	true	"Defined Component ID"
//	@Success		200					{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.ControlStatementImplementation]
//	@Failure		400					{object}	api.Error
//	@Failure		404					{object}	api.Error
//	@Failure		500					{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/components/{defined-component}/control-implementations/statements [get]
func (h *ComponentDefinitionHandler) GetStatements(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	definedComponentID := ctx.Param("defined-component")
	var definedComponent relational.DefinedComponent
	if err := h.db.
		Preload("ControlImplementations").
		Preload("ControlImplementations.ImplementedRequirements").
		Preload("ControlImplementations.ImplementedRequirements.Statements").
		First(&definedComponent, "id = ?", definedComponentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load defined component", "id", definedComponentID, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var oscalStatements []oscalTypes_1_1_3.ControlStatementImplementation
	for _, controlImpl := range definedComponent.ControlImplementations {
		for _, statement := range controlImpl.ImplementedRequirements {
			for _, stmt := range statement.Statements {
				oscalStatements = append(oscalStatements, *stmt.MarshalOscal())
			}
		}
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.ControlStatementImplementation]{Data: oscalStatements})
}

// GetImportComponentDefinitions godoc
//
//	@Summary		Get import component definitions for a defined component
//	@Description	Retrieves all import component definitions for a given defined component.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id					path		string	true	"Component Definition ID"
//	@Param			defined-component	path		string	true	"Defined Component ID"
//	@Success		200					{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.ImportComponentDefinition]
//	@Failure		400					{object}	api.Error
//	@Failure		404					{object}	api.Error
//	@Failure		500					{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/import-component-definitions [get]
func (h *ComponentDefinitionHandler) GetImportComponentDefinitions(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var oscalImportComponentDefinitions []oscalTypes_1_1_3.ImportComponentDefinition
	for _, importComponentDefinition := range componentDefinition.ImportComponentDefinitions {
		oscalImportComponentDefinitions = append(oscalImportComponentDefinitions, *importComponentDefinition.MarshalOscal())
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.ImportComponentDefinition]{Data: oscalImportComponentDefinitions})
}

// CreateImportComponentDefinitions godoc
//
//	@Summary		Create import component definitions for a component definition
//	@Description	Creates new import component definitions for a given component definition.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id								path		string											true	"Component Definition ID"
//	@Param			import-component-definitions	body		[]oscalTypes_1_1_3.ImportComponentDefinition	true	"Import Component Definitions"
//	@Success		200								{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.ImportComponentDefinition]
//	@Failure		400								{object}	api.Error
//	@Failure		404								{object}	api.Error
//	@Failure		500								{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/import-component-definitions [post]
func (h *ComponentDefinitionHandler) CreateImportComponentDefinitions(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var importComponentDefinitions []oscalTypes_1_1_3.ImportComponentDefinition
	if err := ctx.Bind(&importComponentDefinitions); err != nil {
		h.sugar.Warnw("Failed to bind import component definitions", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Begin a transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		h.sugar.Errorf("Failed to begin transaction: %v", tx.Error)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(tx.Error))
	}

	// Convert to relational model
	var newImportDefs []relational.ImportComponentDefinition
	for _, importDef := range importComponentDefinitions {
		relationalImportDef := relational.ImportComponentDefinition{}
		relationalImportDef.UnmarshalOscal(importDef)
		newImportDefs = append(newImportDefs, relationalImportDef)
	}

	// Append new import definitions to existing ones
	existingImports := componentDefinition.ImportComponentDefinitions
	updatedImports := append(existingImports, newImportDefs...)

	// Update the component definition with the new import definitions
	if err := tx.Model(&componentDefinition).Update("import_component_definitions", updatedImports).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update import component definitions: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update metadata
	now := time.Now()
	metadataUpdates := map[string]interface{}{
		"last_modified": now,
		"oscal_version": versioning.GetLatestSupportedVersion(),
	}
	if err := tx.Model(&componentDefinition.Metadata).Updates(metadataUpdates).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update metadata: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		h.sugar.Errorf("Failed to commit transaction: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.ImportComponentDefinition]{
		Data: importComponentDefinitions,
	})
}

// UpdateImportComponentDefinitions godoc
//
//	@Summary		Update import component definitions for a component definition
//	@Description	Updates the import component definitions for a given component definition.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id								path		string											true	"Component Definition ID"
//	@Param			import-component-definitions	body		[]oscalTypes_1_1_3.ImportComponentDefinition	true	"Import Component Definitions"
//	@Success		200								{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.ImportComponentDefinition]
//	@Failure		400								{object}	api.Error
//	@Failure		404								{object}	api.Error
//	@Failure		500								{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/import-component-definitions [put]
func (h *ComponentDefinitionHandler) UpdateImportComponentDefinitions(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.First(&componentDefinition, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var importComponentDefinitions []oscalTypes_1_1_3.ImportComponentDefinition
	if err := ctx.Bind(&importComponentDefinitions); err != nil {
		h.sugar.Warnw("Failed to bind import component definitions", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Begin a transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		h.sugar.Errorf("Failed to begin transaction: %v", tx.Error)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(tx.Error))
	}

	// Convert to relational model
	var newImportDefs []relational.ImportComponentDefinition
	for _, importDef := range importComponentDefinitions {
		relationalImportDef := relational.ImportComponentDefinition{}
		relationalImportDef.UnmarshalOscal(importDef)
		newImportDefs = append(newImportDefs, relationalImportDef)
	}

	// Update the import component definitions using Updates
	if err := tx.Model(&componentDefinition).Updates(map[string]interface{}{
		"import_component_definitions": newImportDefs,
	}).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update import component definitions: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update metadata
	now := time.Now()
	metadataUpdates := map[string]interface{}{
		"last_modified": now,
		"oscal_version": versioning.GetLatestSupportedVersion(),
	}
	if err := tx.Model(&componentDefinition.Metadata).Updates(metadataUpdates).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update metadata: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		h.sugar.Errorf("Failed to commit transaction: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.ImportComponentDefinition]{
		Data: importComponentDefinitions,
	})
}

// GetIncorporatesComponents godoc
//
//	@Summary		Get incorporates components for a component definition
//	@Description	Retrieves all incorporates components for a given component definition.
//	@Tags			Oscal
//	@Produce		json
//	@Param			id	path		string	true	"Component Definition ID"
//	@Success		200	{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.IncorporatesComponent]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/oscal/component-definitions/{id}/capabilities/incorporates-components [get]
func (h *ComponentDefinitionHandler) GetIncorporatesComponents(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid component definition id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var componentDefinition relational.ComponentDefinition
	if err := h.db.First(&componentDefinition, "id = ?", id).Error; err != nil {
		h.sugar.Warnw("Failed to load component definition", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var oscalIncorporatesComponents []oscalTypes_1_1_3.IncorporatesComponent
	for _, capability := range componentDefinition.Capabilities {
		for _, component := range capability.IncorporatesComponents {
			oscalIncorporatesComponents = append(oscalIncorporatesComponents, oscalTypes_1_1_3.IncorporatesComponent(component))
		}
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.IncorporatesComponent]{Data: oscalIncorporatesComponents})
}
