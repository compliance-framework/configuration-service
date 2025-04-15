package relational

import (
	"database/sql"
	"github.com/google/uuid"
	"time"
)

type Catalog struct {
	ID         uuid.UUID   `json:"id"`
	Metadata   Metadata    `json:"metadata"`
	Params     []Parameter `json:"params"`
	Controls   []Control   `json:"controls"`
	Groups     []Group     `json:"groups"`
	BackMatter BackMatter  `json:"back-matter"`
	/**
	"required": [
		"uuid",
		"metadata"
	],
	*/
}

type Metadata struct {
	Title              string             `json:"title"`
	Published          time.Time          `json:"published"`
	LastModified       time.Time          `json:"last-modified"`
	Version            string             `json:"version"`
	OscalVersion       string             `json:"oscal-version"`
	Revisions          []Revision         `json:"revisions"`    // -> Revision
	DocumentIDs        []DocumentID       `json:"document-ids"` // -> DocumentID
	Props              Props              `json:"props"`
	Links              Links              `json:"links"`
	Roles              []Role             `json:"roles"`               // -> Role
	Locations          []Location         `json:"locations"`           // -> Location
	Parties            []Party            `json:"parties"`             // -> Party
	ResponsibleParties []ResponsibleParty `json:"responsible-parties"` // -> ResponsibleParty
	Actions            []Action           `json:"actions"`             // -> Action
	Remarks            string             `json:"remarks"`

	/**
	"required": [
		"title",
		"last-modified",
		"version",
		"oscal-version"
	],
	*/
}

type BackMatter struct {
	Resources []BackMatterResource `json:"resources"`
}

type Group struct {
	ID       string      `json:"id"`
	Class    string      `json:"class"`
	Title    string      `json:"title"`
	Params   []Parameter `json:"params"`
	Parts    []Party     `json:"parts"`
	Props    Props       `json:"props"`
	Links    Links       `json:"links"`
	Groups   []Group     `json:"groups"`
	Controls []Control   `json:"controls"`

	/**
	"required": [
		"title"
	],
	*/
}

type Control struct {
	ID       string      `json:"id"`
	Class    string      `json:"class"`
	Title    string      `json:"title"`
	Params   []Parameter `json:"params"`
	Parts    []Part      `json:"parts"`
	Props    Props       `json:"props"`
	Links    Links       `json:"links"`
	Controls []Control   `json:"controls"` // -> Control

	/**
	"required": [
		"id",
		"title"
	],
	*/
}

type Citation struct {
	Text  string `json:"text"`
	Props Props  `json:"props"`
	Links Links  `json:"links"`

	/**
	"required": [
	  "text"
	],
	*/
}

type HashAlgorithm string

const (
	HashAlgorithmSHA_224  = "SHA-224"
	HashAlgorithmSHA_256  = "SHA-256"
	HashAlgorithmSHA_384  = "SHA-384"
	HashAlgorithmSHA_512  = "SHA-512"
	HashAlgorithmSHA3_224 = "SHA3-224"
	HashAlgorithmSHA3_256 = "SHA3-256"
	HashAlgorithmSHA3_384 = "SHA3-384"
	HashAlgorithmSHA3_512 = "SHA3-512"
)

type Hash struct {
	Algorithm HashAlgorithm `json:"algorithm"`
	Value     string        `json:"value"`

	/**
	"required": [
		"value",
		"algorithm"
	],
	*/
}

type RLink struct {
	Href      string `json:"href"`
	MediaType string `json:"media-type"`
	Hashes    []Hash `json:"hashes"`

	/**
	"required": [
		"href"
	],
	*/
}

type Base64 struct {
	Filename  string `json:"filename"`
	MediaType string `json:"media-type"`
	Value     string `json:"value"`

	/**
	"required": [
	  "value"
	],
	*/
}

type BackMatterResource struct {
	ID          uuid.UUID    `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Props       Props        `json:"props"`
	DocumentIDs []DocumentID `json:"document-ids"`
	Citations   Citation     `json:"citation"`
	RLinks      []RLink      `json:"rlinks"`
	Base64      Base64       `json:"base64"`
	Remarks     string       `json:"remarks"`

	/**
	"required": [
		"uuid"
	],
	*/
}

type DocumentIDScheme string

const DocumentIDSchemeDoi DocumentIDScheme = "http://www.doi.org/"

type DocumentID struct {
	Scheme     DocumentIDScheme `json:"scheme"`
	Identifier string           `json:"identifier"`
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
	Uuid                  uuid.UUID         `json:"uuid"`
	Type                  PartyType         `json:"type"`
	Name                  string            `json:"name"`
	ShortName             string            `json:"short-name"`
	ExternalIds           []PartyExternalID `json:"external-ids"`
	Props                 Props             `json:"props"`
	Links                 Links             `json:"links"`
	EmailAddresses        []string          `json:"email-addresses"`
	TelephoneNumbers      []TelephoneNumber `json:"telephone-numbers"`
	Addresses             []Address         `json:"addresses"`
	LocationUuids         []uuid.UUID       `json:"location-uuids"`          // -> Location
	MemberOfOrganizations []uuid.UUID       `json:"member-of-organizations"` // -> Party
	Remarks               string            `json:"remarks"`

	/**
	"required": [
		"uuid",
		"type"
	],
	*/
}

type ResponsibleParty struct {
	RoleID     string      `json:"role-id"`     // -> Role
	PartyUuids []uuid.UUID `json:"party-uuids"` // -> Party
	Props      Props       `json:"props"`
	Links      Links       `json:"links"`
	Remarks    string      `json:"remarks"`

	/**
	"required": [
		"role-id",
		"party-uuids"
	],
	*/
}

type Action struct {
	Uuid               uuid.UUID          `json:"uuid"`
	Date               time.Time          `json:"date"`
	Type               string             `json:"type"`
	System             string             `json:"system"`
	Props              Props              `json:"props"`
	Links              Links              `json:"links"`
	ResponsibleParties []ResponsibleParty `json:"responsible-parties"`
	Remarks            string             `json:"remarks"`

	/**
	"required": [
		"uuid",
		"type",
		"system"
	],
	*/
}

type Location struct {
	ID               uuid.UUID         `json:"id"`
	Title            string            `json:"title"`
	Address          Address           `json:"address"`
	EmailAddresses   []string          `json:"email-addresses"`
	TelephoneNumbers []TelephoneNumber `json:"telephone-numbers"`
	Urls             []string          `json:"urls"`
	Props            Props             `json:"props"`
	Links            Links             `json:"links"`
	Remarks          string            `json:"remarks"`
	/**
	"required": [
		"uuid"
	],
	*/
}

type Role struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	ShortName   *string `json:"short-name"`
	Description *string `json:"description"`
	Props       Props   `json:"props"`
	Links       Links   `json:"links"`
	Remarks     *string `json:"remarks"`

	/**
	"required": [
	  "id",
	  "title"
	],
	*/
}

type Revision struct {
	// Only version is required
	ID           uuid.UUID    `json:"id" gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Title        *string      `json:"title"`
	Published    sql.NullTime `json:"published"`
	LastModified sql.NullTime `json:"last-modified"`
	Version      string       `json:"version"`
	OscalVersion *string      `json:"oscal-version"`
	Props        Props        `json:"props"`
	Links        Links        `json:"links"`
	Remarks      *string      `json:"remarks"`

	/**
	"required": [
	  "version"
	],
	*/
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
