package oscal

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/defenseunicorns/go-oscal/src/pkg/versioning"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/compliance-framework/api/internal/api"
	"github.com/compliance-framework/api/internal/api/handler"
	"github.com/compliance-framework/api/internal/service/relational"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
)

// AssessmentResultsHandler handles OSCAL Assessment Results endpoints.
//
//	@Tags	Assessment Results
//
// All types are defined in oscalTypes_1_1_3 (see types.go)
type AssessmentResultsHandler struct {
	sugar *zap.SugaredLogger
	db    *gorm.DB
}

// NewAssessmentResultsHandler creates a new handler for Assessment Results endpoints.
func NewAssessmentResultsHandler(sugar *zap.SugaredLogger, db *gorm.DB) *AssessmentResultsHandler {
	return &AssessmentResultsHandler{
		sugar: sugar,
		db:    db,
	}
}

// verifyAssessmentResultsExists checks if an Assessment Results exists by ID and returns appropriate HTTP error if not
func (h *AssessmentResultsHandler) verifyAssessmentResultsExists(ctx echo.Context, arID uuid.UUID) error {
	var ar relational.AssessmentResult
	if err := h.db.First(&ar, "id = ?", arID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(fmt.Errorf("assessment results not found")))
		}
		h.sugar.Errorf("Failed to find assessment results: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return nil
}

// Register registers Assessment Results endpoints to the API group.
func (h *AssessmentResultsHandler) Register(api *echo.Group) {
	api.GET("", h.List)          // GET /oscal/assessment-results
	api.POST("", h.Create)       // POST /oscal/assessment-results
	api.GET("/:id", h.Get)       // GET /oscal/assessment-results/:id
	api.PUT("/:id", h.Update)    // PUT /oscal/assessment-results/:id
	api.DELETE("/:id", h.Delete) // DELETE /oscal/assessment-results/:id
	api.GET("/:id/full", h.Full) // GET /oscal/assessment-results/:id/full
	api.GET("/:id/metadata", h.GetMetadata)
	api.PUT("/:id/metadata", h.UpdateMetadata)
	api.GET("/:id/import-ap", h.GetImportAp)
	api.PUT("/:id/import-ap", h.UpdateImportAp)
	api.GET("/:id/local-definitions", h.GetLocalDefinitions)
	api.PUT("/:id/local-definitions", h.UpdateLocalDefinitions)
	api.GET("/:id/results", h.GetResults)
	api.POST("/:id/results", h.CreateResult)
	api.GET("/:id/results/:resultId", h.GetResult)
	api.PUT("/:id/results/:resultId", h.UpdateResult)
	api.DELETE("/:id/results/:resultId", h.DeleteResult)
	api.GET("/:id/results/:resultId/observations", h.GetResultObservations)
	api.POST("/:id/results/:resultId/observations", h.CreateResultObservation)
	api.PUT("/:id/results/:resultId/observations/:obsId", h.UpdateResultObservation)
	api.DELETE("/:id/results/:resultId/observations/:obsId", h.DeleteResultObservation)
	api.GET("/:id/results/:resultId/risks", h.GetResultRisks)
	api.POST("/:id/results/:resultId/risks", h.CreateResultRisk)
	api.PUT("/:id/results/:resultId/risks/:riskId", h.UpdateResultRisk)
	api.DELETE("/:id/results/:resultId/risks/:riskId", h.DeleteResultRisk)
	api.GET("/:id/results/:resultId/findings", h.GetResultFindings)
	api.POST("/:id/results/:resultId/findings", h.CreateResultFinding)
	api.PUT("/:id/results/:resultId/findings/:findingId", h.UpdateResultFinding)
	api.DELETE("/:id/results/:resultId/findings/:findingId", h.DeleteResultFinding)
	api.GET("/:id/results/:resultId/attestations", h.GetResultAttestations)
	api.POST("/:id/results/:resultId/attestations", h.CreateResultAttestation)
	api.PUT("/:id/results/:resultId/attestations/:index", h.UpdateResultAttestation)
	api.DELETE("/:id/results/:resultId/attestations/:index", h.DeleteResultAttestation)
	api.GET("/:id/back-matter", h.GetBackMatter)
	api.POST("/:id/back-matter", h.CreateBackMatter)
	api.PUT("/:id/back-matter", h.UpdateBackMatter)
	api.DELETE("/:id/back-matter", h.DeleteBackMatter)
	api.GET("/:id/back-matter/resources", h.GetBackMatterResources)
	api.POST("/:id/back-matter/resources", h.CreateBackMatterResource)
	api.PUT("/:id/back-matter/resources/:resourceId", h.UpdateBackMatterResource)
	api.DELETE("/:id/back-matter/resources/:resourceId", h.DeleteBackMatterResource)
}

