package daemon

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/bilbo290/automagic/pkg/claude"
	"github.com/bilbo290/automagic/pkg/enum"
	"github.com/bilbo290/automagic/pkg/provider"
)

// processIssueWithLabelUpdate processes an issue and updates its labels
func (d *Daemon) processIssueWithLabelUpdate(issue *provider.Issue) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	d.logProcessingStart(timestamp, issue)

	if err := d.updateIssueLabels(issue, timestamp); err != nil {
		return err
	}

	// Process the issue asynchronously with completion callback
	if err := d.processIssueAsync(issue.IID); err != nil {
		fmt.Printf("[%s] Error starting process for issue #%d: %v\n", time.Now().Format("2006-01-02 15:04:05"), issue.IID, err)
		return err
	}

	fmt.Printf("[%s] Started processing issue #%d\n", time.Now().Format("2006-01-02 15:04:05"), issue.IID)
	return nil
}

// logProcessingStart logs the start of issue processing based on run type
func (d *Daemon) logProcessingStart(timestamp string, issue *provider.Issue) {
	switch d.runType {
	case enum.RunTypeDryRun:
		fmt.Printf("[%s] [DRY RUN] Would process issue #%d: %s\n", timestamp, issue.IID, issue.Title)
	case enum.RunTypeSemiDryRun:
		fmt.Printf("[%s] [SEMI-DRY RUN] Processing issue #%d: %s\n", timestamp, issue.IID, issue.Title)
	default:
		fmt.Printf("[%s] Processing issue #%d: %s\n", timestamp, issue.IID, issue.Title)
	}
}

// updateIssueLabels updates the labels for an issue being processed
func (d *Daemon) updateIssueLabels(issue *provider.Issue, timestamp string) error {
	// Check if this is a human review response
	isHumanReviewResponse := d.isHumanReviewResponse(issue)

	// Remove both claude and waiting_human_review labels, add process label
	newLabels := d.buildNewLabels(issue)

	return d.applyLabelUpdate(issue.IID, newLabels, isHumanReviewResponse, timestamp)
}

// isHumanReviewResponse checks if the issue has the waiting_human_review label
func (d *Daemon) isHumanReviewResponse(issue *provider.Issue) bool {
	for _, label := range issue.Labels {
		if label == d.config.ReviewLabel {
			return true
		}
	}
	return false
}

// buildNewLabels creates the new label set for the issue
func (d *Daemon) buildNewLabels(issue *provider.Issue) []string {
	newLabels := make([]string, 0)
	for _, label := range issue.Labels {
		// Remove both claude and waiting_human_review labels
		if label != d.config.ClaudeLabel && label != d.config.ReviewLabel {
			newLabels = append(newLabels, label)
		}
	}
	// Always add picked_up_by_claude label
	newLabels = append(newLabels, d.config.ProcessLabel)
	return newLabels
}

// applyLabelUpdate applies the label update based on run type
func (d *Daemon) applyLabelUpdate(issueIID int, newLabels []string, isHumanReviewResponse bool, timestamp string) error {
	switch d.runType {
	case enum.RunTypeDryRun:
		d.logDryRunLabelUpdate(isHumanReviewResponse, timestamp)
	case enum.RunTypeSemiDryRun:
		d.logSemiDryRunLabelUpdate(isHumanReviewResponse, timestamp)
	default:
		return d.actuallyUpdateLabels(issueIID, newLabels, isHumanReviewResponse, timestamp)
	}
	return nil
}

// logDryRunLabelUpdate logs what would happen in dry run mode
func (d *Daemon) logDryRunLabelUpdate(isHumanReviewResponse bool, timestamp string) {
	if isHumanReviewResponse {
		fmt.Printf("[%s] [DRY RUN] Would update labels: remove '%s', add '%s' (human review response)\n",
			timestamp, d.config.ReviewLabel, d.config.ProcessLabel)
	} else {
		fmt.Printf("[%s] [DRY RUN] Would update labels: remove '%s', add '%s'\n",
			timestamp, d.config.ClaudeLabel, d.config.ProcessLabel)
	}
}

