package daemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bilbo290/automagic/pkg/claude"
	"github.com/bilbo290/automagic/pkg/enum"
	"github.com/bilbo290/automagic/pkg/provider"
	"github.com/bilbo290/automagic/pkg/session"
	"github.com/bilbo290/automagic/pkg/utils"
)

// resumeSessionWithCommentsWithContext resumes a Claude session with new comments
func (d *Daemon) resumeSessionWithCommentsWithContext(ctx context.Context, session *session.CompletedSession, newComments []provider.Note) error {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Build comment context
	commentContext := d.buildCommentContext(session, newComments)

	// Validate session ID format
	if !utils.IsValidUUID(session.SessionID) {
		fmt.Printf("[%s] Skipping resume for issue #%d: session ID '%s' is not a valid UUID (likely from old format)\n",
			timestamp, session.IssueIID, session.SessionID)
		return nil
	}

	// Handle dry run modes
	if d.runType.IsDryRun() {
		return d.handleDryRunResume(session, commentContext, timestamp)
	}

	// Check again before starting the process
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create and execute resume command
	return d.executeResumeCommand(ctx, session, commentContext, timestamp)
}

// buildCommentContext builds the context string from new comments
func (d *Daemon) buildCommentContext(session *session.CompletedSession, newComments []provider.Note) string {
	commentContext := fmt.Sprintf("# New Comments on Issue #%d\n\n", session.IssueIID)
	commentContext += "The following comments were added after you completed this issue:\n\n"

	for i, comment := range newComments {
		commentContext += fmt.Sprintf("## Comment %d by @%s\n", i+1, comment.Author.Username)
		commentContext += fmt.Sprintf("**Posted:** %s\n\n", comment.CreatedAt)
		commentContext += fmt.Sprintf("%s\n\n", comment.Body)
		commentContext += "---\n\n"
	}

	commentContext += "Please review these comments and take any necessary follow-up actions. "
	commentContext += "You can update your previous work, answer questions, or make additional changes as needed."

	return commentContext
}

// handleDryRunResume handles session resume in dry run modes
func (d *Daemon) handleDryRunResume(session *session.CompletedSession, commentContext, timestamp string) error {
	switch d.runType {
	case enum.RunTypeDryRun:
		fmt.Printf("[%s] [DRY RUN] Would resume session %s with comment context:\n%s\n", timestamp, session.SessionID, commentContext)
	case enum.RunTypeSemiDryRun:
		fmt.Printf("[%s] [SEMI-DRY RUN] Would resume session %s with comment context:\n%s\n", timestamp, session.SessionID, commentContext)
	}
	return nil
}

// executeResumeCommand creates and executes the resume command
func (d *Daemon) executeResumeCommand(ctx context.Context, session *session.CompletedSession, commentContext, timestamp string) error {
	// Use stored environment context to recreate the exact same execution environment
	args := []string{}
	claudeCommand := session.ClaudeCommand
	claudeFlags := session.ClaudeFlags
	workingDir := session.WorkingDir

	// Fallback to config values if not stored in session (for backward compatibility)
	if claudeCommand == "" {
		claudeCommand = d.config.ClaudeCommand
	}
	if claudeFlags == "" {
		claudeFlags = d.config.ClaudeFlags
	}
	if workingDir == "" {
		// Fallback to detection logic for old sessions
		detectedDir, _, err := claude.DetectProjectDirectory(session.ProjectPath)
		if err != nil {
			return fmt.Errorf("failed to detect working directory: %v", err)
		}
		workingDir = detectedDir
	}

	// Build command arguments using the stored flags
	if claudeFlags != "" {
		args = strings.Fields(claudeFlags)
	}
	args = append(args, "-r", session.SessionID, "-p", commentContext)

	fmt.Printf("[%s] Resuming Claude session %s for issue #%d with new comments\n", timestamp, session.SessionID, session.IssueIID)
	fmt.Printf("[%s] Using stored environment: command=%s, working_dir=%s\n", timestamp, claudeCommand, workingDir)

	// Create a context-aware command execution using the stored command and environment
	cmd := exec.CommandContext(ctx, claudeCommand, args...)
	cmd.Dir = workingDir

	// Use stored environment variables if available, otherwise fall back to current environment
	d.setupCommandEnvironment(cmd, session, timestamp)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the resume command asynchronously
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start resume session: %v", err)
	}

	// Track this process for graceful shutdown
	d.resumeProcesses[session.IssueIID] = cmd

	fmt.Printf("[%s] Started resume session for issue #%d (PID: %d)\n", timestamp, session.IssueIID, cmd.Process.Pid)

	// Don't wait for completion - let it run in background
	d.monitorResumeProcess(cmd, session, ctx)

	return nil
}

// setupCommandEnvironment sets up the environment for the resume command
func (d *Daemon) setupCommandEnvironment(cmd *exec.Cmd, session *session.CompletedSession, timestamp string) {
	if len(session.EnvVars) > 0 {
		// Convert stored environment map back to slice format
		envSlice := make([]string, 0, len(session.EnvVars))
		for key, value := range session.EnvVars {
			envSlice = append(envSlice, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = envSlice
		fmt.Printf("[%s] Using %d stored environment variables\n", timestamp, len(session.EnvVars))
	} else {
		// Fallback for backward compatibility
		cmd.Env = os.Environ()
		fmt.Printf("[%s] Using current environment (no stored env vars)\n", timestamp)
	}
}

// monitorResumeProcess monitors the resume process in the background
func (d *Daemon) monitorResumeProcess(cmd *exec.Cmd, session *session.CompletedSession, ctx context.Context) {
	// The process will complete on its own and respect context cancellation
	go func() {
		err := cmd.Wait()

		// Remove from tracking when completed
		delete(d.resumeProcesses, session.IssueIID)

		if err != nil {
			d.handleResumeProcessError(err, session, ctx)
		} else {
			fmt.Printf("[%s] Resume session for issue #%d completed successfully\n",
				time.Now().Format("2006-01-02 15:04:05"), session.IssueIID)
		}
	}()
}

// handleResumeProcessError handles errors from resume process completion
func (d *Daemon) handleResumeProcessError(err error, session *session.CompletedSession, ctx context.Context) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Check if it was cancelled due to context
	if ctx.Err() != nil {
		fmt.Printf("[%s] Resume session for issue #%d cancelled\n", timestamp, session.IssueIID)
		return
	}

	errorMsg := err.Error()
	fmt.Printf("[%s] Resume session for issue #%d completed with error: %v\n", timestamp, session.IssueIID, err)

	// Check if the error indicates the session is no longer valid
	if d.isSessionInvalidError(errorMsg) {
		fmt.Printf("[%s] Session %s appears to be invalid/expired, removing from database\n", timestamp, session.SessionID)

		// Remove the invalid session from the store
		if removeErr := d.sessionStore.RemoveSession(session.IssueIID); removeErr != nil {
			fmt.Printf("[%s] Warning: Failed to remove invalid session for issue #%d: %v\n", timestamp, session.IssueIID, removeErr)
		} else {
			fmt.Printf("[%s] Removed invalid session for issue #%d from database\n", timestamp, session.IssueIID)
		}
	}
}

// isSessionInvalidError checks if an error indicates an invalid session
func (d *Daemon) isSessionInvalidError(errorMsg string) bool {
	return strings.Contains(errorMsg, "No conversation found") ||
		strings.Contains(errorMsg, "session ID") ||
		strings.Contains(errorMsg, "not found")
}
