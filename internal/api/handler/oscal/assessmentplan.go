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
	"gorm.io/gorm"

	"github.com/compliance-framework/api/internal/api"
	"github.com/compliance-framework/api/internal/api/handler"
	"github.com/compliance-framework/api/internal/service/relational"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
)

type AssessmentPlanHandler struct {
	sugar *zap.SugaredLogger
	db    *gorm.DB
}

func NewAssessmentPlanHandler(sugar *zap.SugaredLogger, db *gorm.DB) *AssessmentPlanHandler {
	return &AssessmentPlanHandler{
		sugar: sugar,
		db:    db,
	}
}

// Register registers Assessment Plan endpoints to the API group.
func (h *AssessmentPlanHandler) Register(api *echo.Group) {
	// Core CRUD operations
	api.GET("", h.List)          // GET /oscal/assessment-plans
	api.POST("", h.Create)       // POST /oscal/assessment-plans
	api.GET("/:id", h.Get)       // GET /oscal/assessment-plans/:id
	api.PUT("/:id", h.Update)    // PUT /oscal/assessment-plans/:id
	api.GET("/:id/full", h.Full) // GET /oscal/assessment-plans/:id/full
	api.DELETE("/:id", h.Delete) // DELETE /oscal/assessment-plans/:id

	api.GET("/:id/metadata", h.GetMetadata)
	api.GET("/:id/import-ssp", h.GetImportSsp)
	api.GET("/:id/local-definitions", h.GetLocalDefinitions)
	api.GET("/:id/terms-and-conditions", h.GetTermsAndConditions)
	api.GET("/:id/back-matter", h.GetBackMatter)

	// Tasks sub-resource management
	api.GET("/:id/tasks", h.GetTasks)
	api.POST("/:id/tasks", h.CreateTask)

	api.PUT("/:id/tasks/:taskId", h.UpdateTask)
	api.DELETE("/:id/tasks/:taskId", h.DeleteTask)

	api.GET("/:id/tasks/:taskId/associated-activities", h.GetTaskActivities)
	api.POST("/:id/tasks/:taskId/associated-activities/:activityId", h.AssociateTaskActivity)
	api.DELETE("/:id/tasks/:taskId/associated-activities/:activityId", h.DisassociateTaskActivity)

	// Assessment Subjects sub-resource management
	api.GET("/:id/assessment-subjects", h.GetAssessmentSubjects)
	api.POST("/:id/assessment-subjects", h.CreateAssessmentSubject)
	api.PUT("/:id/assessment-subjects/:subjectId", h.UpdateAssessmentSubject)
	api.DELETE("/:id/assessment-subjects/:subjectId", h.DeleteAssessmentSubject)

	// Assessment Assets sub-resource management
	api.GET("/:id/assessment-assets", h.GetAssessmentAssets)
	api.POST("/:id/assessment-assets", h.CreateAssessmentAsset)
	api.PUT("/:id/assessment-assets/:assetId", h.UpdateAssessmentAsset)
	api.DELETE("/:id/assessment-assets/:assetId", h.DeleteAssessmentAsset)
}

