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
		Props:     ConvertOscalToProps(data.Props),
		Links:     ConvertOscalToLinks(data.Links),
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
		Props:     ConvertOscalToProps(data.Props),
		Links:     ConvertOscalToLinks(data.Links),
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
		Props:   ConvertOscalToProps(data.Props),
		Links:   ConvertOscalToLinks(data.Links),
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
		Props: ConvertOscalToProps(data.Props),
		Links: ConvertOscalToLinks(data.Links),
		Parts: ConvertList(data.Parts, func(data oscalTypes_1_1_3.Part) Part {
			output := Part{}
			output.UnmarshalOscal(data)
			return output
		}),
	}
	return l
}

func (p *Part) MarshalOscal() *oscalTypes_1_1_3.Part {
	op := &oscalTypes_1_1_3.Part{
		ID:    p.ID,
		Name:  p.Name,
		Ns:    p.NS,
		Class: p.Class,
		Title: p.Title,
		Prose: p.Prose,
		Props: ConvertPropsToOscal(p.Props),
		Links: ConvertLinksToOscal(p.Links),
	}
	if len(p.Parts) > 0 {
		sub := make([]oscalTypes_1_1_3.Part, len(p.Parts))
		for i, sp := range p.Parts {
			sub[i] = *sp.MarshalOscal()
		}
		op.Parts = &sub
	}
	return op
}

