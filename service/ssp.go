package service

import (
	"context"
	"errors"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrSSPNotFound = errors.New("SSP not found")

type SSPService struct {
	sspCollection *mongo.Collection
}

func NewSSPService(database *mongo.Database) *SSPService {
	return &SSPService{
		sspCollection: database.Collection("ssp"),
	}
}

func (s *SSPService) Create(ssp *oscaltypes113.SystemSecurityPlan) (string, error) {
	result, err := s.sspCollection.InsertOne(context.TODO(), ssp)
	if err != nil {
		return "", err
	}

	return result.InsertedID.(primitive.ObjectID).Hex(), nil
}

func (s *SSPService) GetByID(id string) (*oscaltypes113.SystemSecurityPlan, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var ssp oscaltypes113.SystemSecurityPlan
	err = s.sspCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&ssp)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrSSPNotFound
		}
		return nil, err
	}

	return &ssp, nil
}

func (s *SSPService) List() ([]*oscaltypes113.SystemSecurityPlan, error) {
	ctx := context.Background()
	ssps := []*oscaltypes113.SystemSecurityPlan{}
	var tErr error
	cur, err := s.sspCollection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	for cur.Next(ctx) {
		ssp := &oscaltypes113.SystemSecurityPlan{}
		err = cur.Decode(&ssp)
		if err != nil {
			jErr := errors.Join(tErr, err)
			if jErr != nil {
				panic(jErr)
			}
		}
		ssps = append(ssps, ssp)

	}
	return ssps, err
}

func (s *SSPService) Update(id string, ssp *oscaltypes113.SystemSecurityPlan) (*oscaltypes113.SystemSecurityPlan, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": ssp}

	result, err := s.sspCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, err
	}

	if result.ModifiedCount == 0 {
		return nil, ErrSSPNotFound
	}

	return ssp, nil
}

func (s *SSPService) Delete(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": objID}
	result, err := s.sspCollection.DeleteOne(context.TODO(), filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrSSPNotFound
	}

	return nil
}
