package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/compliance-framework/configuration-service/internal/service/relational"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
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
		&relational.ComponentDefinition{},
		"metadata_responsible_parties",
		"party_locations",
		"party_member_of_organisations",
		"responsible_party_parties",
		"action_responsible_parties",
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

	// First, the catalog
	def := &relational.ComponentDefinition{}
	def.UnmarshalOscal(*input.ComponentDefinition)
	out := db.Create(def)
	if out.Error != nil {
		return out.Error
	}

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
	catalog := &relational.Catalog{}
	catalog.UnmarshalOscal(*input.Catalog)
	out := db.Create(catalog)
	if out.Error != nil {
		return out.Error
	}

	return nil
}