//// MarshalOscal converts Catalog to its OSCAL representation.
//func (c *Catalog) MarshalOscal() oscalTypes_1_1_3.Catalog {
//	oc := oscalTypes_1_1_3.Catalog{
//		UUID:     c.ID.String(),
//		Metadata: c.Metadata.MarshalOscal(),
//	}
//	if c.BackMatter != (BackMatter{}) {
//		bm := c.BackMatter.MarshalOscal()
//		oc.BackMatter = &bm
//	}
//	if len(c.Params) > 0 {
//		ps := make([]oscalTypes_1_1_3.Parameter, len(c.Params))
//		for i, p := range c.Params {
//			ps[i] = *p.MarshalOscal()
//		}
//		oc.Params = &ps
//	}
//	if len(c.Groups) > 0 {
//		gs := make([]oscalTypes_1_1_3.Group, len(c.Groups))
//		for i, g := range c.Groups {
//			gs[i] = *g.MarshalOscal()
//		}
//		oc.Groups = &gs
//	}
//	if len(c.Controls) > 0 {
//		cs := make([]oscalTypes_1_1_3.Control, len(c.Controls))
//		for i, ctrl := range c.Controls {
//			cs[i] = *ctrl.MarshalOscal()
//		}
//		oc.Controls = &cs
//	}
//	return oc
//}
//
//// MarshalOscal converts Metadata to OSCAL.
//func (m *Metadata) MarshalOscal() oscalTypes_1_1_3.Metadata {
//	// assuming Metadata has its own MarshalOscal (add if missing)
//	return m.MarshalOscal()
//}
//
//// MarshalOscal converts BackMatter to OSCAL.
//func (b *BackMatter) MarshalOscal() oscalTypes_1_1_3.BackMatter {
//	// assuming BackMatter has its own MarshalOscal (add if missing)
//	return b.MarshalOscal()
//}
//
//// MarshalOscal converts Group to OSCAL.
//func (g *Group) MarshalOscal() *oscalTypes_1_1_3.Group {
//	og := &oscalTypes_1_1_3.Group{
//		ID:    g.ID,
//		Title: g.Title,
//		Class: g.Class,
//		Props: ConvertPropsToOscal(g.Props),
//		Links: ConvertLinksToOscal(g.Links),
//	}
//	if len(g.Params) > 0 {
//		ps := make([]oscalTypes_1_1_3.Parameter, len(g.Params))
//		for i, p := range g.Params {
//			ps[i] = *p.MarshalOscal()
//		}
//		og.Params = &ps
//	}
//	if len(g.Parts) > 0 {
//		parts := make([]oscalTypes_1_1_3.Part, len(g.Parts))
//		for i, pt := range g.Parts {
//			parts[i] = *pt.MarshalOscal()
//		}
//		og.Parts = &parts
//	}
//	if len(g.Groups) > 0 {
//		subgroups := make([]oscalTypes_1_1_3.Group, len(g.Groups))
//		for i, sg := range g.Groups {
//			subgroups[i] = *sg.MarshalOscal()
//		}
//		og.Groups = &subgroups
//	}
//	if len(g.Controls) > 0 {
//		ctrls := make([]oscalTypes_1_1_3.Control, len(g.Controls))
//		for i, c := range g.Controls {
//			ctrls[i] = *c.MarshalOscal()
//		}
//		og.Controls = &ctrls
//	}
//	return og
//}
//
//// MarshalOscal converts Control to OSCAL.
//func (c *Control) MarshalOscal() *oscalTypes_1_1_3.Control {
//	oc := &oscalTypes_1_1_3.Control{
//		ID:    c.ID,
//		Title: c.Title,
//		Class: derefString(c.Class),
//		Props: ConvertPropsToOscal(c.Props),
//		Links: ConvertLinksToOscal(c.Links),
//	}
//	if len(c.Params) > 0 {
//		ps := make([]oscalTypes_1_1_3.Parameter, len(c.Params))
//		for i, p := range c.Params {
//			ps[i] = *p.MarshalOscal()
//		}
//		oc.Params = &ps
//	}
//	if len(c.Parts) > 0 {
//		pts := make([]oscalTypes_1_1_3.Part, len(c.Parts))
//		for i, pt := range c.Parts {
//			pts[i] = *pt.MarshalOscal()
//		}
//		oc.Parts = &pts
//	}
//	if len(c.Controls) > 0 {
//		ctrls := make([]oscalTypes_1_1_3.Control, len(c.Controls))
//		for i, cl := range c.Controls {
//			ctrls[i] = *cl.MarshalOscal()
//		}
//		oc.Controls = &ctrls
//	}
//	return oc
//}
//
//// MarshalOscal converts Parameter to OSCAL.
//func (p *Parameter) MarshalOscal() *oscalTypes_1_1_3.Parameter {
//	op := &oscalTypes_1_1_3.Parameter{
//		ID:      p.ID,
//		Class:   derefString(p.Class),
//		Label:   derefString(p.Label),
//		Usage:   derefString(p.Usage),
//		Remarks: derefString(p.Remarks),
//		Props:   ConvertPropsToOscal(p.Props),
//		Links:   ConvertLinksToOscal(p.Links),
//	}
//	if p.Select.Valid {
//		sel := p.Select.Value
//		op.Select = &oscalTypes_1_1_3.ParameterSelection{
//			HowMany: string(sel.HowMany),
//			Choice:  sel.Choice,
//		}
//	}
//	if len(p.Values) > 0 {
//		op.Values = &p.Values
//	}
//	if len(p.Constraints) > 0 {
//		cs := make([]oscalTypes_1_1_3.ParameterConstraint, len(p.Constraints))
//		for i, c := range p.Constraints {
//			cs[i] = *c.MarshalOscal()
//		}
//		op.Constraints = &cs
//	}
//	if len(p.Guidelines) > 0 {
//		gs := make([]oscalTypes_1_1_3.ParameterGuideline, len(p.Guidelines))
//		for i, g := range p.Guidelines {
//			gs[i] = *g.MarshalOscal()
//		}
//		op.Guidelines = &gs
//	}
//	return op
//}
//
//// MarshalOscal converts ParameterSelection to OSCAL.
//func (p *ParameterSelection) MarshalOscal() *oscalTypes_1_1_3.ParameterSelection {
//	return &oscalTypes_1_1_3.ParameterSelection{
//		HowMany: string(p.HowMany),
//		Choice:  p.Choice,
//	}
//}
//
//// MarshalOscal converts ParameterGuideline to OSCAL.
//func (g *ParameterGuideline) MarshalOscal() *oscalTypes_1_1_3.ParameterGuideline {
//	pg := oscalTypes_1_1_3.ParameterGuideline(g.Prose)
//	return &pg
//}
//
//// MarshalOscal converts ParameterConstraint to OSCAL.
//func (c *ParameterConstraint) MarshalOscal() *oscalTypes_1_1_3.ParameterConstraint {
//	pc := &oscalTypes_1_1_3.ParameterConstraint{
//		Description: c.Description,
//	}
//	if len(c.Tests) > 0 {
//		ts := make([]oscalTypes_1_1_3.ConstraintTest, len(c.Tests))
//		for i, t := range c.Tests {
//			ts[i] = *t.MarshalOscal()
//		}
//		pc.Tests = &ts
//	}
//	return pc
//}
//
//// MarshalOscal converts ParameterConstraintTest to OSCAL.
//func (t *ParameterConstraintTest) MarshalOscal() *oscalTypes_1_1_3.ConstraintTest {
//	ct := oscalTypes_1_1_3.ConstraintTest(*t)
//	return &ct
//}

// helper to dereference *string safely
func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
