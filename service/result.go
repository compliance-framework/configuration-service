package service

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"

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
	result, err := mongoStore.FindMany[domain.Result](context.Background(), "result", bson.M{"planUuid": id})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
