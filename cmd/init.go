package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func NewInitCommand() *cobra.Command {
	opts := &InitOptions{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new GoZap project",
		Long:  `Initialize a new GoZap project with the specified name and stage.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.ProjectName == "" {
				return cmd.Help()
			}
			if opts.Stage == "" {
				return cmd.Help()
			}
			if len(args) > 0 {
				opts.ProjectName = args[0]
			}
			return runInit(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.ProjectName, "name", "n", "", "Name of the project")
	cmd.Flags().StringVarP(&opts.Stage, "stage", "s", "", "Stage of the project (e.g., dev, prod)")
	cmd.Flags().StringVarP(&opts.S3Bucket, "bucket", "b", "", "S3 bucket for deployment artifacts")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("stage")
	cmd.MarkFlagRequired("bucket")

	return cmd
}

func runInit(opts *InitOptions) error {
	fmt.Printf("🚀 Creating new GoZap project: %s (Environment: %s)\n", opts.ProjectName, opts.Stage)

	// Check if S3 Bucket exists
	s3Check := exec.Command("aws", "s3api", "head-bucket", "--bucket", opts.S3Bucket)
	if err := s3Check.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("S3 bucket %s does not exist: %w", opts.S3Bucket, err)
		}
		return fmt.Errorf("failed to check S3 bucket: %w", err)
	}

	// Step 1: Load existing config.json or initialize new map
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

	// Step 2: Update or add the current stage config
	config[opts.Stage] = DeploymentConfig{
		FunctionName: opts.ProjectName,
		S3Bucket:     opts.S3Bucket,
		S3Key:        "deployment.zip",
		Timeout:      30,
		Memory:       128,
		Stage:        opts.Stage,
	}

	// Step 3: Marshal and write back
	configJSON, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	if err := os.WriteFile(configFile, configJSON, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Println("✅ Project initialized successfully!")
	fmt.Println("📂 Project file(s) created:")
	fmt.Println("  - config.json")

	return nil
}
