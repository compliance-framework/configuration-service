package templates

import (
	"errors"
	"net/http"
	"time"

	"github.com/compliance-framework/api/internal/api"
	"github.com/compliance-framework/api/internal/converters/labelfilter"
	svc "github.com/compliance-framework/api/internal/service"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/service/relational/ssptemplates"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SSPTemplateHandler serves /admin/ssp-templates.
type SSPTemplateHandler struct {
	service    *ssptemplates.Service
	sugar      *zap.SugaredLogger
	pagination *svc.PaginationConfig
}

func NewSSPTemplateHandler(sugar *zap.SugaredLogger, db *gorm.DB) *SSPTemplateHandler {
	return &SSPTemplateHandler{
		service:    ssptemplates.NewService(db),
		sugar:      sugar,
		pagination: svc.NewPaginationConfig(),
	}
}

// Register mounts the routes on a group guarded by JWT and admin:manage.
func (h *SSPTemplateHandler) Register(group *echo.Group) {
	group.GET("", h.List)
	group.POST("", h.Create)
	group.GET("/:id", h.Get)
	group.PUT("/:id", h.Update)
	group.DELETE("/:id", h.Delete)
}

// sspTemplateRequest is ssptemplates.Input with JSON tags; it converts to it directly.
type sspTemplateRequest struct {
	Name                string                       `json:"name"`
	Enabled             *bool                        `json:"enabled"`
	Keys                []string                     `json:"keys"`
	Filter              *labelfilter.Filter          `json:"filter"`
	ScopeLabels         [][]string                   `json:"scope-labels"`
	MinStreams          *int                         `json:"min-streams"`
	TitleTemplate       string                       `json:"title-template"`
	DescriptionTemplate *string                      `json:"description-template"`
	Props               []relational.SSPTemplateProp `json:"props"`
	ProfileIDs          []uuid.UUID                  `json:"profile-ids"`
}

type sspTemplateResponse struct {
	ID                  uuid.UUID                    `json:"id"`
	Name                string                       `json:"name"`
	Enabled             bool                         `json:"enabled"`
	Keys                []string                     `json:"keys"`
	Filter              *labelfilter.Filter          `json:"filter"`
	ScopeLabels         [][]string                   `json:"scope-labels"`
	MinStreams          int                          `json:"min-streams"`
	TitleTemplate       string                       `json:"title-template"`
	DescriptionTemplate *string                      `json:"description-template"`
	Props               []relational.SSPTemplateProp `json:"props"`
	ProfileIDs          []uuid.UUID                  `json:"profile-ids"`
	CreatedAt           time.Time                    `json:"created-at"`
	UpdatedAt           time.Time                    `json:"updated-at"`
}

type sspTemplateDataResponse struct {
	Data sspTemplateResponse `json:"data"`
}

// toSSPTemplateResponse maps a stored template. Build stores every list non-nil, so the
// lists render as [] rather than null.
func toSSPTemplateResponse(t relational.SSPTemplate) sspTemplateResponse {
	resp := sspTemplateResponse{
		ID:                  *t.ID,
		Name:                t.Name,
		Enabled:             t.Enabled,
		Keys:                t.Keys,
		ScopeLabels:         t.ScopeLabels.Data(),
		MinStreams:          t.MinStreams,
		TitleTemplate:       t.TitleTemplate,
		DescriptionTemplate: t.DescriptionTemplate,
		Props:               t.Props,
		ProfileIDs:          t.ProfileIDs,
		CreatedAt:           t.CreatedAt,
		UpdatedAt:           t.UpdatedAt,
	}
	if t.Filter != nil {
		f := t.Filter.Data()
		resp.Filter = &f
	}
	return resp
}

// List godoc
//
//	@Summary		List SSP templates
//	@Description	List SSP templates, newest first.
//	@Tags			SSP Templates
//	@Produce		json
//	@Param			page	query		int	false	"Page number"
//	@Param			limit	query		int	false	"Page size"
//	@Success		200		{object}	svc.ListResponse[sspTemplateResponse]
//	@Failure		400		{object}	api.Error
//	@Failure		500		{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/admin/ssp-templates [get]
func (h *SSPTemplateHandler) List(ctx echo.Context) error {
	pagination, err := h.pagination.ParseParams(ctx)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	rows, total, err := h.service.List(pagination.Limit, pagination.Offset)
	if err != nil {
		return templateInternalServerError(ctx, h.sugar, "failed to list SSP templates", err)
	}
	resp := make([]sspTemplateResponse, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, toSSPTemplateResponse(row))
	}
	return ctx.JSON(http.StatusOK, svc.NewListResponse(resp, total, pagination.Page, pagination.Limit))
}

