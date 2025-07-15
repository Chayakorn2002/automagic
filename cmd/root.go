package cmd

import (
	"fmt"
	"os"

	"github.com/bilbo290/automagic/pkg/config"
	"github.com/bilbo290/automagic/pkg/provider"
	"github.com/bilbo290/automagic/pkg/version"
	"github.com/spf13/cobra"
)

var (
	cfg              *config.Config
	providerInstance provider.Provider
)

var rootCmd = &cobra.Command{
	Use:   "automagic",
	Short: "Automagic Multi-Provider Automation (GitLab & GitHub)",
	Long: `Automagic is a CLI tool that automates issue processing for GitLab and GitHub.
It can run in interactive mode to help you select projects and issues, or in daemon mode
to automatically process issues with specific labels.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Print version info
		version.PrintVersionInfo()

		// Skip config loading for certain commands
		if cmd.Name() == "generate" || cmd.Name() == "version" || cmd.Name() == "help" {
			return nil
		}

		// Load and validate configuration
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("error loading configuration: %v", err)
		}

		if err := config.Validate(cfg); err != nil {
			return fmt.Errorf("configuration error: %v", err)
		}

		// Create provider instance
		factory := provider.NewProviderFactory()
		providerConfig := cfg.GetProviderConfig()

		providerInstance, err = factory.CreateProvider(providerConfig)
		if err != nil {
			return fmt.Errorf("error creating provider: %v", err)
		}

		// Test connection
		fmt.Printf("Testing %s connection...\n", providerConfig.Type)
		if err := providerInstance.TestConnection(); err != nil {
			return fmt.Errorf("%s connection test failed: %v", providerConfig.Type, err)
		}
		fmt.Printf("%s connection successful!\n", providerConfig.Type)

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
