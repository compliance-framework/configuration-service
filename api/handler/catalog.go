package handler

import (
	"fmt"
	"github.com/compliance-framework/configuration-service/api"
	"github.com/compliance-framework/configuration-service/domain"
	"github.com/compliance-framework/configuration-service/store"
	"github.com/labstack/echo/v4"
	"net/http"
)

type CatalogHandler struct {
	store store.CatalogStore
}

func NewCatalogHandler(s store.CatalogStore) *CatalogHandler {
	return &CatalogHandler{store: s}
}

func (h *CatalogHandler) Register(api *echo.Group) {
	api.POST("/catalog", h.CreateCatalog)
	api.GET("/catalog/:id", h.GetCatalog)
	api.PATCH("/catalog/:id", h.UpdateCatalog)
	api.DELETE("/catalog/:id", h.DeleteCatalog)
	api.POST("/catalog/:id/controls", h.CreateControl)
}

// CreateCatalog godoc
// @Summary 		Create a catalog
// @Description 	Create a catalog with the given title
// @Tags 		curl -X 'PATCH' 'http://localhost:8080/api/catalog/654b70acbcd83fba9c216045' -H 'accept: application/json' -H 'Content-Type: application/json' -d '{ "catalog": { "title": "new title" } }'	Catalog
// @Accept  		json
// @Produce  		json
// @Param   		catalog body createCatalogRequest true "Catalog to add"
// @Success 		201 {object} catalogIdResponse
// @Failure 		401 {object} api.Error
// @Failure 		422 {object} api.Error
// @Failure 		500 {object} api.Error
// @Router 			/api/catalog [post]
func (h *CatalogHandler) CreateCatalog(ctx echo.Context) error {
	var c domain.Catalog
	req := newCreateCatalogRequest()
	if err := req.bind(ctx, &c); err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, api.NewError(err))
	}

	id, err := h.store.CreateCatalog(&c)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return ctx.JSON(http.StatusCreated, catalogIdResponse{
		Id: id.(string),
	})
}

func (h *CatalogHandler) GetCatalog(ctx echo.Context) error {
	id := ctx.Param("id")
	c, err := h.store.GetCatalog(id)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return ctx.JSON(http.StatusOK, c)
}

func (h *CatalogHandler) UpdateCatalog(ctx echo.Context) error {
	id := ctx.Param("id")
	var c domain.Catalog
	req := newCreateCatalogRequest()
	if err := req.bind(ctx, &c); err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, api.NewError(err))
	}

	err := h.store.UpdateCatalog(id, &c)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	updatedCatalog, err := h.store.GetCatalog(id)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, updatedCatalog)
}

func (h *CatalogHandler) DeleteCatalog(ctx echo.Context) error {
	id := ctx.Param("id")
	var c domain.Catalog

	// Check if the catalog exists before attempting to delete
	existingCatalog, err := h.store.GetCatalog(id)
	if err != nil || existingCatalog == nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"message": "Catalog not found"})
	}

	req := newCreateCatalogRequest()

	if err := req.bind(ctx, &c); err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, api.NewError(err))
	}

	err = h.store.DeleteCatalog(id)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	return ctx.JSON(http.StatusOK, map[string]string{"message": "Catalog has been deleted"})
}

func (h *CatalogHandler) CreateControl(ctx echo.Context) error {
	var c domain.Control
	req := newCreateControlRequest()
	if err := req.bind(ctx, &c); err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, api.NewError(err))
	}

	catalogId := ctx.Param("id")
	id, err := h.store.CreateControl(catalogId, &c)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	} else {
		fmt.Println("CreateControl has been called")
	}

	return ctx.JSON(http.StatusCreated, catalogIdResponse{
		Id: id.(string),
	})
}
