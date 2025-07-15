package cmd

import (
	"fmt"

	"github.com/bilbo290/automagic/pkg/handlers"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  `Commands for managing automagic configuration`,
}

var generateConfigCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a template configuration file",
	Long:  `Generate a template .env configuration file with all available options`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := handlers.RunGenerateConfigTemplate(); err != nil {
			return fmt.Errorf("error generating config template: %v", err)
		}
		fmt.Println("Generated .env template file. Please edit it with your provider credentials.")
		return nil
	},
}

func init() {
	configCmd.AddCommand(generateConfigCmd)
	rootCmd.AddCommand(configCmd)
}
