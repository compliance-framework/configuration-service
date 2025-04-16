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
		&relational.CatalogMetadata{},
		&relational.Catalog{},
		"catalog_roles",
		&relational.ComponentDefinitionMetadata{},
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
		&relational.CatalogMetadata{},
		&relational.Catalog{},
		&relational.ComponentDefinitionMetadata{},
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
	metadata := ComponentDefinitionMetadataFromOscal(&input.ComponentDefinition.Metadata)

	db.Create(&relational.ComponentDefinition{
		UUIDModel: relational.UUIDModel{
			ID: &cdId,
		},
		Metadata: metadata,
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
	metadata := CatalogMetadataFromOscal(&input.Catalog.Metadata)

	db.Create(&relational.Catalog{
		UUIDModel: relational.UUIDModel{
			ID: &catalogId,
		},
		Metadata: metadata,
	})

	return nil
}

// incomplete
func CatalogMetadataFromOscal(metadata *oscaltypes113.Metadata) relational.CatalogMetadata {
	published := sql.NullTime{}
	if metadata.Published != nil {
		published = sql.NullTime{
			Time: *metadata.Published,
		}
	}
	return relational.CatalogMetadata{
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
		Parties:            nil,
		ResponsibleParties: nil,
		Actions:            nil,
		Remarks:            metadata.Remarks,
	}
}

// incomplete
func ComponentDefinitionMetadataFromOscal(metadata *oscaltypes113.Metadata) relational.ComponentDefinitionMetadata {
	published := sql.NullTime{}
	if metadata.Published != nil {
		published = sql.NullTime{
			Time: *metadata.Published,
		}
	}
	return relational.ComponentDefinitionMetadata{
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
		Props: ConvertList(metadata.Props, PropFromOscal),
		Links: ConvertList(metadata.Links, LinkFromOscal),
		//Revisions:          ConvertList(metadata.Revisions, RevisionFromOscal),
		Roles:              ConvertList(metadata.Roles, RoleFromOscal),
		Locations:          nil,
		Parties:            nil,
		ResponsibleParties: nil,
		//Actions:            nil,
		Remarks: metadata.Remarks,
	}
}

func CatalogParameterFromOscal(parameter oscaltypes113.Parameter) relational.Parameter {
	return relational.Parameter{
		ID:          parameter.ID,
		Class:       parameter.Class,
		Props:       ConvertList(parameter.Props, PropFromOscal),
		Links:       ConvertList(parameter.Links, LinkFromOscal),
		Label:       parameter.Label,
		Usage:       parameter.Usage,
		Constraints: ConvertList(parameter.Constraints, ParameterConstraintFromOscal),
		Guidelines:  ConvertList(parameter.Guidelines, ParameterGuidelineFromOscal),
		Values:      *parameter.Values,
		Select: relational.ParameterSelection{
			HowMany: relational.ParameterSelectionCount(parameter.Select.HowMany),
			Choice:  *parameter.Select.Choice,
		},
		Remarks: parameter.Remarks,
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
