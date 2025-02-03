package store

import (
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
)

type CatalogStore interface {
	CreateCatalog(catalog *oscaltypes113.Catalog) (interface{}, error)
	GetCatalog(id string) (*oscaltypes113.Catalog, error)
	UpdateCatalog(id string, catalog *oscaltypes113.Catalog) error
	DeleteCatalog(id string) error
	CreateControl(catalogId string, control *oscaltypes113.Control) (interface{}, error)
	GetControl(catalogId string, controlId string) (*oscaltypes113.Control, error)
	UpdateControl(catalogId string, controlId string, control *oscaltypes113.Control) (*oscaltypes113.Catalog, error)
}
