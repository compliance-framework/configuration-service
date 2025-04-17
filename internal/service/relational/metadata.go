package relational

import (
	"database/sql"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"time"
)

type Metadata struct {
	UUIDModel

	// Metadata is shared across many resources, and so it mapped using a polymorphic relationship
	ParentID   *string
	ParentType *string

	Title              string                          `json:"title"`
	Published          sql.NullTime                    `json:"published"`
	LastModified       time.Time                       `json:"last-modified"`
	Version            string                          `json:"version"`
	OscalVersion       string                          `json:"oscal-version"`
	DocumentIDs        datatypes.JSONSlice[DocumentID] `json:"document-ids"` // -> DocumentID
	Props              datatypes.JSONSlice[Prop]       `json:"props"`
	Links              datatypes.JSONSlice[Link]       `json:"links"`
	ResponsibleParties []ResponsibleParty              `gorm:"many2many:metadata_responsible_parties;"`
	Revisions          []Revision                      `json:"revisions"`
	Roles              []Role                          `json:"roles"`
	Locations          []Location                      `json:"locations"`
	Parties            []Party                         `json:"parties"`
	Actions            []Action                        `json:"actions"`
	Remarks            string                          `json:"remarks"`

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
					doc := DocumentID{}
					doc.UnmarshalOscal(document)
					list = append(list, doc)
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

type Action struct {
	UUIDModel

	// Actions only exist on a metadata object. We'll link them straight there with a BelongsTo relationship
	MetadataID uuid.UUID `json:"metadata-id"`

	Date               sql.NullTime              `json:"date"`
	Type               string                    `json:"type"`   // required
	System             string                    `json:"system"` // required
	Props              datatypes.JSONSlice[Prop] `json:"props"`
	Links              datatypes.JSONSlice[Link] `json:"links"`
	ResponsibleParties []ResponsibleParty        `gorm:"many2many:action_responsible_parties;"`
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

	// Parties only exist on a metadata object. We'll link them straight there with a BelongsTo relationship
	MetadataID uuid.UUID `json:"metadata-id"`

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

type Revision struct {
	// Only version is required
	UUIDModel

	// Revision only exist on a metadata object. We'll link them straight there with a BelongsTo relationship
	MetadataID uuid.UUID `json:"metadata-id"`

	Title        *string                   `json:"title"`
	Published    sql.NullTime              `json:"published"`
	LastModified sql.NullTime              `json:"last-modified"`
	Version      string                    `json:"version"` // required
	OscalVersion *string                   `json:"oscal-version"`
	Props        datatypes.JSONSlice[Prop] `json:"props"`
	Links        datatypes.JSONSlice[Link] `json:"links"`
	Remarks      *string                   `json:"remarks"`
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
	ID string `json:"id" gorm:"primary_key;"`

	// Roles only exist on a metadata object. We'll link them straight there with a BelongsTo relationship
	MetadataID uuid.UUID `json:"metadata-id"`

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

type Location struct {
	UUIDModel

	// Locations only exist on a metadata object. We'll link them straight there with a BelongsTo relationship
	MetadataID uuid.UUID `json:"metadata-id"`

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
