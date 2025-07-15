package daemon

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bilbo290/automagic/pkg/provider"
	"github.com/bilbo290/automagic/pkg/session"
)

// checkForNewClaudeIssuesWithContext monitors for new issues with the claude label
func (d *Daemon) checkForNewClaudeIssuesWithContext(ctx context.Context, processedIssues map[int]bool, timestamp string) (int, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// Fetch issues with the claude label (new work) with timeout
	fmt.Printf("[%s] DEBUG: Fetching issues with label '%s' from project '%s'...\n", timestamp, d.config.ClaudeLabel, d.selectedProject)

	// Create a timeout context for the API call
	apiCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use a channel to make the API call cancellable
	type result struct {
		issues []provider.Issue
		err    error
	}

	resultCh := make(chan result, 1)
	go func() {
		issues, err := d.provider.GetProjectIssues(d.selectedProject, []string{d.config.ClaudeLabel}, "opened")
		resultCh <- result{issues: issues, err: err}
	}()

	// Wait for either the result or context cancellation
	var issues []provider.Issue
	var err error
	select {
	case <-apiCtx.Done():
		fmt.Printf("[%s] DEBUG: API call timed out or was cancelled\n", timestamp)
		return 0, apiCtx.Err()
	case res := <-resultCh:
		issues = res.issues
		err = res.err
	}

	if err != nil {
		fmt.Printf("[%s] DEBUG: Failed to fetch claude issues: %v\n", timestamp, err)
		return 0, fmt.Errorf("failed to fetch claude issues: %v", err)
	}
	fmt.Printf("[%s] DEBUG: Successfully fetched %d issues with claude label\n", timestamp, len(issues))

	newIssues := 0
	for _, issue := range issues {
		// Check for cancellation between issues
		select {
		case <-ctx.Done():
			return newIssues, ctx.Err()
		default:
		}

		// Skip if already processed in this cycle
		if processedIssues[issue.IID] {
			fmt.Printf("[%s] DEBUG: Issue #%d already processed in this cycle, skipping\n", timestamp, issue.IID)
			continue
		}

		// Mark as processed to avoid duplicates
		processedIssues[issue.IID] = true

		// Check if there's currently a process running for this issue
		if d.isProcessRunning(issue.IID) {
			fmt.Printf("[%s] DEBUG: Issue #%d is already being processed, skipping\n", timestamp, issue.IID)
			continue
		}

		// Process the issue
		fmt.Printf("[%s] DEBUG: Processing new issue #%d: %s\n", timestamp, issue.IID, issue.Title)
		if err := d.processIssueWithLabelUpdate(&issue); err != nil {
			fmt.Printf("[%s] Error processing issue #%d: %v\n", timestamp, issue.IID, err)
			continue
		}

		newIssues++
	}

	return newIssues, nil
}

// checkForReviewIssuesWithCommentsWithContext monitors for issues under review with new comments
func (d *Daemon) checkForReviewIssuesWithCommentsWithContext(ctx context.Context, timestamp string) (int, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// Fetch issues with the review label (waiting for human review) with timeout
	fmt.Printf("[%s] DEBUG: Fetching issues with label '%s' from project '%s'...\n", timestamp, d.config.ReviewLabel, d.selectedProject)

	// Create a timeout context for the API call
	apiCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use a channel to make the API call cancellable
	type result struct {
		issues []provider.Issue
		err    error
	}

	resultCh := make(chan result, 1)
	go func() {
		issues, err := d.provider.GetProjectIssues(d.selectedProject, []string{d.config.ReviewLabel}, "opened")
		resultCh <- result{issues: issues, err: err}
	}()

	// Wait for either the result or context cancellation
	var issues []provider.Issue
	var err error
	select {
	case <-apiCtx.Done():
		fmt.Printf("[%s] DEBUG: API call timed out or was cancelled\n", timestamp)
		return 0, apiCtx.Err()
	case res := <-resultCh:
		issues = res.issues
		err = res.err
	}

	if err != nil {
		fmt.Printf("[%s] DEBUG: Failed to fetch review issues: %v\n", timestamp, err)
		return 0, fmt.Errorf("failed to fetch review issues: %v", err)
	}
	fmt.Printf("[%s] DEBUG: Successfully fetched %d issues with review label\n", timestamp, len(issues))

	resumedIssues := 0
	for _, issue := range issues {
		// Check for cancellation between issues
		select {
		case <-ctx.Done():
			return resumedIssues, ctx.Err()
		default:
		}

		// Skip if there's currently a process running for this issue
		if d.isProcessRunning(issue.IID) {
			fmt.Printf("[%s] DEBUG: Issue #%d is already being processed, skipping\n", timestamp, issue.IID)
			continue
		}

		// Check if there's a completed session for this issue
		session, exists := d.sessionStore.GetCompletedSession(issue.IID)
		if !exists {
			fmt.Printf("[%s] DEBUG: No completed session found for issue #%d\n", timestamp, issue.IID)
			continue
		}

		// Check for new comments since last processed
		hasNewComments, newComments, err := d.checkForNewComments(ctx, &issue, session, timestamp)
		if err != nil {
			fmt.Printf("[%s] Error checking for new comments on issue #%d: %v\n", timestamp, issue.IID, err)
			continue
		}

		if hasNewComments {
			fmt.Printf("[%s] DEBUG: Found %d new comments on issue #%d, resuming session\n", timestamp, len(newComments), issue.IID)
			if err := d.resumeSessionWithCommentsWithContext(ctx, session, newComments); err != nil {
				fmt.Printf("[%s] Error resuming session for issue #%d: %v\n", timestamp, issue.IID, err)
				continue
			}
			resumedIssues++
		}
	}

	return resumedIssues, nil
}

