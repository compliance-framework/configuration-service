package handler

import (
	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/service"
	store "github.com/compliance-framework/configuration-service/internal/store/mongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func RegisterHandlers(server *api.Server, database *mongo.Database, logger *zap.SugaredLogger) {
	catalogStore := store.NewCatalogStore(database)
	catalogHandler := NewCatalogHandler(catalogStore)
	catalogHandler.Register(server.API().Group("/catalog"))

	planService := service.NewPlanService(database)
	planHandler := NewPlanHandler(logger, planService)
	planHandler.Register(server.API().Group("/plan"))

	resultService := service.NewResultsService(database)
	resultHandler := NewResultsHandler(logger, resultService, planService)
	resultHandler.Register(server.API().Group("/results"))

	plansService := service.NewPlansService(database)
	plansHandler := NewPlansHandler(logger, plansService)
	plansHandler.Register(server.API().Group("/plans"))

	systemPlanService := service.NewSSPService(database)
	systemPlanHandler := NewSSPHandler(systemPlanService)
	systemPlanHandler.Register(server.API())
}
