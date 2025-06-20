package relational

import (
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"time"
)

type Evidence struct {
	// ID is the unique ID for this specific observation, and will be used as the primary key in the database.
	UUIDModel

	// UUID needs to remain consistent when automation runs again, but unique for each subject.
	// It represents the "stream" of the same observation being made over time.
	UUID uuid.UUID `gorm:"index"`

	Title       *string `json:"title,omitempty" yaml:"title,omitempty"`
	Description string  `json:"description" yaml:"description"`
	Remarks     *string `json:"remarks,omitempty" yaml:"remarks,omitempty"`

	// Assigning labels to Evidence makes it searchable and easily usable in the UI
	Labels []Labels `gorm:"many2many:evidence_labels;"`

	// When did we start collecting the evidence, and when did the process end, and how long is it valid for ?
	Start   time.Time
	End     time.Time
	Expires *time.Time

	Props []Prop
	Links []Link

	// Who or What is generating this evidence
	Origins datatypes.JSONSlice[Origin]

	// Who or What are we providing evidence for. What's under test.
	Subjects []AssessmentSubject `gorm:"many2many:evidence_subjects;"`

	// What steps did we take to create this evidence
	Activities []Activity `gorm:"many2many:evidence_activities"`

	// Which components of the subject are being observed. A tool, user, policy etc.
	Components []SystemComponent `gorm:"many2many:evidence_components"`

	// Did we satisfy what was being tested for, or did we fail ?
	Status datatypes.JSONType[oscalTypes_1_1_3.ObjectiveStatus]
}
