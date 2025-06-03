package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

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

func init() {
	rootCmd.AddCommand(runCmd)
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error executing root command:", err)
		return err
	}
	return nil
}
