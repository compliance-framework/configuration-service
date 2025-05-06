package oscal

import (
	"errors"
	"github.com/compliance-framework/configuration-service/internal/api"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/compliance-framework/configuration-service/internal/api/handler"
	"github.com/compliance-framework/configuration-service/internal/service/relational"
)

type CatalogHandler struct {
	sugar *zap.SugaredLogger
	db    *gorm.DB
}

func NewCatalogHandler(l *zap.SugaredLogger, db *gorm.DB) *CatalogHandler {
	return &CatalogHandler{
		sugar: l,
		db:    db,
	}
}

func (h *CatalogHandler) Register(api *echo.Group) {
	api.GET("", h.List)
	api.GET("/:id", h.Get)
	api.GET("/:id/back-matter", h.GetBackMatter)
	//api.POST("", h.Create)
	//api.PUT("/:id", h.Update)
	//api.DELETE("/:id", h.Delete)
}

func (h *CatalogHandler) List(ctx echo.Context) error {
	var catalogs []relational.Catalog
	if err := h.db.
		Preload("Metadata").
		Preload("Metadata.Revisions").
		Find(&catalogs).Error; err != nil {
		h.sugar.Warnw("Failed to load catalogs", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	oscalCatalogs := []oscalTypes_1_1_3.Catalog{}
	for _, catalog := range catalogs {
		oscalCatalogs = append(oscalCatalogs, *catalog.MarshalOscal())
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataListResponse[oscalTypes_1_1_3.Catalog]{Data: oscalCatalogs})
}

func (h *CatalogHandler) Get(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid catalog id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var catalog relational.Catalog
	if err := h.db.
		Preload("Metadata").
		Preload("Metadata.Revisions").
		First(&catalog, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load catalog", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[oscalTypes_1_1_3.Catalog]{Data: *catalog.MarshalOscal()})
}

func (h *CatalogHandler) GetBackMatter(ctx echo.Context) error {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.sugar.Warnw("Invalid catalog id", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var catalog relational.Catalog
	if err := h.db.
		Preload("BackMatter").
		Preload("BackMatter.Resources").
		First(&catalog, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, api.NewError(err))
		}
		h.sugar.Warnw("Failed to load catalog", "id", idParam, "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[*oscalTypes_1_1_3.BackMatter]{Data: catalog.BackMatter.MarshalOscal()})
}

//
//func (h *CatalogHandler) Create(ctx echo.Context) error {
//	var catalog relational.Catalog
//	if err := ctx.Bind(&catalog); err != nil {
//		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
//	}
//	if err := h.catalogService.CreateCatalog(&catalog); err != nil {
//		h.sugar.Errorw("Failed to create catalog", "error", err)
//		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
//	}
//	return ctx.JSON(http.StatusCreated, handler.GenericDataResponse[relational.Catalog]{Data: catalog})
//}
//
//func (h *CatalogHandler) Update(ctx echo.Context) error {
//	idParam := ctx.Param("id")
//	id, err := uuid.Parse(idParam)
//	if err != nil {
//		h.sugar.Warnw("Invalid catalog id", "id", idParam, "error", err)
//		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
//	}
//	var catalog relational.Catalog
//	if err := ctx.Bind(&catalog); err != nil {
//		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
//	}
//	// Enforce the URL ID
//	catalog.ID = &id
//	if err := h.catalogService.UpdateCatalog(&catalog); err != nil {
//		h.sugar.Errorw("Failed to update catalog", "id", id, "error", err)
//		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
//	}
//	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[relational.Catalog]{Data: catalog})
//}
//
//func (h *CatalogHandler) Delete(ctx echo.Context) error {
//	idParam := ctx.Param("id")
//	id, err := uuid.Parse(idParam)
//	if err != nil {
//		h.sugar.Warnw("Invalid catalog id", "id", idParam, "error", err)
//		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
//	}
//	if err := h.catalogService.DeleteCatalog(id); err != nil {
//		h.sugar.Errorw("Failed to delete catalog", "id", id, "error", err)
//		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
//	}
//	return ctx.NoContent(http.StatusNoContent)
//}
