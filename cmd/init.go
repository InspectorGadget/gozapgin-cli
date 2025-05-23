package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewInitCommand() *cobra.Command {
	opts := &InitOptions{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new GoZap project",
		Long:  `Initialize a new GoZap project with the specified name and stage.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If project name is provided as argument, use it
			if len(args) > 0 {
				opts.ProjectName = args[0]
			}

			// Validate required fields
			if opts.ProjectName == "" {
				return fmt.Errorf("❌ project name is required")
			}
			if opts.Stage == "" {
				return fmt.Errorf("❌ stage is required")
			}
			if opts.S3Bucket == "" {
				return fmt.Errorf("❌ S3 bucket is required")
			}

			return runInit(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.ProjectName, "name", "n", "", "Name of the project")
	cmd.Flags().StringVarP(&opts.Stage, "stage", "s", "", "Stage of the project (e.g., dev, prod)")
	cmd.Flags().StringVarP(&opts.S3Bucket, "bucket", "b", "", "S3 bucket for deployment artifacts")
	cmd.Flags().IntVarP(&opts.Timeout, "timeout", "t", 30, "Lambda function timeout in seconds")
	cmd.Flags().IntVarP(&opts.Memory, "memory", "m", 128, "Lambda function memory in MB")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("stage")
	cmd.MarkFlagRequired("bucket")

	return cmd
}

func runInit(opts *InitOptions) error {
	fmt.Printf("🚀 Initializing GoZap project: %s (Stage: %s)\n", opts.ProjectName, opts.Stage)

	// 1. Check if S3 bucket exists and is accessible
	if err := checkS3Bucket(opts.S3Bucket); err != nil {
		return err
	}

	// 2. Load existing config.json or initialize new map
	config, err := loadOrCreateConfig("config.json")
	if err != nil {
		return err
	}

	// 3. Check if stage already exists
	if _, exists := config[opts.Stage]; exists {
		return fmt.Errorf("❌ stage '%s' already exists in config.json. Use a different stage name or update the existing configuration", opts.Stage)
	}

	// 4. Create new stage configuration with provided values
	config[opts.Stage] = DeploymentConfig{
		FunctionName: fmt.Sprintf("%s-%s", opts.ProjectName, opts.Stage),
		S3Bucket:     opts.S3Bucket,
		S3Key:        "", // Will be set during deployment
		Timeout:      opts.Timeout,
		Memory:       opts.Memory,
		Stage:        opts.Stage,
	}

	// 5. Write config back to file
	if err := writeConfig("config.json", config); err != nil {
		return err
	}

	// 6. Display summary
	fmt.Println("✅ Project initialized successfully!")
	fmt.Println("📂 Configuration created:")
	fmt.Printf("  - Function Name: %s\n", config[opts.Stage].FunctionName)
	fmt.Printf("  - Stage: %s\n", config[opts.Stage].Stage)
	fmt.Printf("  - S3 Bucket: %s\n", config[opts.Stage].S3Bucket)
	fmt.Printf("  - Timeout: %d seconds\n", config[opts.Stage].Timeout)
	fmt.Printf("  - Memory: %d MB\n", config[opts.Stage].Memory)
	fmt.Println("📝 Config saved to: config.json")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Run 'gozap deploy --stage %s' to deploy your Lambda function\n", opts.Stage)
	fmt.Printf("  2. Run 'gozap update --stage %s' to update an existing deployment\n", opts.Stage)

	return nil
}

// Helper function to load existing config or create new one
func loadOrCreateConfig(configFile string) (map[string]DeploymentConfig, error) {
	config := map[string]DeploymentConfig{}

	if _, err := os.Stat(configFile); err == nil {
		fmt.Println("📖 Loading existing config.json...")
		content, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err := json.Unmarshal(content, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	} else {
		fmt.Println("📝 Creating new config.json...")
	}

	return config, nil
}

// Helper function to write config to file
func writeConfig(configFile string, config map[string]DeploymentConfig) error {
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	if err := os.WriteFile(configFile, configJSON, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
