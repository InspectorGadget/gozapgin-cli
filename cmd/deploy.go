package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"

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

	// 1. Build the project
	build := exec.Command("go", "build", "-o", "bin/bootstrap", "-ldflags", "-s -w", ".")
	build.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	if err := build.Run(); err != nil {
		return fmt.Errorf("failed to build project: %w", err)
	}

	// 2. Zip the project
	zip := exec.Command("zip", "-j", "deployment.zip", "bin/bootstrap")
	if err := zip.Run(); err != nil {
		return fmt.Errorf("failed to zip project: %w", err)
	}

	// 3. Read config file
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

	// 4. Create CloudFormation file
	if _, ok := config[opts.Stage]; !ok {
		return fmt.Errorf("stage %s not found in config.json", opts.Stage)
	}

	// 5. Parse the deployment configuration and generate the CloudFormation template
	templateContent, err := templateFS.ReadFile("templates/template.yaml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	tmpl, err := template.New("cfTemplate").Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	outFile, err := os.Create("template.yaml")
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if err := tmpl.Execute(outFile, config[opts.Stage]); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// 6. Check if S3 is created
	s3Check := exec.Command("aws", "s3api", "head-bucket", "--bucket", config[opts.Stage].S3Bucket)
	if err := s3Check.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("S3 bucket %s does not exist: %w", config[opts.Stage].S3Bucket, err)
		}
		return fmt.Errorf("failed to check S3 bucket: %w", err)
	}

	// 5. Upload the zip file to S3
	s3Upload := exec.Command("aws", "s3", "cp", "deployment.zip", fmt.Sprintf("s3://%s/%s", config[opts.Stage].S3Bucket, config[opts.Stage].S3Key))
	if err := s3Upload.Run(); err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	// 6. Deploy the CloudFormation stack
	stackName := fmt.Sprintf("%s-%s", config[opts.Stage].FunctionName, opts.Stage)
	deploy := exec.Command("aws", "cloudformation", "deploy", "--template-file", "template.yaml", "--stack-name", stackName, "--capabilities", "CAPABILITY_NAMED_IAM")
	if err := deploy.Run(); err != nil {
		return fmt.Errorf("failed to deploy CloudFormation stack: %w", err)
	}

	// 7. Wait for stack creation to complete
	wait := exec.Command("aws", "cloudformation", "wait", "stack-create-complete", "--stack-name", stackName)
	if err := wait.Run(); err != nil {
		return fmt.Errorf("failed to wait for stack creation: %w", err)
	}

	// 8. Output the stack details
	describe := exec.Command("aws", "cloudformation", "describe-stacks", "--stack-name", stackName)
	output, err := describe.Output()
	if err != nil {
		return fmt.Errorf("failed to describe stack: %w", err)
	}

	// 9. Parse the output to get the stack details
	var stackDetails map[string]any
	if err := json.Unmarshal(output, &stackDetails); err != nil {
		return fmt.Errorf("failed to parse stack details: %w", err)
	}

	fmt.Println("Stack Details:")
	// Get the stack outputs
	if outputs, ok := stackDetails["Stacks"].([]any); ok && len(outputs) > 0 {
		if stack, ok := outputs[0].(map[string]any); ok {
			if stackOutputs, ok := stack["Outputs"].([]any); ok {
				for _, output := range stackOutputs {
					if outputMap, ok := output.(map[string]any); ok {
						if outputKey, ok := outputMap["OutputKey"].(string); ok {
							if outputValue, ok := outputMap["OutputValue"].(string); ok {
								fmt.Printf("  %s: %s\n", outputKey, outputValue)
							}
						}
					}
				}
			}
		}
	} else {
		return fmt.Errorf("failed to get stack outputs")
	}

	// 10. Output the stack details
	fmt.Println("✅ Deployment complete!")

	return nil
}
