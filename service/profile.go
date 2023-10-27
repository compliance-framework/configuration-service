package service

import (
	"context"

	"github.com/compliance-framework/configuration-service/domain"
	"github.com/compliance-framework/configuration-service/event"
	mongoStore "github.com/compliance-framework/configuration-service/store/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

//todo
//create an endpoint to save a profile (only with its title for now),
// get a list of profiles

type ProfileService struct {
	profileCollection *mongo.Collection
	publisher         event.Publisher
}

func NewProfileService(p event.Publisher) *ProfileService {
	return &ProfileService{
		profileCollection: mongoStore.Collection("profile"),
		publisher:         p,
	}
}

func (s *ProfileService) GetById(id string) (*domain.Profile, error) {
	profile, err := mongoStore.FindById[domain.Profile](context.Background(), "profile", id)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *ProfileService) GetByTitle(title string) (*domain.Profile, error) {
	filter := bson.M{"title": title}
	profile, err := mongoStore.FindOne[domain.Profile](context.Background(), "profile", filter)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *ProfileService) Create(profile *domain.Profile) (string, error) {
	result, err := s.profileCollection.InsertOne(context.TODO(), profile)
	if err != nil {
		return "", err
	}

	return result.InsertedID.(primitive.ObjectID).Hex(), nil
}