// logSemiDryRunLabelUpdate logs what would happen in semi-dry run mode
func (d *Daemon) logSemiDryRunLabelUpdate(isHumanReviewResponse bool, timestamp string) {
	if isHumanReviewResponse {
		fmt.Printf("[%s] [SEMI-DRY RUN] Would update labels: remove '%s', add '%s' (human review response)\n",
			timestamp, d.config.ReviewLabel, d.config.ProcessLabel)
	} else {
		fmt.Printf("[%s] [SEMI-DRY RUN] Would update labels: remove '%s', add '%s'\n",
			timestamp, d.config.ClaudeLabel, d.config.ProcessLabel)
	}
}

// actuallyUpdateLabels performs the actual label update
func (d *Daemon) actuallyUpdateLabels(issueIID int, newLabels []string, isHumanReviewResponse bool, timestamp string) error {
	if err := d.provider.UpdateIssueLabels(d.selectedProject, issueIID, newLabels); err != nil {
		return fmt.Errorf("failed to update issue labels: %v", err)
	}

	if isHumanReviewResponse {
		fmt.Printf("[%s] Updated labels: removed '%s', added '%s' (continuing human review loop)\n",
			timestamp, d.config.ReviewLabel, d.config.ProcessLabel)
	} else {
		fmt.Printf("[%s] Updated labels: removed '%s', added '%s'\n",
			timestamp, d.config.ClaudeLabel, d.config.ProcessLabel)
	}
	return nil
}

// processIssue processes a single issue (legacy method for backward compatibility)
func (d *Daemon) processIssue(issueNumber int) error {
	processManager := claude.NewProcessManager()

	fmt.Printf("Processing issue #%d...\n", issueNumber)

	processID := fmt.Sprintf("issue-%d-%d", issueNumber, time.Now().Unix())

	process, err := claude.CreateProcess(
		issueNumber,
		processID,
		d.config.ClaudeCommand,
		d.config.ClaudeFlags,
		d.selectedProject,
		d.config.ProviderUsername,
	)
	if err != nil {
		return fmt.Errorf("error creating claude process: %v", err)
	}

	processManager.AddProcess(process)

	if err := claude.RunProcess(process); err != nil {
		return fmt.Errorf("error executing claude command: %v", err)
	}

	return nil
}

// processIssueAsync processes an issue asynchronously with completion callback
func (d *Daemon) processIssueAsync(issueNumber int) error {
	d.logAsyncProcessStart(issueNumber)

	processID := fmt.Sprintf("issue-%d-%d", issueNumber, time.Now().Unix())

	// Define completion labels - remove process label and add review label
	completionLabels := []string{d.config.ReviewLabel}

	// Create completion callback
	onCompletion := d.createCompletionCallback()

	process, err := claude.CreateProcessWithCallbackAndProviderDryRun(
		issueNumber,
		processID,
		d.config.ClaudeCommand,
		d.config.ClaudeFlags,
		d.selectedProject,
		d.config.ProviderUsername,
		d.config.ProviderURL,
		d.config.ProviderType,
		!d.runType.RequiresRepositoryClone(),
		completionLabels,
		onCompletion,
	)
	if err != nil {
		return fmt.Errorf("error creating claude process: %v", err)
	}

	if d.runType.IsDryRun() {
		d.handleDryRunMode(process, issueNumber, processID)
		return nil
	}

	// Normal execution mode
	d.processManager.AddProcess(process)
	claude.RunProcessAsync(process, d.processManager)

	return nil
}

// logAsyncProcessStart logs the start of async processing based on run type
func (d *Daemon) logAsyncProcessStart(issueNumber int) {
	switch d.runType {
	case enum.RunTypeDryRun:
		fmt.Printf("[DRY RUN] Would start async process for issue #%d...\n", issueNumber)
	case enum.RunTypeSemiDryRun:
		fmt.Printf("[SEMI-DRY RUN] Starting repository check for issue #%d...\n", issueNumber)
	default:
		fmt.Printf("Starting async process for issue #%d...\n", issueNumber)
	}
}