// Create godoc
//
//	@Summary		Create SSP template
//	@Description	Create an SSP template. Discovery creates one SSP per distinct combination of its key label values.
//	@Tags			SSP Templates
//	@Accept			json
//	@Produce		json
//	@Param			template	body		sspTemplateRequest	true	"SSP template"
//	@Success		201			{object}	sspTemplateDataResponse
//	@Failure		400			{object}	api.Error
//	@Failure		500			{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/admin/ssp-templates [post]
func (h *SSPTemplateHandler) Create(ctx echo.Context) error {
	var req sspTemplateRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	row, err := h.service.Create(ssptemplates.Input(req))
	if err != nil {
		return h.serviceError(ctx, "failed to create SSP template", err)
	}
	return ctx.JSON(http.StatusCreated, sspTemplateDataResponse{Data: toSSPTemplateResponse(row)})
}

// Get godoc
//
//	@Summary	Get SSP template
//	@Tags		SSP Templates
//	@Produce	json
//	@Param		id	path		string	true	"SSP template ID"
//	@Success	200	{object}	sspTemplateDataResponse
//	@Failure	400	{object}	api.Error
//	@Failure	404	{object}	api.Error
//	@Failure	500	{object}	api.Error
//	@Security	OAuth2Password
//	@Router		/admin/ssp-templates/{id} [get]
func (h *SSPTemplateHandler) Get(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	row, err := h.service.Get(id)
	if err != nil {
		return h.serviceError(ctx, "failed to get SSP template", err)
	}
	return ctx.JSON(http.StatusOK, sspTemplateDataResponse{Data: toSSPTemplateResponse(row)})
}

// Update godoc
//
//	@Summary		Update SSP template
//	@Description	Replace an SSP template's settings. Keys cannot change (400): create a new template instead. SSPs already created are not changed.
//	@Tags			SSP Templates
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string				true	"SSP template ID"
//	@Param			template	body		sspTemplateRequest	true	"SSP template"
//	@Success		200			{object}	sspTemplateDataResponse
//	@Failure		400			{object}	api.Error
//	@Failure		404			{object}	api.Error
//	@Failure		500			{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/admin/ssp-templates/{id} [put]
func (h *SSPTemplateHandler) Update(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	var req sspTemplateRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	row, err := h.service.Update(id, ssptemplates.Input(req))
	if err != nil {
		return h.serviceError(ctx, "failed to update SSP template", err)
	}
	return ctx.JSON(http.StatusOK, sspTemplateDataResponse{Data: toSSPTemplateResponse(row)})
}

// Delete godoc
//
//	@Summary		Delete SSP template
//	@Description	Delete an SSP template. The SSPs it created and their scope entries are kept.
//	@Tags			SSP Templates
//	@Param			id	path	string	true	"SSP template ID"
//	@Success		204
//	@Failure		400	{object}	api.Error
//	@Failure		404	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Security		OAuth2Password
//	@Router			/admin/ssp-templates/{id} [delete]
func (h *SSPTemplateHandler) Delete(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}
	if err := h.service.Delete(id); err != nil {
		return h.serviceError(ctx, "failed to delete SSP template", err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (h *SSPTemplateHandler) serviceError(ctx echo.Context, message string, err error) error {
	switch {
	case errors.Is(err, ssptemplates.ErrNotFound):
		return ctx.JSON(http.StatusNotFound, api.NotFound())
	case errors.Is(err, ssptemplates.ErrInvalidTemplate):
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	default:
		return templateInternalServerError(ctx, h.sugar, message, err)
	}
}
