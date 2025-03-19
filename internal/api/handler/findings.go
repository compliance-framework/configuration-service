package handler

import (
	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/service"
	"github.com/compliance-framework/configuration-service/sdk"
	"github.com/compliance-framework/configuration-service/sdk/types"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
)

type FindingsHandler struct {
	findingService   *service.FindingService
	subjectService   *service.SubjectService
	componentService *service.ComponentService
	sugar            *zap.SugaredLogger
}

func (h *FindingsHandler) Register(api *echo.Group) {
	api.POST("/", h.Create)
}

func NewFindingsHandler(
	l *zap.SugaredLogger,
	findingService *service.FindingService,
	subjectService *service.SubjectService,
	componentService *service.ComponentService,
) *FindingsHandler {
	return &FindingsHandler{
		sugar:            l,
		findingService:   findingService,
		subjectService:   subjectService,
		componentService: componentService,
	}
}

// Create godoc
//
//	@Summary		Create new findings
//	@Description	Creates multiple findings in the CCF API, as well as their subject and component counterparts
//	@Tags			Findings
//	@Accept			json
//	@Produce		json
//	@Success		201	"Created"
//	@Failure		400	{object}	api.Error
//	@Failure		500	{object}	api.Error
//	@Router			/findings [post]
func (h *FindingsHandler) Create(ctx echo.Context) error {
	// Bind the incoming JSON payload into a slice of findings.
	var findings []*types.Finding
	if err := ctx.Bind(&findings); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	// Process each finding.
	for _, finding := range findings {
		// First we need to create and map the subjects that were passed.
		// For the moment we will simply seed a UUID with the subject attribute map, but this will alter
		// change to only include some attributes to map across different subject attributes lists that reference
		// the same subject
		// Process the SDK subjects if provided.
		subjectIds := make([]uuid.UUID, 0)
		if finding.Subjects != nil {
			var subjectIDs []uuid.UUID
			// Iterate over each SDK subject reference.
			for _, subject := range *finding.Subjects {
				// Generate a seeded (consistent) UUID for the subject based on its attributes.
				seededID, err := sdk.SeededUUID(subject.Attributes)
				if err != nil {
					return ctx.JSON(http.StatusBadRequest, api.NewError(err))
				}
				// Use the SubjectService's FindOrCreate method.
				createdSubject, err := h.subjectService.FindOrCreate(ctx.Request().Context(), &seededID, &service.Subject{
					// The FindOrCreate method will ensure the ID is set using the seeded ID.
					Type:       subject.Type,
					Title:      subject.Title,
					Remarks:    subject.Remarks,
					Attributes: subject.Attributes,
					Links:      subject.Links,
					Props:      subject.Props,
				})
				if err != nil {
					return ctx.JSON(http.StatusBadRequest, api.NewError(err))
				}
				// Append the subject's ID to the list.
				subjectIDs = append(subjectIDs, *createdSubject.ID)
			}
		}

		// We also want to compile the OSCAL style observation ids to our local
		observationIds := make([]uuid.UUID, 0)
		for _, relatedObservation := range *finding.RelatedObservations {
			observationIds = append(observationIds, relatedObservation.ObservationUuid)
		}

		// We also want to ensure the components are in the database.
		// For not they will be placeholders, but eventually their information will be pulled in from
		// the common components library
		componentIds := make([]uuid.UUID, 0)
		for _, componentReference := range *finding.Components {
			component, err := h.componentService.FindOrCreate(ctx.Request().Context(), componentReference.Identifier, &service.Component{
				Identifier: componentReference.Identifier,
				Title:      componentReference.Identifier, // Using identifier for title for now. This will eventually be pulled from common components.
			})
			if err != nil {
				return ctx.JSON(http.StatusBadRequest, api.NewError(err))
			}
			componentIds = append(componentIds, *component.ID)
		}

		newFinding := &service.Finding{
			ID:             &finding.ID,
			UUID:           finding.UUID,
			Title:          finding.Title,
			Description:    finding.Description,
			Collected:      finding.Collected,
			Remarks:        &finding.Remarks,
			Labels:         finding.Labels,
			Origins:        finding.Origins,
			SubjectIDs:     &subjectIds,
			ComponentIDs:   &componentIds,
			ObservationIDs: &observationIds,
			Controls:       finding.Controls,
			Risks:          finding.Risks,
			Status:         finding.Status,
			Links:          finding.Links,
			Props:          finding.Props,
		}

		if _, err := h.findingService.Create(ctx.Request().Context(), newFinding); err != nil {
			h.sugar.Errorw("failed to create finding", "error", err)
			return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
		}
	}

	// Return a 201 Created response with no content.
	return ctx.NoContent(http.StatusCreated)
}
