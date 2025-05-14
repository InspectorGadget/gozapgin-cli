package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func NewUndeployCommand() *cobra.Command {
	opts := &DeployOptions{}

	cmd := &cobra.Command{
		Use:   "undeploy",
		Short: "Undeploy a GoZap application",
		Long:  `Undeploy a GoZap application from the serverless environment.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUndeploy(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Stage, "stage", "s", "", "Stage of the project (e.g., dev, prod)")
	cmd.MarkFlagRequired("stage")

	return cmd
}

func runUndeploy(opts *DeployOptions) error {
	fmt.Println("🗑️  Undeploying GoZap application...")

	// 1. Load the configuration file
	config := map[string]DeploymentConfig{}
	configFile := "config.json"
	if _, err := os.Stat(configFile); err == nil {
		content, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		if err := json.Unmarshal(content, &config); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// 2. Validate the configuration
	if _, ok := config[opts.Stage]; !ok {
		return fmt.Errorf("no configuration found for the specified stage")
	}

	// 3. Check if the application is already deployed via CloudFormation stack
	cloudFormationStackName := fmt.Sprintf("%s-%s", config[opts.Stage].FunctionName, opts.Stage)
	checkStack := exec.Command("aws", "cloudformation", "describe-stacks", "--stack-name", cloudFormationStackName)
	if err := checkStack.Run(); err != nil {
		return fmt.Errorf("failed to check CloudFormation stack: %w", err)
	}

	// 4. Undeploy the application
	undeployStack := exec.Command("aws", "cloudformation", "delete-stack", "--stack-name", cloudFormationStackName)
	if err := undeployStack.Run(); err != nil {
		return fmt.Errorf("failed to undeploy CloudFormation stack: %w", err)
	}

	// 5. Wait for the stack to be deleted
	waitStack := exec.Command("aws", "cloudformation", "wait", "stack-delete-complete", "--stack-name", cloudFormationStackName)
	if err := waitStack.Run(); err != nil {
		return fmt.Errorf("failed to wait for CloudFormation stack deletion: %w", err)
	}

	// 6. Print success message
	fmt.Printf("✅ Successfully undeployed the application from stage %s\n", opts.Stage)

	return nil
}
