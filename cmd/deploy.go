package cmd

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

//go:embed templates/template.yaml.tmpl
var templateFS embed.FS

func NewDeployCommand() *cobra.Command {
	opts := &DeployOptions{}

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy the GoZap project to the specified environment",
		Long:  `Deploy the GoZap project to the specified environment using the provided configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Stage, "stage", "s", "", "Stage of the project (e.g., dev, prod)")
	cmd.MarkFlagRequired("stage")

	return cmd
}

func runDeploy(opts *DeployOptions) error {
	fmt.Println("🚀 Deploying GoZap project...")

	// 1. Read config file
	config, err := readConfig("config.json")
	if err != nil {
		return err
	}

	// Validate stage exists in config
	stageConfig, exists := config[opts.Stage]
	if !exists {
		return fmt.Errorf("❌ stage '%s' not found in configuration", opts.Stage)
	}

	// 2. Check if CloudFormation stack already exists
	stackName := fmt.Sprintf("%s-%s", stageConfig.FunctionName, opts.Stage)
	if err := checkStackExists(stackName); err == nil {
		return fmt.Errorf("❌ Stack '%s' already exists. Use 'update' command instead", stackName)
	}

	// Create temporary directories
	tempDir := "bin"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Set up cleanup for local files
	currentTime := time.Now().Format("20060102150405")
	zipFileName := fmt.Sprintf("deployment-%s.zip", currentTime)
	defer cleanupFiles([]string{zipFileName, "template.yaml", tempDir})

	// 3. Build the project
	if err := buildProject(tempDir); err != nil {
		return err
	}

	// 4. Zip the project
	if err := zipProject(zipFileName, filepath.Join(tempDir, "bootstrap")); err != nil {
		return err
	}

	// 5. Check if S3 bucket exists
	if err := checkS3Bucket(stageConfig.S3Bucket); err != nil {
		return err
	}

	// Update config with new S3Key
	stageConfig.S3Key = zipFileName
	stageConfig.Stage = opts.Stage

	// 6. Upload to S3
	if err := uploadToS3(zipFileName, stageConfig.S3Bucket, zipFileName); err != nil {
		return err
	}

	// Wait for the S3 upload to propagate
	fmt.Println("Waiting for S3 upload to propagate...")
	if err := waitForS3Object(stageConfig.S3Bucket, zipFileName); err != nil {
		return err
	}

	// 7. Generate CloudFormation template
	if err := generateTemplate("template.yaml", stageConfig); err != nil {
		return err
	}

	// 8. Deploy CloudFormation stack
	if err := deployStack(stackName); err != nil {
		return err
	}

	// 9. Wait for stack creation to complete
	if err := waitForStackCreation(stackName); err != nil {
		return err
	}

	// 10. Output the stack details
	if err := outputStackDetails(stackName); err != nil {
		return err
	}

	// 11. NOW it's safe to delete the zip file from S3
	if err := deleteFromS3(stageConfig.S3Bucket, zipFileName); err != nil {
		fmt.Printf("Warning: failed to clean up S3 file: %v\n", err)
	}

	fmt.Println("✅ Deployment complete!")
	return nil
}

func checkS3Bucket(bucket string) error {
	fmt.Printf("Checking if S3 bucket '%s' exists...\n", bucket)
	s3Check := exec.Command("aws", "s3api", "head-bucket", "--bucket", bucket)
	if output, err := s3Check.CombinedOutput(); err != nil {
		return fmt.Errorf("❌ S3 bucket '%s' does not exist or is not accessible: %w\n%s", bucket, err, output)
	}
	return nil
}

func deployStack(stackName string) error {
	fmt.Printf("Deploying CloudFormation stack '%s'...\n", stackName)
	deploy := exec.Command(
		"aws", "cloudformation", "deploy",
		"--template-file", "template.yaml",
		"--stack-name", stackName,
		"--capabilities", "CAPABILITY_NAMED_IAM",
	)
	if output, err := deploy.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to deploy CloudFormation stack: %w\n%s", err, output)
	}
	return nil
}

func waitForStackCreation(stackName string) error {
	fmt.Printf("Waiting for stack '%s' creation to complete...\n", stackName)

	waitCmd := exec.Command(
		"aws", "cloudformation", "wait", "stack-create-complete",
		"--stack-name", stackName,
	)

	if output, err := waitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("stack creation failed or timed out: %w\n%s", err, output)
	}

	fmt.Println("Stack creation completed successfully!")
	return nil
}
