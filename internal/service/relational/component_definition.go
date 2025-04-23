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

	ImportComponentDefinitions datatypes.JSONSlice[ImportComponentDefinition] `json:"import-component-definitions"`
	Components                 []DefinedComponent                             `json:"components"`

	//oscaltypes113.ComponentDefinition
}

func (c *ComponentDefinition) UnmarshalOscal(ocd oscalTypes_1_1_3.ComponentDefinition) *ComponentDefinition {
	metadata := &Metadata{}
	metadata.UnmarshalOscal(ocd.Metadata)

	id := uuid.MustParse(ocd.UUID)

	importComponentDefs := ConvertList(ocd.ImportComponentDefinitions, func(oicd oscalTypes_1_1_3.ImportComponentDefinition) ImportComponentDefinition {
		compDef := ImportComponentDefinition{}
		compDef.UnmarshalOscal(oicd)
		return compDef
	})

	components := ConvertList(ocd.Components, func(odc oscalTypes_1_1_3.DefinedComponent) DefinedComponent {
		dc := &DefinedComponent{}
		dc.UnmarshalOscal(odc)
		return *dc
	})

	backMatter := &BackMatter{}
	backMatter.UnmarshalOscal(*ocd.BackMatter)

	*c = ComponentDefinition{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		Metadata:                   *metadata,
		ImportComponentDefinitions: datatypes.NewJSONSlice[ImportComponentDefinition](importComponentDefs),
		Components:                 components,
		BackMatter:                 *backMatter,
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

	// TODO: Convert to a linker table that maps between roles that exist on UUID in the metadata
	ResponsibleRoles       datatypes.JSONSlice[ResponsibleRole] `json:"responsible-roles"`
	ControlImplementations []ControlImplementationSet           `json:"control-implementations"`

	Props     datatypes.JSONSlice[Prop]     `json:"props"`
	Links     datatypes.JSONSlice[Link]     `json:"links"`
	Protocols datatypes.JSONSlice[Protocol] `json:"protocols"`

	ComponentDefinitionID uuid.UUID

	// oscalTypes113.DefinedComponent
}

func (dc *DefinedComponent) UnmarshalOscal(odc oscalTypes_1_1_3.DefinedComponent) *DefinedComponent {
	id := uuid.MustParse(odc.UUID)

	protocols := ConvertList(odc.Protocols, func(op oscalTypes_1_1_3.Protocol) Protocol {
		protocol := Protocol{}
		protocol.UnmarshalOscal(op)
		return protocol
	})

	cis := ConvertList(odc.ControlImplementations, func(ci oscalTypes_1_1_3.ControlImplementationSet) ControlImplementationSet {
		impl := ControlImplementationSet{}
		impl.UnmarshalOscal(ci)
		return impl
	})

	roles := ConvertList(odc.ResponsibleRoles, func(rr oscalTypes_1_1_3.ResponsibleRole) ResponsibleRole {
		r := ResponsibleRole{}
		r.UnmarshalOscal(rr)
		return r
	})

	links := ConvertOscalLinks(odc.Links)
	props := ConvertOscalProps(odc.Props)

	*dc = DefinedComponent{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		Type:                   odc.Type,
		Title:                  odc.Title,
		Purpose:                odc.Purpose,
		Remarks:                odc.Remarks,
		Description:            odc.Description,
		Protocols:              protocols,
		Links:                  links,
		Props:                  props,
		ControlImplementations: cis,
		ResponsibleRoles:       datatypes.NewJSONSlice[ResponsibleRole](roles),
	}
	return dc
}

type Protocol oscalTypes_1_1_3.Protocol

func (p *Protocol) UnmarshalOscal(op oscalTypes_1_1_3.Protocol) *Protocol {
	*p = Protocol(op)
	return p
}

type SetParameter oscalTypes_1_1_3.SetParameter

func (sp *SetParameter) UnmarshalOscal(osp oscalTypes_1_1_3.SetParameter) *SetParameter {
	*sp = SetParameter(osp)
	return sp
}

type ControlImplementationSet struct {
	UUIDModel
	Source        string                            `json:"source"`
	Description   string                            `json:"description"`
	SetParameters datatypes.JSONSlice[SetParameter] `json:"set-parameters"`

	ImplementedRequirements []ImplementedRequirementControlImplementation `json:"implemented-requirements"`

	Props datatypes.JSONSlice[Prop] `json:"props"`
	Links datatypes.JSONSlice[Link] `json:"links"`

	DefinedComponentID uuid.UUID

	/** required: [
		"uuid:,
		"source",
		"description",
		"implemented-requirements"
	]
	*/

	// oscalType_1_1_3.ControlImplementationSet
}

func (ci *ControlImplementationSet) UnmarshalOscal(oci oscalTypes_1_1_3.ControlImplementationSet) *ControlImplementationSet {
	id := uuid.MustParse(oci.UUID)

	setParms := ConvertList(oci.SetParameters, func(osp oscalTypes_1_1_3.SetParameter) SetParameter {
		sp := SetParameter{}
		sp.UnmarshalOscal(osp)
		return sp
	})

	links := ConvertOscalLinks(oci.Links)
	props := ConvertOscalProps(oci.Props)

	implReqs := ConvertList(&oci.ImplementedRequirements, func(oirci oscalTypes_1_1_3.ImplementedRequirementControlImplementation) ImplementedRequirementControlImplementation {
		irci := ImplementedRequirementControlImplementation{}
		irci.UnmarshalOscal(oirci)
		return irci
	})

	*ci = ControlImplementationSet{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		Source:                  oci.Source,
		Description:             oci.Description,
		SetParameters:           setParms,
		Links:                   links,
		Props:                   props,
		ImplementedRequirements: implReqs,
	}
	return ci
}

type ImplementedRequirementControlImplementation struct {
	UUIDModel
	ControlId        string                               `json:"control-id"`
	Description      string                               `json:"description"`
	SetParameters    datatypes.JSONSlice[SetParameter]    `json:"set-parameters"`
	Props            datatypes.JSONSlice[Prop]            `json:"props"`
	Links            datatypes.JSONSlice[Link]            `json:"links"`
	Remarks          string                               `json:"remarks"`
	ResponsibleRoles datatypes.JSONSlice[ResponsibleRole] `json:"responsible-roles"`
	Statements       []ControlStatementImplementation     `json:"statements"`

	ControlImplementationSetID uuid.UUID

	/** Required:
	UUID
	Control-ID
	Description
	*/
	// oscalType_1_1_3.ImplementedRequirementControlImplementation
}

func (irci *ImplementedRequirementControlImplementation) UnmarshalOscal(oirci oscalTypes_1_1_3.ImplementedRequirementControlImplementation) *ImplementedRequirementControlImplementation {
	id := uuid.MustParse(oirci.UUID)

	links := ConvertOscalLinks(oirci.Links)
	props := ConvertOscalProps(oirci.Props)

	setParms := ConvertList(oirci.SetParameters, func(osp oscalTypes_1_1_3.SetParameter) SetParameter {
		sp := SetParameter{}
		sp.UnmarshalOscal(osp)
		return sp
	})

	roles := ConvertList(oirci.ResponsibleRoles, func(rr oscalTypes_1_1_3.ResponsibleRole) ResponsibleRole {
		r := ResponsibleRole{}
		r.UnmarshalOscal(rr)
		return r
	})

	statements := ConvertList(oirci.Statements, func(s oscalTypes_1_1_3.ControlStatementImplementation) ControlStatementImplementation {
		stmt := ControlStatementImplementation{}
		stmt.UnmarshalOscal(s)
		return stmt
	})

	*irci = ImplementedRequirementControlImplementation{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		ControlId:        oirci.ControlId,
		Description:      oirci.Description,
		Links:            links,
		Props:            props,
		Remarks:          oirci.Remarks,
		SetParameters:    setParms,
		ResponsibleRoles: datatypes.NewJSONSlice[ResponsibleRole](roles),
		Statements:       statements,
	}
	return irci
}

type ControlStatementImplementation struct {
	UUIDModel
	StatementId      string                               `json:"statement-id"`
	Description      string                               `json:"description"`
	Props            datatypes.JSONSlice[Prop]            `json:"props"`
	Links            datatypes.JSONSlice[Link]            `json:"links"`
	ResponsibleRoles datatypes.JSONSlice[ResponsibleRole] `json:"responsible-roles"`
	Remarks          string                               `json:"remarks"`

	ImplementedRequirementControlImplementationId uuid.UUID
}

func (s *ControlStatementImplementation) UnmarshalOscal(oci oscalTypes_1_1_3.ControlStatementImplementation) *ControlStatementImplementation {
	id := uuid.MustParse(oci.UUID)
	links := ConvertOscalLinks(oci.Links)
	props := ConvertOscalProps(oci.Props)
	roles := ConvertList(oci.ResponsibleRoles, func(rr oscalTypes_1_1_3.ResponsibleRole) ResponsibleRole {
		r := ResponsibleRole{}
		r.UnmarshalOscal(rr)
		return r
	})

	*s = ControlStatementImplementation{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		StatementId:      oci.StatementId,
		Description:      oci.Description,
		Links:            links,
		Props:            props,
		Remarks:          oci.Remarks,
		ResponsibleRoles: datatypes.NewJSONSlice[ResponsibleRole](roles),
	}

	return s
}

type ResponsibleRole oscalTypes_1_1_3.ResponsibleRole

func (rr *ResponsibleRole) UnmarshalOscal(osc oscalTypes_1_1_3.ResponsibleRole) *ResponsibleRole {
	*rr = ResponsibleRole(osc)
	return rr
}

type ImportComponentDefinition oscalTypes_1_1_3.ImportComponentDefinition

func (icd *ImportComponentDefinition) UnmarshalOscal(oicd oscalTypes_1_1_3.ImportComponentDefinition) *ImportComponentDefinition {
	*icd = ImportComponentDefinition(oicd)
	return icd
}
