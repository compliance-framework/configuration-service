package relational

import (
	"time"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SSPTemplateProp is a prop an SSP template writes on the SSPs it creates. Value is a Go
// text/template rendered with the SSP's key values.
type SSPTemplateProp struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SSPTemplate creates one SSP per distinct combination of its key label values seen in
// the latest evidence, and keeps each SSP's scope up to date (see ssptemplates).
type SSPTemplate struct {
	UUIDModel
	Name    string `gorm:"not null"`
	Enabled bool   `gorm:"not null;default:true"`
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
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (SSPTemplate) TableName() string { return "ssp_templates" }

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

	Template *SSPTemplate `gorm:"foreignKey:TemplateID;constraint:OnDelete:CASCADE"`
}

func (SSPTemplateInstance) TableName() string { return "ssp_template_instances" }

// EnsureSSPTemplateConstraints adds the foreign keys of ssp_template_instances: deleting
// a template deletes its instances (its SSPs and their scope entries are kept), and
// deleting an SSP nulls its instance's ssp_id. AutoMigrate runs with
// DisableForeignKeyConstraintWhenMigrating, so the models' constraint tags alone are not
// enforced. Idempotent; postgres only.
func EnsureSSPTemplateConstraints(db *gorm.DB) error {
	if db.Name() != "postgres" {
		return nil
	}
	return ensureForeignKeys(db, []foreignKey{
		{"ssp_template_instances", "fk_ssp_template_instances_template", "template_id", "ssp_templates(id)", "CASCADE"},
		{"ssp_template_instances", "fk_ssp_template_instances_ssp", "ssp_id", "system_security_plans(id)", "SET NULL"},
	})
}
