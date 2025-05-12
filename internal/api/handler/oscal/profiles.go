package oscal

import (
	"errors"
	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/api/handler"
	"github.com/compliance-framework/configuration-service/internal/service/relational"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
)

type ProfileHandler struct {
	sugar *zap.SugaredLogger
	db    *gorm.DB
}

func NewProfileHandler(sugar *zap.SugaredLogger, db *gorm.DB) *ProfileHandler {
	return &ProfileHandler{
		sugar: sugar,
		db:    db,
	}
}

func (h *ProfileHandler) Register(api *echo.Group) {
	api.GET("", h.List)
	api.GET("/:id", h.Get)
}

// List godoc
//
//	@Summary		List Profiles
//	@Description	Retrieves all OSCAL profiles
//	@Tags			Oscal, Profiles
//	@Produce		json
//	@Success		200	{object}	handler.GenericDataListResponse[oscal.List.response]
//	@Failure		400	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/oscal/profiles [get]
func (h *ProfileHandler) List(ctx echo.Context) error {
	type response struct {
		UUID     uuid.UUID                 `json:"uuid"`
		Metadata oscalTypes_1_1_3.Metadata `json:"metadata"`
	}

	var profiles []relational.Profile
	if err := h.db.
		Preload("Metadata").
		Preload("Metadata.Revisions").
		Find(&profiles).Error; err != nil {
		h.sugar.Errorw("error listing profiles", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	respProfiles := make([]response, len(profiles))
	for i, profile := range profiles {
		respProfiles[i] = response{
			UUID:     *profile.UUIDModel.ID,
			Metadata: *profile.Metadata.MarshalOscal(),
		}
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[response]{Data: respProfiles})
}

// Get godoc
//
//	@Summary		Get Profile
//	@Description	Get an OSCAL profile with the uuid provided
//	@Tags			Oscal, Profiles
//	@Param			id	path	string	true	"Profile ID"
//	@Produce		json
//	@Success		200	{object}	handler.GenericDataResponse[oscal.Get.response]
//	@Failure		404	{object}	api.Error
//	@Failure		400	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/oscal/profiles/{id} [get]
func (h *ProfileHandler) Get(ctx echo.Context) error {
	type response struct {
		UUID     uuid.UUID                 `json:"uuid"`
		Metadata oscalTypes_1_1_3.Metadata `json:"metadata"`
	}

	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Errorw("error parsing UUID", "id", idParam, "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	var profile relational.Profile
	if err := h.db.
		Preload("Metadata").
		Preload("Metadata.Revisions").
		Where("id = ?", id).
		First(&profile).Error; err != nil {
		h.sugar.Errorw("error getting profile", "id", idParam, "error", err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	var responseProfile response
	responseProfile = response{
		UUID:     *profile.UUIDModel.ID,
		Metadata: *profile.Metadata.MarshalOscal(),
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[response]{Data: responseProfile})
}
