package relational

import (
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Catalog struct {
	UUIDModel
	Metadata   Metadata                       `json:"metadata" gorm:"polymorphic:Parent;"`
	Params     datatypes.JSONSlice[Parameter] `json:"params"`
	Controls   []Control                      `json:"controls"`
	Groups     []Group                        `json:"groups"`
	BackMatter BackMatter                     `json:"back-matter" gorm:"polymorphic:Parent;"`
	/**
	"required": [
		"uuid",
		"metadata"
	],
	*/
}

type Group struct {
	ID     string                         `json:"id" gorm:"primary_key"` // required
	Class  string                         `json:"class"`
	Title  string                         `json:"title"` // required
	Params datatypes.JSONSlice[Parameter] `json:"params"`
	Parts  datatypes.JSONSlice[Part]      `json:"parts"`
	Props  datatypes.JSONSlice[Prop]      `json:"props"`
	Links  datatypes.JSONSlice[Link]      `json:"links"`

	CatalogID  uuid.UUID
	ParentID   *string
	ParentType *string

	Groups   []Group   `json:"groups" gorm:"polymorphic:Parent;"`
	Controls []Control `json:"controls" gorm:"polymorphic:Parent;"`
}

type Control struct {
	ID     string                         `json:"id" gorm:"primary_key"` // required
	Title  string                         `json:"title"`                 // required
	Class  *string                        `json:"class"`
	Params datatypes.JSONSlice[Parameter] `json:"params"`
	Parts  datatypes.JSONSlice[Part]      `json:"parts"`
	Props  datatypes.JSONSlice[Prop]      `json:"props"`
	Links  datatypes.JSONSlice[Link]      `json:"links"`

	CatalogID  uuid.UUID
	ParentID   *string
	ParentType *string

	Controls []Control `json:"controls" gorm:"polymorphic:Parent;"`
}

type Parameter struct {
	ID          string                `json:"id"`
	Class       string                `json:"class"`
	Props       []Prop                `json:"props"`
	Links       []Link                `json:"links"`
	Label       string                `json:"label"`
	Usage       string                `json:"usage"`
	Constraints []ParameterConstraint `json:"constraints"`
	Guidelines  []ParameterGuideline  `json:"guidelines"`
	Values      []string              `json:"values"`
	Select      ParameterSelection    `json:"select"`
	Remarks     string                `json:"remarks"`

	/**
	"required": [
		"id"
	],
	*/
}

type ParameterSelectionCount string

const (
	ParameterSelectionCountOne       ParameterSelectionCount = "one"
	ParameterSelectionCountOneOrMore ParameterSelectionCount = "one-or-more"
)

type ParameterSelection struct {
	HowMany ParameterSelectionCount `json:"how-many"`
	Choice  []string                `json:"choice"`
}

type ParameterGuideline struct {
	Prose string `json:"prose"`

	/**
	"required": [
		"prose"
	],
	*/
}

type ParameterConstraint struct {
	Description string                    `json:"description"`
	Tests       []ParameterConstraintTest `json:"tests"`
}

type ParameterConstraintTest struct {
	Expression string `json:"expression"`
	Remarks    string `json:"remarks"`
}

func (l *ParameterConstraintTest) UnmarshalOscal(data oscalTypes_1_1_3.ConstraintTest) *ParameterConstraintTest {
	*l = ParameterConstraintTest(data)
	return l
}

type Part struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	NS    string `json:"ns"`
	Class string `json:"class"`
	Title string `json:"title"`
	Prose string `json:"prose"`
	Props []Prop `json:"props"`
	Links []Link `json:"links"`
	Parts []Part `json:"parts"` // -> Part

	/**
	"required": [
		"name"
	],
	*/
}
