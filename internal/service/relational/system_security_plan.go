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

	SystemIds         datatypes.JSONSlice[SystemId] `json:"system-ids"`
	SystemInformation SystemInformation             `json:"system-information"`

	Links datatypes.JSONSlice[Link] `json:"links"`
	Props datatypes.JSONSlice[Prop] `json:"props"`

	// SystemInformation
	// SecurityImpactLevel
	// Status
	// AuthorizationBoundary
	// Network Architecture
	// DataFlow
	// ResponsibleParties

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
		description:           oit.Description,
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
