package oscal

import (
	"errors"
	"github.com/google/uuid"
	"net/http"

	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/api/handler"
	"github.com/compliance-framework/configuration-service/internal/service/relational"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SystemSecurityPlanHandler struct {
	sugar *zap.SugaredLogger
	db    *gorm.DB
}

func NewSystemSecurityPlanHandler(sugar *zap.SugaredLogger, db *gorm.DB) *SystemSecurityPlanHandler {
	return &SystemSecurityPlanHandler{
		sugar: sugar,
		db:    db,
	}
}

func (h *SystemSecurityPlanHandler) Register(api *echo.Group) {
	api.GET("", h.List)
	api.GET("/:id", h.Get)
	api.GET("/:id/system-characteristics", h.GetCharacteristics)
}

// List godoc
//
// @Summary List System Security Plans
// @Description Retrieves all System Security Plans.
// @Tags Oscal
// @Produce json
// @Success 200 {object} handler.GenericDataListResponse[oscalTypes_1_1_3.SystemSecurityPlan]
// @Failure 400 {object} api.Error
// @Failure 500 {object} api.Error
// @Router /oscal/system-security-plans [get]
func (h *SystemSecurityPlanHandler) List(ctx echo.Context) error {
	type response struct {
		UUID     uuid.UUID                 `json:"uuid"`
		Metadata oscalTypes_1_1_3.Metadata `json:"metadata"`
	}

	var ssps []relational.SystemSecurityPlan

	if err := h.db.
		Preload("Metadata").
		Preload("Metadata.Revisions").
		Find(&ssps).Error; err != nil {
		h.sugar.Error(err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	oscalSSP := make([]oscalTypes_1_1_3.SystemSecurityPlan, len(ssps))
	for i, ssp := range ssps {
		oscalSSP[i] = *ssp.MarshalOscal()
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.SystemSecurityPlan]{Data: oscalSSP})
}

// Get godoc
//
// @Summary Get a System Security Plan
// @Description Retrieves a single System Security Plan by its unique ID.
// @Tags Oscal
// @Produce json
// @Param id path string true "System Security Plan ID"
// @Success 200 {object} handler.GenericDataResponse[*oscalTypes_1_1_3.SystemSecurityPlan]
// @Failure 400 {object} api.Error
// @Failure 404 {object} api.Error
// @Failure 500 {object} api.Error
// @Router /oscal/system-security-plans/{id} [get]
func (h *SystemSecurityPlanHandler) Get(ctx echo.Context) error {
	type response struct {
		UUID     uuid.UUID                 `json:"uuid"`
		Metadata oscalTypes_1_1_3.Metadata `json:"metadata"`
	}

	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid system security plan id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var ssp relational.SystemSecurityPlan
	if err := h.db.
		Preload("Metadata").
		Preload("Metadata.Revisions").
		First(&ssp, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load system security plan", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[*oscalTypes_1_1_3.SystemSecurityPlan]{Data: ssp.MarshalOscal()})
}

// GetCharacteristics godoc
//
// @Summary Get System Characteristics
// @Description Retrieves the System Characteristics for a given System Security Plan.
// @Tags Oscal
// @Produce json
// @Param id path string true "System Security Plan ID"
// @Success 200 {object} handler.GenericDataResponse[oscalTypes_1_1_3.SystemCharacteristics]
// @Failure 400 {object} api.Error
// @Failure 404 {object} api.Error
// @Failure 500 {object} api.Error
// @Router /oscal/system-security-plans/{id}/system-characteristics [get]
func (h *SystemSecurityPlanHandler) GetCharacteristics(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid system security plan id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var ssp relational.SystemSecurityPlan
	if err := h.db.
		Preload("SystemCharacteristics").
		First(&ssp, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load system security plan", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.SystemCharacteristics]{Data: ssp.MarshalOscal().SystemCharacteristics})
}
