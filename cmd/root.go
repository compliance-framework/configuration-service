package cmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	DriverOptions = []string{"postgres"}
)

type Config struct {
	MongoURI           string
	AppPort            string
	DBDriver           string
	DBConnectionString string
	DBDebug            bool
}

var (
	rootCmd = &cobra.Command{
		Use:   "conf-service",
		Short: "Compliance Framework Configuration Service",
	}

	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the configuration service API",
		Run:   RunServer,
	}
)

func NewConfig(logger *zap.SugaredLogger) *Config {
	// for non-default but required variables, make sure the user is aware
	if !viper.IsSet("mongo_uri") {
		logger.Fatal("MONGO_URI is not set. Please set it in the environment or .env file.")
	}

	if !viper.IsSet("db_driver") {
		logger.Fatal(
			"CCF_DB_DRIVER is not set. Please set it in the environment or .env file. Expected values: ",
			strings.Join(DriverOptions, ", "),
		)
	}

	dbDriver := strings.ToLower(viper.GetString("db_driver"))

	if !slices.Contains(DriverOptions, dbDriver) {
		logger.Fatal(
			"CCF_DB_DRIVER is set to an unsupported value: ",
			viper.GetString("db_driver"),
			". Supported values are: ",
			strings.Join(DriverOptions, ", "),
		)
	}

	if !viper.IsSet("db_connection") {
		logger.Fatal("CCF_DB_CONNECTION is not set. Please set it in the environment or .env file.")
	}

	appPort := viper.GetString("app_port")
	if !strings.HasPrefix(appPort, ":") {
		appPort = ":" + appPort
	}

	return &Config{
		MongoURI:           viper.GetString("mongo_uri"),
		AppPort:            appPort,
		DBDriver:           dbDriver,
		DBConnectionString: viper.GetString("db_connection"),
		DBDebug:            viper.GetBool("db_debug"),
	}

}

func configSetDefaults() {
	viper.SetDefault("app_port", ":8080")
	viper.SetDefault("db_debug", "false")
}

func configEnvKeys() {
	viper.SetEnvPrefix("ccf")
	viper.BindEnv("mongo_uri", "MONGO_URI")
	viper.BindEnv("app_port", "APP_PORT")
	viper.BindEnv("db_driver")
	viper.BindEnv("db_connection")
	viper.BindEnv("db_debug")
}

func init() {
	if err := godotenv.Load(".env"); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			panic("Error loading .env file: " + err.Error())
		}
	}
	configSetDefaults()
	configEnvKeys()

	rootCmd.AddCommand(runCmd)

}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error executing root command:", err)
		return err
	}
	return nil
}
