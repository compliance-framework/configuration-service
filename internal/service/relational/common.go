package relational

import (
	"database/sql"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

type Prop oscaltypes113.Property

func (p *Prop) UnmarshalOscal(data oscaltypes113.Property) *Prop {
	*p = Prop(data)
	return p
}

type Link oscaltypes113.Link

func (l *Link) UnmarshalOscal(data oscaltypes113.Link) *Link {
	*l = Link(data)
	return l
}

type Metadata struct {
	UUIDModel
	Title              string                          `json:"title"`
	Published          sql.NullTime                    `json:"published"`
	LastModified       time.Time                       `json:"last-modified"`
	Version            string                          `json:"version"`
	OscalVersion       string                          `json:"oscal-version"`
	DocumentIDs        datatypes.JSONSlice[DocumentID] `json:"document-ids"` // -> DocumentID
	Props              datatypes.JSONSlice[Prop]       `json:"props"`
	Links              datatypes.JSONSlice[Link]       `json:"links"`
	Revisions          []Revision                      `json:"revisions" gorm:"polymorphic:Parent;"`
	Roles              []Role                          `gorm:"many2many:metadata_roles;"`
	Locations          []Location                      `gorm:"many2many:metadata_locations;"`
	Parties            []Party                         `gorm:"many2many:metadata_parties;"`
	ResponsibleParties []ResponsibleParty              `gorm:"many2many:metadata_responsible_parties;"`
	Actions            []Action                        `json:"actions"` // -> Action
	Remarks            string                          `json:"remarks"`

	// Metadata is shared across many resources, and so it mapped using a polymorphic relationship
	ParentID   *string
	ParentType *string

	/**
	"required": [
		"title",
		"last-modified",
		"version",
		"oscal-version"
	],
	*/
}

func (m *Metadata) UnmarshalOscal(metadata oscaltypes113.Metadata) *Metadata {
	published := sql.NullTime{}
	if metadata.Published != nil {
		published = sql.NullTime{
			Time: *metadata.Published,
		}
	}
	*m = Metadata{
		Title:        metadata.Title,
		Published:    published,
		LastModified: metadata.LastModified,
		Version:      metadata.Version,
		OscalVersion: metadata.OscalVersion,
		DocumentIDs: func() datatypes.JSONSlice[DocumentID] {
			list := make([]DocumentID, 0)
			if metadata.DocumentIds != nil {
				for _, document := range *metadata.DocumentIds {
					doc := &DocumentID{}
					doc.FromOscal(document)
					list = append(list, *doc)
				}
			}
			return datatypes.NewJSONSlice[DocumentID](list)
		}(),
		Props: ConvertList(metadata.Props, func(property oscaltypes113.Property) Prop {
			prop := Prop{}
			prop.UnmarshalOscal(property)
			return prop
		}),
		Links: ConvertList(metadata.Links, func(olink oscaltypes113.Link) Link {
			link := Link{}
			link.UnmarshalOscal(olink)
			return link
		}),
		Revisions: ConvertList(metadata.Revisions, func(entry oscaltypes113.RevisionHistoryEntry) Revision {
			revision := Revision{}
			revision.UnmarshalOscal(entry)
			return revision
		}),
		Roles: ConvertList(metadata.Roles, func(orole oscaltypes113.Role) Role {
			role := Role{}
			role.UnmarshalOscal(orole)
			return role
		}),
		Locations: ConvertList(metadata.Locations, func(oloc oscaltypes113.Location) Location {
			location := Location{}
			location.UnmarshalOscal(oloc)
			return location
		}),
		Parties: ConvertList(metadata.Parties, func(oparty oscaltypes113.Party) Party {
			party := Party{}
			party.UnmarshalOscal(oparty)
			return party
		}),
		ResponsibleParties: ConvertList(metadata.ResponsibleParties, func(oparty oscaltypes113.ResponsibleParty) ResponsibleParty {
			party := ResponsibleParty{}
			party.UnmarshalOscal(oparty)
			return party
		}),
		Actions: ConvertList(metadata.Actions, func(oaction oscaltypes113.Action) Action {
			action := Action{}
			action.UnmarshalOscal(oaction)
			return action
		}),
		Remarks: metadata.Remarks,
	}
	return m
}

type Citation struct {
	Text  string `json:"text"` // required
	Props []Prop `json:"props"`
	Links []Link `json:"links"`
}

type HashAlgorithm string

const (
	HashAlgorithmSHA_224  HashAlgorithm = "SHA-224"
	HashAlgorithmSHA_256  HashAlgorithm = "SHA-256"
	HashAlgorithmSHA_384  HashAlgorithm = "SHA-384"
	HashAlgorithmSHA_512  HashAlgorithm = "SHA-512"
	HashAlgorithmSHA3_224 HashAlgorithm = "SHA3-224"
	HashAlgorithmSHA3_256 HashAlgorithm = "SHA3-256"
	HashAlgorithmSHA3_384 HashAlgorithm = "SHA3-384"
	HashAlgorithmSHA3_512 HashAlgorithm = "SHA3-512"
)

type Hash struct {
	Algorithm HashAlgorithm `json:"algorithm"` // required
	Value     string        `json:"value"`     // required
}

type RLink struct {
	Href      string `json:"href"` // required
	MediaType string `json:"media-type"`
	Hashes    []Hash `json:"hashes"`
}

type Base64 struct {
	Filename  string `json:"filename"`
	MediaType string `json:"media-type"`
	Value     string `json:"value"` // required
}

type DocumentIDScheme string

const DocumentIDSchemeDoi DocumentIDScheme = "http://www.doi.org/"

type DocumentID struct {
	Scheme     DocumentIDScheme `json:"scheme"`
	Identifier string           `json:"identifier"`
}

func (d *DocumentID) FromOscal(id oscaltypes113.DocumentId) {
	d.Scheme = DocumentIDScheme(id.Scheme)
	d.Identifier = id.Identifier
}

type BackMatter struct {
	UUIDModel
	Resources  []BackMatterResource `json:"resources"`
	ParentID   *string
	ParentType *string
}

type BackMatterResource struct {
	UUIDModel                                    // required
	BackMatterID uuid.UUID                       `json:"back-matter-id"`
	Title        *string                         `json:"title"`
	Description  *string                         `json:"description"`
	Props        datatypes.JSONSlice[Prop]       `json:"props"`
	DocumentIDs  datatypes.JSONSlice[DocumentID] `json:"document-ids"`
	Citations    datatypes.JSONType[Citation]    `json:"citation"`
	RLinks       datatypes.JSONSlice[RLink]      `json:"rlinks"`
	Base64       datatypes.JSONType[Base64]      `json:"base64"`
	Remarks      *string                         `json:"remarks"`
}

type Revision struct {
	// Only version is required
	UUIDModel
	Title        *string                   `json:"title"`
	Published    sql.NullTime              `json:"published"`
	LastModified sql.NullTime              `json:"last-modified"`
	Version      string                    `json:"version"` // required
	OscalVersion *string                   `json:"oscal-version"`
	Props        datatypes.JSONSlice[Prop] `json:"props"`
	Links        datatypes.JSONSlice[Link] `json:"links"`
	Remarks      *string                   `json:"remarks"`
	ParentID     *uuid.UUID
	ParentType   *string
}

func (r *Revision) UnmarshalOscal(entry oscaltypes113.RevisionHistoryEntry) *Revision {
	if entry.Published != nil {
		r.Published = sql.NullTime{Time: *entry.Published}
	}
	if entry.LastModified != nil {
		r.LastModified = sql.NullTime{Time: *entry.LastModified}
	}
	r.Version = entry.Version
	r.OscalVersion = &entry.OscalVersion
	r.Title = &entry.Title
	r.Remarks = &entry.Remarks
	r.Props = ConvertList(entry.Props, func(property oscaltypes113.Property) Prop {
		prop := Prop{}
		prop.UnmarshalOscal(property)
		return prop
	})
	r.Links = ConvertList(entry.Links, func(olink oscaltypes113.Link) Link {
		link := Link{}
		link.UnmarshalOscal(olink)
		return link
	})

	return r
}

type Role struct {
	ID          string                    `json:"id" gorm:"primary_key;"`
	Title       string                    `json:"title"`
	ShortName   *string                   `json:"short-name"`
	Description *string                   `json:"description"`
	Props       datatypes.JSONSlice[Prop] `json:"props"`
	Links       datatypes.JSONSlice[Link] `json:"links"`
	Remarks     *string                   `json:"remarks"`
}

func (r *Role) UnmarshalOscal(entry oscaltypes113.Role) *Role {
	r.ID = entry.ID
	r.Title = entry.Title
	r.ShortName = &entry.ShortName
	r.Description = &entry.Description
	r.Remarks = &entry.Remarks
	r.Props = ConvertList(entry.Props, func(property oscaltypes113.Property) Prop {
		prop := Prop{}
		prop.UnmarshalOscal(property)
		return prop
	})
	r.Links = ConvertList(entry.Links, func(olink oscaltypes113.Link) Link {
		link := Link{}
		link.UnmarshalOscal(olink)
		return link
	})

	return r
}

type AddressType string

const AddressTypeWork AddressType = "work"
const AddressTypeHome AddressType = "home"

type Address struct {
	Type       AddressType `json:"type"`
	AddrLines  []string    `json:"lines"`
	City       string      `json:"city"`
	State      string      `json:"state"`
	PostalCode string      `json:"postal-code"`
	Country    string      `json:"country"`
}

func (a *Address) UnmarshalOscal(oaddress oscaltypes113.Address) *Address {
	*a = Address{
		Type:       AddressType(oaddress.Type),
		AddrLines:  *oaddress.AddrLines,
		City:       oaddress.City,
		State:      oaddress.State,
		PostalCode: oaddress.PostalCode,
		Country:    oaddress.Country,
	}
	return a
}

type TelephoneNumberType string

const TelephoneNumberTypeHome TelephoneNumberType = "home"
const TelephoneNumberTypeOffice TelephoneNumberType = "office"
const TelephoneNumberTypeMobile TelephoneNumberType = "mobile"

type TelephoneNumber struct {
	Type   *TelephoneNumberType `json:"type"`
	Number string               `json:"number"`

	/**
	"required": [
		"number"
	],
	*/
}

func (t *TelephoneNumber) UnmarshalOscal(number oscaltypes113.TelephoneNumber) *TelephoneNumber {
	ntype := TelephoneNumberType(number.Type)
	*t = TelephoneNumber{
		Type:   &ntype,
		Number: number.Number,
	}
	return t
}

type PartyType string

const PartyTypePerson PartyType = "person"
const PartyTypeOrganization PartyType = "organization"

type PartyExternalIDScheme string

const PartyExternalIDSchemeOrchid PartyExternalIDScheme = "http://orcid.org/"

type PartyExternalID struct {
	ID     string                `json:"id"`
	Scheme PartyExternalIDScheme `json:"scheme"`

	/**
	"required": [
		"id",
		"scheme"
	],
	*/
}

func (p *PartyExternalID) UnmarshalOscal(oid oscaltypes113.PartyExternalIdentifier) *PartyExternalID {
	p.ID = oid.ID
	p.Scheme = PartyExternalIDScheme(oid.Scheme)

	return p
}

type Party struct {
	UUIDModel
	MetadataID            uuid.UUID
	Type                  PartyType                            `json:"type"`
	Name                  *string                              `json:"name"`
	ShortName             *string                              `json:"short-name"`
	ExternalIds           datatypes.JSONSlice[PartyExternalID] `json:"external-ids"`
	Props                 datatypes.JSONSlice[Prop]            `json:"props"`
	Links                 datatypes.JSONSlice[Link]            `json:"links"`
	EmailAddresses        datatypes.JSONSlice[string]          `json:"email-addresses"`
	TelephoneNumbers      datatypes.JSONSlice[TelephoneNumber] `json:"telephone-numbers"`
	Addresses             datatypes.JSONSlice[Address]         `json:"addresses"`
	Locations             []Location                           `json:"locations" gorm:"many2many:party_locations;"`
	MemberOfOrganizations []Party                              `json:"member-of-organizations" gorm:"many2many:party_member_of_organisations;"` // -> Party
	Remarks               *string                              `json:"remarks"`

	/**
	"required": [
		"uuid",
		"type"
	],
	*/
}

func (p *Party) UnmarshalOscal(oparty oscaltypes113.Party) *Party {
	*p = Party{
		UUIDModel: UUIDModel{
			ID: func() *uuid.UUID {
				id := uuid.MustParse(oparty.UUID)
				return &id
			}(),
		},
		Type:      PartyType(oparty.Type),
		Name:      &oparty.Name,
		ShortName: &oparty.Name,
		ExternalIds: ConvertList(oparty.ExternalIds, func(id oscaltypes113.PartyExternalIdentifier) PartyExternalID {
			pei := PartyExternalID{}
			pei.UnmarshalOscal(id)
			return pei
		}),
		Props: ConvertList(oparty.Props, func(property oscaltypes113.Property) Prop {
			prop := Prop{}
			prop.UnmarshalOscal(property)
			return prop
		}),
		Links: ConvertList(oparty.Links, func(olink oscaltypes113.Link) Link {
			link := Link{}
			link.UnmarshalOscal(olink)
			return link
		}),
		EmailAddresses:   nil,
		TelephoneNumbers: nil,
		Addresses: ConvertList(oparty.Addresses, func(oaddress oscaltypes113.Address) Address {
			address := Address{}
			address.UnmarshalOscal(oaddress)
			return address
		}),
		Locations:             nil,
		MemberOfOrganizations: nil,
		Remarks:               &oparty.Remarks,
	}

	return p
}

type ResponsibleParty struct {
	UUIDModel
	MetadataID uuid.UUID
	Props      datatypes.JSONSlice[Prop] `json:"props"`
	Links      datatypes.JSONSlice[Link] `json:"links"`
	Remarks    string                    `json:"remarks"`

	RoleID  string `json:"role-id"` // required
	Role    Role
	Parties []Party `gorm:"many2many:responsible_party_parties;"`
}

func (r *ResponsibleParty) UnmarshalOscal(or oscaltypes113.ResponsibleParty) *ResponsibleParty {
	*r = ResponsibleParty{
		Props: ConvertList(or.Props, func(property oscaltypes113.Property) Prop {
			prop := Prop{}
			prop.UnmarshalOscal(property)
			return prop
		}),
		Links: ConvertList(or.Links, func(olink oscaltypes113.Link) Link {
			link := Link{}
			link.UnmarshalOscal(olink)
			return link
		}),
		Remarks: or.Remarks,
		RoleID:  or.RoleId,
		Parties: ConvertList(&or.PartyUuids, func(olink string) Party {
			id := uuid.MustParse(olink)
			return Party{
				UUIDModel: UUIDModel{
					ID: &id,
				},
			}
		}),
	}

	return r
}

type Action struct {
	UUIDModel
	MetadataID         uuid.UUID                 // required
	Date               sql.NullTime              `json:"date"`
	Type               string                    `json:"type"`   // required
	System             string                    `json:"system"` // required
	Props              datatypes.JSONSlice[Prop] `json:"props"`
	Links              datatypes.JSONSlice[Link] `json:"links"`
	ResponsibleParties []ResponsibleParty        `gorm:"many2many:action_responsible_party;"`
	Remarks            string                    `json:"remarks"`
}

func (a *Action) UnmarshalOscal(action oscaltypes113.Action) *Action {
	date := sql.NullTime{}
	if action.Date != nil {
		date = sql.NullTime{
			Time: *action.Date,
		}
	}
	*a = Action{
		UUIDModel: UUIDModel{
			ID: func() *uuid.UUID {
				id := uuid.MustParse(action.UUID)
				return &id
			}(),
		},
		Date:   date,
		Type:   action.Type,
		System: action.System,
		Props: ConvertList(action.Props, func(property oscaltypes113.Property) Prop {
			prop := Prop{}
			prop.UnmarshalOscal(property)
			return prop
		}),
		Links: ConvertList(action.Links, func(olink oscaltypes113.Link) Link {
			link := Link{}
			link.UnmarshalOscal(olink)
			return link
		}),
		ResponsibleParties: ConvertList(action.ResponsibleParties, func(oparty oscaltypes113.ResponsibleParty) ResponsibleParty {
			party := ResponsibleParty{}
			party.UnmarshalOscal(oparty)
			return party
		}),
		Remarks: action.Remarks,
	}

	return a
}

type Location struct {
	UUIDModel
	MetadataID       uuid.UUID
	Title            *string                              `json:"title"`
	Address          datatypes.JSONType[Address]          `json:"address"`
	EmailAddresses   datatypes.JSONSlice[string]          `json:"email-addresses"`
	TelephoneNumbers datatypes.JSONSlice[TelephoneNumber] `json:"telephone-numbers"`
	Urls             datatypes.JSONSlice[string]          `json:"urls"`
	Props            datatypes.JSONSlice[Prop]            `json:"props"`
	Links            datatypes.JSONSlice[Link]            `json:"links"`
	Remarks          *string                              `json:"remarks"`
	/**
	"required": [
		"uuid"
	],
	*/
}

func (l *Location) UnmarshalOscal(olocation oscaltypes113.Location) *Location {
	*l = Location{
		UUIDModel: UUIDModel{
			ID: func() *uuid.UUID {
				id := uuid.MustParse(olocation.UUID)
				return &id
			}(),
		},
		Props: ConvertList(olocation.Props, func(property oscaltypes113.Property) Prop {
			prop := Prop{}
			prop.UnmarshalOscal(property)
			return prop
		}),
		Links: ConvertList(olocation.Links, func(olink oscaltypes113.Link) Link {
			link := Link{}
			link.UnmarshalOscal(olink)
			return link
		}),
		EmailAddresses: *olocation.EmailAddresses,
		TelephoneNumbers: ConvertList(olocation.TelephoneNumbers, func(onumb oscaltypes113.TelephoneNumber) TelephoneNumber {
			numb := TelephoneNumber{}
			numb.UnmarshalOscal(onumb)
			return numb
		}),
		Remarks: &olocation.Remarks,
	}

	return l
}

type UUIDModel struct {
	ID *uuid.UUID `json:"id" gorm:"type:uuid;primary_key;"`
}

func (u *UUIDModel) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == nil {
		id := uuid.New()
		u.ID = &id
	}
	return
}
