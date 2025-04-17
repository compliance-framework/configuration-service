package relational

import (
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
)

type ComponentDefinition struct {
	UUIDModel
	Metadata   Metadata   `json:"metadata" gorm:"polymorphic:Parent;"`
	BackMatter BackMatter `json:"back-matter" gorm:"polymorphic:Parent;"`

	//oscaltypes113.ComponentDefinition
}

func (c *ComponentDefinition) UnmarshalOscal(ocd oscalTypes_1_1_3.ComponentDefinition) *ComponentDefinition {
	metadata := &Metadata{}
	metadata.UnmarshalOscal(ocd.Metadata)

	id := uuid.MustParse(ocd.UUID)
	*c = ComponentDefinition{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		Metadata: *metadata,
	}
	return c
}
