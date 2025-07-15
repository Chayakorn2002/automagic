package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/bilbo290/automagic/pkg/enum"
	"github.com/bilbo290/automagic/pkg/handlers"
	"github.com/bilbo290/automagic/pkg/ui"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Issue management commands",
	Long:  `Commands for listing, selecting, and processing issues`,
}

var listIssuesCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues in the selected project",
	Long:  `List all issues in the currently selected project with optional label filtering`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.DefaultProjectPath == "" {
			return fmt.Errorf("no project selected. Please run: automagic project interactive")
		}

		filterLabel, _ := cmd.Flags().GetString("label")
		
		var labels []string
		if filterLabel != "" {
			labels = append(labels, filterLabel)
		}

		issues, err := providerInstance.GetProjectIssues(cfg.DefaultProjectPath, labels, "opened")
		if err != nil {
			return fmt.Errorf("error fetching issues: %v", err)
		}

		fmt.Printf("Found %d issues in project %s:\n\n", len(issues), cfg.DefaultProjectPath)
		for _, issue := range issues {
			fmt.Printf("Issue #%d: %s\n", issue.IID, issue.Title)
			fmt.Printf("  State: %s\n", issue.State)
			if len(issue.Labels) > 0 {
				fmt.Printf("  Labels: %s\n", strings.Join(issue.Labels, ", "))
			}
			fmt.Printf("  Author: %s\n", issue.Author.Name)
			if issue.Assignee.Name != "" {
				fmt.Printf("  Assignee: %s\n", issue.Assignee.Name)
			}
			fmt.Printf("  Created: %s\n", issue.CreatedAt)
			fmt.Printf("  URL: %s\n\n", issue.WebURL)
		}
		return nil
	},
}

var selectIssueCmd = &cobra.Command{
	Use:   "select",
	Short: "Interactive issue selection",
	Long:  `Interactively select an issue from the current project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.DefaultProjectPath == "" {
			return fmt.Errorf("no project selected. Please run: automagic project interactive")
		}

		filterLabel, _ := cmd.Flags().GetString("label")
		
		var labels []string
		if filterLabel != "" {
			labels = append(labels, filterLabel)
		}

		issues, err := providerInstance.GetProjectIssues(cfg.DefaultProjectPath, labels, "opened")
		if err != nil {
			return fmt.Errorf("error fetching issues: %v", err)
		}

		selectedIssue, err := ui.SelectIssue(issues)
		if err != nil {
			return fmt.Errorf("error selecting issue: %v", err)
		}

		fmt.Printf("Selected issue: #%d %s\n", selectedIssue.IID, selectedIssue.Title)
		fmt.Printf("You can now run: automagic issue process %d\n", selectedIssue.IID)
		return nil
	},
}

var processIssueCmd = &cobra.Command{
	Use:   "process <issue-number>",
	Short: "Process a specific issue",
	Long:  `Process a specific issue by number with optional run type`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		issueNumber, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid issue number: %v", err)
		}
		
		runTypeFlag, _ := cmd.Flags().GetString("run-type")
		
		runType, valid := enum.ParseRunType(runTypeFlag)
		if !valid {
			return fmt.Errorf("invalid run type '%s'. Valid types: %s", 
				runTypeFlag, strings.Join(func() []string {
					types := enum.GetValidValues()
					result := make([]string, len(types))
					for i, t := range types {
						result[i] = string(t)
					}
					return result
				}(), ", "))
		}

		return handlers.ProcessIssueWithRunType(issueNumber, cfg, runType)
	},
}

func init() {
	// Add flags
	listIssuesCmd.Flags().String("label", "", "Filter issues by label (solved, open, picked_up_by_claude)")
	selectIssueCmd.Flags().String("label", "", "Filter issues by label (solved, open, picked_up_by_claude)")
	processIssueCmd.Flags().String("run-type", "normal", "Execution mode: normal, dry-run, semi-dry-run")
	
	// Add subcommands
	issueCmd.AddCommand(listIssuesCmd)
	issueCmd.AddCommand(selectIssueCmd)
	issueCmd.AddCommand(processIssueCmd)
	
	rootCmd.AddCommand(issueCmd)
}