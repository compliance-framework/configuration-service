package relational

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Origins of an SSP scope entry.
const (
	// ScopeEntryOriginManual marks an entry a user added.
	ScopeEntryOriginManual = "manual"
	// ScopeEntryOriginTemplate marks an entry an SSP template learned from evidence.
	ScopeEntryOriginTemplate = "template"
)

// ScopeLabel is one key=value pair of a scope entry or exception.
type ScopeLabel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SSPScopeSettings holds per-SSP scope settings. A missing row means the defaults.
type SSPScopeSettings struct {
	SSPID       uuid.UUID `json:"ssp-id" gorm:"type:uuid;primaryKey"`
	AutoFilters bool      `json:"auto-filters" gorm:"not null;default:false"`
	UpdatedAt   time.Time `json:"updated-at"`

	SystemSecurityPlan *SystemSecurityPlan `json:"-" gorm:"foreignKey:SSPID;references:ID;constraint:OnDelete:CASCADE"`
}

func (SSPScopeSettings) TableName() string { return "ssp_scope_settings" }

// SSPScopeEntry is one entry of an SSP's scope: evidence carrying all of its labels is
// in scope. MatchHash identifies the evidence the entry matches (see sspscope.Normalize),
// so two entries matching the same evidence cannot coexist.
type SSPScopeEntry struct {
	UUIDModel
	SSPID      uuid.UUID                       `json:"ssp-id" gorm:"type:uuid;not null;uniqueIndex:idx_ssp_scope_entries_match,priority:1"`
	Labels     datatypes.JSONSlice[ScopeLabel] `json:"labels" gorm:"type:jsonb;not null"`
	MatchHash  string                          `json:"-" gorm:"not null;uniqueIndex:idx_ssp_scope_entries_match,priority:2"`
	Origin     string                          `json:"origin" gorm:"not null;default:manual"` // manual | template
	TemplateID *uuid.UUID                      `json:"template-id" gorm:"type:uuid;index"`    // set when Origin is template
	CreatedBy  *uuid.UUID                      `json:"created-by,omitempty" gorm:"type:uuid"`
	CreatedAt  time.Time                       `json:"created-at"`

	SystemSecurityPlan *SystemSecurityPlan `json:"-" gorm:"foreignKey:SSPID;references:ID;constraint:OnDelete:CASCADE"`
}

func (SSPScopeEntry) TableName() string { return "ssp_scope_entries" }

// SSPScopeException removes an entry from an SSP's scope for good: an entry with the same
// MatchHash is deleted when the exception is added, and templates never learn it again.
type SSPScopeException struct {
	UUIDModel
	SSPID     uuid.UUID                       `json:"ssp-id" gorm:"type:uuid;not null;uniqueIndex:idx_ssp_scope_exceptions_match,priority:1"`
	Labels    datatypes.JSONSlice[ScopeLabel] `json:"labels" gorm:"type:jsonb;not null"`
	MatchHash string                          `json:"-" gorm:"not null;uniqueIndex:idx_ssp_scope_exceptions_match,priority:2"`
	CreatedBy *uuid.UUID                      `json:"created-by,omitempty" gorm:"type:uuid"`
	CreatedAt time.Time                       `json:"created-at"`

	SystemSecurityPlan *SystemSecurityPlan `json:"-" gorm:"foreignKey:SSPID;references:ID;constraint:OnDelete:CASCADE"`
}

func (SSPScopeException) TableName() string { return "ssp_scope_exceptions" }

// SSPScopePolicyFilter records that an SSP's automatic per-policy filter was created for a
// _policy value, so a deleted automatic filter is not recreated: deleting the filter nulls
// FilterID and keeps the row.
type SSPScopePolicyFilter struct {
	SSPID     uuid.UUID  `json:"ssp-id" gorm:"type:uuid;primaryKey"`
	Policy    string     `json:"policy" gorm:"primaryKey"` // the _policy label value
	FilterID  *uuid.UUID `json:"filter-id" gorm:"type:uuid;index"`
	CreatedAt time.Time  `json:"created-at"`

	SystemSecurityPlan *SystemSecurityPlan `json:"-" gorm:"foreignKey:SSPID;references:ID;constraint:OnDelete:CASCADE"`
	Filter             *Filter             `json:"-" gorm:"foreignKey:FilterID;references:ID;constraint:OnDelete:SET NULL"`
}

func (SSPScopePolicyFilter) TableName() string { return "ssp_scope_policy_filters" }

// sspScopeTables are the tables whose ssp_id cascades from system_security_plans.
var sspScopeTables = []string{"ssp_scope_settings", "ssp_scope_entries", "ssp_scope_exceptions", "ssp_scope_policy_filters"}

// EnsureSSPConstraints adds the ON DELETE CASCADE foreign keys from the scope tables
// to system_security_plans, the ON DELETE SET NULL foreign key from
// ssp_scope_policy_filters to filters, and ssp_template_instances' foreign keys (see
// SSPTemplateInstance). AutoMigrate runs with
// DisableForeignKeyConstraintWhenMigrating, so the models' constraint tags alone are not
// enforced. Idempotent; postgres only.
func EnsureSSPConstraints(db *gorm.DB) error {
	if db.Name() != "postgres" {
		return nil
	}
	type foreignKey struct{ table, name, column, references, onDelete string }
	keys := make([]foreignKey, 0, len(sspScopeTables)+3)
	for _, table := range sspScopeTables {
		keys = append(keys, foreignKey{table, "fk_" + table + "_system_security_plan", "ssp_id", "system_security_plans(id)", "CASCADE"})
	}
	keys = append(keys, foreignKey{"ssp_scope_policy_filters", "fk_ssp_scope_policy_filters_filter", "filter_id", "filters(id)", "SET NULL"})
	keys = append(keys, foreignKey{"ssp_template_instances", "fk_ssp_template_instances_template", "template_id", "ssp_templates(id)", "CASCADE"})
	keys = append(keys, foreignKey{"ssp_template_instances", "fk_ssp_template_instances_ssp", "ssp_id", "system_security_plans(id)", "SET NULL"})
	for _, key := range keys {
		if err := db.Exec(fmt.Sprintf(`
			DO $$
			BEGIN
			  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = '%[2]s') THEN
			    ALTER TABLE %[1]s
			      ADD CONSTRAINT %[2]s
			      FOREIGN KEY (%[3]s)
			      REFERENCES %[4]s
			      ON DELETE %[5]s;
			  END IF;
			END $$;
		`, key.table, key.name, key.column, key.references, key.onDelete)).Error; err != nil {
			return err
		}
	}
	return nil
}