// handleDryRunMode handles dry run and semi-dry run execution
func (d *Daemon) handleDryRunMode(process *claude.Process, issueNumber int, processID string) {
	switch d.runType {
	case enum.RunTypeDryRun:
		fmt.Println("\n=== DRY RUN MODE (Daemon) ===")
	case enum.RunTypeSemiDryRun:
		fmt.Println("\n=== SEMI-DRY RUN MODE (Daemon) ===")
		fmt.Println("Repository has been cloned/verified.")
	}

	fmt.Printf("Issue #%d\n", issueNumber)
	fmt.Printf("Process ID: %s\n", processID)
	fmt.Printf("Working Directory: %s\n", process.Cmd.Dir)
	fmt.Printf("Command: %s\n", process.Cmd.Path)
	fmt.Printf("Arguments: %v\n", process.Cmd.Args)
	fmt.Println("\n=== PROMPT ===")

	// Extract the prompt from the command args
	for i, arg := range process.Cmd.Args {
		if arg == "-p" && i+1 < len(process.Cmd.Args) {
			fmt.Println(process.Cmd.Args[i+1])
			break
		}
	}

	if d.runType == enum.RunTypeSemiDryRun {
		d.showSemiDryRunDetails(process)
	} else {
		fmt.Println("=== END DRY RUN ===")
		fmt.Printf("[DRY RUN] Would update labels: remove '%s', add '%s' on completion\n",
			d.config.ProcessLabel, d.config.ReviewLabel)
	}
}

// showSemiDryRunDetails shows additional details for semi-dry run mode
func (d *Daemon) showSemiDryRunDetails(process *claude.Process) {
	fmt.Println("=== END SEMI-DRY RUN ===")

	// Additional repository checks in semi-dry-run mode
	fmt.Println("\n=== REPOSITORY STATUS ===")

	// Run git status in the repository directory
	statusCmd := exec.Command("git", "status", "--short")
	statusCmd.Dir = process.Cmd.Dir
	if output, err := statusCmd.Output(); err == nil {
		if len(output) == 0 {
			fmt.Println("Git status: Clean working directory")
		} else {
			fmt.Printf("Git status:\n%s", output)
		}
	}

	// Show current branch
	branchCmd := exec.Command("git", "branch", "--show-current")
	branchCmd.Dir = process.Cmd.Dir
	if output, err := branchCmd.Output(); err == nil {
		fmt.Printf("Current branch: %s", output)
	}

	fmt.Printf("\n[SEMI-DRY RUN] Would update labels: remove '%s', add '%s' on completion\n",
		d.config.ProcessLabel, d.config.ReviewLabel)

	// Show what cleanup would do
	fmt.Printf("\n=== REPOSITORY CLEANUP ===\n")
	fmt.Printf("After Claude finishes, the following cleanup would occur:\n")
	fmt.Printf("- Reset any uncommitted changes (git reset --hard HEAD)\n")
	fmt.Printf("- Remove untracked files (git clean -fd)\n")
	fmt.Printf("- Switch back to main branch\n")
	fmt.Printf("- Delete any issue-* branches\n")
	fmt.Printf("- Pull latest changes\n")
	fmt.Printf("Repository will be ready for the next parallel session\n")
}

// createCompletionCallback creates the completion callback for async processing
func (d *Daemon) createCompletionCallback() func(*claude.Process, bool) error {
	return func(process *claude.Process, success bool) error {
		// Run completion tasks asynchronously to avoid blocking the main process
		go func() {
			timestamp := time.Now().Format("2006-01-02 15:04:05")

			if success {
				d.handleSuccessfulCompletion(process, timestamp)
			} else {
				d.handleFailedCompletion(process, timestamp)
			}
		}() // End of async goroutine

		return nil // Return immediately from callback
	}
}

// handleSuccessfulCompletion handles successful issue completion
func (d *Daemon) handleSuccessfulCompletion(process *claude.Process, timestamp string) {
	fmt.Printf("[%s] Successfully completed issue #%d\n", timestamp, process.IssueNum)

	// Post completion comment
	d.postCompletionComment(process, timestamp)

	// Add delay to ensure comment is processed
	time.Sleep(2 * time.Second)

	// Update labels
	d.updateCompletionLabels(process, timestamp)

	// Store session information
	d.storeSessionInfo(process, timestamp)
}

