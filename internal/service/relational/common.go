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
	Roles              []Role                          `gorm:"many2many:catalog_roles;"`
	Locations          []Location                      `gorm:"many2many:catalog_locations;"`
	Parties            []Party                         `gorm:"many2many:catalog_parties;"`
	ResponsibleParties []ResponsibleParty              `gorm:"many2many:catalog_responsible_parties;"`
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
