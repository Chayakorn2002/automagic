package cmd

import (
	"fmt"

	"github.com/bilbo290/automagic/pkg/handlers"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Project management commands",
	Long:  `Commands for managing and interacting with projects`,
}

var listProjectsCmd = &cobra.Command{
	Use:   "list",
	Short: "List accessible projects",
	Long:  `List all accessible projects with optional filtering by keyword and visibility`,
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword, _ := cmd.Flags().GetString("keyword")
		visibility, _ := cmd.Flags().GetString("visibility")

		return handlers.RunProjectListing(providerInstance, keyword, visibility)
	},
}

var searchProjectsCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for projects by name",
	Long:  `Search for projects by name using the provided query`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		searchQuery := args[0]

		projects, err := providerInstance.SearchProjects(searchQuery)
		if err != nil {
			return fmt.Errorf("error searching projects: %v", err)
		}

		fmt.Printf("Found %d projects matching '%s':\n\n", len(projects), searchQuery)
		for _, project := range projects {
			fmt.Printf("ID: %d\nName: %s\nPath: %s\nDescription: %s\nWebURL: %s\n\n",
				project.ID, project.Name, project.PathWithNamespace, project.Description, project.WebURL)
		}
		return nil
	},
}

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Interactive project and issue selection",
	Long:  `Launch interactive mode to select projects and issues`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handlers.RunInteractiveWorkflow(providerInstance, cfg)
	},
}

func init() {
	// Add flags to list command
	listProjectsCmd.Flags().String("keyword", "", "Filter projects by keyword (searches name, path, and description)")
	listProjectsCmd.Flags().String("visibility", "", "Filter projects by visibility (public, private, internal)")

	// Add subcommands
	projectCmd.AddCommand(listProjectsCmd)
	projectCmd.AddCommand(searchProjectsCmd)
	projectCmd.AddCommand(interactiveCmd)

	rootCmd.AddCommand(projectCmd)
}
