package cmd

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewUpdateCommand creates a new update command
func NewUpdateCommand() *cobra.Command {
	opts := &UpdateOptions{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the GoZap project to the specified environment",
		Long:  `Update the GoZap project to the specified environment using the provided configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Stage, "stage", "s", "", "Stage of the project (e.g., dev, prod)")
	cmd.MarkFlagRequired("stage")

	return cmd
}

func runUpdate(opts *UpdateOptions) error {
	fmt.Println("🔄 Updating GoZap project...")

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

	// Set the deployment file name
	currentTime := time.Now().Format("20060102150405")
	zipFileName := fmt.Sprintf("deployment-%s.zip", currentTime)

	// Update config with new S3Key
	stageConfig.S3Key = zipFileName
	stageConfig.Stage = opts.Stage
	config[opts.Stage] = stageConfig

	// 2. Check if the CloudFormation stack exists
	stackName := fmt.Sprintf("%s-%s", stageConfig.FunctionName, opts.Stage)
	if err := checkStackExists(stackName); err != nil {
		return err
	}

	// Create temporary directories
	tempDir := "bin"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Setup cleanup for local files only (not S3 yet)
	defer cleanupFiles([]string{zipFileName, "template.yaml", tempDir})

	// 3. Build the project
	if err := buildProject(tempDir); err != nil {
		return err
	}

	// 4. Zip the project
	if err := zipProject(zipFileName, filepath.Join(tempDir, "bootstrap")); err != nil {
		return err
	}

	// 5. Upload to S3
	if err := uploadToS3(zipFileName, stageConfig.S3Bucket, zipFileName); err != nil {
		return err
	}

	// Wait for the S3 upload to propagate
	fmt.Println("Waiting for S3 upload to propagate...")
	if err := waitForS3Object(stageConfig.S3Bucket, zipFileName); err != nil {
		return err
	}

	// 6. Generate CloudFormation template
	if err := generateTemplate("template.yaml", stageConfig); err != nil {
		return err
	}

	// 7. Update CloudFormation stack
	if err := updateStack(stackName); err != nil {
		return err
	}

	// 8. Wait for CloudFormation stack update to complete
	if err := waitForStackUpdate(stackName); err != nil {
		return err
	}

	// 9. Output the stack details
	if err := outputStackDetails(stackName); err != nil {
		return err
	}

	// 10. NOW it's safe to delete the zip file from S3
	if err := deleteFromS3(stageConfig.S3Bucket, zipFileName); err != nil {
		fmt.Printf("Warning: failed to clean up S3 file: %v\n", err)
	}

	fmt.Println("✅ Deployment updated successfully!")
	return nil
}

func waitForStackUpdate(stackName string) error {
	fmt.Printf("Waiting for stack '%s' update to complete...\n", stackName)

	waitCmd := exec.Command(
		"aws", "cloudformation", "wait", "stack-update-complete",
		"--stack-name", stackName,
	)

	if output, err := waitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("stack update failed or timed out: %w\n%s", err, output)
	}

	fmt.Println("Stack update completed successfully!")
	return nil
}

func readConfig(configFile string) (map[string]DeploymentConfig, error) {
	config := map[string]DeploymentConfig{}

	if _, err := os.Stat(configFile); err != nil {
		return nil, fmt.Errorf("config file not found: %w", err)
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

func checkStackExists(stackName string) error {
	fmt.Printf("Checking if stack '%s' exists...\n", stackName)
	describeStack := exec.Command("aws", "cloudformation", "describe-stacks", "--stack-name", stackName)
	if output, err := describeStack.CombinedOutput(); err != nil {
		return fmt.Errorf("❌ Stack does not exist or cannot be accessed: %w\n%s", err, output)
	}
	return nil
}

func buildProject(binDir string) error {
	fmt.Println("Building project...")
	build := exec.Command("go", "build", "-o", filepath.Join(binDir, "bootstrap"), "-ldflags", "-s -w", ".")
	build.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	if output, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build project: %w\n%s", err, output)
	}
	return nil
}

func zipProject(zipFileName, filePath string) error {
	fmt.Println("Creating deployment package...")
	zip := exec.Command("zip", "-j", zipFileName, filePath)
	if output, err := zip.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to zip project: %w\n%s", err, output)
	}
	return nil
}

func uploadToS3(localFile, bucket, key string) error {
	fmt.Printf("Uploading to S3 bucket '%s'...\n", bucket)
	s3Upload := exec.Command("aws", "s3", "cp", localFile, fmt.Sprintf("s3://%s/%s", bucket, key))
	if output, err := s3Upload.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to upload to S3: %w\n%s", err, output)
	}
	return nil
}

func generateTemplate(outFile string, config DeploymentConfig) error {
	fmt.Println("Generating CloudFormation template...")
	templateContent, err := templateFS.ReadFile("templates/template.yaml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	tmpl, err := template.New("cfTemplate").Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, config); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}

func updateStack(stackName string) error {
	fmt.Printf("Updating CloudFormation stack '%s'...\n", stackName)
	cloudformation := exec.Command(
		"aws", "cloudformation", "update-stack",
		"--stack-name", stackName,
		"--template-body", "file://template.yaml",
		"--capabilities", "CAPABILITY_NAMED_IAM",
	)
	if output, err := cloudformation.CombinedOutput(); err != nil {
		// Check if it's the "No updates are to be performed" error
		if string(output) != "" && len(output) > 0 {
			outputStr := string(output)
			if strings.Contains(outputStr, "No updates are to be performed") {
				fmt.Println("No changes detected in CloudFormation template")
				return nil
			}
		}
		return fmt.Errorf("failed to update stack: %w\n%s", err, output)
	}
	return nil
}

func outputStackDetails(stackName string) error {
	fmt.Printf("Fetching details for stack '%s'...\n", stackName)
	describeStack := exec.Command("aws", "cloudformation", "describe-stacks", "--stack-name", stackName)
	output, err := describeStack.Output()
	if err != nil {
		return fmt.Errorf("failed to describe stack: %w", err)
	}

	var response struct {
		Stacks []struct {
			Outputs []struct {
				OutputKey   string `json:"OutputKey"`
				OutputValue string `json:"OutputValue"`
			} `json:"Outputs"`
		} `json:"Stacks"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return fmt.Errorf("failed to parse stack details: %w", err)
	}

	if len(response.Stacks) == 0 || len(response.Stacks[0].Outputs) == 0 {
		return fmt.Errorf("no stack outputs found")
	}

	fmt.Println("\nStack Outputs:")
	for _, output := range response.Stacks[0].Outputs {
		fmt.Printf("  %s: %s\n", output.OutputKey, output.OutputValue)
	}

	return nil
}