// checkForHumanReviewIssuesWithContext monitors for issues with human review labels that have new comments
func (d *Daemon) checkForHumanReviewIssuesWithContext(ctx context.Context, processedIssues map[int]bool, timestamp string) (int, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// Fetch issues with the review label (waiting for human review) with timeout
	fmt.Printf("[%s] DEBUG: Fetching issues with label '%s' from project '%s'...\n", timestamp, d.config.ReviewLabel, d.selectedProject)

	// Create a timeout context for the API call
	apiCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use a channel to make the API call cancellable
	type result struct {
		issues []provider.Issue
		err    error
	}

	resultCh := make(chan result, 1)
	go func() {
		issues, err := d.provider.GetProjectIssues(d.selectedProject, []string{d.config.ReviewLabel}, "opened")
		resultCh <- result{issues: issues, err: err}
	}()

	// Wait for either the result or context cancellation
	var issues []provider.Issue
	var err error
	select {
	case <-apiCtx.Done():
		fmt.Printf("[%s] DEBUG: API call timed out or was cancelled\n", timestamp)
		return 0, apiCtx.Err()
	case res := <-resultCh:
		issues = res.issues
		err = res.err
	}

	if err != nil {
		fmt.Printf("[%s] DEBUG: Failed to fetch review issues: %v\n", timestamp, err)
		return 0, fmt.Errorf("failed to fetch review issues: %v", err)
	}
	fmt.Printf("[%s] DEBUG: Successfully fetched %d issues with review label\n", timestamp, len(issues))

	newIssues := 0
	for _, issue := range issues {
		// Check for cancellation between issues
		select {
		case <-ctx.Done():
			return newIssues, ctx.Err()
		default:
		}

		// Skip if already processed in this cycle
		if processedIssues[issue.IID] {
			fmt.Printf("[%s] DEBUG: Issue #%d already processed in this cycle, skipping\n", timestamp, issue.IID)
			continue
		}

		// Mark as processed to avoid duplicates
		processedIssues[issue.IID] = true

		// Skip if there's currently a process running for this issue
		if d.isProcessRunning(issue.IID) {
			fmt.Printf("[%s] DEBUG: Issue #%d is already being processed, skipping\n", timestamp, issue.IID)
			continue
		}

		// Check if there are new human comments
		hasNewComments, err := d.checkForNewHumanComments(ctx, &issue, timestamp)
		if err != nil {
			fmt.Printf("[%s] Error checking for new human comments on issue #%d: %v\n", timestamp, issue.IID, err)
			continue
		}

		if hasNewComments {
			fmt.Printf("[%s] DEBUG: Found new human comments on issue #%d, processing\n", timestamp, issue.IID)
			if err := d.processIssueWithLabelUpdate(&issue); err != nil {
				fmt.Printf("[%s] Error processing issue #%d: %v\n", timestamp, issue.IID, err)
				continue
			}
			newIssues++
		}
	}

	return newIssues, nil
}

