package handler

import (
	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
	"strings"
	"time"
)

func ParseIntervalListQueryParam(intervalQuery string, def []time.Duration) ([]time.Duration, error) {
	if intervalQuery == "" {
		return def, nil
	}

	var intervals []time.Duration
	userIntervals := strings.Split(intervalQuery, ",")
	for _, interval := range userIntervals {
		dur, err := time.ParseDuration(interval)
		if err != nil {
			return nil, err
		}
		intervals = append(intervals, dur)
	}
	return intervals, nil
}

// createPlanRequest defines the request payload for method Create
type createFilterRequest struct {
	Name     string             `json:"name" yaml:"name" validate:"required"`
	Filter   labelfilter.Filter `json:"filter" yaml:"filter" validate:"required"`
	Controls *[]string          `json:"controls" yaml:"controls"`
}

func (r *createFilterRequest) bind(ctx echo.Context, p *relational.Filter) error {
	if err := ctx.Bind(r); err != nil {
		return err
	}
	p.Name = r.Name
	p.Filter = datatypes.NewJSONType(r.Filter)
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
