package handler

import (
	"github.com/compliance-framework/configuration-service/converters/labelfilter"
	"github.com/compliance-framework/configuration-service/domain"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/labstack/echo/v4"
)

// createCatalogRequest defines the request payload for method CreateCatalog
type createCatalogRequest struct {
	Catalog struct {
		Title string `json:"title" yaml:"title" validate:"required"`
	}
}

func newCreateCatalogRequest() *createCatalogRequest {
	return &createCatalogRequest{}
}

func (r *createCatalogRequest) bind(ctx echo.Context, c *oscaltypes113.Catalog) error {
	if err := ctx.Bind(r); err != nil {
		return err
	}
	c.Metadata.Title = r.Catalog.Title
	return nil
}

// createControlRequest defines the request payload for method CreateControl
type createControlRequest struct {
	Control struct {
		Title string `json:"title" yaml:"title" validate:"required"`
	}
}

func newCreateControlRequest() *createControlRequest {
	return &createControlRequest{}
}

func (r *createControlRequest) bind(ctx echo.Context, c *oscaltypes113.Control) error {
	if err := ctx.Bind(r); err != nil {
		return err
	}
	c.Title = r.Control.Title
	return nil
}

// createSSPRequest defines the request payload for method CreateSSP
type CreateSSPRequest struct {
	Title string `json:"title" yaml:"title" validate:"required"`
}

func (r *CreateSSPRequest) bind(ctx echo.Context, ssp *oscaltypes113.SystemSecurityPlan) error {
	if err := ctx.Bind(r); err != nil {
		return err
	}

	ssp.Metadata.Title = r.Title
	return nil
}

// createPlanRequest defines the request payload for method Create
// TODO: Using minimal data for now, we might need to expand it later
type createPlanRequest struct {
	Title  string             `json:"title" yaml:"title" validate:"required"`
	Filter labelfilter.Filter `json:"filter" yaml:"filter" validate:"required"`
}

func (r *createPlanRequest) bind(ctx echo.Context, p *domain.Plan) error {
	if err := ctx.Bind(r); err != nil {
		return err
	}
	p.Metadata.Title = r.Title
	p.ResultFilter = r.Filter
	return nil
}

// createPlanRequest defines the request payload for method Create
// TODO: Using minimal data for now, we might need to expand it later
type filteredSearchRequest struct {
	Filter labelfilter.Filter `json:"filter" yaml:"filter" validate:"required"`
}

func (r *filteredSearchRequest) bind(ctx echo.Context, p *labelfilter.Filter) error {
	if err := ctx.Bind(r); err != nil {
		return err
	}
	p.Scope = r.Filter.Scope
	return nil
}

// addTaskRequest defines the request payload for method CreateTask
// TODO: these are not currently used anywhere - When it is used, remove nolints:
type addAssetRequest struct { //nolint
	AssetId string `json:"assetId" yaml:"assetId" validate:"required"`
	Type    string `json:"type" yaml:"type" validate:"required"`
}

func (r *addAssetRequest) bind(ctx echo.Context, p *domain.Plan) error { //nolint
	if err := ctx.Bind(r); err != nil {
		return err
	}

	return nil
}

// createTaskRequest defines the request payload for method CreateTask
type createTaskRequest struct {
	// TODO: We are keeping it minimal for now for the demo
	Title       string `json:"title" yaml:"title" validate:"required"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string `json:"type" yaml:"type" validate:"required"`
	Schedule    string `json:"schedule" yaml:"schedule" validate:"required"`
}

func (r *createTaskRequest) Bind(ctx echo.Context, t *oscaltypes113.Task) error {
	if err := ctx.Bind(r); err != nil {
		return err
	}
	t.Title = r.Title
	t.Description = r.Description
	t.Type = r.Type
	return nil
}

// updateSSPRequest defines the request payload for method UpdateSSP
type UpdateSSPRequest struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
}

func (r *UpdateSSPRequest) bind(ctx echo.Context, ssp *oscaltypes113.SystemSecurityPlan) error {
	if err := ctx.Bind(r); err != nil {
		return err
	}

	ssp.Metadata.Title = r.Title
	return nil
}
