package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bilbo290/automagic/pkg/enum"
)

// Run starts the daemon with SQLite session storage enabled
func (d *Daemon) Run() error {
	return d.RunWithMemory()
}

// RunWithMemory starts the daemon with SQLite session storage and resumption enabled
func (d *Daemon) RunWithMemory() error {
	// Step 1: Select project interactively
	fmt.Printf("=== Project Selection for Daemon Mode ===\n")
	projects, err := d.provider.GetAccessibleProjects()
	if err != nil {
		return fmt.Errorf("error fetching projects: %v", err)
	}

	selectedProject, err := d.selectProject(projects)
	if err != nil {
		return fmt.Errorf("error selecting project: %v", err)
	}

	d.selectedProject = selectedProject.PathWithNamespace
	fmt.Printf("Project selected: %s\n", d.selectedProject)

	// Step 2: Start daemon monitoring
	fmt.Printf("\n=== Starting Daemon Mode ===\n")
	d.logDaemonStartup()
	fmt.Printf("Monitoring project: %s\n", d.selectedProject)
	fmt.Printf("Monitoring for issues with label: %s\n", d.config.ClaudeLabel)
	fmt.Printf("Processing interval: %d seconds\n", d.config.DaemonInterval)
	fmt.Printf("Session storage: SQLite (session resumption enabled)\n")
	fmt.Printf("Press Ctrl+C to stop...\n\n")

	// Set up signal handling for graceful shutdown with context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d.setupSignalHandling(ctx, cancel)

	// Keep track of processed issues to avoid duplicates
	processedIssues := make(map[int]bool)

	ticker := time.NewTicker(time.Duration(d.config.DaemonInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return d.handleShutdown()

		case <-ticker.C:
			// Check if context was cancelled before starting work
			select {
			case <-ctx.Done():
				fmt.Printf("\nOperation cancelled before processing\n")
				return nil
			default:
			}

			timestamp := time.Now().Format("2006-01-02 15:04:05")
			fmt.Printf("[%s] Checking for issues to process...\n", timestamp)

			// 1. Check for new issues with 'claude' label (spawn new sessions)
			fmt.Printf("[%s] DEBUG: Starting checkForNewClaudeIssues...\n", timestamp)
			newIssues, err := d.checkForNewClaudeIssuesWithContext(ctx, processedIssues, timestamp)
			if err != nil {
				if ctx.Err() != nil {
					fmt.Printf("[%s] Operation cancelled by user\n", timestamp)
					continue
				}
				fmt.Printf("[%s] Error checking for new claude issues: %v\n", timestamp, err)
			}
			fmt.Printf("[%s] DEBUG: Finished checkForNewClaudeIssues, found %d new issues\n", timestamp, newIssues)

			// 2. Check for issues under review with new comments (resume sessions)
			fmt.Printf("[%s] DEBUG: Starting checkForReviewIssuesWithComments...\n", timestamp)
			resumedIssues, err := d.checkForReviewIssuesWithCommentsWithContext(ctx, timestamp)
			if err != nil {
				if ctx.Err() != nil {
					fmt.Printf("[%s] Operation cancelled by user\n", timestamp)
					continue
				}
				fmt.Printf("[%s] Error checking for review issues with comments: %v\n", timestamp, err)
			}
			fmt.Printf("[%s] DEBUG: Finished checkForReviewIssuesWithComments, resumed %d sessions\n", timestamp, resumedIssues)

			// Summary
			if newIssues > 0 || resumedIssues > 0 {
				fmt.Printf("[%s] Activity: %d new sessions started, %d sessions resumed\n", timestamp, newIssues, resumedIssues)
			} else {
				fmt.Printf("[%s] No new activity found\n", timestamp)
			}
			fmt.Printf("[%s] DEBUG: Finished polling cycle, waiting for next tick...\n", timestamp)
		}
	}
}

// logDaemonStartup logs the daemon startup information based on run type
func (d *Daemon) logDaemonStartup() {
	switch d.runType {
	case enum.RunTypeDryRun:
		fmt.Printf("*** DRY RUN MODE - No actual processing will occur ***\n")
	case enum.RunTypeSemiDryRun:
		fmt.Printf("*** SEMI-DRY RUN MODE - Will clone repositories but not execute Claude ***\n")
	}
}

// setupSignalHandling sets up signal handling for graceful shutdown
func (d *Daemon) setupSignalHandling(ctx context.Context, cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Cancel context when signal received
	go func() {
		<-sigCh
		fmt.Printf("\nReceived shutdown signal. Cancelling operations...\n")
		cancel()
	}()
}

// handleShutdown handles graceful shutdown with memory mode
func (d *Daemon) handleShutdown() error {
	fmt.Printf("\nReceived shutdown signal. Stopping daemon...\n")

	// Gracefully stop any running processes
	runningProcesses := d.processManager.GetRunningProcesses()
	totalProcesses := len(runningProcesses) + len(d.resumeProcesses)

	if totalProcesses > 0 {
		fmt.Printf("Terminating %d running Claude processes...\n", totalProcesses)

		// Terminate regular Claude processes
		for _, process := range runningProcesses {
			if process.Cmd != nil && process.Cmd.Process != nil {
				fmt.Printf("  Terminating process for issue #%d (PID: %d)\n", process.IssueNum, process.Cmd.Process.Pid)
				process.Cmd.Process.Signal(syscall.SIGTERM)
			}
		}

		// Terminate resume processes
		for issueID, cmd := range d.resumeProcesses {
			if cmd != nil && cmd.Process != nil {
				fmt.Printf("  Terminating resume process for issue #%d (PID: %d)\n", issueID, cmd.Process.Pid)
				cmd.Process.Signal(syscall.SIGTERM)
			}
		}

		// Give processes a moment to terminate gracefully
		fmt.Printf("Waiting 3 seconds for processes to terminate...\n")
		time.Sleep(3 * time.Second)

		// Force kill any remaining processes
		for _, process := range runningProcesses {
			if process.Cmd != nil && process.Cmd.Process != nil {
				process.Cmd.Process.Kill()
			}
		}

		for _, cmd := range d.resumeProcesses {
			if cmd != nil && cmd.Process != nil {
				cmd.Process.Kill()
			}
		}
	}

	fmt.Printf("Daemon stopped.\n")
	return nil
}
