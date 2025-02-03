package domain

import (
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Result struct {
	Id       *primitive.ObjectID `json:"_id,omitempty" yaml:"_id,omitempty" bson:"_id,omitempty"`
	StreamID uuid.UUID           `json:"streamId,omitempty" yaml:"streamId,omitempty" bson:"streamId,omitempty"`
	Labels   map[string]string   `json:"labels,omitempty" yaml:"labels,omitempty" bson:"labels,omitempty"`
	oscaltypes113.Result
}

type ObservationMethod string

const (
	ObservationMethodExamine   ObservationMethod = "examine"
	ObservationMethodInterview ObservationMethod = "interview"
	ObservationMethodTest      ObservationMethod = "test"
	ObservationMethodUnknown   ObservationMethod = "unknown"
)

type ObservationType string

const (
	ObservationTypeSSPStatementIssue ObservationType = "ssp-statement-issue"
	ObservationTypeControlObjective  ObservationType = "control-objective"
	ObservationTypeMitigation        ObservationType = "mitigation"
	ObservationTypeFinding           ObservationType = "finding"
	ObservationTypeHistoric          ObservationType = "historic"
)

type RiskStatus string

const (
	RiskStatusOpen               RiskStatus = "open"
	RiskStatusInvestigating      RiskStatus = "investigating"
	RiskStatusRemediating        RiskStatus = "remediating"
	RiskStatusDeviationRequested RiskStatus = "deviation-requested"
	RiskStatusDeviationApproved  RiskStatus = "deviation-approved"
	RiskStatusClosed             RiskStatus = "closed"
)
