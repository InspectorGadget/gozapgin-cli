package main

import (
	"os"

	"github.com/InspectorGadget/ginzapgin-cmd/cmd"
	"github.com/spf13/cobra"
)

var (
	Version    = "0.0.1"
	Verbose    bool
	ConfigPath string
)

var rootCmd = &cobra.Command{
	Use:     "gozap",
	Short:   "GoZap - Deploy Go applications to serverless environments",
	Long:    `GoZap is a serverless framework for Go applications that makes it easy to build and deploy to platforms like AWS Lambda. Compatible with popular frameworks like Gin.`,
	Version: Version,
}

func init() {
	rootCmd.AddCommand(cmd.NewInitCommand())
	rootCmd.AddCommand(cmd.NewDeployCommand())

	rootCmd.SetVersionTemplate("GoZap Version: {{.Version}}\n")
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
