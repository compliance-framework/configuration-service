package relational

import (
	"database/sql"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"time"
)

type SystemSecurityPlan struct {
	UUIDModel
	Metadata   Metadata   `json:"metadata" gorm:"polymorphic:Parent;"`
	BackMatter BackMatter `json:"back-matter" gorm:"polymorphic:Parent;"`

	ImportProfile         datatypes.JSONType[ImportProfile] `json:"import-profile"`
	SystemCharacteristics SystemCharacteristics             `json:"system-characteristics"`
}

func (s *SystemSecurityPlan) UnmarshalOscal(os oscalTypes_1_1_3.SystemSecurityPlan) *SystemSecurityPlan {
	id := uuid.MustParse(os.UUID)
	metadata := Metadata{}
	metadata.UnmarshalOscal(os.Metadata)

	backMatter := BackMatter{}
	backMatter.UnmarshalOscal(*os.BackMatter)

	importProfile := ImportProfile{}
	importProfile.UnmarshalOscal(os.ImportProfile)

	systemCharacteristics := SystemCharacteristics{}
	systemCharacteristics.UnmarshalOscal(os.SystemCharacteristics)

	*s = SystemSecurityPlan{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		Metadata:              metadata,
		BackMatter:            backMatter,
		ImportProfile:         datatypes.NewJSONType[ImportProfile](importProfile),
		SystemCharacteristics: systemCharacteristics,
	}

	return s
}

type ImportProfile oscalTypes_1_1_3.ImportProfile

func (ip *ImportProfile) UnmarshalOscal(oip oscalTypes_1_1_3.ImportProfile) *ImportProfile {
	*ip = ImportProfile(oip)
	return ip
}

type SystemCharacteristics struct {
	UUIDModel
	SystemName               string       `json:"system-name"`
	SystemNameShort          string       `json:"system-name-short"`
	Description              string       `json:"description"`
	DateAuthorized           sql.NullTime `json:"date-authorized"`
	SecuritySensitivityLevel string       `json:"security-sensitivity-level"`
	Remarks                  string       `json:"remarks"`

	SystemIds            datatypes.JSONSlice[SystemId] `json:"system-ids"`
	SystemInformation    SystemInformation             `json:"system-information"`
	Status               datatypes.JSONType[Status]    `json:"status"`
	AuthorizationBoundry AuthorizationBoundary         `json:"authorization-boundary"`
	NetworkArchitecture  NetworkArchitecture           `json:"network-architecture"`
	DataFlow             DataFlow                      `json:"data-flow"`

	Links              datatypes.JSONSlice[Link]             `json:"links"`
	Props              datatypes.JSONSlice[Prop]             `json:"props"`
	ResponsibleParties datatypes.JSONSlice[ResponsibleParty] `json:"responsible-role"`

	SystemSecurityPlanId uuid.UUID
}

func (sc *SystemCharacteristics) UnmarshalOscal(osc oscalTypes_1_1_3.SystemCharacteristics) *SystemCharacteristics {
	props := ConvertOscalProps(osc.Props)
	links := ConvertOscalLinks(osc.Links)

	dateAuthorized := sql.NullTime{}
	if len(osc.DateAuthorized) > 0 {
		// Todo: Assume that a timezone is attached as per the spec - https://pages.nist.gov/metaschema/specification/datatypes/#date
		parsed, err := time.Parse(time.DateOnly, osc.DateAuthorized)
		if err != nil {
			panic(err)
		}
		dateAuthorized = sql.NullTime{
			Time:  parsed,
			Valid: true,
		}

	}

	systemIds := ConvertList(&osc.SystemIds, func(osi oscalTypes_1_1_3.SystemId) SystemId {
		sid := SystemId(osi)
		return sid
	})

	systemInformation := SystemInformation{}
	systemInformation.UnmarshalOscal(osc.SystemInformation)

	status := Status{}
	status.UnmarshalOscal(osc.Status)

	authBoundary := AuthorizationBoundary{}
	authBoundary.UnmarshalOscal(osc.AuthorizationBoundary)

	var networkArchitecture NetworkArchitecture
	if osc.NetworkArchitecture != nil {
		networkArchitecture = NetworkArchitecture{}
		networkArchitecture.UnmarshalOscal(*osc.NetworkArchitecture)
	}

	var dataflow DataFlow
	if osc.DataFlow != nil {
		dataflow = DataFlow{}
		dataflow.UnmarshalOscal(*osc.DataFlow)
	}

	responsibleParties := ConvertList(osc.ResponsibleParties, func(orp oscalTypes_1_1_3.ResponsibleParty) ResponsibleParty {
		rp := ResponsibleParty{}
		rp.UnmarshalOscal(orp)
		return rp
	})

	*sc = SystemCharacteristics{
		UUIDModel:                UUIDModel{},
		SystemName:               osc.SystemName,
		SystemNameShort:          osc.SystemNameShort,
		Description:              osc.Description,
		DateAuthorized:           dateAuthorized,
		SecuritySensitivityLevel: osc.SecuritySensitivityLevel,
		Remarks:                  osc.Remarks,
		Links:                    links,
		Props:                    props,
		SystemIds:                systemIds,
		SystemInformation:        systemInformation,
		Status:                   datatypes.NewJSONType[Status](status),
		AuthorizationBoundry:     authBoundary,
		NetworkArchitecture:      networkArchitecture,
		DataFlow:                 dataflow,
		ResponsibleParties:       datatypes.NewJSONSlice[ResponsibleParty](responsibleParties),
	}

	return sc
}

