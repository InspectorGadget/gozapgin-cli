package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

func NewUndeployCommand() *cobra.Command {
	opts := &UndeployOptions{}

	cmd := &cobra.Command{
		Use:   "undeploy",
		Short: "Undeploy a GoZap application",
		Long:  `Undeploy a GoZap application from the serverless environment by deleting the CloudFormation stack.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUndeploy(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Stage, "stage", "s", "", "Stage of the project (e.g., dev, prod)")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Skip confirmation prompt")
	cmd.MarkFlagRequired("stage")

	return cmd
}

func runUndeploy(opts *UndeployOptions) error {
	fmt.Printf("🗑️  Undeploying GoZap application for stage: %s\n", opts.Stage)

	// 1. Read config file
	config, err := readConfig("config.json")
	if err != nil {
		return err
	}

	// 2. Validate stage exists in config
	stageConfig, exists := config[opts.Stage]
	if !exists {
		return fmt.Errorf("❌ stage '%s' not found in configuration", opts.Stage)
	}

	// 3. Determine stack name
	stackName := fmt.Sprintf("%s-%s", stageConfig.FunctionName, opts.Stage)

	// 4. Check if the CloudFormation stack exists
	fmt.Printf("Checking if stack '%s' exists...\n", stackName)
	if err := checkStackExists(stackName); err != nil {
		return fmt.Errorf("❌ Stack '%s' does not exist or cannot be accessed", stackName)
	}

	// 5. Confirmation prompt (unless --force is used)
	if !opts.Force {
		fmt.Printf("\n⚠️  WARNING: This will permanently delete the following resources:\n")
		fmt.Printf("  - Lambda function: %s\n", stageConfig.FunctionName)
		fmt.Printf("  - CloudFormation stack: %s\n", stackName)
		fmt.Printf("  - All associated AWS resources\n\n")

		if !confirmAction("Are you sure you want to proceed with undeployment?") {
			fmt.Println("❌ Undeployment cancelled")
			return nil
		}
	}

	// 6. Delete the CloudFormation stack
	if err := deleteStack(stackName); err != nil {
		return err
	}

	// 7. Wait for stack deletion to complete
	if err := waitForStackDeletion(stackName); err != nil {
		return err
	}

	// 8. Success message
	fmt.Printf("✅ Successfully undeployed application from stage '%s'\n", opts.Stage)
	fmt.Println("💡 The configuration in config.json has been preserved for future deployments")

	return nil
}

func deleteStack(stackName string) error {
	fmt.Printf("Deleting CloudFormation stack '%s'...\n", stackName)

	deleteCmd := exec.Command("aws", "cloudformation", "delete-stack", "--stack-name", stackName)
	if output, err := deleteCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete CloudFormation stack: %w\n%s", err, output)
	}

	fmt.Println("Stack deletion initiated...")
	return nil
}

func waitForStackDeletion(stackName string) error {
	fmt.Printf("Waiting for stack '%s' deletion to complete...\n", stackName)

	waitCmd := exec.Command(
		"aws", "cloudformation", "wait", "stack-delete-complete",
		"--stack-name", stackName,
	)

	if output, err := waitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("stack deletion failed or timed out: %w\n%s", err, output)
	}

	fmt.Println("Stack deletion completed successfully!")
	return nil
}

func confirmAction(message string) bool {
	fmt.Printf("%s (y/N): ", message)
	var response string
	fmt.Scanln(&response)

	return response == "y" || response == "Y" || response == "yes" || response == "Yes"
}
