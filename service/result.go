package service

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/compliance-framework/configuration-service/domain"
	mongoStore "github.com/compliance-framework/configuration-service/store/mongo"
	"go.mongodb.org/mongo-driver/mongo"
)

type ResultService struct {
	resultCollection *mongo.Collection
}

func NewResultService() *ResultService {
	return &ResultService{
		resultCollection: mongoStore.Collection("result"),
	}
}

func (s *ResultService) GetById(id string) (*domain.Result, error) {
	result, err := mongoStore.FindById[domain.Result](context.Background(), "result", id)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *ResultService) FindByPlanId(id string) (*[]domain.Result, error) {
	results, err := mongoStore.FindMany[domain.Result](context.Background(), "result", bson.M{"planUuid": id})
	if err != nil {
		return nil, err
	}
	return &results, nil
}

func (s *ResultService) Create(plan *domain.Result) (string, error) {
	result, err := s.resultCollection.InsertOne(context.Background(), plan)
	if err != nil {
		return "", err
	}
	return result.InsertedID.(primitive.ObjectID).Hex(), nil
}
