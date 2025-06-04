package main

import (
	"os"

	"github.com/compliance-framework/configuration-service/cmd"
	// adjust as needed
)

// @title						Continuous Compliance Framework API
// @version					1
// @description				This is the API for the Continuous Compliance Framework.
// @host						localhost:8080
// @accept						json
// @produce					json
// @BasePath					/api
// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
