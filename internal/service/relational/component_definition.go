package relational

import oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"

type ComponentDefinition struct {
	UUIDModel
	Metadata   Metadata   `json:"metadata" gorm:"polymorphic:Parent;"`
	BackMatter BackMatter `json:"back-matter" gorm:"polymorphic:Parent;"`

	oscaltypes113.ComponentDefinition
}
