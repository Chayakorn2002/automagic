package handlers

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bilbo290/automagic/pkg/claude"
	"github.com/bilbo290/automagic/pkg/config"
	"github.com/bilbo290/automagic/pkg/constants"
	"github.com/bilbo290/automagic/pkg/enum"
	"github.com/bilbo290/automagic/pkg/provider"
	"github.com/bilbo290/automagic/pkg/ui"
)

func RunInteractiveWorkflow(providerInstance provider.Provider, cfg *config.Config) error {
	fmt.Printf("=== Project Selection ===\n")
	projects, err := providerInstance.GetAccessibleProjects()
	if err != nil {
		return fmt.Errorf("error fetching projects: %v", err)
	}

	selectedProject, err := ui.SelectProject(projects)
	if err != nil {
		return fmt.Errorf("error selecting project: %v", err)
	}

	if err := config.SaveProjectSelection(selectedProject.PathWithNamespace); err != nil {
		fmt.Printf("Warning: Could not save project selection: %v\n", err)
	} else {
		fmt.Printf("Project selection saved to automagic.yaml\n")
	}

	fmt.Printf("\n=== Issue Filtering ===\n")
	labelFilter := ui.SelectLabelFilter()

	fmt.Printf("\n=== Issue Selection ===\n")
	var labels []string
	if labelFilter != "" {
		labels = append(labels, labelFilter)
		fmt.Printf("Fetching issues with label '%s' from project %s...\n", labelFilter, selectedProject.PathWithNamespace)
	} else {
		fmt.Printf("Fetching all open issues from project %s...\n", selectedProject.PathWithNamespace)
	}

	issues, err := providerInstance.GetProjectIssues(selectedProject.PathWithNamespace, labels, "opened")
	if err != nil {
		return fmt.Errorf("error fetching issues: %v", err)
	}

	if len(issues) == 0 {
		fmt.Printf("No issues found with the selected criteria.\n")
		return nil
	}

	fmt.Printf("Found %d issues:\n", len(issues))

	selectedIssue, err := ui.SelectIssue(issues)
	if err != nil {
		return fmt.Errorf("error selecting issue: %v", err)
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Selected Project: %s\n", selectedProject.PathWithNamespace)
	fmt.Printf("Selected Issue: #%d %s\n", selectedIssue.IID, selectedIssue.Title)
	fmt.Printf("Issue URL: %s\n", selectedIssue.WebURL)

	fmt.Printf("\nWhat would you like to do?\n")
	fmt.Printf("1. Process this issue now\n")
	fmt.Printf("2. Debug MCP integration for this issue\n")
	fmt.Printf("3. Exit\n")
	fmt.Printf("Enter your choice (1-3): ")

	var choice int
	fmt.Scanf("%d", &choice)

	switch choice {
	case 1:
		return ProcessIssue(selectedIssue.IID, cfg)
	case 2:
		return DebugMCPForIssue(selectedIssue.IID, cfg)
	case 3:
		fmt.Printf("You can process this issue later with: go run main.go -issue %d\n", selectedIssue.IID)
		return nil
	default:
		fmt.Printf("Invalid choice. You can process this issue later with: go run main.go -issue %d\n", selectedIssue.IID)
		return nil
	}
}

func ProcessIssue(issueNumber int, cfg *config.Config) error {
	return ProcessIssueWithRunType(issueNumber, cfg, enum.RunTypeNormal)
}

func ProcessIssueWithRunType(issueNumber int, cfg *config.Config, runType enum.RunType) error {
	processManager := claude.NewProcessManager()

	fmt.Printf("Processing issue #%d with run type: %s...\n", issueNumber, runType)

	processID := fmt.Sprintf("issue-%d-%d", issueNumber, time.Now().Unix())
	shouldCloneRepo := runType.RequiresRepositoryClone()

	process, err := claude.CreateProcessWithCallbackAndProviderDryRun(
		issueNumber,
		processID,
		cfg.ClaudeCommand,
		cfg.ClaudeFlags,
		cfg.DefaultProjectPath,
		cfg.ProviderUsername,
		cfg.ProviderURL,
		cfg.ProviderType,
		!shouldCloneRepo,
		nil,
		nil,
	)
	if err != nil {
		return fmt.Errorf("error creating claude process: %v", err)
	}

	if runType.IsDryRun() {
		return handleDryRun(runType, process)
	}

	processManager.AddProcess(process)

	if err := claude.RunProcess(process); err != nil {
		return fmt.Errorf("error executing claude command: %v", err)
	}

	return nil
}

func handleDryRun(runType enum.RunType, process *claude.Process) error {
	switch runType {
	case enum.RunTypeDryRun:
		fmt.Println("\n=== DRY RUN MODE ===")
	case enum.RunTypeSemiDryRun:
		fmt.Println("\n=== SEMI-DRY RUN MODE ===")
		fmt.Println("Repository has been cloned/verified.")
	}

	fmt.Printf("Working Directory: %s\n", process.Cmd.Dir)
	fmt.Printf("Command: %s\n", process.Cmd.Path)
	fmt.Printf("Arguments: %v\n", process.Cmd.Args)
	fmt.Println("\n=== PROMPT ===")

	for i, arg := range process.Cmd.Args {
		if arg == "-p" && i+1 < len(process.Cmd.Args) {
			fmt.Println(process.Cmd.Args[i+1])
			break
		}
	}

	switch runType {
	case enum.RunTypeDryRun:
		fmt.Println("=== END DRY RUN ===")
	case enum.RunTypeSemiDryRun:
		fmt.Println("=== END SEMI-DRY RUN ===")
		showRepositoryStatus(process)
	}
	return nil
}

func showRepositoryStatus(process *claude.Process) {
	fmt.Println("\n=== REPOSITORY STATUS ===")

	statusCmd := exec.Command("git", "status", "--short")
	statusCmd.Dir = process.Cmd.Dir
	if output, err := statusCmd.Output(); err == nil {
		if len(output) == 0 {
			fmt.Println("Git status: Clean working directory")
		} else {
			fmt.Printf("Git status:\n%s", output)
		}
	}

	branchCmd := exec.Command("git", "branch", "--show-current")
	branchCmd.Dir = process.Cmd.Dir
	if output, err := branchCmd.Output(); err == nil {
		fmt.Printf("Current branch: %s", output)
	}

	fmt.Printf("\n=== REPOSITORY CLEANUP ===\n")
	fmt.Printf("After Claude finishes, the following cleanup would occur:\n")
	fmt.Printf("- Reset any uncommitted changes (git reset --hard HEAD)\n")
	fmt.Printf("- Remove untracked files (git clean -fd)\n")
	fmt.Printf("- Switch back to main branch\n")
	fmt.Printf("- Delete any issue-* branches\n")
	fmt.Printf("- Pull latest changes\n")
	fmt.Printf("Repository will be ready for the next parallel session\n")
}

func DebugMCPForIssue(issueNumber int, cfg *config.Config) error {
	fmt.Printf("Starting MCP debug session for issue #%d...\n", issueNumber)

	projectPath := cfg.DefaultProjectPath
	if projectPath == "" {
		fmt.Println("No project configured. Please select a project:")
		return fmt.Errorf("no project configured for MCP debug")
	}

	processManager := claude.NewProcessManager()

	process, err := claude.CreateMCPDebugProcess(issueNumber, cfg, projectPath)
	if err != nil {
		return fmt.Errorf("error creating MCP debug process: %v", err)
	}

	processManager.AddProcess(process)

	if err := claude.RunProcess(process); err != nil {
		return fmt.Errorf("error running MCP debug process: %v", err)
	}

	return nil
}

func TestLabelFiltering(providerInstance provider.Provider, cfg *config.Config) error {
	if cfg.DefaultProjectPath == "" {
		return fmt.Errorf("no project configured. Please run with -interactive first")
	}

	fmt.Printf("Testing label filtering for project: %s\n\n", cfg.DefaultProjectPath)

	fmt.Printf("=== Test 1: All Open Issues ===\n")
	allIssues, err := providerInstance.GetProjectIssues(cfg.DefaultProjectPath, []string{}, "opened")
	if err != nil {
		return fmt.Errorf("failed to fetch all issues: %v", err)
	}

	fmt.Printf("Found %d total open issues:\n", len(allIssues))
	for _, issue := range allIssues {
		fmt.Printf("  #%d: %s\n", issue.IID, issue.Title)
		fmt.Printf("    Labels: [%s]\n", strings.Join(issue.Labels, ", "))
		fmt.Printf("    State: %s\n\n", issue.State)
	}

	fmt.Printf("=== Test 2: Issues with '%s' label ===\n", cfg.ClaudeLabel)
	claudeIssues, err := providerInstance.GetProjectIssues(cfg.DefaultProjectPath, []string{cfg.ClaudeLabel}, "opened")
	if err != nil {
		return fmt.Errorf("failed to fetch claude issues: %v", err)
	}

	fmt.Printf("Found %d issues with '%s' label:\n", len(claudeIssues), cfg.ClaudeLabel)
	for _, issue := range claudeIssues {
		fmt.Printf("  #%d: %s\n", issue.IID, issue.Title)
		fmt.Printf("    Labels: [%s]\n", strings.Join(issue.Labels, ", "))
	}

	fmt.Printf("\n=== Test 3: Manual Filter Check ===\n")
	fmt.Printf("Manually filtering all issues for label '%s':\n", cfg.ClaudeLabel)

	manualCount := 0
	for _, issue := range allIssues {
		for _, label := range issue.Labels {
			if label == cfg.ClaudeLabel {
				manualCount++
				fmt.Printf("  Manual match #%d: %s\n", issue.IID, issue.Title)
				break
			}
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("- API filtered results: %d issues\n", len(claudeIssues))
	fmt.Printf("- Manual filtering: %d issues\n", manualCount)

	if len(claudeIssues) != manualCount {
		fmt.Printf("⚠️  MISMATCH! API filtering may not be working correctly.\n")
	} else {
		fmt.Printf("✅ API filtering matches manual filtering.\n")
	}

	return nil
}

func RunProjectListing(providerInstance provider.Provider, keyword, visibility string) error {
	selectedOrgs, err := ui.SelectOrganizations(providerInstance)
	if err != nil {
		fmt.Printf("Error during organization selection: %v\n", err)
		os.Exit(1)
	}

	options := provider.ProjectListOptions{
		Organizations: selectedOrgs,
		Keyword:       keyword,
		Visibility:    visibility,
	}

	projects, err := providerInstance.GetAccessibleProjectsWithOptions(options)
	if err != nil {
		fmt.Printf("Error fetching projects: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== Project Listing Results ===\n")
	if len(options.Organizations) > 0 {
		fmt.Printf("Organizations: %s\n", strings.Join(options.Organizations, ", "))
	}
	if options.Keyword != "" {
		fmt.Printf("Keyword filter: %s\n", options.Keyword)
	}
	if options.Visibility != "" {
		fmt.Printf("Visibility filter: %s\n", options.Visibility)
	}
	fmt.Printf("\nFound %d projects:\n\n", len(projects))

	if len(projects) == 0 {
		fmt.Printf("No projects match your criteria.\n")
		return nil
	}

	for _, project := range projects {
		fmt.Printf("Name: %s\n", project.Name)
		fmt.Printf("Path: %s\n", project.PathWithNamespace)
		if project.Description != "" {
			fmt.Printf("Description: %s\n", project.Description)
		}
		fmt.Printf("Visibility: %s\n", project.Visibility)
		fmt.Printf("URL: %s\n", project.WebURL)
		fmt.Printf("Last Activity: %s\n\n", project.LastActivityAt)
	}

	return nil
}

func RunGenerateConfigTemplate() error {
	return os.WriteFile(".env", []byte(constants.ConfigTemplate), 0644)
}
