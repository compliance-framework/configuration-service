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
	Groups     []Group                        `json:"groups" gorm:"polymorphic:Parent;"`
	Controls   []Control                      `json:"controls" gorm:"polymorphic:Parent;"`
	BackMatter BackMatter                     `json:"back-matter" gorm:"polymorphic:Parent;"`
	/**
	"required": [
		"uuid",
		"metadata"
	],
	*/
}

func (c *Catalog) UnmarshalOscal(ocatalog oscalTypes_1_1_3.Catalog) *Catalog {
	metadata := &Metadata{}
	metadata.UnmarshalOscal(ocatalog.Metadata)

	id := uuid.MustParse(ocatalog.UUID)
	*c = Catalog{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		Metadata: *metadata,
	}

	if ocatalog.BackMatter != nil {
		backmatter := &BackMatter{}
		backmatter.UnmarshalOscal(*ocatalog.BackMatter)
		c.BackMatter = *backmatter
	}

	if ocatalog.Params != nil {
		c.Params = ConvertList(ocatalog.Params, func(data oscalTypes_1_1_3.Parameter) Parameter {
			output := Parameter{}
			output.UnmarshalOscal(data)
			return output
		})
	}

	if ocatalog.Controls != nil {
		c.Controls = ConvertList(ocatalog.Controls, func(data oscalTypes_1_1_3.Control) Control {
			output := Control{}
			output.UnmarshalOscal(data, id)
			return output
		})
	}

	if ocatalog.Groups != nil {
		c.Groups = ConvertList(ocatalog.Groups, func(data oscalTypes_1_1_3.Group) Group {
			output := Group{}
			output.UnmarshalOscal(data, id)
			return output
		})
	}

	return c
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

func (c *Group) UnmarshalOscal(data oscalTypes_1_1_3.Group, catalogId uuid.UUID) *Group {
	*c = Group{
		ID:        data.ID,
		Title:     data.Title,
		Class:     data.Class,
		Props:     ConvertOscalProps(data.Props),
		Links:     ConvertOscalLinks(data.Links),
		CatalogID: catalogId,
	}
	if data.Params != nil {
		params := ConvertList(data.Params, func(data oscalTypes_1_1_3.Parameter) Parameter {
			output := Parameter{}
			output.UnmarshalOscal(data)
			return output
		})
		c.Params = params
	}
	if data.Parts != nil {
		parts := ConvertList(data.Parts, func(data oscalTypes_1_1_3.Part) Part {
			output := Part{}
			output.UnmarshalOscal(data)
			return output
		})
		c.Parts = parts
	}
	if data.Groups != nil {
		groups := ConvertList(data.Groups, func(data oscalTypes_1_1_3.Group) Group {
			output := Group{}
			output.UnmarshalOscal(data, c.CatalogID)
			return output
		})
		c.Groups = groups
	}
	if data.Controls != nil {
		controls := ConvertList(data.Controls, func(data oscalTypes_1_1_3.Control) Control {
			output := Control{}
			output.UnmarshalOscal(data, c.CatalogID)
			return output
		})
		c.Controls = controls
	}
	return c
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

func (c *Control) UnmarshalOscal(data oscalTypes_1_1_3.Control, catalogId uuid.UUID) *Control {
	*c = Control{
		ID:        data.ID,
		Title:     data.Title,
		Class:     &data.Class,
		Props:     ConvertOscalProps(data.Props),
		Links:     ConvertOscalLinks(data.Links),
		CatalogID: catalogId,
	}
	if data.Params != nil {
		params := ConvertList(data.Params, func(data oscalTypes_1_1_3.Parameter) Parameter {
			output := Parameter{}
			output.UnmarshalOscal(data)
			return output
		})
		c.Params = params
	}
	if data.Parts != nil {
		parts := ConvertList(data.Parts, func(data oscalTypes_1_1_3.Part) Part {
			output := Part{}
			output.UnmarshalOscal(data)
			return output
		})
		c.Parts = parts
	}
	if data.Controls != nil {
		controls := ConvertList(data.Controls, func(data oscalTypes_1_1_3.Control) Control {
			output := Control{}
			output.UnmarshalOscal(data, c.CatalogID)
			return output
		})
		c.Controls = controls
	}
	return c
}

type Parameter struct {
	ID          string                                   `json:"id"`
	Class       *string                                  `json:"class"`
	Label       *string                                  `json:"label"`
	Usage       *string                                  `json:"usage"`
	Remarks     *string                                  `json:"remarks"`
	Constraints datatypes.JSONSlice[ParameterConstraint] `json:"constraints"`
	Guidelines  datatypes.JSONSlice[ParameterGuideline]  `json:"guidelines"`
	Select      datatypes.JSONType[ParameterSelection]   `json:"select"`
	Values      datatypes.JSONSlice[string]              `json:"values"`
	Props       datatypes.JSONSlice[Prop]                `json:"props"`
	Links       datatypes.JSONSlice[Link]                `json:"links"`

	/**
	"required": [
		"id"
	],
	*/
}

func (l *Parameter) UnmarshalOscal(data oscalTypes_1_1_3.Parameter) *Parameter {
	*l = Parameter{
		ID:      data.ID,
		Class:   &data.Class,
		Props:   ConvertOscalProps(data.Props),
		Links:   ConvertOscalLinks(data.Links),
		Label:   &data.Label,
		Usage:   &data.Usage,
		Remarks: &data.Remarks,
	}
	if data.Select != nil {
		selection := ParameterSelection{}
		selection.UnmarshalOscal(*data.Select)
		l.Select = datatypes.NewJSONType(selection)
	}
	if data.Constraints != nil {
		l.Constraints = ConvertList(data.Constraints, func(pc oscalTypes_1_1_3.ParameterConstraint) ParameterConstraint {
			constraint := ParameterConstraint{}
			constraint.UnmarshalOscal(pc)
			return constraint
		})
	}
	if data.Guidelines != nil {
		l.Guidelines = ConvertList(data.Guidelines, func(data oscalTypes_1_1_3.ParameterGuideline) ParameterGuideline {
			output := ParameterGuideline{}
			output.UnmarshalOscal(data)
			return output
		})
	}
	if data.Values != nil {
		l.Values = *data.Values
	}
	return l
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

func (l *ParameterSelection) UnmarshalOscal(data oscalTypes_1_1_3.ParameterSelection) *ParameterSelection {
	*l = ParameterSelection{
		HowMany: ParameterSelectionCount(data.HowMany),
	}
	if data.Choice != nil {
		l.Choice = *data.Choice
	}
	return l
}

type ParameterGuideline struct {
	Prose string `json:"prose"`

	/**
	"required": [
		"prose"
	],
	*/
}

func (l *ParameterGuideline) UnmarshalOscal(data oscalTypes_1_1_3.ParameterGuideline) *ParameterGuideline {
	*l = ParameterGuideline(data)
	return l
}

type ParameterConstraint struct {
	Description string                    `json:"description"`
	Tests       []ParameterConstraintTest `json:"tests"`
}

func (l *ParameterConstraint) UnmarshalOscal(data oscalTypes_1_1_3.ParameterConstraint) *ParameterConstraint {
	*l = ParameterConstraint{
		Description: data.Description,
	}
	if data.Tests != nil {
		l.Tests = ConvertList(data.Tests, func(t oscalTypes_1_1_3.ConstraintTest) ParameterConstraintTest {
			test := ParameterConstraintTest{}
			test.UnmarshalOscal(t)
			return test
		})
	}
	return l
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
	ID     string                    `json:"id"`
	Name   string                    `json:"name"`
	NS     string                    `json:"ns"`
	Class  string                    `json:"class"`
	Title  string                    `json:"title"`
	Prose  string                    `json:"prose"`
	Props  datatypes.JSONSlice[Prop] `json:"props"`
	Links  datatypes.JSONSlice[Link] `json:"links"`
	PartID string                    `json:"part_id"`
	Parts  []Part                    `json:"parts"` // -> Part

	/**
	"required": [
		"name"
	],
	*/
}

func (l *Part) UnmarshalOscal(data oscalTypes_1_1_3.Part) *Part {
	*l = Part{
		ID:    data.ID,
		Name:  data.Name,
		NS:    data.Ns,
		Class: data.Class,
		Title: data.Title,
		Prose: data.Prose,
		Props: ConvertOscalProps(data.Props),
		Links: ConvertOscalLinks(data.Links),
		Parts: ConvertList(data.Parts, func(data oscalTypes_1_1_3.Part) Part {
			output := Part{}
			output.UnmarshalOscal(data)
			return output
		}),
	}
	return l
}