// validateAssessmentResultsInput validates Assessment Results input following OSCAL requirements
func (h *AssessmentResultsHandler) validateAssessmentResultsInput(ar *oscalTypes_1_1_3.AssessmentResults) error {
	if ar.UUID == "" {
		return fmt.Errorf("UUID is required")
	}
	if _, err := uuid.Parse(ar.UUID); err != nil {
		return fmt.Errorf("invalid UUID format: %v", err)
	}
	if ar.Metadata.Title == "" {
		return fmt.Errorf("metadata.title is required")
	}
	if ar.Metadata.Version == "" {
		return fmt.Errorf("metadata.version is required")
	}
	if ar.ImportAp.Href == "" {
		return fmt.Errorf("import-ap.href is required")
	}
	if len(ar.Results) == 0 {
		return fmt.Errorf("at least one result is required")
	}
	return nil
}

// validateResultInput validates Result input
func (h *AssessmentResultsHandler) validateResultInput(result *oscalTypes_1_1_3.Result) error {
	if result.UUID == "" {
		return fmt.Errorf("UUID is required")
	}
	if _, err := uuid.Parse(result.UUID); err != nil {
		return fmt.Errorf("invalid UUID format: %v", err)
	}
	if result.Title == "" {
		return fmt.Errorf("title is required")
	}
	if result.Description == "" {
		return fmt.Errorf("description is required")
	}
	if result.Start.IsZero() {
		return fmt.Errorf("start time is required")
	}
	return nil
}

// List godoc
//
//	@Summary		List Assessment Results
//	@Description	Retrieves all Assessment Results.
//	@Tags			Assessment Results
//	@Produce		json
//	@Success		200	{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.AssessmentResults]
//	@Failure		400	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results [get]
func (h *AssessmentResultsHandler) List(ctx echo.Context) error {
	var assessmentResults []relational.AssessmentResult
	if err := h.db.Preload("Metadata").Preload("Metadata.Revisions").Find(&assessmentResults).Error; err != nil {
		h.sugar.Errorw("failed to list assessment results", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	oscalARs := make([]oscalTypes_1_1_3.AssessmentResults, len(assessmentResults))
	for i, ar := range assessmentResults {
		oscalARs[i] = *ar.MarshalOscal()
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.AssessmentResults]{Data: oscalARs})
}

// Get godoc
//
//	@Summary		Get an Assessment Results
//	@Description	Retrieves a single Assessment Results by its unique ID.
//	@Tags			Assessment Results
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Results ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentResults]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id} [get]
func (h *AssessmentResultsHandler) Get(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	var ar relational.AssessmentResult
	if err := h.db.Preload("Metadata").Preload("Metadata.Revisions").First(&ar, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load assessment results", "id", idParam, "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentResults]{Data: *ar.MarshalOscal()})
}

// Full godoc
//
//	@Summary		Get a complete Assessment Results
//	@Description	Retrieves a complete Assessment Results by its ID, including all metadata and related objects.
//	@Tags			Assessment Results
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Results ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentResults]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/full [get]
func (h *AssessmentResultsHandler) Full(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	var ar relational.AssessmentResult
	if err := h.db.
		Preload("Metadata").
		Preload("Metadata.Revisions").
		Preload("Results").
		Preload("Results.Observations").
		Preload("Results.Risks").
		Preload("Results.Findings").
		Preload("BackMatter").
		Preload("BackMatter.Resources").
		First(&ar, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Errorw("failed to get assessment results", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentResults]{Data: *ar.MarshalOscal()})
}

