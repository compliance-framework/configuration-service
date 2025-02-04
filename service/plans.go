package service

import (
	"context"
	"github.com/compliance-framework/configuration-service/domain"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type PlansService struct {
	planCollection *mongo.Collection
}

func NewPlansService(database *mongo.Database) *PlansService {
	return &PlansService{
		planCollection: database.Collection("plan"),
	}
}

func (s *PlansService) GetPlans() (*[]domain.Plan, error) {
	log.Println("GetPlans")

	results := &[]domain.Plan{}

	cursor, err := s.planCollection.Find(context.Background(), bson.D{})
	if err != nil {
		return nil, err
	}

	if err = cursor.All(context.Background(), results); err != nil {
		return nil, err
	}

	return results, nil
}
