package session

import "time"

// CompletedSession tracks information about a completed Claude session
type CompletedSession struct {
	IssueIID        int        `json:"issue_iid"`
	SessionID       string     `json:"session_id"`
	ProjectPath     string     `json:"project_path"`
	CompletionTime  time.Time  `json:"completion_time"`
	LastCommentTime *time.Time `json:"last_comment_time,omitempty"`
	// Environment context for session resumption
	WorkingDir    string            `json:"working_dir"`
	ClaudeCommand string            `json:"claude_command"`
	ClaudeFlags   string            `json:"claude_flags"`
	EnvVars       map[string]string `json:"env_vars"`
}