// verifyAssessmentPlanExists checks if an assessment plan exists in the database
func (h *AssessmentPlanHandler) verifyAssessmentPlanExists(ctx echo.Context, planID uuid.UUID) error {
	var count int64
	if err := h.db.Model(&relational.AssessmentPlan{}).Where("id = ?", planID).Count(&count).Error; err != nil {
		h.sugar.Errorw("failed to count assessment plans", "err", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	if count == 0 {
		return api.NotFoundError(fmt.Errorf("assessment plan not found"))
	}
	return nil
}

// List godoc
//
//	@Summary		List Assessment Plans
//	@Description	Retrieves all Assessment Plans.
//	@Tags			Assessment Plans
//	@Produce		json
//	@Success		200	{object}	handler.GenericDataListResponse[oscalTypes_1_1_3.AssessmentPlan]
//	@Failure		400	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-plans [get]
func (h *AssessmentPlanHandler) List(ctx echo.Context) error {
	var plans []relational.AssessmentPlan
	if err := h.db.
		Preload("Metadata").
		Preload("Metadata.Revisions").
		Find(&plans).Error; err != nil {
		h.sugar.Warnw("Failed to load assessment plans", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	oscalPlans := []oscalTypes_1_1_3.AssessmentPlan{}
	for _, plan := range plans {
		oscalPlans = append(oscalPlans, *plan.MarshalOscal())
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.AssessmentPlan]{Data: oscalPlans})
}

// Get godoc
//
//	@Summary		Get an Assessment Plan
//	@Description	Retrieves a single Assessment Plan by its unique ID.
//	@Tags			Assessment Plans
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Plan ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentPlan]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-plans/{id} [get]
func (h *AssessmentPlanHandler) Get(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return api.InvalidUUIDError(err)
	}

	var plan relational.AssessmentPlan
	err = h.db.
		Preload("Metadata").
		Preload("Metadata.Revisions").
		First(&plan, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.NotFoundError(fmt.Errorf("assessment plan not found: %w", err))
		}
		h.sugar.Errorw("Failed to load assessment plan", "id", id.String(), "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[*oscalTypes_1_1_3.AssessmentPlan]{Data: plan.MarshalOscal()})
}

// Create godoc
//
//	@Summary		Create an Assessment Plan
//	@Description	Creates a new OSCAL Assessment Plan with comprehensive validation.
//	@Tags			Assessment Plans
//	@Accept			json
//	@Produce		json
//	@Param			plan	body		oscalTypes_1_1_3.AssessmentPlan									true	"Assessment Plan object with required fields: UUID, metadata (title, version), import-ssp"
//	@Success		201		{object}	handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentPlan]	"Successfully created assessment plan"
//	@Failure		400		{object}	api.Error														"Bad request - validation errors or malformed input"
//	@Failure		401		{object}	api.Error														"Unauthorized - invalid or missing JWT token"
//	@Failure		500		{object}	api.Error														"Internal server error"
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-plans [post]
func (h *AssessmentPlanHandler) Create(ctx echo.Context) error {
	var request AssessmentPlanCreateRequest
	err := ctx.Bind(&request.Data)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	errs := request.Validate()
	if len(errs) > 0 {
		return NewValidationErrorResponse(errs)
	}

	// Set metadata timestamps
	now := time.Now()
	request.Data.Metadata.LastModified = now
	request.Data.Metadata.OscalVersion = versioning.GetLatestSupportedVersion()

	// Convert to a relational model
	relationalPlan := &relational.AssessmentPlan{}
	relationalPlan.UnmarshalOscal(*request.Data)

	// Save to the database
	if err := h.db.Create(relationalPlan).Error; err != nil {
		h.sugar.Errorw("failed to create assessment plan", "err", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusCreated, handler.GenericDataResponse[*oscalTypes_1_1_3.AssessmentPlan]{Data: relationalPlan.MarshalOscal()})
}

// Update godoc
//
//	@Summary		Update an Assessment Plan
//	@Description	Updates an existing Assessment Plan.
//	@Tags			Assessment Plans
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Assessment Plan ID"
//	@Param			plan	body		oscalTypes_1_1_3.AssessmentPlan	true	"Assessment Plan object"
//	@Success		200		{object}	handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentPlan]
//	@Failure		400		{object}	api.Error
//	@Failure		404		{object}	api.Error
//	@Failure		500		{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-plans/{id} [put]
func (h *AssessmentPlanHandler) Update(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return api.InvalidUUIDError(err)
	}

	if err = h.verifyAssessmentPlanExists(ctx, id); err != nil {
		return err
	}

	var request AssessmentPlanUpdateRequest
	if err := ctx.Bind(&request.Data); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	errs := request.Validate()
	if len(errs) > 0 {
		return NewValidationErrorResponse(errs)
	}

	// Update metadata
	now := time.Now()
	request.Data.Metadata.LastModified = now

	// Convert to a relational model
	relationalPlan := &relational.AssessmentPlan{}
	relationalPlan.UnmarshalOscal(*request.Data)

	// Update in database
	relationalPlan.ID = &id
	if err := h.db.Where("id = ?", id).Updates(relationalPlan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.NotFoundError(fmt.Errorf("assessment plan not found: %w", err))
		}
		h.sugar.Errorw("failed to update assessment plan", "err", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[*oscalTypes_1_1_3.AssessmentPlan]{Data: relationalPlan.MarshalOscal()})
}

// Delete godoc
//
//	@Summary		Delete an Assessment Plan
//	@Description	Deletes an Assessment Plan by its unique ID.
//	@Tags			Assessment Plans
//	@Param			id	path	string	true	"Assessment Plan ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-plans/{id} [delete]
func (h *AssessmentPlanHandler) Delete(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return api.InvalidUUIDError(err)
	}

	// Verify plan exists
	if err := h.verifyAssessmentPlanExists(ctx, id); err != nil {
		return err
	}

	// Delete from database
	if err := h.db.Delete(&relational.AssessmentPlan{}, "id = ?", id).Error; err != nil {
		h.sugar.Errorw("failed to delete assessment plan", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.NoContent(http.StatusNoContent)
}

// GetMetadata godoc
//
//	@Summary		Get Assessment Plan Metadata
//	@Description	Retrieves metadata for an Assessment Plan.
//	@Tags			Assessment Plans
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Plan ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.Metadata]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-plans/{id}/metadata [get]
func (h *AssessmentPlanHandler) GetMetadata(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return api.InvalidUUIDError(err)
	}

	var plan relational.AssessmentPlan
	if err := h.db.Preload("Metadata").Where("id = ?", id).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.NotFoundError(fmt.Errorf("assessment plan not found: %w", err))
		}
		h.sugar.Errorw("failed to retrieve assessment plan metadata", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[*oscalTypes_1_1_3.Metadata]{Data: plan.Metadata.MarshalOscal()})
}