// checkForNewComments checks if there are new comments on an issue since the last processed time
func (d *Daemon) checkForNewComments(ctx context.Context, issue *provider.Issue, session *session.CompletedSession, timestamp string) (bool, []provider.Note, error) {
	// Create a timeout context for the API call
	apiCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use a channel to make the API call cancellable
	type result struct {
		comments []provider.Note
		err      error
	}

	resultCh := make(chan result, 1)
	go func() {
		discussions, err := d.provider.GetIssueDiscussions(d.selectedProject, issue.IID)
		var comments []provider.Note
		if err == nil {
			for _, discussion := range discussions {
				comments = append(comments, discussion.Notes...)
			}
		}
		resultCh <- result{comments: comments, err: err}
	}()

	// Wait for either the result or context cancellation
	var comments []provider.Note
	var err error
	select {
	case <-apiCtx.Done():
		fmt.Printf("[%s] DEBUG: API call timed out or was cancelled\n", timestamp)
		return false, nil, apiCtx.Err()
	case res := <-resultCh:
		comments = res.comments
		err = res.err
	}

	if err != nil {
		return false, nil, fmt.Errorf("failed to fetch issue comments: %v", err)
	}

	// Filter comments to only include those after the last processed time
	var newComments []provider.Note
	lastProcessedTime := d.lastCommentTime[issue.IID]
	
	if lastProcessedTime == "" {
		// If no last processed time, use session completion time
		lastProcessedTime = session.CompletionTime.Format(time.RFC3339)
	}

	for _, comment := range comments {
		// Skip system comments or comments from Claude
		if d.isSystemComment(comment) {
			continue
		}

		// Check if comment is after last processed time
		if comment.CreatedAt > lastProcessedTime {
			newComments = append(newComments, comment)
		}
	}

	// Sort comments by creation time (oldest first)
	sort.Slice(newComments, func(i, j int) bool {
		return newComments[i].CreatedAt < newComments[j].CreatedAt
	})

	return len(newComments) > 0, newComments, nil
}

// checkForNewHumanComments checks if there are new human comments on an issue
func (d *Daemon) checkForNewHumanComments(ctx context.Context, issue *provider.Issue, timestamp string) (bool, error) {
	// Create a timeout context for the API call
	apiCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use a channel to make the API call cancellable
	type result struct {
		comments []provider.Note
		err      error
	}

	resultCh := make(chan result, 1)
	go func() {
		discussions, err := d.provider.GetIssueDiscussions(d.selectedProject, issue.IID)
		var comments []provider.Note
		if err == nil {
			for _, discussion := range discussions {
				comments = append(comments, discussion.Notes...)
			}
		}
		resultCh <- result{comments: comments, err: err}
	}()

	// Wait for either the result or context cancellation
	var comments []provider.Note
	var err error
	select {
	case <-apiCtx.Done():
		fmt.Printf("[%s] DEBUG: API call timed out or was cancelled\n", timestamp)
		return false, apiCtx.Err()
	case res := <-resultCh:
		comments = res.comments
		err = res.err
	}

	if err != nil {
		return false, fmt.Errorf("failed to fetch issue comments: %v", err)
	}

	// Check if there are new human comments since last processed time
	lastProcessedTime := d.lastCommentTime[issue.IID]
	
	for _, comment := range comments {
		// Skip system comments or comments from Claude
		if d.isSystemComment(comment) {
			continue
		}

		// Check if comment is after last processed time
		if lastProcessedTime == "" || comment.CreatedAt > lastProcessedTime {
			return true, nil
		}
	}

	return false, nil
}

// isProcessRunning checks if there's a process running for the given issue ID
func (d *Daemon) isProcessRunning(issueID int) bool {
	runningProcesses := d.processManager.GetRunningProcesses()
	for _, process := range runningProcesses {
		if process.IssueNum == issueID {
			return true
		}
	}
	return false
}

// isSystemComment checks if a comment is a system comment that should be ignored
func (d *Daemon) isSystemComment(comment provider.Note) bool {
	// Skip system comments (usually have no author or special markers)
	if comment.Author.Username == "" {
		return true
	}

	// Skip comments that are task completion notifications
	if strings.Contains(comment.Body, "✅ **Task completed successfully**") {
		return true
	}

	// Skip comments that are automated status updates
	if strings.Contains(comment.Body, "automated") || strings.Contains(comment.Body, "system") {
		return true
	}

	return false
}

// GetProcessStatus returns the current status of all processes
func (d *Daemon) GetProcessStatus() {
	fmt.Printf("=== Process Status ===\n")

	running := d.processManager.GetRunningProcesses()
	completed := d.processManager.GetProcessesByStatus("completed")
	failed := d.processManager.GetProcessesByStatus("failed")

	fmt.Printf("Running processes: %d\n", len(running))
	for _, process := range running {
		fmt.Printf("  - Issue #%d (ID: %s) - Running for %v\n",
			process.IssueNum, process.ID, time.Since(process.StartTime))
	}

	fmt.Printf("Completed processes: %d\n", len(completed))
	for _, process := range completed {
		fmt.Printf("  - Issue #%d (ID: %s) - Completed in %v\n",
			process.IssueNum, process.ID, time.Since(process.StartTime))
	}

	fmt.Printf("Failed processes: %d\n", len(failed))
	for _, process := range failed {
		fmt.Printf("  - Issue #%d (ID: %s) - Failed after %v\n",
			process.IssueNum, process.ID, time.Since(process.StartTime))
	}

	fmt.Printf("Resume processes: %d\n", len(d.resumeProcesses))
	for issueID, cmd := range d.resumeProcesses {
		if cmd != nil && cmd.Process != nil {
			fmt.Printf("  - Issue #%d (PID: %d) - Resume session\n", issueID, cmd.Process.Pid)
		}
	}
}