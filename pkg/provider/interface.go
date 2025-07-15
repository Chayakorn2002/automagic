package provider

import (
	"context"
	"time"
)

// Provider defines the interface that both GitLab and GitHub implementations must satisfy
type Provider interface {
	// Connection and authentication
	TestConnection() error

	// Organization operations
	GetUserOrganizations() ([]Organization, error)
	
	// Project/Repository operations
	GetAccessibleProjects() ([]Project, error)
	GetAccessibleProjectsWithOptions(options ProjectListOptions) ([]Project, error)
	GetProject(projectID string) (*Project, error)
	SearchProjects(query string) ([]Project, error)

	// Issue operations
	GetProjectIssues(projectPath string, labels []string, state string) ([]Issue, error)
	GetIssue(projectPath string, issueIID int) (*Issue, error)
	UpdateIssueLabels(projectPath string, issueIID int, labels []string) error

	// Discussion/Comment operations
	GetIssueDiscussions(projectPath string, issueIID int) ([]Discussion, error)
	GetIssueDiscussionsWithContext(ctx context.Context, projectPath string, issueIID int) ([]Discussion, error)
	GetIssueCommentsAfter(projectPath string, issueIID int, afterTime time.Time) ([]Note, error)
	GetIssueCommentsAfterWithContext(ctx context.Context, projectPath string, issueIID int, afterTime time.Time) ([]Note, error)
	GetLatestCommentTime(projectPath string, issueIID int) (*time.Time, error)
	CreateIssueNote(projectPath string, issueIID int, body string) (*Note, error)

	// Provider information
	GetProviderType() ProviderType
	GetBaseURL() string
	GetUsername() string
}

// Factory creates provider instances based on configuration
type Factory interface {
	CreateProvider(config *Config) (Provider, error)
}