func waitForS3Object(bucket, key string) error {
	fmt.Printf("Verifying S3 object '%s' exists...\n", key)
	maxAttempts := 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		cmd := exec.Command("aws", "s3api", "head-object", "--bucket", bucket, "--key", key)
		if _, err := cmd.CombinedOutput(); err == nil {
			fmt.Println("S3 object verified successfully!")
			return nil
		} else {
			fmt.Printf("Waiting for S3 object to be available (attempt %d/%d)...\n", attempt, maxAttempts)
			time.Sleep(2 * time.Second)
		}
	}
	return fmt.Errorf("S3 object not available after %d attempts", maxAttempts)
}

func deleteFromS3(bucket, key string) error {
	fmt.Printf("Cleaning up S3 file '%s'...\n", key)
	s3Delete := exec.Command("aws", "s3", "rm", fmt.Sprintf("s3://%s/%s", bucket, key))
	if output, err := s3Delete.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete from S3: %w\n%s", err, output)
	}
	return nil
}

func cleanupFiles(files []string) {
	fmt.Println("Cleaning up local files...")
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			var err error
			if info, _ := os.Stat(file); info.IsDir() {
				err = os.RemoveAll(file)
			} else {
				err = os.Remove(file)
			}
			if err != nil {
				fmt.Printf("Warning: failed to clean up '%s': %v\n", file, err)
			}
		}
	}
}
