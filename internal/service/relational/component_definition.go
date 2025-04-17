package relational

import (
	"database/sql"
	"time"

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
	LastModified         time.Time                       `json:"last-modified"`
	Version              string                          `json:"version"`
	OscalVersion         string                          `json:"oscal-version"`
	DocumentIDs          datatypes.JSONSlice[DocumentID] `json:"document-ids"` // -> DocumentID
	Props                datatypes.JSONSlice[Prop]       `json:"properties"`
	Links                datatypes.JSONSlice[Link]       `json:"links"`

	// TODO: Revisions is currently a 1:* with direct ties to catalog, either needs to be shifted to JSON ot many-to-many
	// Revisions are tied to a specific resource to denote it's history.
	// Many 2 Many would work, but wouldn't properly communicate it's use.
	// A polymorphic relationship on Revision would be better as that allows us to emulate a BelongsTo->HasMany relationship, without tying it to a specific parent model.
	Revisions          []Revision         `json:"revisions" gorm:"polymorphic:Parent;"`
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