type SystemId oscalTypes_1_1_3.SystemId

func (si *SystemId) UnmarshalOscal(osi oscalTypes_1_1_3.SystemId) *SystemId {
	*si = SystemId(osi)
	return si
}

type SystemInformation struct {
	UUIDModel
	Props datatypes.JSONSlice[Prop] `json:"props"`
	Links datatypes.JSONSlice[Link] `json:"links"`

	InformationTypes []InformationType `json:"information-types"`

	SystemCharacteristicsId uuid.UUID
}

func (si *SystemInformation) UnmarshalOscal(osi oscalTypes_1_1_3.SystemInformation) *SystemInformation {
	props := ConvertOscalProps(osi.Props)
	links := ConvertOscalLinks(osi.Links)

	informationTypes := ConvertList(&osi.InformationTypes, func(oit oscalTypes_1_1_3.InformationType) InformationType {
		informationType := InformationType{}
		informationType.UnmarshalOscal(oit)
		return informationType
	})

	*si = SystemInformation{
		UUIDModel:        UUIDModel{},
		Props:            props,
		Links:            links,
		InformationTypes: informationTypes,
	}

	return si
}

type InformationType struct {
	UUIDModel
	Title       string `json:"title"`
	Description string `json:"description"`

	Props datatypes.JSONSlice[Prop] `json:"props"`
	Links datatypes.JSONSlice[Link] `json:"links"`

	ConfidentialityImpact datatypes.JSONType[Impact] `json:"confidentiality-impact"`
	IntegrityImpact       datatypes.JSONType[Impact] `json:"integrity-impact"`
	AvailabilityImpact    datatypes.JSONType[Impact] `json:"availability-impact"`

	Categorizations datatypes.JSONSlice[InformationTypeCategorization] `json:"categorizations"`

	SystemInformationId uuid.UUID
}

func (it *InformationType) UnmarshalOscal(oit oscalTypes_1_1_3.InformationType) *InformationType {
	id := uuid.MustParse(oit.UUID)

	props := ConvertOscalProps(oit.Props)
	links := ConvertOscalLinks(oit.Links)

	confImpact := Impact{}
	confImpact.UnmarshalOscal(*oit.ConfidentialityImpact)

	integrityImpact := Impact{}
	integrityImpact.UnmarshalOscal(*oit.IntegrityImpact)

	availabilityImpact := Impact{}
	availabilityImpact.UnmarshalOscal(*oit.AvailabilityImpact)

	categorizations := ConvertList(oit.Categorizations, func(oitc oscalTypes_1_1_3.InformationTypeCategorization) InformationTypeCategorization {
		category := InformationTypeCategorization{}
		category.UnmarshalOscal(oitc)
		return category
	})

	*it = InformationType{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		Title:                 oit.Title,
		Description:           oit.Description,
		Props:                 props,
		Links:                 links,
		ConfidentialityImpact: datatypes.NewJSONType[Impact](confImpact),
		IntegrityImpact:       datatypes.NewJSONType[Impact](integrityImpact),
		AvailabilityImpact:    datatypes.NewJSONType[Impact](availabilityImpact),
		Categorizations:       categorizations,
	}

	return it
}

type Impact oscalTypes_1_1_3.Impact

func (i *Impact) UnmarshalOscal(osi oscalTypes_1_1_3.Impact) *Impact {
	*i = Impact(osi)
	return i
}

type InformationTypeCategorization oscalTypes_1_1_3.InformationTypeCategorization

func (itc *InformationTypeCategorization) UnmarshalOscal(oitc oscalTypes_1_1_3.InformationTypeCategorization) *InformationTypeCategorization {
	*itc = InformationTypeCategorization(oitc)
	return itc
}