// GetImportSsp godoc
//
//	@Summary		Get Assessment Plan Import SSP
//	@Description	Retrieves import SSP information for an Assessment Plan.
//	@Tags			Assessment Plans
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Plan ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.ImportSsp]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-plans/{id}/import-ssp [get]
func (h *AssessmentPlanHandler) GetImportSsp(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return api.InvalidUUIDError(err)
	}

	var plan relational.AssessmentPlan
	if err := h.db.Where("id = ?", id).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.NotFoundError(fmt.Errorf("assessment plan not found: %w", err))
		}
		h.sugar.Errorw("failed to retrieve assessment plan", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	importSsp := oscalTypes_1_1_3.ImportSsp(plan.ImportSSP.Data())
	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[*oscalTypes_1_1_3.ImportSsp]{Data: &importSsp})
}

// GetLocalDefinitions godoc
//
//	@Summary		Get Assessment Plan Local Definitions
//	@Description	Retrieves local definitions for an Assessment Plan.
//	@Tags			Assessment Plans
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Plan ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.LocalDefinitions]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-plans/{id}/local-definitions [get]
func (h *AssessmentPlanHandler) GetLocalDefinitions(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return api.InvalidUUIDError(err)
	}

	var plan relational.AssessmentPlan
	if err := h.db.Preload("LocalDefinitions").Where("id = ?", id).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.NotFoundError(fmt.Errorf("assessment plan not found: %w", err))
		}
		h.sugar.Errorw("failed to retrieve assessment plan", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	if plan.LocalDefinitions.ID == nil {
		return api.NotFoundError(fmt.Errorf("local definitions not found for assessment plan: %s", id.String()))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[*oscalTypes_1_1_3.LocalDefinitions]{Data: plan.LocalDefinitions.MarshalOscal()})
}

// GetTermsAndConditions godoc
//
//	@Summary		Get Assessment Plan Terms and Conditions
//	@Description	Retrieves terms and conditions for an Assessment Plan.
//	@Tags			Assessment Plans
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Plan ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentPlanTermsAndConditions]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-plans/{id}/terms-and-conditions [get]
func (h *AssessmentPlanHandler) GetTermsAndConditions(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return api.InvalidUUIDError(err)
	}

	var plan relational.AssessmentPlan
	if err := h.db.Preload("TermsAndConditions").Where("id = ?", id).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.NotFoundError(fmt.Errorf("assessment plan not found: %w", err))
		}
		h.sugar.Errorw("failed to retrieve assessment plan", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	if plan.TermsAndConditions.ID == nil {
		return api.NotFoundError(fmt.Errorf("terms and conditions not found for assessment plan: %s", id.String()))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[*oscalTypes_1_1_3.AssessmentPlanTermsAndConditions]{Data: plan.TermsAndConditions.MarshalOscal()})
}

// GetBackMatter godoc
//
//	@Summary		Get Assessment Plan Back Matter
//	@Description	Retrieves back matter for an Assessment Plan.
//	@Tags			Assessment Plans
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Plan ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.BackMatter]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-plans/{id}/back-matter [get]
func (h *AssessmentPlanHandler) GetBackMatter(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return api.InvalidUUIDError(err)
	}

	var plan relational.AssessmentPlan
	if err := h.db.Preload("BackMatter").Where("id = ?", id).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.NotFoundError(fmt.Errorf("assessment plan not found: %w", err))
		}
		h.sugar.Errorw("failed to retrieve assessment plan", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	if plan.BackMatter == nil {
		return api.NotFoundError(fmt.Errorf("back matter not found for assessment plan: %s", id.String()))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[*oscalTypes_1_1_3.BackMatter]{Data: plan.BackMatter.MarshalOscal()})
}

// Full godoc
//
//	@Summary		Get a full Assessment Plan
//	@Description	Retrieves a single Assessment Plan by its unique ID with all related data preloaded.
//	@Tags			Assessment Plans
//	@Produce		json
//	@Param			id	path		string	true	"Assessment Plan ID"
//	@Success		200	{object}	handler.GenericDataResponse[oscalTypes_1_1_3.AssessmentPlan]
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/oscal/assessment-plans/{id}/full [get]
func (h *AssessmentPlanHandler) Full(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return api.InvalidUUIDError(err)
	}

	var plan relational.AssessmentPlan
	if err := h.db.
		Preload("Metadata").
		Preload("Metadata.Revisions").
		Preload("Tasks").
		Preload("Tasks.Dependencies").
		Preload("Tasks.Tasks").
		Preload("Tasks.AssociatedActivities").
		Preload("Tasks.AssociatedActivities.Steps").
		Preload("Tasks.Subjects").
		Preload("Tasks.ResponsibleRole").
		Preload("AssessmentAssets").
		Preload("AssessmentSubjects").
		Preload("LocalDefinitions").
		Preload("TermsAndConditions").
		Preload("BackMatter").
		First(&plan, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.NotFoundError(fmt.Errorf("assessment plan not found: %w", err))
		}
		h.sugar.Errorw("failed to load assessment plan", "id", id.String(), "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[*oscalTypes_1_1_3.AssessmentPlan]{Data: plan.MarshalOscal()})
}
