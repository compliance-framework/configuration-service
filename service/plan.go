package service

import (
	"context"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"log"

	"github.com/compliance-framework/configuration-service/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type PlanService struct {
	planCollection    *mongo.Collection
	subjectCollection *mongo.Collection
}

func NewPlanService(db *mongo.Database) *PlanService {
	return &PlanService{
		planCollection:    db.Collection("plan"),
		subjectCollection: db.Collection("subject"),
	}
}

func (s *PlanService) GetById(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	output := s.planCollection.FindOne(ctx, bson.D{bson.E{Key: "_id", Value: id}})
	if output.Err() != nil {
		return nil, output.Err()
	}

	result := &domain.Plan{}
	err := output.Decode(result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (s *PlanService) Create(plan *domain.Plan) (*domain.Plan, error) {
	log.Println("Create")
	if plan.UUID == nil {
		newId := uuid.New()
		plan.UUID = &newId
	}
	_, err := s.planCollection.InsertOne(context.TODO(), plan)
	if err != nil {
		return plan, err
	}
	return plan, nil
}

func (s *PlanService) SaveSubject(subject oscaltypes113.SubjectReference) error {
	log.Println("SaveSubject")
	_, err := s.subjectCollection.InsertOne(context.Background(), subject)
	if err != nil {
		return err
	}
	return nil
}

func (s *PlanService) Risks(planId string, resultId string) ([]oscaltypes113.Risk, error) {
	log.Println("Risks", "planId: ", planId, "resultId: ", resultId)
	pipeline := bson.A{
		bson.D{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$tasks"}}}},
		bson.D{
			{Key: "$project",
				Value: bson.D{
					{Key: "_id", Value: 0},
				},
			},
		},
	}

	cursor, err := s.planCollection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, err
	}

	var risks []oscaltypes113.Risk
	if err = cursor.All(context.Background(), &risks); err != nil {
		return nil, err
	}

	return risks, nil
}