type SecurityImpactLevel oscalTypes_1_1_3.SecurityImpactLevel

type Status oscalTypes_1_1_3.Status

func (s *Status) UnmarshalOscal(osi oscalTypes_1_1_3.Status) *Status {
	*s = Status(osi)
	return s
}

type AuthorizationBoundary struct {
	UUIDModel
	Description string                    `json:"description"`
	Remarks     string                    `json:"remarks"`
	Props       datatypes.JSONSlice[Prop] `json:"props"`
	Links       datatypes.JSONSlice[Link] `json:"links"`
	Diagrams    []Diagram                 `json:"diagrams" gorm:"many2many:authorization_boundary_diagrams;"`

	SystemCharacteristicsId uuid.UUID
}

func (ab *AuthorizationBoundary) UnmarshalOscal(oab oscalTypes_1_1_3.AuthorizationBoundary) *AuthorizationBoundary {
	links := ConvertOscalLinks(oab.Links)
	props := ConvertOscalProps(oab.Props)

	diagrams := ConvertList(oab.Diagrams, func(od oscalTypes_1_1_3.Diagram) Diagram {
		diagram := Diagram{}
		diagram.UnmarshalOscal(od)
		return diagram
	})

	*ab = AuthorizationBoundary{
		UUIDModel:   UUIDModel{},
		Description: oab.Description,
		Remarks:     oab.Remarks,
		Props:       props,
		Links:       links,
		Diagrams:    diagrams,
	}
	return ab
}

type NetworkArchitecture struct {
	UUIDModel
	Description string                    `json:"description"`
	Props       datatypes.JSONSlice[Prop] `json:"props"`
	Links       datatypes.JSONSlice[Link] `json:"links"`
	Remarks     string                    `json:"remarks"`
	Diagrams    []Diagram                 `json:"diagrams" gorm:"many2many:network_architecture_diagrams;"`

	SystemCharacteristicsId uuid.UUID
}

func (na *NetworkArchitecture) UnmarshalOscal(ona oscalTypes_1_1_3.NetworkArchitecture) *NetworkArchitecture {
	props := ConvertOscalProps(ona.Props)
	links := ConvertOscalLinks(ona.Links)

	diagrams := ConvertList(ona.Diagrams, func(od oscalTypes_1_1_3.Diagram) Diagram {
		diagram := Diagram{}
		diagram.UnmarshalOscal(od)
		return diagram
	})

	*na = NetworkArchitecture{
		UUIDModel:   UUIDModel{},
		Description: ona.Description,
		Props:       props,
		Links:       links,
		Remarks:     ona.Remarks,
		Diagrams:    diagrams,
	}
	return na
}

type DataFlow struct {
	UUIDModel
	Description string                    `json:"description"`
	Props       datatypes.JSONSlice[Prop] `json:"props"`
	Links       datatypes.JSONSlice[Link] `json:"links"`
	Remarks     string                    `json:"remarks"`
	Diagrams    []Diagram                 `json:"diagrams" gorm:"many2many:data_flow_diagrams;"`

	SystemCharacteristicsId uuid.UUID
}

func (df *DataFlow) UnmarshalOscal(odf oscalTypes_1_1_3.DataFlow) *DataFlow {
	props := ConvertOscalProps(odf.Props)
	links := ConvertOscalLinks(odf.Links)

	diagrams := ConvertList(odf.Diagrams, func(od oscalTypes_1_1_3.Diagram) Diagram {
		diagram := Diagram{}
		diagram.UnmarshalOscal(od)
		return diagram
	})

	*df = DataFlow{
		UUIDModel:   UUIDModel{},
		Description: odf.Description,
		Props:       props,
		Links:       links,
		Remarks:     odf.Remarks,
		Diagrams:    diagrams,
	}

	return df
}

type Diagram struct {
	UUIDModel
	Description string                    `json:"description"`
	Props       datatypes.JSONSlice[Prop] `json:"props"`
	Links       datatypes.JSONSlice[Link] `json:"links"`
	Caption     string                    `json:"caption"`
	Remarks     string                    `json:"remarks"`
}

func (d *Diagram) UnmarshalOscal(od oscalTypes_1_1_3.Diagram) *Diagram {
	id := uuid.MustParse(od.UUID)
	links := ConvertOscalLinks(od.Links)
	props := ConvertOscalProps(od.Props)

	*d = Diagram{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		Description: od.Description,
		Props:       props,
		Links:       links,
		Caption:     od.Caption,
		Remarks:     od.Remarks,
	}

	return d
}
