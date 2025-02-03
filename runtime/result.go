package runtime

import (
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
)

type ExecutionStatus int
type LogType int

const (
	ExecutionStatusSuccess ExecutionStatus = iota
	ExecutionStatusFailure
)

// ExecutionResult holds the result of an compliance check execution for each subject.
type ExecutionResult struct {
	Title        string                         `json:"title" yaml:"title"`
	Labels       map[string]string              `json:"labels" yaml:"labels"`
	Status       ExecutionStatus                `json:"status" yaml:"status"`
	StreamId     uuid.UUID                      `json:"streamId" yaml:"streamId"`
	ComponentId  string                         `json:"componentId" yaml:"componentId"`
	ControlId    string                         `json:"controlId" yaml:"controlId"`
	TaskId       string                         `json:"taskId" yaml:"taskId"`
	ActivityId   string                         `json:"activityId" yaml:"activityId"`
	Error        error                          `json:"error" yaml:"error"`
	Subject      oscaltypes113.SubjectReference `json:"subject" yaml:"subject"`
	Observations []oscaltypes113.Observation    `json:"observations" yaml:"observations"`
	Findings     []oscaltypes113.Finding        `json:"findings" yaml:"findings"`
	Risks        []oscaltypes113.Risk           `json:"risks" yaml:"risks"`
	Logs         oscaltypes113.AssessmentLog    `json:"logs" yaml:"logs"`
}
