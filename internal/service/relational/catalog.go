package relational

import (
	"database/sql"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"time"
)

type Catalog struct {
	UUIDModel
	Metadata   CatalogMetadata                `json:"metadata"`
	Params     datatypes.JSONSlice[Parameter] `json:"params"`
	Controls   []Control                      `json:"controls"`
	Groups     []Group                        `json:"groups"`
	BackMatter BackMatter                     `json:"back-matter"`
	/**
	"required": [
		"uuid",
		"metadata"
	],
	*/
}

type CatalogMetadata struct {
	UUIDModel
	Title              string                          `json:"title"`
	Published          sql.NullTime                    `json:"published"`
	LastModified       time.Time                       `json:"last-modified"`
	Version            string                          `json:"version"`
	OscalVersion       string                          `json:"oscal-version"`
	DocumentIDs        datatypes.JSONSlice[DocumentID] `json:"document-ids"` // -> DocumentID
	Props              datatypes.JSONSlice[Prop]       `json:"props"`
	Links              datatypes.JSONSlice[Link]       `json:"links"`
	Revisions          []Revision                      `json:"revisions"` // -> Revision
	Roles              []Role                          `gorm:"many2many:catalog_roles;"`
	Locations          []Location                      `gorm:"many2many:catalog_locations;"`
	Parties            []Party                         `gorm:"many2many:catalog_parties;"`
	ResponsibleParties []ResponsibleParty              `gorm:"many2many:catalog_responsible_parties;"`
	Actions            []Action                        `json:"actions"` // -> Action
	Remarks            string                          `json:"remarks"`
	CatalogID          uuid.UUID

	/**
	"required": [
		"title",
		"last-modified",
		"version",
		"oscal-version"
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

type Citation struct {
	Text  string `json:"text"` // required
	Props Props  `json:"props"`
	Links Links  `json:"links"`
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

type BackMatter struct {
	UUIDModel
	CatalogID uuid.UUID
	Resources []BackMatterResource `json:"resources"`
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

type DocumentIDScheme string

const DocumentIDSchemeDoi DocumentIDScheme = "http://www.doi.org/"

type DocumentID struct {
	Scheme     DocumentIDScheme `json:"scheme"`
	Identifier string           `json:"identifier"`
}

func (d *DocumentID) FromOscal(id oscalTypes_1_1_3.DocumentId) {
	d.Scheme = DocumentIDScheme(id.Scheme)
	d.Identifier = id.Identifier
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

type Party struct {
	UUIDModel
	CatalogMetadataID     uuid.UUID
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

type ResponsibleParty struct {
	UUIDModel
	CatalogMetadataID uuid.UUID
	Props             datatypes.JSONSlice[Prop] `json:"props"`
	Links             datatypes.JSONSlice[Link] `json:"links"`
	Remarks           string                    `json:"remarks"`

	RoleID  string `json:"role-id"` // required
	Role    Role
	Parties []Party `gorm:"many2many:responsible_party_parties;"`
}

type Action struct {
	UUIDModel
	CatalogMetadataID  uuid.UUID                 // required
	Date               sql.NullTime              `json:"date"`
	Type               string                    `json:"type"`   // required
	System             string                    `json:"system"` // required
	Props              datatypes.JSONSlice[Prop] `json:"props"`
	Links              datatypes.JSONSlice[Link] `json:"links"`
	ResponsibleParties []ResponsibleParty        `gorm:"many2many:action_responsible_party;"`
	Remarks            string                    `json:"remarks"`
}

type Location struct {
	UUIDModel
	CatalogMetadataID uuid.UUID
	Title             *string                               `json:"title"`
	Address           datatypes.JSONType[Address]           `json:"address"`
	EmailAddresses    datatypes.JSONType[[]string]          `json:"email-addresses"`
	TelephoneNumbers  datatypes.JSONType[[]TelephoneNumber] `json:"telephone-numbers"`
	Urls              datatypes.JSONType[[]string]          `json:"urls"`
	Props             datatypes.JSONType[[]Prop]            `json:"props"`
	Links             datatypes.JSONType[[]Link]            `json:"links"`
	Remarks           *string                               `json:"remarks"`
	/**
	"required": [
		"uuid"
	],
	*/
}

type Role struct {
	ID                string `json:"id" gorm:"primary_key;"`
	CatalogMetadataID uuid.UUID
	Title             string                    `json:"title"`
	ShortName         *string                   `json:"short-name"`
	Description       *string                   `json:"description"`
	Props             datatypes.JSONSlice[Prop] `json:"props"`
	Links             datatypes.JSONSlice[Link] `json:"links"`
	Remarks           *string                   `json:"remarks"`
}

type Revision struct {
	// Only version is required
	UUIDModel
	CatalogMetadataID uuid.UUID
	Title             *string                   `json:"title"`
	Published         sql.NullTime              `json:"published"`
	LastModified      sql.NullTime              `json:"last-modified"`
	Version           string                    `json:"version"` // required
	OscalVersion      *string                   `json:"oscal-version"`
	Props             datatypes.JSONSlice[Prop] `json:"props"`
	Links             datatypes.JSONSlice[Link] `json:"links"`
	Remarks           *string                   `json:"remarks"`
}

func (d *Revision) FromOscal(rev oscalTypes_1_1_3.RevisionHistoryEntry) {
	if rev.Published != nil {
		d.Published = sql.NullTime{Time: *rev.Published}
	}
	if rev.LastModified != nil {
		d.LastModified = sql.NullTime{Time: *rev.LastModified}
	}
	d.Version = rev.Version
	d.OscalVersion = &rev.OscalVersion
	d.Title = &rev.Title
	d.Remarks = &rev.Remarks
}

type Parameter struct {
	ID          string                `json:"id"`
	Class       string                `json:"class"`
	Props       Props                 `json:"props"`
	Links       Links                 `json:"links"`
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

	/**
	"required": [
		"expression"
	],
	*/
}

type Part struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	NS    string `json:"ns"`
	Class string `json:"class"`
	Title string `json:"title"`
	Prose string `json:"prose"`
	Props Props  `json:"props"`
	Links Links  `json:"links"`
	Parts []Part `json:"parts"` // -> Part

	/**
	"required": [
		"name"
	],
	*/
}
