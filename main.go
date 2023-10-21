package main

import (
	"log"
	"os"
	"sync"

	"github.com/Valgard/godotenv"
	"github.com/compliance-framework/configuration-service/internal/jobs"
	"github.com/compliance-framework/configuration-service/internal/server"
	"github.com/compliance-framework/configuration-service/internal/stores/mongo"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

const (
	DefaultMongoURI = "mongodb://localhost:27017"
	DefaultNATSURI  = "nats://localhost:4222"
	DefaultPort     = ":8080"
)

type Config struct {
	MongoURI string
	NATSURI  string
}

func main() {
	var wg sync.WaitGroup
	config := loadConfig()

	driver := &mongo.MongoDriver{Url: config.MongoURI, Database: "cf"}
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Can't initialize zap logger: %v", err)
	}
	sugar := logger.Sugar()
	e := echo.New()
	e.Use(middleware.Logger())

	job := jobs.RuntimeJobManager{Log: sugar, Driver: driver}
	checkErr(job.Init())
	wg.Add(1)
	go func() {
		defer wg.Done()
		job.Run()
	}()

	sub := jobs.EventSubscriber{Log: sugar}
	checkErr(sub.Connect(config.NATSURI))
	ch := sub.Subscribe("assessment.result")

	process := jobs.EventProcessor{Log: sugar, Driver: driver}
	checkErr(process.Init(ch))
	wg.Add(1)
	go func() {
		defer wg.Done()
		process.Run()
	}()

	pub := jobs.EventPublisher{Log: sugar}
	checkErr(pub.Connect(config.NATSURI))
	wg.Add(1)
	go func() {
		defer wg.Done()
		pub.Run()
	}()

	sv := server.Server{Driver: driver}
	checkErr(sv.RegisterOSCAL(e))
	checkErr(sv.RegisterRuntime(e))
	checkErr(sv.RegisterProcess(e))

	checkErr(e.Start(DefaultPort))

	wg.Wait()
}

func loadConfig() (config Config) {
	dotenv := godotenv.New()
	if err := dotenv.Load(".env"); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = DefaultMongoURI
	}

	natsURI := os.Getenv("NATS_URI")
	if natsURI == "" {
		natsURI = DefaultNATSURI
	}

	config = Config{
		MongoURI: mongoURI,
		NATSURI:  natsURI,
	}
	return config
}

func checkErr(err error) {
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}
}
