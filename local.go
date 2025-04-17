package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"github.com/compliance-framework/configuration-service/internal/service/relational"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.Migrator().DropTable(
		&relational.Location{},
		&relational.Party{},
		&relational.BackMatterResource{},
		&relational.BackMatter{},
		&relational.Role{},
		&relational.Revision{},
		&relational.Control{},
		&relational.Group{},
		&relational.ResponsibleParty{},
		&relational.Action{},
		&relational.Metadata{},
		&relational.Catalog{},
		"catalog_roles",
		&relational.ComponentDefinition{},
	)
	if err != nil {
		panic(err)
	}

	// Migrate the schema
	err = db.AutoMigrate(
		&relational.Location{},
		&relational.Party{},
		&relational.BackMatterResource{},
		&relational.BackMatter{},
		&relational.Role{},
		&relational.Revision{},
		&relational.Control{},
		&relational.Group{},
		&relational.ResponsibleParty{},
		&relational.Action{},
		&relational.Metadata{},
		&relational.Catalog{},
		&relational.ComponentDefinition{},
	)
	if err != nil {
		panic(err)
	}

	err = LoadCatalogDataFromJSON(db, "testdata/sp800_53_catalog.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	err = LoadComponentDefinitionDataFromJSON(db, "testdata/sp800_53_component_definition_sample.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func LoadComponentDefinitionDataFromJSON(db *gorm.DB, jsonPath string) error {
	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		return err
	}
	fmt.Println("Successfully Opened Component Definition")
	defer jsonFile.Close()
	input := &struct {
		ComponentDefinition *oscaltypes113.ComponentDefinition `json:"component-definition"`
	}{}

	err = json.NewDecoder(jsonFile).Decode(input)
	if err != nil {
		return err
	}

	cdId := uuid.MustParse(input.ComponentDefinition.UUID)
	metadata := &relational.Metadata{}
	metadata.UnmarshalOscal(input.ComponentDefinition.Metadata)

	db.Create(&relational.ComponentDefinition{
		UUIDModel: relational.UUIDModel{
			ID: &cdId,
		},
		Metadata: *metadata,
	})

	return nil
}

func LoadCatalogDataFromJSON(db *gorm.DB, jsonPath string) error {
	jsonFile, err := os.Open(jsonPath)
	// if we os.Open returns an error then handle it
	if err != nil {
		return err
	}
	fmt.Println("Successfully Opened Catalog")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	input := &struct {
		Catalog *oscaltypes113.Catalog
	}{}
	err = json.NewDecoder(jsonFile).Decode(input)
	if err != nil {
		return err
	}

	// First, the catalog
	catalogId := uuid.MustParse(input.Catalog.UUID)

	metadata := &relational.Metadata{}
	metadata.UnmarshalOscal(input.Catalog.Metadata)
	db.Create(&relational.Catalog{
		UUIDModel: relational.UUIDModel{
			ID: &catalogId,
		},
		Metadata: *metadata,
	})

	return nil
}

// incomplete
func MetadataFromOscal(metadata *oscaltypes113.Metadata) relational.Metadata {
	published := sql.NullTime{}
	if metadata.Published != nil {
		published = sql.NullTime{
			Time: *metadata.Published,
		}
	}
	return relational.Metadata{
		Title:        metadata.Title,
		Published:    published,
		LastModified: metadata.LastModified,
		Version:      metadata.Version,
		OscalVersion: metadata.OscalVersion,
		DocumentIDs: func() datatypes.JSONSlice[relational.DocumentID] {
			list := make([]relational.DocumentID, 0)
			if metadata.DocumentIds != nil {
				for _, document := range *metadata.DocumentIds {
					doc := &relational.DocumentID{}
					doc.FromOscal(document)
					list = append(list, *doc)
				}
			}
			return datatypes.NewJSONSlice[relational.DocumentID](list)
		}(),
		Props:              ConvertList(metadata.Props, PropFromOscal),
		Links:              ConvertList(metadata.Links, LinkFromOscal),
		Revisions:          ConvertList(metadata.Revisions, RevisionFromOscal),
		Roles:              ConvertList(metadata.Roles, RoleFromOscal),
		Locations:          nil,
		Parties:            ConvertList(metadata.Parties, PartyFromOscal),
		ResponsibleParties: nil,
		Actions:            nil,
		Remarks:            metadata.Remarks,
	}
}

