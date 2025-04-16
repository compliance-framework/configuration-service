package relational

import (
	"database/sql"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ComponentDefinition struct {
	UUIDModel
	Metadata ComponentDefinitionMetadata `json:"metadata" gorm:"foreignKey:ComponentDefintionID"`
}

// `ComponentDefinitionMetadata` belongs to `ComponentDefinition`, `ComponentDefinitionID` is the foriegn key
type ComponentDefinitionMetadata struct {
	UUIDModel
	ComponentDefintionID uuid.UUID
	Title                string                          `json:"title"`
	Published            sql.NullTime                    `json:"published"`
	LastModified         sql.NullTime                    `json:"last-modified"`
	Version              string                          `json:"version"`
	OscalVersion         string                          `json:"oscal-version"`
	DocumentIDs          datatypes.JSONSlice[DocumentID] `json:"document-ids"` // -> DocumentID
	Properties           datatypes.JSONSlice[Prop]       `json:"properties"`
	Links                datatypes.JSONSlice[Link]       `json:"links"`

	// TODO: Revisions is currently a 1:* with direct ties to catalog, either needs to be shifted to JSON ot many-to-many
	//Revisions          []Revision         `json:"revisions"` // -> Revision

	Roles              []Role             `gorm:"many2many:component_definition_roles;"`
	Locations          []Location         `gorm:"many2many:component_definition_locations;"`
	Parties            []Party            `gorm:"many2many:component_definition_parties;"`
	ResponsibleParties []ResponsibleParty `gorm:"many2many:component_definition_responsible_parties;"`

	// TODO: Actions is currently a 1:* with direct ties to catalog, either needs to be shifted to JSON ot many-to-many
	//Actions []Action `json:"actions"` // -> Action
	Remarks string `json:"remarks"`

	/**
	"required": [
		"title",
		"last-modified",
		"version",
		"oscal-version"
	],
	*/
}
