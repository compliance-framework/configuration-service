package handler

import (
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"net/http"

	"github.com/compliance-framework/configuration-service/api"
	"github.com/compliance-framework/configuration-service/store"
	"github.com/labstack/echo/v4"
)

type CatalogHandler struct {
	store store.CatalogStore
}

func NewCatalogHandler(s store.CatalogStore) *CatalogHandler {
	return &CatalogHandler{store: s}
}

func (h *CatalogHandler) Register(api *echo.Group) {
	api.POST("", h.CreateCatalog)
	api.GET("/:id", h.GetCatalog)
}

// CreateCatalog godoc
//
//	@Summary		Create a catalog
//	@Description	Create a catalog with the given title
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			catalog	body		createCatalogRequest	true	"Catalog to add"
//	@Success		201		{object}	catalogIdResponse
//	@Failure		401		{object}	api.Error
//	@Failure		422		{object}	api.Error
//	@Failure		500		{object}	api.Error
//	@Router			/catalog [post]
func (h *CatalogHandler) CreateCatalog(ctx echo.Context) error {
	var c oscaltypes113.Catalog
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

// GetCatalog godoc
//
//	@Summary		Get a catalog
//	@Description	Get a specific catalog by its ID
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Catalog ID"
//	@Success		200	{object}	domain.Catalog
//	@Failure		401	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/catalog/{id} [get]
func (h *CatalogHandler) GetCatalog(ctx echo.Context) error {
	id := ctx.Param("id")
	c, err := h.store.GetCatalog(id)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return ctx.JSON(http.StatusOK, c)
}
