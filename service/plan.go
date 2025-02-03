package service

import (
	"context"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"log"

	. "github.com/compliance-framework/configuration-service/domain"
	"github.com/compliance-framework/configuration-service/event"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PlanService struct {
	planCollection    *mongo.Collection
	subjectCollection *mongo.Collection
	publisher         event.Publisher
}

func NewPlanService(db *mongo.Database, p event.Publisher) *PlanService {
	return &PlanService{
		planCollection:    db.Collection("plan"),
		subjectCollection: db.Collection("subject"),
		publisher:         p,
	}
}

func (s *PlanService) GetById(ctx context.Context, id string) (*Plan, error) {
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	output := s.planCollection.FindOne(ctx, bson.D{bson.E{Key: "_id", Value: objectId}})
	if output.Err() != nil {
		return nil, output.Err()
	}

	result := &Plan{}
	err = output.Decode(result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (s *PlanService) Create(plan *Plan) (string, error) {
	log.Println("Create")
	result, err := s.planCollection.InsertOne(context.TODO(), plan)
	if err != nil {
		return "", err
	}
	return result.InsertedID.(primitive.ObjectID).Hex(), nil
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

type RiskSeverity string

const (
	Medium RiskSeverity = "medium"
	Low    RiskSeverity = "low"
	High   RiskSeverity = "high"
)

type RiskState string

const (
	Pass          RiskState = "pass"
	Warn          RiskState = "warn"
	Fail          RiskState = "fail"
	Indeterminate RiskState = "indeterminate"
)

type RiskLevels struct {
	Low    int `json:"low" yaml:"low"`
	Medium int `json:"medium" yaml:"medium"`
	High   int `json:"high" yaml:"high"`
}

type RiskScore struct {
	Score    int          `json:"score" yaml:"score"`
	Severity RiskSeverity `json:"severity" yaml:"severity"`
}

type PlanSummary struct {
	Published        string     `json:"published" yaml:"published"`
	EndDate          string     `json:"endDate" yaml:"endDate"`
	Description      string     `json:"description" yaml:"description"`
	Status           string     `json:"status" yaml:"status"`
	NumControls      int        `json:"numControls" yaml:"numControls"`
	NumSubjects      int        `json:"numSubjects" yaml:"numSubjects"`
	NumObservations  int        `json:"numObservations" yaml:"numObservations"`
	NumRisks         int        `json:"numRisks" yaml:"numRisks"`
	RiskScore        RiskScore  `json:"riskScore" yaml:"riskScore"`
	ComplianceStatus float64    `json:"complianceStatus" yaml:"complianceStatus"`
	RiskLevels       RiskLevels `json:"riskLevels" yaml:"riskLevels"`
}

type ComplianceStatusByTargets struct {
	Control    string      `json:"control" yaml:"control"`
	Target     string      `json:"target" yaml:"target"`
	Compliance []RiskState `json:"compliance" yaml:"compliance"`
}

type ComplianceStatusOverTime struct {
	Date         string `json:"date" yaml:"date"`
	Findings     int    `json:"findings" yaml:"findings"`
	Observations int    `json:"observations" yaml:"observations"`
	Risks        int    `json:"risks" yaml:"risks"`
}

type RemediationVsTime struct {
	Control     string `json:"control" yaml:"control"`
	Remediation string `json:"remediation" yaml:"remediation"`
}
