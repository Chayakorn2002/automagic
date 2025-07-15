package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/bilbo290/automagic/pkg/claude"
	"github.com/bilbo290/automagic/pkg/handlers"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug and testing commands",
	Long:  `Commands for debugging and testing various functionality`,
}

var debugMCPCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Debug MCP (Model Context Protocol) integration",
	Long:  `Test the MCP integration with the current project configuration`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath := cfg.DefaultProjectPath
		if projectPath == "" {
			return fmt.Errorf("no project configured. Please run: automagic project interactive")
		}

		return claude.TestProviderMCPIntegration(cfg, providerInstance, projectPath)
	},
}

var debugMCPIssueCmd = &cobra.Command{
	Use:   "mcp-issue <issue-number>",
	Short: "Debug MCP integration for a specific issue",
	Long:  `Start MCP debug session for a specific issue`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		issueNumber, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid issue number: %v", err)
		}
		
		return handlers.DebugMCPForIssue(issueNumber, cfg)
	},
}

var testLabelsCmd = &cobra.Command{
	Use:   "labels",
	Short: "Test label filtering functionality",
	Long:  `Test the label filtering functionality against the current project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handlers.TestLabelFiltering(providerInstance, cfg)
	},
}

var listOrgsCmd = &cobra.Command{
	Use:   "orgs",
	Short: "List user's organizations/groups",
	Long:  `List all organizations/groups that the user has access to`,
	RunE: func(cmd *cobra.Command, args []string) error {
		orgs, err := providerInstance.GetUserOrganizations()
		if err != nil {
			return fmt.Errorf("error fetching organizations: %v", err)
		}

		if len(orgs) == 0 {
			fmt.Printf("No organizations found for user %s\n", cfg.ProviderUsername)
		} else {
			fmt.Printf("Found %d organizations for user %s:\n\n", len(orgs), cfg.ProviderUsername)
			for _, org := range orgs {
				fmt.Printf("Login: %s\nName: %s\nDescription: %s\nWebURL: %s\n\n",
					org.Login, org.Name, org.Description, org.WebURL)
			}
		}
		return nil
	},
}

func init() {
	// Add subcommands
	debugCmd.AddCommand(debugMCPCmd)
	debugCmd.AddCommand(debugMCPIssueCmd)
	debugCmd.AddCommand(testLabelsCmd)
	debugCmd.AddCommand(listOrgsCmd)
	
	rootCmd.AddCommand(debugCmd)
}