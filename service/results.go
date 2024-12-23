package service

import (
	"context"
	"errors"
	"github.com/compliance-framework/configuration-service/domain"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ResultsService struct {
	resultsCollection *mongo.Collection
}

func NewResultsService(db *mongo.Database) *ResultsService {
	return &ResultsService{
		resultsCollection: db.Collection("results"),
	}
}

func (s *ResultsService) Create(ctx context.Context, result *domain.Result) error {
	output, err := s.resultsCollection.InsertOne(ctx, result)
	if err != nil {
		return err
	}
	insertedId, ok := output.InsertedID.(primitive.ObjectID)
	if !ok {
		return errors.New("result ID is not a primitive.ObjectID")
	}
	result.Id = &insertedId
	return nil
}

func (s *ResultsService) GetAll(ctx context.Context) ([]*domain.Result, error) {
	cursor, err := s.resultsCollection.Find(ctx, bson.D{})

	if err != nil {
		return nil, err
	}
	if cursor.Err() != nil {
		return nil, cursor.Err()
	}
	defer cursor.Close(ctx)

	var results []*domain.Result
	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (s *ResultsService) GetAllForPlan(ctx context.Context, planId *primitive.ObjectID) (results []*domain.Result, err error) {
	cursor, err := s.resultsCollection.Find(ctx, bson.D{
		{Key: "relatedPlans", Value: planId},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if cursor.Err() != nil {
		return nil, cursor.Err()
	}

	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *ResultsService) GetAllForStream(ctx context.Context, streamId uuid.UUID) (results []*domain.Result, err error) {
	cursor, err := s.resultsCollection.Find(ctx, bson.D{
		bson.E{Key: "streamId", Value: streamId},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if cursor.Err() != nil {
		return nil, cursor.Err()
	}

	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}
