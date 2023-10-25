package service

import (
	"context"

	"github.com/compliance-framework/configuration-service/domain"
	"github.com/compliance-framework/configuration-service/event"
	mongoStore "github.com/compliance-framework/configuration-service/store/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ResultService struct {
	resultCollection *mongo.Collection
	publisher        event.Publisher
}

func NewResultService(p event.Publisher) *ResultService {
	return &ResultService{
		resultCollection: mongoStore.Collection("result"),
		publisher:        p,
	}
}

func (s *ResultService) GetById(id string) (*domain.Result, error) {
	result, err := mongoStore.FindById[domain.Result](context.Background(), "result", id)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *ResultService) Create(result *domain.Result) (string, error) {
	result, err := s.resultCollection.InsertOne(context.TODO(), result)
	if err != nil {
		return "", err
	}
	return result.InsertedID.(primitive.ObjectID).Hex(), nil
}

func (s *ResultService) Update(result *domain.Result) error {
	_, err := s.resultCollection.ReplaceOne(context.Background(), primitive.M{"uuid": result.Uuid}, result)
	if err != nil {
		return err
	}

	if result.Ready() {
		err = s.publisher(event.ResultUpdated{Uuid: result.Uuid}, event.TopicTypeResult)
		if err != nil {
			return err
		}
	}

	return nil
}
