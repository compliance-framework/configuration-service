package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/compliance-framework/configuration-service/internal/service/relational"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
)

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
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
	)
	if err != nil {
		panic(err)
	}

	jsonFile, err := os.Open("testdata/sp800_53_catalog.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened Catalog")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	input := &struct {
		Catalog *oscaltypes113.Catalog
	}{}
	err = json.NewDecoder(jsonFile).Decode(input)
	if err != nil {
		fmt.Println(err)
	}

	catalog := input.Catalog

	// First, the catalog
	catalogId := uuid.MustParse(catalog.UUID)
	metadata := relational.CatalogMetadata{
		Title:        catalog.Metadata.Title,
		Published:    sql.NullTime{},
		LastModified: catalog.Metadata.LastModified,
		Version:      catalog.Metadata.Version,
		OscalVersion: catalog.Metadata.OscalVersion,
		DocumentIDs: func() datatypes.JSONSlice[relational.DocumentID] {
			list := make([]relational.DocumentID, 0)
			if catalog.Metadata.DocumentIds != nil {
				for _, document := range *catalog.Metadata.DocumentIds {
					doc := &relational.DocumentID{}
					doc.FromOscal(document)
					list = append(list, *doc)
				}
			}
			return datatypes.NewJSONSlice[relational.DocumentID](list)
		}(),
		Revisions: func() []relational.Revision {
			list := make([]relational.Revision, 0)
			for _, revision := range *catalog.Metadata.Revisions {
				doc := &relational.Revision{}
				doc.FromOscal(revision)
				list = append(list, *doc)
			}
			return list
		}(),
		Remarks: catalog.Metadata.Remarks,
	}
	db.Create(&relational.Catalog{
		UUIDModel: relational.UUIDModel{
			ID: &catalogId,
		},
		Metadata: metadata,
	})
}