// Create godoc
//
//	@Summary		Create an Assessment Results
//	@Description	Creates an Assessment Results from input.
//	@Tags			Assessment Results
//	@Accept			json
//	@Produce		json
//	@Param			ar	body		oscalTypes_1_1_3.AssessmentResults	true	"Assessment Results data"
//	@Success		201	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentResults]
//	@Failure		400	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results [post]
func (h *AssessmentResultsHandler) Create(ctx echo.Context) error {
	var oscalAR oscalTypes_1_1_3.AssessmentResults
	if err := ctx.Bind(&oscalAR); err != nil {
		h.sugar.Warnw("Invalid create assessment results request", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Validate input
	if err := h.validateAssessmentResultsInput(&oscalAR); err != nil {
		h.sugar.Warnw("Invalid assessment results input", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	now := time.Now()
	relAR := &relational.AssessmentResult{}
	relAR.UnmarshalOscal(oscalAR)
	relAR.Metadata.LastModified = &now
	relAR.Metadata.OscalVersion = versioning.GetLatestSupportedVersion()

	if err := h.db.Create(relAR).Error; err != nil {
		h.sugar.Errorf("Failed to create assessment results: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusCreated, handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentResults]{Data: *relAR.MarshalOscal()})
}

// Update godoc
//
//	@Summary		Update an Assessment Results
//	@Description	Updates an existing Assessment Results.
//	@Tags			Assessment Results
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string								true	"Assessment Results ID"
//	@Param			ar	body		oscalTypes_1_1_3.AssessmentResults	true	"Updated Assessment Results object"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentResults]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id} [put]
func (h *AssessmentResultsHandler) Update(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid assessment results id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var oscalAR oscalTypes_1_1_3.AssessmentResults
	if err := ctx.Bind(&oscalAR); err != nil {
		h.sugar.Warnw("Invalid update assessment results request", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	
	h.sugar.Infof("DEBUG: Received update request - Title: '%s', UUID: '%s'", oscalAR.Metadata.Title, oscalAR.UUID)

	// Validate required fields
	if err := h.validateAssessmentResultsInput(&oscalAR); err != nil {
		h.sugar.Warnw("Invalid assessment results input", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Begin a transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		h.sugar.Errorf("Failed to begin transaction: %v", tx.Error)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(tx.Error))
	}

	// Check if assessment results exists and preload metadata
	var existingAR relational.AssessmentResult
	if err := tx.Preload("Metadata").Preload("LocalDefinitions").First(&existingAR, "id = ?", id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Errorf("Failed to find assessment results: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update assessment results
	now := time.Now()
	relAR := &relational.AssessmentResult{}
	relAR.UnmarshalOscal(oscalAR)
	relAR.ID = &id // Ensure ID is set correctly

	// Update the main assessment results record (only simple fields)
	updateFields := map[string]any{
		"import_ap": relAR.ImportAp,
	}

	if err := tx.Model(&existingAR).Updates(updateFields).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update assessment results: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update metadata using the incoming OSCAL metadata fields
	// Build update map with all fields from the incoming metadata
	metadataUpdates := map[string]interface{}{
		"title":          oscalAR.Metadata.Title,
		"version":        oscalAR.Metadata.Version,
		"published":      oscalAR.Metadata.Published,
		"remarks":        oscalAR.Metadata.Remarks,
		"last_modified":  &now,
		"oscal_version":  versioning.GetLatestSupportedVersion(),
	}
	
	if err := tx.Model(&relational.Metadata{}).Where("id = ?", existingAR.Metadata.ID).Updates(metadataUpdates).Error; err != nil {
		tx.Rollback()
		h.sugar.Errorf("Failed to update metadata: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Handle LocalDefinitions if provided
	if relAR.LocalDefinitions != nil {
		if existingAR.LocalDefinitions != nil {
			// Update existing LocalDefinitions
			relAR.LocalDefinitions.ID = existingAR.LocalDefinitions.ID
			if err := tx.Model(&relational.LocalDefinitions{}).Where("id = ?", existingAR.LocalDefinitions.ID).Updates(relAR.LocalDefinitions).Error; err != nil {
				tx.Rollback()
				h.sugar.Errorf("Failed to update local definitions: %v", err)
				return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
			}
		} else {
			// Create new LocalDefinitions
			relAR.LocalDefinitions.ParentID = id
			relAR.LocalDefinitions.ParentType = "assessment_results"
			if err := tx.Create(relAR.LocalDefinitions).Error; err != nil {
				tx.Rollback()
				h.sugar.Errorf("Failed to create local definitions: %v", err)
				return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		h.sugar.Errorf("Failed to commit transaction: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Reload the updated assessment results with all associations
	var updatedAR relational.AssessmentResult
	if err := h.db.Preload("Metadata").
		Preload("LocalDefinitions").
		Preload("BackMatter").
		Where("id = ?", id).
		First(&updatedAR).Error; err != nil {
		h.sugar.Errorf("Failed to reload assessment results: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentResults]{Data: *updatedAR.MarshalOscal()})
}

// Delete godoc
//
//	@Summary		Delete an Assessment Results
//	@Description	Deletes an Assessment Results by its ID.
//	@Tags			Assessment Results
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Results ID"
//	@Success		200	{object}	api.Response
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id} [delete]
func (h *AssessmentResultsHandler) Delete(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Verify assessment results exists
	if err := h.verifyAssessmentResultsExists(ctx, id); err != nil {
		return err
	}

	if err := h.db.Delete(&relational.AssessmentResult{}, "id = ?", id).Error; err != nil {
		h.sugar.Errorf("Failed to delete assessment results: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, struct{Message string `json:"message"`}{Message: "Assessment results deleted successfully"})
}

// GetMetadata godoc
//
//	@Summary		Get Assessment Results metadata
//	@Description	Retrieves metadata for a given Assessment Results.
//	@Tags			Assessment Results
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Results ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.Metadata]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/metadata [get]
func (h *AssessmentResultsHandler) GetMetadata(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	var ar relational.AssessmentResult
	if err := h.db.Preload("Metadata").First(&ar, "id = ?", id).Error; err != nil {
		h.sugar.Errorw("failed to get assessment results", "error", err)
		return ctx.JSON(http.StatusNotFound, api.NewError(err))
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.Metadata]{Data: *ar.Metadata.MarshalOscal()})
}

// UpdateMetadata godoc
//
//	@Summary		Update Assessment Results metadata
//	@Description	Updates metadata for a given Assessment Results.
//	@Tags			Assessment Results
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string						true	"Assessment Results ID"
//	@Param			metadata	body		oscalTypes_1_1_3.Metadata	true	"Metadata data"
//	@Success		200			{object}	handler.GenericDataResponse[oscalTypes_1_1_3.Metadata]
//	@Failure		400			{object}	api.Error
//	@Failure		404			{object}	api.Error
//	@Failure		500			{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/metadata [put]
func (h *AssessmentResultsHandler) UpdateMetadata(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Verify assessment results exists
	if err := h.verifyAssessmentResultsExists(ctx, id); err != nil {
		return err
	}

	var oscalMetadata oscalTypes_1_1_3.Metadata
	if err := ctx.Bind(&oscalMetadata); err != nil {
		h.sugar.Warnw("Invalid update metadata request", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Validate required fields
	if oscalMetadata.Title == "" {
		h.sugar.Warnw("Invalid metadata input", "error", "title is required")
		return ctx.JSON(http.StatusBadRequest, api.NewError(fmt.Errorf("title is required")))
	}
	if oscalMetadata.Version == "" {
		h.sugar.Warnw("Invalid metadata input", "error", "version is required")
		return ctx.JSON(http.StatusBadRequest, api.NewError(fmt.Errorf("version is required")))
	}

	var ar relational.AssessmentResult
	if err := h.db.Preload("Metadata").First(&ar, "id = ?", id).Error; err != nil {
		h.sugar.Errorw("failed to get assessment results", "error", err)
		return ctx.JSON(http.StatusNotFound, api.NewError(err))
	}

	// Update metadata with current timestamp
	now := time.Now()
	relMetadata := &relational.Metadata{}
	relMetadata.UnmarshalOscal(oscalMetadata)
	relMetadata.LastModified = &now
	relMetadata.OscalVersion = versioning.GetLatestSupportedVersion()

	// Update the metadata
	if err := h.db.Model(&ar.Metadata).Updates(relMetadata).Error; err != nil {
		h.sugar.Errorf("Failed to update metadata: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.Metadata]{Data: *relMetadata.MarshalOscal()})
}

// GetImportAp godoc
//
//	@Summary		Get Assessment Results import-ap
//	@Description	Retrieves import-ap for a given Assessment Results.
//	@Tags			Assessment Results
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Results ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.ImportAp]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/import-ap [get]
func (h *AssessmentResultsHandler) GetImportAp(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	var ar relational.AssessmentResult
	if err := h.db.First(&ar, "id = ?", id).Error; err != nil {
		h.sugar.Errorw("failed to get assessment results", "error", err)
		return ctx.JSON(http.StatusNotFound, api.NewError(err))
	}
	importAp := ar.ImportAp.Data()
	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.ImportAp]{Data: oscalTypes_1_1_3.ImportAp(importAp)})
}

// UpdateImportAp godoc
//
//	@Summary		Update Assessment Results import-ap
//	@Description	Updates import-ap for a given Assessment Results.
//	@Tags			Assessment Results
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string						true	"Assessment Results ID"
//	@Param			importAp	body		oscalTypes_1_1_3.ImportAp	true	"Import AP data"
//	@Success		200			{object}	handler.GenericDataResponse[oscalTypes_1_1_3.ImportAp]
//	@Failure		400			{object}	api.Error
//	@Failure		404			{object}	api.Error
//	@Failure		500			{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/import-ap [put]
func (h *AssessmentResultsHandler) UpdateImportAp(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Verify assessment results exists
	if err := h.verifyAssessmentResultsExists(ctx, id); err != nil {
		return err
	}

	var oscalImportAp oscalTypes_1_1_3.ImportAp
	if err := ctx.Bind(&oscalImportAp); err != nil {
		h.sugar.Warnw("Invalid update import-ap request", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Validate required fields
	if oscalImportAp.Href == "" {
		h.sugar.Warnw("Invalid import-ap input", "error", "href is required")
		return ctx.JSON(http.StatusBadRequest, api.NewError(fmt.Errorf("href is required")))
	}

	var ar relational.AssessmentResult
	if err := h.db.First(&ar, "id = ?", id).Error; err != nil {
		h.sugar.Errorw("failed to get assessment results", "error", err)
		return ctx.JSON(http.StatusNotFound, api.NewError(err))
	}

	// Convert to relational model
	relImportAp := relational.ImportAp{
		Href: oscalImportAp.Href,
		Remarks: oscalImportAp.Remarks,
	}

	// Update the import-ap
	if err := h.db.Model(&ar).Update("import_ap", datatypes.NewJSONType(relImportAp)).Error; err != nil {
		h.sugar.Errorf("Failed to update import-ap: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update metadata last-modified
	now := time.Now()
	if err := h.db.Model(&relational.Metadata{}).Where("id = ?", ar.Metadata.ID).Update("last_modified", now).Error; err != nil {
		h.sugar.Errorf("Failed to update metadata: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.ImportAp]{Data: oscalImportAp})
}

// GetLocalDefinitions godoc
//
//	@Summary		Get Assessment Results local-definitions
//	@Description	Retrieves local-definitions for a given Assessment Results.
//	@Tags			Assessment Results
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Results ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.LocalDefinitions]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/local-definitions [get]
func (h *AssessmentResultsHandler) GetLocalDefinitions(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	var ar relational.AssessmentResult
	if err := h.db.First(&ar, "id = ?", id).Error; err != nil {
		h.sugar.Errorw("failed to get assessment results", "error", err)
		return ctx.JSON(http.StatusNotFound, api.NewError(err))
	}
	// LocalDefinitions in AssessmentResult is already a pointer to LocalDefinitions struct
	if ar.LocalDefinitions == nil {
		return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.LocalDefinitions]{Data: oscalTypes_1_1_3.LocalDefinitions{}})
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.LocalDefinitions]{Data: *ar.LocalDefinitions.MarshalOscal()})
}

// UpdateLocalDefinitions godoc
//
//	@Summary		Update Assessment Results local-definitions
//	@Description	Updates local-definitions for a given Assessment Results.
//	@Tags			Assessment Results
//	@Accept			json
//	@Produce		json
//	@Param			id					path		string								true	"Assessment Results ID"
//	@Param			localDefinitions	body		oscalTypes_1_1_3.LocalDefinitions	true	"Local definitions data"
//	@Success		200					{object}	handler.GenericDataResponse[oscalTypes_1_1_3.LocalDefinitions]
//	@Failure		400					{object}	api.Error
//	@Failure		404					{object}	api.Error
//	@Failure		500					{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/local-definitions [put]
func (h *AssessmentResultsHandler) UpdateLocalDefinitions(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Verify assessment results exists
	if err := h.verifyAssessmentResultsExists(ctx, id); err != nil {
		return err
	}

	var oscalLocalDefs oscalTypes_1_1_3.LocalDefinitions
	if err := ctx.Bind(&oscalLocalDefs); err != nil {
		h.sugar.Warnw("Invalid update local-definitions request", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var ar relational.AssessmentResult
	if err := h.db.First(&ar, "id = ?", id).Error; err != nil {
		h.sugar.Errorw("failed to get assessment results", "error", err)
		return ctx.JSON(http.StatusNotFound, api.NewError(err))
	}

	// Convert to relational model
	relLocalDefs := relational.LocalDefinitions{}
	relLocalDefs.UnmarshalOscal(oscalLocalDefs)

	// Update the local-definitions
	if err := h.db.Model(&ar).Update("local_definitions", datatypes.NewJSONType(relLocalDefs)).Error; err != nil {
		h.sugar.Errorf("Failed to update local-definitions: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update metadata last-modified
	now := time.Now()
	if err := h.db.Model(&relational.Metadata{}).Where("id = ?", ar.Metadata.ID).Update("last_modified", now).Error; err != nil {
		h.sugar.Errorf("Failed to update metadata: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.LocalDefinitions]{Data: oscalLocalDefs})
}

// GetResults godoc
//
//	@Summary		Get results for an Assessment Results
//	@Description	Retrieves all results for a given Assessment Results.
//	@Tags			Assessment Results
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Results ID"
//	@Success		200	{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.Result]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/results [get]
func (h *AssessmentResultsHandler) GetResults(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	var ar relational.AssessmentResult
	if err := h.db.Preload("Results").First(&ar, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load assessment results", "id", idParam, "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	oscalResults := make([]oscalTypes_1_1_3.Result, len(ar.Results))
	for i, result := range ar.Results {
		oscalResults[i] = *result.MarshalOscal()
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.Result]{Data: oscalResults})
}

// CreateResult godoc
//
//	@Summary		Create a result for an Assessment Results
//	@Description	Creates a new result for a given Assessment Results.
//	@Tags			Assessment Results
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Assessment Results ID"
//	@Param			result	body		oscalTypes_1_1_3.Result	true	"Result data"
//	@Success		201		{object}	handler.GenericDataResponse[oscalTypes_1_1_3.Result]
//	@Failure		400		{object}	api.Error
//	@Failure		404		{object}	api.Error
//	@Failure		500		{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/results [post]
func (h *AssessmentResultsHandler) CreateResult(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Verify assessment results exists
	if err := h.verifyAssessmentResultsExists(ctx, id); err != nil {
		return err
	}

	var oscalResult oscalTypes_1_1_3.Result
	if err := ctx.Bind(&oscalResult); err != nil {
		h.sugar.Warnw("Invalid create result request", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Validate input
	if err := h.validateResultInput(&oscalResult); err != nil {
		h.sugar.Warnw("Invalid result input", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Convert to relational model
	relResult := relational.Result{}
	relResult.UnmarshalOscal(oscalResult)
	relResult.AssessmentResultID = id

	// Create the result
	if err := h.db.Create(&relResult).Error; err != nil {
		h.sugar.Errorf("Failed to create result: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update metadata last-modified
	now := time.Now()
	var ar relational.AssessmentResult
	if err := h.db.First(&ar, "id = ?", id).Error; err == nil {
		h.db.Model(&relational.Metadata{}).Where("id = ?", ar.Metadata.ID).Update("last_modified", now)
	}

	return ctx.JSON(http.StatusCreated, handler.GenericDataResponse[oscalTypes_1_1_3.Result]{Data: *relResult.MarshalOscal()})
}

// GetResult godoc
//
//	@Summary		Get a specific result
//	@Description	Retrieves a specific result from an Assessment Results.
//	@Tags			Assessment Results
//	@Produce		json
//	@Param			id			path		string	true	"Assessment Results ID"
//	@Param			resultId	path		string	true	"Result ID"
//	@Success		200			{object}	handler.GenericDataResponse[oscalTypes_1_1_3.Result]
//	@Failure		400			{object}	api.Error
//	@Failure		404			{object}	api.Error
//	@Failure		500			{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/results/{resultId} [get]
func (h *AssessmentResultsHandler) GetResult(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid assessment results id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	resultIdParam := ctx.Param("resultId")
	resultId, err := uuid.Parse(resultIdParam)
	if err != nil {
		h.sugar.Errorw("invalid result id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var result relational.Result
	if err := h.db.
		Preload("Observations").
		Preload("Risks").
		Preload("Findings").
		First(&result, "id = ? AND assessment_result_id = ?", resultId, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load result", "id", resultIdParam, "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.Result]{Data: *result.MarshalOscal()})
}

// UpdateResult godoc
//
//	@Summary		Update a result
//	@Description	Updates a specific result in an Assessment Results.
//	@Tags			Assessment Results
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string					true	"Assessment Results ID"
//	@Param			resultId	path		string					true	"Result ID"
//	@Param			result		body		oscalTypes_1_1_3.Result	true	"Result data"
//	@Success		200			{object}	handler.GenericDataResponse[oscalTypes_1_1_3.Result]
//	@Failure		400			{object}	api.Error
//	@Failure		404			{object}	api.Error
//	@Failure		500			{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/results/{resultId} [put]
func (h *AssessmentResultsHandler) UpdateResult(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid assessment results id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	resultIdParam := ctx.Param("resultId")
	resultId, err := uuid.Parse(resultIdParam)
	if err != nil {
		h.sugar.Errorw("invalid result id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var oscalResult oscalTypes_1_1_3.Result
	if err := ctx.Bind(&oscalResult); err != nil {
		h.sugar.Warnw("Invalid update result request", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Validate input
	if err := h.validateResultInput(&oscalResult); err != nil {
		h.sugar.Warnw("Invalid result input", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Check if result exists
	var existingResult relational.Result
	if err := h.db.First(&existingResult, "id = ? AND assessment_result_id = ?", resultId, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Errorf("Failed to find result: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Convert to relational model
	relResult := relational.Result{}
	relResult.UnmarshalOscal(oscalResult)
	relResult.ID = &resultId
	relResult.AssessmentResultID = id

	// Update the result - Result type in relational model doesn't have these fields
	// directly, need to update the whole object
	if err := h.db.Model(&existingResult).Updates(relResult).Error; err != nil {
		h.sugar.Errorf("Failed to update result: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update metadata last-modified
	now := time.Now()
	var ar relational.AssessmentResult
	if err := h.db.First(&ar, "id = ?", id).Error; err == nil {
		h.db.Model(&relational.Metadata{}).Where("id = ?", ar.Metadata.ID).Update("last_modified", now)
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.Result]{Data: *relResult.MarshalOscal()})
}

// DeleteResult godoc
//
//	@Summary		Delete a result
//	@Description	Deletes a specific result from an Assessment Results.
//	@Tags			Assessment Results
//	@Produce		json
//	@Param			id			path		string	true	"Assessment Results ID"
//	@Param			resultId	path		string	true	"Result ID"
//	@Success		200			{object}	api.Response
//	@Failure		400			{object}	api.Error
//	@Failure		404			{object}	api.Error
//	@Failure		500			{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/results/{resultId} [delete]
func (h *AssessmentResultsHandler) DeleteResult(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid assessment results id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	resultIdParam := ctx.Param("resultId")
	resultId, err := uuid.Parse(resultIdParam)
	if err != nil {
		h.sugar.Errorw("invalid result id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Check if result exists
	var result relational.Result
	if err := h.db.First(&result, "id = ? AND assessment_result_id = ?", resultId, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Errorf("Failed to find result: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Delete the result
	if err := h.db.Delete(&result).Error; err != nil {
		h.sugar.Errorf("Failed to delete result: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	// Update metadata last-modified
	now := time.Now()
	var ar relational.AssessmentResult
	if err := h.db.First(&ar, "id = ?", id).Error; err == nil {
		h.db.Model(&relational.Metadata{}).Where("id = ?", ar.Metadata.ID).Update("last_modified", now)
	}

	return ctx.JSON(http.StatusOK, struct{Message string `json:"message"`}{Message: "Result deleted successfully"})
}

// Additional endpoint implementations would follow the same pattern for:
// - GetResultObservations, CreateResultObservation, UpdateResultObservation, DeleteResultObservation
// - GetResultRisks, CreateResultRisk, UpdateResultRisk, DeleteResultRisk
// - GetResultFindings, CreateResultFinding, UpdateResultFinding, DeleteResultFinding
// - GetResultAttestations, CreateResultAttestation, UpdateResultAttestation, DeleteResultAttestation
// - GetBackMatter, CreateBackMatter, UpdateBackMatter, DeleteBackMatter
// - GetBackMatterResources, CreateBackMatterResource, UpdateBackMatterResource, DeleteBackMatterResource

// GetResultObservations godoc
//
//	@Summary		Get observations for a result
//	@Description	Retrieves all observations for a given result.
//	@Tags			Assessment Results
//	@Produce		json
//	@Param			id			path		string	true	"Assessment Results ID"
//	@Param			resultId	path		string	true	"Result ID"
//	@Success		200			{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.Observation]
//	@Failure		400			{object}	api.Error
//	@Failure		404			{object}	api.Error
//	@Failure		500			{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-results/{id}/results/{resultId}/observations [get]
func (h *AssessmentResultsHandler) GetResultObservations(ctx echo.Context) error {
	idParam := ctx.Param("id")
	_, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("invalid assessment results id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	resultIdParam := ctx.Param("resultId")
	resultId, err := uuid.Parse(resultIdParam)
	if err != nil {
		h.sugar.Errorw("invalid result id", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var result relational.Result
	if err := h.db.Preload("Observations").First(&result, "id = ?", resultId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load result", "id", resultIdParam, "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	oscalObs := make([]oscalTypes_1_1_3.Observation, len(result.Observations))
	for i, obs := range result.Observations {
		oscalObs[i] = *obs.MarshalOscal()
	}
	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.Observation]{Data: oscalObs})
}

// Placeholder implementations for remaining endpoints
func (h *AssessmentResultsHandler) CreateResultObservation(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) UpdateResultObservation(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) DeleteResultObservation(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) GetResultRisks(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) CreateResultRisk(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) UpdateResultRisk(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) DeleteResultRisk(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) GetResultFindings(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) CreateResultFinding(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) UpdateResultFinding(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) DeleteResultFinding(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) GetResultAttestations(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) CreateResultAttestation(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) UpdateResultAttestation(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) DeleteResultAttestation(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) GetBackMatter(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) CreateBackMatter(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) UpdateBackMatter(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) DeleteBackMatter(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) GetBackMatterResources(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) CreateBackMatterResource(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) UpdateBackMatterResource(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}

func (h *AssessmentResultsHandler) DeleteBackMatterResource(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, api.NewError(fmt.Errorf("not implemented")))
}