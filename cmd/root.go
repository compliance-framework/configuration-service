package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

func NewConfig() *Config {
	return &Config{
		MongoURI:           viper.GetString("mongo_uri"),
		AppPort:            viper.GetString("app_port"),
		DBDriver:           viper.GetString("db_driver"),
		DBConnectionString: viper.GetString("db_connection"),
		DBDebug:            viper.GetBool("db_debug"),
	}

}

func configSetDefaults() {
	viper.SetDefault("app_port", "8080")
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
