package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/api/handler"
	"github.com/compliance-framework/configuration-service/internal/api/handler/oscal"
	"github.com/compliance-framework/configuration-service/internal/service"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	DefaultMongoURI = "mongodb://localhost:27017"
	DefaultPort     = ":8080"
	DefaultDBDriver = "postgres"
)

type Config struct {
	MongoURI           string
	AppPort            string
	DBDriver           string
	DBConnectionString string
	DBDebug            bool
}

func RunServer(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Can't initialize zap logger: %v", err)
	}
	defer zapLogger.Sync() // flushes buffer, if any
	sugar := zapLogger.Sugar()

	config := loadConfig(sugar)

	mongoDatabase, err := connectMongo(ctx, options.Client().ApplyURI(config.MongoURI), "cf")
	if err != nil {
		sugar.Fatal(err)
	}
	defer mongoDatabase.Client().Disconnect(ctx)

	server := api.NewServer(ctx, sugar)

	handler.RegisterHandlers(server, mongoDatabase, sugar)

	db, err := connectSQLDb(config, sugar)
	if err != nil {
		sugar.Fatal("Failed to connect to SQL database", "err", err)
	}

	err = service.MigrateUp(db)
	if err != nil {
		sugar.Fatal("Failed to migrate database", "err", err)
	}
	oscal.RegisterHandlers(server, sugar, db)

	server.PrintRoutes()

	checkErr(server.Start(config.AppPort), sugar)
}

func loadConfig(logger *zap.SugaredLogger) (config Config) {
	if err := godotenv.Load(".env"); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.Fatalf("Error loading .env file: %v", err)
		}
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = DefaultMongoURI
	}

	port := DefaultPort
	appPort, portSet := os.LookupEnv("APP_PORT")
	if portSet {
		port = fmt.Sprintf(":%s", appPort)
	}

	dbDriver := os.Getenv("CCF_DB_DRIVER")
	if dbDriver == "" {
		dbDriver = DefaultDBDriver
	}

	dbConnectionString, ok := os.LookupEnv("CCF_DB_CONNECTION")
	if !ok {
		switch dbDriver {
		case "postgres":
			dbConnectionString = "host=db user=postgres password=postgres dbname=ccf port=5432 sslmode=disable"
		default:
			logger.Fatalf("Unrecognised db driver: %s", dbDriver)
		}
	}

	dbDebugEnv := os.Getenv("CCF_DB_DEBUG")
	if dbDebugEnv == "" {
		dbDebugEnv = "false"
	}
	dbDebug := dbDebugEnv == "1" || strings.ToLower(dbDebugEnv) == "true"

	logger.Infof("dbConnectionString: %s", dbConnectionString)

	config = Config{
		MongoURI:           mongoURI,
		AppPort:            port,
		DBDriver:           dbDriver,
		DBConnectionString: dbConnectionString,
		DBDebug:            dbDebug,
	}

	return config
}

func checkErr(err error, logger *zap.SugaredLogger) {
	if err != nil {
		logger.Fatalf("An error occurred: %v", err)
	}
}
