package oscal

import (
	"fmt"

	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
)

type Validatable interface {
	Validate() []ValidationError
}

type ValidationError struct {
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

var _ Validatable = (*AssessmentPlanCreateRequest)(nil)

type AssessmentPlanCreateRequest struct {
	Data *oscalTypes.AssessmentPlan
}

func (ap *AssessmentPlanCreateRequest) Validate() []ValidationError {
	errors := []ValidationError{}

	if ap.Data == nil {
		errors = append(errors, ValidationError{
			Message: "assessment plan data is required",
			Field:   "data",
		})
		return errors
	}

	if ap.Data.UUID == "" {
		errors = append(errors, ValidationError{
			Message: "UUID is required",
			Field:   "uuid",
		})
	} else if _, err := uuid.Parse(ap.Data.UUID); err != nil {
		errors = append(errors, ValidationError{
			Message: fmt.Sprintf("invalid UUID format: %s", ap.Data.UUID),
			Field:   "uuid",
		})
	}

	if ap.Data.Metadata.Title == "" {
		errors = append(errors, ValidationError{
			Message: "metadata.title is required",
			Field:   "metadata.title",
		})
	}

	if ap.Data.Metadata.Version == "" {
		errors = append(errors, ValidationError{
			Message: "metadata.version is required",
			Field:   "metadata.version",
		})
	}

	if ap.Data.ImportSsp.Href == "" {
		errors = append(errors, ValidationError{
			Message: "import-ssp.href is required",
			Field:   "import-ssp.href",
		})
	}

	return errors
}
