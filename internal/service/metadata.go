package service

import (
	"context"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type DocWithMetadata struct {
	Uuid     uuid.UUID              `json:"uuid" yaml:"uuid"`
	Metadata oscaltypes113.Metadata `bson:"metadata"`
}

type MetadataService struct {
	database *mongo.Database
}

func NewMetadataService(database *mongo.Database) *MetadataService {
	return &MetadataService{
		database: database,
	}
}

func (s *MetadataService) AttachMetadata(uuid string, collection string, revision oscaltypes113.RevisionHistoryEntry) error {
	_, err := s.database.Collection(collection).UpdateOne(context.TODO(), bson.M{"uuid": uuid}, bson.M{
		"$addToSet": bson.M{
			"metadata.revisions": revision,
		},
	})

	return err
}
