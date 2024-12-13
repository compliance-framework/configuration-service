//go:build integration

package handler

import (
	"context"
	"fmt"
	mongo2 "github.com/compliance-framework/configuration-service/store/mongo"
	"github.com/compliance-framework/configuration-service/tests"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"testing"
)

func TestPlanApi(t *testing.T) {

	suite.Run(t, new(PlanIntegrationSuite))

}

type PlanIntegrationSuite struct {
	tests.IntegrationTestSuite
}

func (suite *PlanIntegrationSuite) TestSomething() {
	fmt.Println("Running a test")

	suite.Run("Some test", func() {
		ctx := context.Background()

		mongo2.NewCatalogStore()

		fmt.Println(mongo2.Collection("plan").Find(ctx, bson.D{}))
	})
}
