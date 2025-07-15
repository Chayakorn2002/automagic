package provider

// Issue represents a generic issue/ticket across different platforms
type Issue struct {
	ID          int      `json:"id"`
	IID         int      `json:"iid"` // Internal ID (GitLab) or Number (GitHub)
	ProjectID   int      `json:"project_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	State       string   `json:"state"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Labels      []string `json:"labels"`
	WebURL      string   `json:"web_url"`
	Author      User     `json:"author"`
	Assignee    User     `json:"assignee"`
}

// Project represents a generic project/repository across different platforms
type Project struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"` // owner/repo format
	Description       string `json:"description"`
	WebURL            string `json:"web_url"`
	DefaultBranch     string `json:"default_branch"`
	Visibility        string `json:"visibility"`
	LastActivityAt    string `json:"last_activity_at"`
}

// Discussion represents a discussion thread (GitLab) or issue comments (GitHub)
type Discussion struct {
	ID    string `json:"id"`
	Notes []Note `json:"notes"`
}

// Note represents a comment/note on an issue
type Note struct {
	ID        int    `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	System    bool   `json:"system"`
	Author    User   `json:"author"`
}

// User represents a user across different platforms
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

// Organization represents an organization/group across different platforms
type Organization struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Login       string `json:"login"`       // GitHub login or GitLab path
	Description string `json:"description"`
	WebURL      string `json:"web_url"`
	AvatarURL   string `json:"avatar_url"`
}

// ProjectListOptions holds options for listing projects
type ProjectListOptions struct {
	Organizations []string // List of organization logins/paths to filter by
	Keyword       string   // Keyword to filter project names
	Visibility    string   // public, private, internal
	Archived      bool     // Include archived projects
}

// ProviderType represents the type of provider
type ProviderType string

const (
	GitLab ProviderType = "gitlab"
	GitHub ProviderType = "github"
)

// Config holds provider-specific configuration
type Config struct {
	Type     ProviderType
	URL      string
	Token    string
	Username string
}
