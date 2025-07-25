package cmd

import (
	"context"
	"log"

	"github.com/compliance-framework/api/internal/api"
	"github.com/compliance-framework/api/internal/api/handler"
	"github.com/compliance-framework/api/internal/api/handler/auth"
	"github.com/compliance-framework/api/internal/api/handler/oscal"
	"github.com/compliance-framework/api/internal/config"
	"github.com/compliance-framework/api/internal/service"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	RunCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the compliance framework API",
		Run:   RunServer,
	}
)

func RunServer(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Can't initialize zap logger: %v", err)
	}
	defer zapLogger.Sync() // flushes buffer, if any
	sugar := zapLogger.Sugar()

	config := config.NewConfig(sugar)

	db, err := service.ConnectSQLDb(ctx, config, sugar)
	if err != nil {
		sugar.Fatal("Failed to connect to SQL database", "err", err)
	}

	err = service.MigrateUp(db)
	if err != nil {
		sugar.Fatal("Failed to migrate database", "err", err)
	}

	server := api.NewServer(ctx, sugar, config)

	handler.RegisterHandlers(server, sugar, db, config)
	oscal.RegisterHandlers(server, sugar, db, config)
	auth.RegisterHandlers(server, sugar, db, config)

	sugar.Infow("Allowed Origins", "origins", config.APIAllowedOrigins)
	server.PrintRoutes()

	checkErr(server.Start(config.AppPort), sugar)
}

func checkErr(err error, logger *zap.SugaredLogger) {
	if err != nil {
		logger.Fatalf("An error occurred: %v", err)
	}
}
