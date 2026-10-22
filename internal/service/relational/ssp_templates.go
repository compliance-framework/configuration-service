package relational

import (
	"time"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// SSPTemplateProp is a prop written on created SSPs; Value is a text/template of the keys.
type SSPTemplateProp struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SSPTemplate creates one SSP per distinct combination of its key label values seen in
// the latest evidence, and keeps each SSP's scope up to date (see ssptemplates).
type SSPTemplate struct {
	UUIDModel
	Name    string `gorm:"not null"`
	Enabled bool   `gorm:"not null"` // no column default: gorm would write it over a false Enabled
	// Keys identify one SSP by their values: sorted, lower-cased, immutable.
	Keys datatypes.JSONSlice[string] `gorm:"type:jsonb;not null"`
	// Filter, when set, limits discovery to the evidence it matches.
	Filter *datatypes.JSONType[labelfilter.Filter] `gorm:"type:jsonb"`
	// ScopeLabels are label groups learned into each SSP's scope (sorted, lower-cased).
	ScopeLabels         datatypes.JSONType[[][]string] `gorm:"type:jsonb"`
	MinStreams          int                            `gorm:"not null;default:1"`
	TitleTemplate       string                         `gorm:"not null"`
	DescriptionTemplate *string
	Props               datatypes.JSONSlice[SSPTemplateProp] `gorm:"type:jsonb"`
	ProfileIDs          datatypes.JSONSlice[uuid.UUID]       `gorm:"type:jsonb"`
	// LastRunAt is when a discovery run for the template last completed.
	LastRunAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (SSPTemplate) TableName() string { return "ssp_templates" }

// SkippedScopeEntry is a learned scope entry discovery could not store, with the reason.
type SkippedScopeEntry struct {
	Labels []ScopeLabel `json:"labels"`
	Reason string       `json:"reason"`
}

// SSPTemplateInstance records an SSP a template created, keyed by its key values.
type SSPTemplateInstance struct {
	TemplateID uuid.UUID `gorm:"type:uuid;primaryKey"`
	// KeyHash is the SHA-256 of the sorted lower(key)=value pairs.
	KeyHash   string            `gorm:"primaryKey"`
	KeyValues datatypes.JSONMap `gorm:"type:jsonb;not null"`
	// SSPID is nulled when the SSP is deleted (ON DELETE SET NULL); discovery then skips
	// the instance, so a deleted SSP is not recreated until an admin forgets it.
	SSPID      *uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt  time.Time
	LastSeenAt time.Time
	// LastError is the last failure applying the instance (a profile attach, say).
	LastError *string
	// ProfilesPending is set while some of the template's profiles failed to attach to
	// the SSP; discovery retries them on its next run.
	ProfilesPending bool `gorm:"not null;default:false"`
	// SkippedEntries counts the learned entries the last run could not store (a value
	// over the length limit, say); SkippedSample keeps a few of them.
	SkippedEntries int                                    `gorm:"not null;default:0"`
	SkippedSample  datatypes.JSONSlice[SkippedScopeEntry] `gorm:"type:jsonb"`

	Template *SSPTemplate `gorm:"foreignKey:TemplateID;constraint:OnDelete:CASCADE"`
}

func (SSPTemplateInstance) TableName() string { return "ssp_template_instances" }
