package oscal

import (
	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/api/middleware"
	"github.com/compliance-framework/configuration-service/internal/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterHandlers(server *api.Server, logger *zap.SugaredLogger, db *gorm.DB, config *config.Config) {
	oscalGroup := server.API().Group("/oscal")
	oscalGroup.Use(middleware.JWTMiddleware(config.JWTPublicKey))

	catalogHandler := NewCatalogHandler(logger, db)
	catalogHandler.Register(oscalGroup.Group("/catalogs"))

	profileHandler := NewProfileHandler(logger, db)
	profileHandler.Register(oscalGroup.Group("/profiles"))

	sspHandler := NewSystemSecurityPlanHandler(logger, db)
	sspHandler.Register(oscalGroup.Group("/system-security-plans"))

	partyHandler := NewPartyHandler(logger, db)
	partyHandler.Register(oscalGroup.Group("/parties"))

	roleHandler := NewRoleHandler(logger, db)
	roleHandler.Register(oscalGroup.Group("/roles"))

	componentDefinitionHandler := NewComponentDefinitionHandler(logger, db)
	componentDefinitionHandler.Register(oscalGroup.Group("/component-definitions"))

	poamHandler := NewPlanOfActionAndMilestonesHandler(logger, db)
	poamHandler.Register(oscalGroup.Group("/plan-of-action-and-milestones"))
}
