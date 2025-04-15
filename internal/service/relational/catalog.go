package relational

import (
	"database/sql"
	"github.com/google/uuid"
	"time"
)

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