// handleFailedCompletion handles failed issue completion
func (d *Daemon) handleFailedCompletion(process *claude.Process, timestamp string) {
	fmt.Printf("[%s] Failed to complete issue #%d\n", timestamp, process.IssueNum)

	// Get current issue and update labels to error
	issue, err := d.provider.GetIssue(d.selectedProject, process.IssueNum)
	if err != nil {
		fmt.Printf("[%s] Warning: failed to get issue #%d for label update: %v\n", timestamp, process.IssueNum, err)
		return
	}

	// Remove process label and add error label
	newLabels := make([]string, 0)
	for _, label := range issue.Labels {
		if label != d.config.ProcessLabel {
			newLabels = append(newLabels, label)
		}
	}
	newLabels = append(newLabels, "error")

	// Update labels
	if err := d.provider.UpdateIssueLabels(d.selectedProject, process.IssueNum, newLabels); err != nil {
		fmt.Printf("[%s] Warning: failed to update error labels for issue #%d: %v\n", timestamp, process.IssueNum, err)
	} else {
		fmt.Printf("[%s] Updated labels for issue #%d to 'error'\n", timestamp, process.IssueNum)
	}
}

// postCompletionComment posts a completion comment to the issue
func (d *Daemon) postCompletionComment(process *claude.Process, timestamp string) {
	completionComment := "✅ **Task completed successfully**\n\nClaude has finished processing this issue. The implementation has been completed and is ready for human review."
	note, err := d.provider.CreateIssueNote(d.selectedProject, process.IssueNum, completionComment)
	if err != nil {
		fmt.Printf("[%s] Warning: failed to post completion comment for issue #%d: %v\n", timestamp, process.IssueNum, err)
	} else {
		fmt.Printf("[%s] Posted completion comment for issue #%d\n", timestamp, process.IssueNum)
		// Update the last comment time to the actual comment timestamp
		d.lastCommentTime[process.IssueNum] = note.CreatedAt
		fmt.Printf("[%s] Updated last comment time for issue #%d to comment timestamp: %s\n", timestamp, process.IssueNum, note.CreatedAt)
	}
}

// updateCompletionLabels updates labels after successful completion
func (d *Daemon) updateCompletionLabels(process *claude.Process, timestamp string) {
	// Get current issue to get current labels
	issue, err := d.provider.GetIssue(d.selectedProject, process.IssueNum)
	if err != nil {
		fmt.Printf("[%s] Warning: failed to get issue #%d for label update: %v\n", timestamp, process.IssueNum, err)
		return
	}

	// Remove process label and add review label
	newLabels := make([]string, 0)
	for _, label := range issue.Labels {
		if label != d.config.ProcessLabel {
			newLabels = append(newLabels, label)
		}
	}
	newLabels = append(newLabels, d.config.ReviewLabel)

	// Update labels
	if err := d.provider.UpdateIssueLabels(d.selectedProject, process.IssueNum, newLabels); err != nil {
		fmt.Printf("[%s] Warning: failed to update completion labels for issue #%d: %v\n", timestamp, process.IssueNum, err)
	} else {
		fmt.Printf("[%s] Updated labels for issue #%d to '%s'\n", timestamp, process.IssueNum, d.config.ReviewLabel)
	}
}

// storeSessionInfo stores session information for future resumption
func (d *Daemon) storeSessionInfo(process *claude.Process, timestamp string) {
	sessionID := process.ClaudeSessionID
	if sessionID == "" {
		fmt.Printf("[%s] Warning: Claude session ID not captured for issue #%d, using fallback ID %s\n", timestamp, process.IssueNum, process.ID)
		sessionID = process.ID // Fallback to internal ID
	}

	// Prepare environment context for storage
	envVars := make(map[string]string)
	if process.Cmd != nil && process.Cmd.Env != nil {
		for _, env := range process.Cmd.Env {
			if strings.Contains(env, "=") {
				parts := strings.SplitN(env, "=", 2)
				envVars[parts[0]] = parts[1]
			}
		}
	}

	if err := d.sessionStore.AddCompletedSession(
		process.IssueNum,
		sessionID,
		d.selectedProject,
		time.Now(),
		process.WorkingDir,
		d.config.ClaudeCommand,
		d.config.ClaudeFlags,
		envVars,
	); err != nil {
		fmt.Printf("[%s] Warning: failed to store session info for issue #%d: %v\n", timestamp, process.IssueNum, err)
	} else {
		fmt.Printf("[%s] Stored session %s for issue #%d (monitoring for new comments)\n", timestamp, sessionID, process.IssueNum)
	}
}
