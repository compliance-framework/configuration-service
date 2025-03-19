package service

import (
	"context"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FindingService struct {
	collection *mongo.Collection
}

// NewFindingService creates a new FindingService connected to the "findings" collection.
func NewFindingService(db *mongo.Database) *FindingService {
	return &FindingService{
		collection: db.Collection("findings"),
	}
}

// Create inserts a new finding. If the finding's ID is nil, a new UUID is generated.
// It returns the newly created finding with its ID populated.
func (s *FindingService) Create(ctx context.Context, finding *Finding) (*Finding, error) {
	if finding.ID == nil {
		id := uuid.New()
		finding.ID = &id
	}
	_, err := s.collection.InsertOne(ctx, finding)
	if err != nil {
		return nil, err
	}
	return finding, nil
}

// FindOneById retrieves a finding by its primary key (_id).
func (s *FindingService) FindOneById(ctx context.Context, id *uuid.UUID) (*Finding, error) {
	filter := bson.M{"_id": id}
	var finding Finding
	err := s.collection.FindOne(ctx, filter).Decode(&finding)
	if err != nil {
		return nil, err
	}
	return &finding, nil
}

// Find retrieves findings based on a provided filter.
func (s *FindingService) Find(ctx context.Context, filter interface{}) ([]*Finding, error) {
	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var findings []*Finding
	for cursor.Next(ctx) {
		var f Finding
		if err := cursor.Decode(&f); err != nil {
			return nil, err
		}
		findings = append(findings, &f)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return findings, nil
}

// Update replaces an existing finding document (identified by _id) with the provided finding.
// If no document matches, it returns mongo.ErrNoDocuments.
func (s *FindingService) Update(ctx context.Context, id *uuid.UUID, finding *Finding) (*Finding, error) {
	filter := bson.M{"_id": id}
	result, err := s.collection.ReplaceOne(ctx, filter, finding)
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, mongo.ErrNoDocuments
	}
	return finding, nil
}

// Delete removes a finding by its _id and returns the deleted finding.
func (s *FindingService) Delete(ctx context.Context, id *uuid.UUID, _ *Finding) (*Finding, error) {
	filter := bson.M{"_id": id}
	var deleted Finding
	err := s.collection.FindOneAndDelete(ctx, filter).Decode(&deleted)
	if err != nil {
		return nil, err
	}
	return &deleted, nil
}

// FindLatest returns the finding with the given stream UUID that has the latest Collected timestamp.
func (s *FindingService) FindLatest(ctx context.Context, streamUuid uuid.UUID) (*Finding, error) {
	filter := bson.M{"uuid": streamUuid}
	opts := options.FindOne().SetSort(bson.D{{Key: "collected", Value: -1}})
	var finding Finding
	err := s.collection.FindOne(ctx, filter, opts).Decode(&finding)
	if err != nil {
		return nil, err
	}
	return &finding, nil
}

// FindByUuid returns up to 200 findings with the specified stream UUID, ordered by Collected in descending order.
func (s *FindingService) FindByUuid(ctx context.Context, streamUuid uuid.UUID) ([]*Finding, error) {
	filter := bson.M{"uuid": streamUuid}
	opts := options.Find().SetSort(bson.D{{Key: "collected", Value: -1}}).SetLimit(200)
	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var findings []*Finding
	for cursor.Next(ctx) {
		var f Finding
		if err := cursor.Decode(&f); err != nil {
			return nil, err
		}
		findings = append(findings, &f)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return findings, nil
}
