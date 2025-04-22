package relational

import (
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ComponentDefinition struct {
	UUIDModel
	Metadata   Metadata   `json:"metadata" gorm:"polymorphic:Parent;"`
	BackMatter BackMatter `json:"back-matter" gorm:"polymorphic:Parent;"`

	Components []DefinedComponent `json:"components"`

	//oscaltypes113.ComponentDefinition
}

func (c *ComponentDefinition) UnmarshalOscal(ocd oscalTypes_1_1_3.ComponentDefinition) *ComponentDefinition {
	metadata := &Metadata{}
	metadata.UnmarshalOscal(ocd.Metadata)

	id := uuid.MustParse(ocd.UUID)

	components := ConvertList(ocd.Components, func(odc oscalTypes_1_1_3.DefinedComponent) DefinedComponent {
		dc := &DefinedComponent{}
		dc.UnmarshalOscal(odc)
		return *dc
	})

	*c = ComponentDefinition{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		Metadata:   *metadata,
		Components: components,
	}
	return c
}

type DefinedComponent struct {
	UUIDModel
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Purpose     string `json:"purpose"`
	Remarks     string `json:"remarks"`

	ResponsibleRoles []Role `json:"responsible-roles" gorm:"many2many:component_definition_responsible_roles"`

	Props     datatypes.JSONSlice[Prop]     `json:"props"`
	Links     datatypes.JSONSlice[Link]     `json:"links"`
	Protocols datatypes.JSONSlice[Protocol] `json:"protocols"`

	ComponentDefinitionID uuid.UUID

	// ResponsibleRoles -> many2many to roles
	// Protocols -> JSON?
	// Control Implementation -> 1 to many?

	// oscalTypes113.DefinedComponent
}

func (dc *DefinedComponent) UnmarshalOscal(odc oscalTypes_1_1_3.DefinedComponent) *DefinedComponent {
	id := uuid.MustParse(odc.UUID)

	protocols := ConvertList(odc.Protocols, func(op oscalTypes_1_1_3.Protocol) Protocol {
		protocol := Protocol{}
		protocol.UnmarshalOscal(op)
		return protocol
	})

	links := ConvertList(odc.Links, func(ol oscalTypes_1_1_3.Link) Link {
		link := Link{}
		link.UnmarshalOscal(ol)
		return link
	})

	props := ConvertList(odc.Props, func(op oscalTypes_1_1_3.Property) Prop {
		prop := Prop{}
		prop.UnmarshalOscal(op)
		return prop
	})

	*dc = DefinedComponent{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		Type:        odc.Type,
		Title:       odc.Title,
		Purpose:     odc.Purpose,
		Remarks:     odc.Remarks,
		Description: odc.Description,
		Protocols:   datatypes.NewJSONSlice[Protocol](protocols),
		Links:       datatypes.NewJSONSlice[Link](links),
		Props:       datatypes.NewJSONSlice[Prop](props),
	}
	return dc
}

type Protocol oscalTypes_1_1_3.Protocol

func (p *Protocol) UnmarshalOscal(op oscalTypes_1_1_3.Protocol) *Protocol {
	*p = Protocol(op)
	return p
}