func ConvertList[in any, out any](list *[]in, mutate func(in) out) []out {
	if list == nil {
		return nil
	}
	output := make([]out, 0)
	for _, i := range *list {
		output = append(output, mutate(i))
	}
	return output
}

func RoleFromOscal(r oscaltypes113.Role) relational.Role {
	return relational.Role{
		ID:          r.ID,
		Title:       r.Title,
		ShortName:   &r.ShortName,
		Description: &r.Description,
		Props:       ConvertList(r.Props, PropFromOscal),
		Links:       ConvertList(r.Links, LinkFromOscal),
		Remarks:     &r.Remarks,
	}
}

func RevisionFromOscal(r oscaltypes113.RevisionHistoryEntry) relational.Revision {
	published := sql.NullTime{}
	if r.Published != nil {
		published = sql.NullTime{Time: *r.Published}
	}
	lastModified := sql.NullTime{}
	if r.LastModified != nil {
		lastModified = sql.NullTime{Time: *r.LastModified}
	}
	return relational.Revision{
		Title:        &r.Title,
		Published:    published,
		LastModified: lastModified,
		Version:      r.Version,
		OscalVersion: &r.OscalVersion,
		Props:        ConvertList(r.Props, PropFromOscal),
		Links:        ConvertList(r.Links, LinkFromOscal),
		Remarks:      &r.Remarks,
	}
}

func ParameterGuidelineFromOscal(c oscaltypes113.ParameterGuideline) relational.ParameterGuideline {
	return relational.ParameterGuideline(c)
}

func ParameterConstraintFromOscal(c oscaltypes113.ParameterConstraint) relational.ParameterConstraint {
	return relational.ParameterConstraint{
		Description: c.Description,
		Tests:       ConvertList(c.Tests, ConstraintTestFromOscal),
	}
}

func ConstraintTestFromOscal(ct oscaltypes113.ConstraintTest) relational.ParameterConstraintTest {
	return relational.ParameterConstraintTest(ct)
}

func LinkFromOscal(olink oscaltypes113.Link) relational.Link {
	link := relational.Link{}
	link.UnmarshalOscal(olink)
	return link
}

func PropFromOscal(property oscaltypes113.Property) relational.Prop {
	prop := relational.Prop{}
	prop.UnmarshalOscal(property)
	return prop
}

// incomplete
func PartyFromOscal(p oscaltypes113.Party) relational.Party {
	var email_addresses *[]string
	party_uuid := uuid.MustParse(p.UUID)

	if p.EmailAddresses != nil {
		email_addresses = p.EmailAddresses
	} else {
		email_addresses = &[]string{}
	}

	return relational.Party{
		UUIDModel: relational.UUIDModel{
			ID: &party_uuid,
		},
		Type:           relational.PartyType(p.Type),
		Name:           &p.Name,
		ShortName:      &p.ShortName,
		Props:          ConvertList(p.Props, PropFromOscal),
		Links:          ConvertList(p.Links, LinkFromOscal),
		EmailAddresses: *email_addresses,
		TelephoneNumbers: ConvertList(p.TelephoneNumbers, func(tn oscaltypes113.TelephoneNumber) relational.TelephoneNumber {
			tn_type := relational.TelephoneNumberType(tn.Type)
			return relational.TelephoneNumber{
				Number: tn.Number,
				Type:   &tn_type,
			}
		}),

		// TODO: manage support for if `location-uuids` is set
		// as spec is choice of addresses OR location-uuids
		Addresses: ConvertList(p.Addresses, func(a oscaltypes113.Address) relational.Address {
			addr_type := relational.AddressType(a.Type)
			return relational.Address{
				Type:       addr_type,
				AddrLines:  *a.AddrLines,
				City:       a.City,
				State:      a.State,
				PostalCode: a.PostalCode,
				Country:    a.Country,
			}
		}),

		ExternalIds: ConvertList(p.ExternalIds, func(e oscaltypes113.PartyExternalIdentifier) relational.PartyExternalID {
			party_scheme := relational.PartyExternalIDScheme(e.Scheme)
			return relational.PartyExternalID{
				ID:     e.ID,
				Scheme: party_scheme,
			}
		}),

		// Locations -> many-2-many relationship (Location)
		// Members of Organizations -> many-2-many relationship (Party)

		Remarks: &p.Remarks,
	}
}
