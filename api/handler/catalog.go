package handler

import (
	"fmt"
	"net/http"

	"github.com/compliance-framework/configuration-service/api"
	"github.com/compliance-framework/configuration-service/domain"
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
	api.POST("/catalog", h.CreateCatalog)
	api.GET("/catalog/:id", h.ListCatalog)
}

// CreateCatalog godoc
// @Summary 		Create a catalog
// @Description 	Create a catalog with the given title
// @Tags 			Catalog
// @Accept  		json
// @Produce  		json
// @Param   		catalog body createCatalogRequest true "Catalog to add"
// @Success 		201 {object} catalogIdResponse
// @Failure 		401 {object} api.Error
// @Failure 		422 {object} api.Error
// @Failure 		500 {object} api.Error
// @Router 			/api/catalog [post]
func (h *CatalogHandler) CreateCatalog(ctx echo.Context) error {
	fmt.Println("CreateCatalog called") // Add this line
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

func (h *CatalogHandler) ListCatalog(ctx echo.Context) error {
	id := ctx.Param("id")
	c, err := h.store.GetCatalog(id)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return ctx.JSON(http.StatusOK, c)
}
