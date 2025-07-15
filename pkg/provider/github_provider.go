package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GitHubProvider implements the Provider interface for GitHub
type GitHubProvider struct {
	client *http.Client
	config *Config
}

// GitHub API types (internal to this provider)
type githubRepository struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	HTMLURL       string `json:"html_url"`
	CloneURL      string `json:"clone_url"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
	PushedAt      string `json:"pushed_at"`
}

type githubIssue struct {
	ID        int           `json:"id"`
	Number    int           `json:"number"`
	Title     string        `json:"title"`
	Body      string        `json:"body"`
	State     string        `json:"state"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
	HTMLURL   string        `json:"html_url"`
	Labels    []githubLabel `json:"labels"`
	User      githubUser    `json:"user"`
	Assignee  *githubUser   `json:"assignee"`
}

type githubLabel struct {
	Name string `json:"name"`
}

type githubUser struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

type githubOrganization struct {
	ID          int    `json:"id"`
	Login       string `json:"login"`
	Name        string `json:"name"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
	AvatarURL   string `json:"avatar_url"`
}

type githubComment struct {
	ID        int        `json:"id"`
	Body      string     `json:"body"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
	HTMLURL   string     `json:"html_url"`
	User      githubUser `json:"user"`
}

// NewGitHubProvider creates a new GitHub provider instance
func NewGitHubProvider(config *Config) *GitHubProvider {
	return &GitHubProvider{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
	}
}

// makeRequest makes an authenticated request to the GitHub API
func (g *GitHubProvider) makeRequest(endpoint string) ([]byte, error) {
	return g.makeRequestWithContext(context.Background(), endpoint)
}

// makeRequestWithContext makes an authenticated request to the GitHub API with context
func (g *GitHubProvider) makeRequestWithContext(ctx context.Context, endpoint string) ([]byte, error) {
	baseURL := g.config.URL
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	url := fmt.Sprintf("%s%s", baseURL, endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "token "+g.config.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// makePostRequest makes a POST request to the GitHub API
func (g *GitHubProvider) makePostRequest(endpoint string, payload interface{}) ([]byte, error) {
	baseURL := g.config.URL
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	url := fmt.Sprintf("%s%s", baseURL, endpoint)

	var reqBody io.Reader
	if payload != nil {
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}
		reqBody = strings.NewReader(string(jsonPayload))
	}

	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "token "+g.config.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// TestConnection tests the connection to GitHub
func (g *GitHubProvider) TestConnection() error {
	_, err := g.makeRequest("/user")
	return err
}

// GetUserOrganizations returns all organizations the user belongs to
func (g *GitHubProvider) GetUserOrganizations() ([]Organization, error) {
	body, err := g.makeRequest("/user/orgs")
	if err != nil {
		return nil, err
	}

	var githubOrgs []githubOrganization
	if err := json.Unmarshal(body, &githubOrgs); err != nil {
		return nil, fmt.Errorf("failed to parse organizations: %v", err)
	}

	orgs := make([]Organization, len(githubOrgs))
	for i, org := range githubOrgs {
		orgs[i] = Organization{
			ID:          org.ID,
			Name:        org.Name,
			Login:       org.Login,
			Description: org.Description,
			WebURL:      org.HTMLURL,
			AvatarURL:   org.AvatarURL,
		}
	}

	return orgs, nil
}

// GetAccessibleProjects returns all accessible repositories
func (g *GitHubProvider) GetAccessibleProjects() ([]Project, error) {
	body, err := g.makeRequest("/user/repos?per_page=100&sort=updated")
	if err != nil {
		return nil, err
	}

	var githubRepos []githubRepository
	if err := json.Unmarshal(body, &githubRepos); err != nil {
		return nil, fmt.Errorf("failed to parse repositories: %v", err)
	}

	projects := make([]Project, len(githubRepos))
	for i, repo := range githubRepos {
		visibility := "public"
		if repo.Private {
			visibility = "private"
		}

		projects[i] = Project{
			ID:                repo.ID,
			Name:              repo.Name,
			Path:              repo.Name,
			PathWithNamespace: repo.FullName,
			Description:       repo.Description,
			WebURL:            repo.HTMLURL,
			DefaultBranch:     repo.DefaultBranch,
			Visibility:        visibility,
			LastActivityAt:    repo.PushedAt,
		}
	}

	return projects, nil
}

// GetAccessibleProjectsWithOptions returns repositories with filtering options
func (g *GitHubProvider) GetAccessibleProjectsWithOptions(options ProjectListOptions) ([]Project, error) {
	var allProjects []Project

	// If no organizations specified, get user repos + all org repos
	if len(options.Organizations) == 0 {
		// Get user's own repositories
		userProjects, err := g.GetAccessibleProjects()
		if err != nil {
			return nil, fmt.Errorf("failed to get user repositories: %v", err)
		}
		allProjects = append(allProjects, userProjects...)

		// Get repositories from all user's organizations
		orgs, err := g.GetUserOrganizations()
		if err != nil {
			return nil, fmt.Errorf("failed to get user organizations: %v", err)
		}

		for _, org := range orgs {
			orgProjects, err := g.getOrganizationRepositories(org.Login)
			if err != nil {
				// Don't fail completely, just log and continue
				fmt.Printf("Warning: failed to get repositories for org %s: %v\n", org.Login, err)
				continue
			}
			allProjects = append(allProjects, orgProjects...)
		}
	} else {
		// Get repositories for specified organizations
		for _, orgLogin := range options.Organizations {
			if orgLogin == g.config.Username {
				// This is the user's own repositories
				userProjects, err := g.GetAccessibleProjects()
				if err != nil {
					return nil, fmt.Errorf("failed to get user repositories: %v", err)
				}
				allProjects = append(allProjects, userProjects...)
			} else {
				// This is an organization
				orgProjects, err := g.getOrganizationRepositories(orgLogin)
				if err != nil {
					return nil, fmt.Errorf("failed to get repositories for org %s: %v", orgLogin, err)
				}
				allProjects = append(allProjects, orgProjects...)
			}
		}
	}

	// Apply keyword filtering
	if options.Keyword != "" {
		filteredProjects := make([]Project, 0)
		keyword := strings.ToLower(options.Keyword)
		for _, project := range allProjects {
			if strings.Contains(strings.ToLower(project.Name), keyword) ||
				strings.Contains(strings.ToLower(project.PathWithNamespace), keyword) ||
				strings.Contains(strings.ToLower(project.Description), keyword) {
				filteredProjects = append(filteredProjects, project)
			}
		}
		allProjects = filteredProjects
	}

	// Apply visibility filtering
	if options.Visibility != "" {
		filteredProjects := make([]Project, 0)
		for _, project := range allProjects {
			if project.Visibility == options.Visibility {
				filteredProjects = append(filteredProjects, project)
			}
		}
		allProjects = filteredProjects
	}

	return allProjects, nil
}

// getOrganizationRepositories gets all repositories for a specific organization
func (g *GitHubProvider) getOrganizationRepositories(orgLogin string) ([]Project, error) {
	endpoint := fmt.Sprintf("/orgs/%s/repos?per_page=100&sort=updated", orgLogin)
	body, err := g.makeRequest(endpoint)
	if err != nil {
		return nil, err
	}

	var githubRepos []githubRepository
	if err := json.Unmarshal(body, &githubRepos); err != nil {
		return nil, fmt.Errorf("failed to parse repositories: %v", err)
	}

	projects := make([]Project, len(githubRepos))
	for i, repo := range githubRepos {
		visibility := "public"
		if repo.Private {
			visibility = "private"
		}

		projects[i] = Project{
			ID:                repo.ID,
			Name:              repo.Name,
			Path:              repo.Name,
			PathWithNamespace: repo.FullName,
			Description:       repo.Description,
			WebURL:            repo.HTMLURL,
			DefaultBranch:     repo.DefaultBranch,
			Visibility:        visibility,
			LastActivityAt:    repo.PushedAt,
		}
	}

	return projects, nil
}

// GetProject returns a specific repository
func (g *GitHubProvider) GetProject(projectID string) (*Project, error) {
	endpoint := fmt.Sprintf("/repos/%s", projectID)
	body, err := g.makeRequest(endpoint)
	if err != nil {
		return nil, err
	}

	var repo githubRepository
	if err := json.Unmarshal(body, &repo); err != nil {
		return nil, fmt.Errorf("failed to parse repository: %v", err)
	}

	visibility := "public"
	if repo.Private {
		visibility = "private"
	}

	return &Project{
		ID:                repo.ID,
		Name:              repo.Name,
		Path:              repo.Name,
		PathWithNamespace: repo.FullName,
		Description:       repo.Description,
		WebURL:            repo.HTMLURL,
		DefaultBranch:     repo.DefaultBranch,
		Visibility:        visibility,
		LastActivityAt:    repo.PushedAt,
	}, nil
}

// SearchProjects searches for repositories by query
func (g *GitHubProvider) SearchProjects(query string) ([]Project, error) {
	encodedQuery := url.QueryEscape(fmt.Sprintf("%s user:%s", query, g.config.Username))
	endpoint := fmt.Sprintf("/search/repositories?q=%s&per_page=50", encodedQuery)

	body, err := g.makeRequest(endpoint)
	if err != nil {
		return nil, err
	}

	var searchResult struct {
		Items []githubRepository `json:"items"`
	}
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %v", err)
	}

	projects := make([]Project, len(searchResult.Items))
	for i, repo := range searchResult.Items {
		visibility := "public"
		if repo.Private {
			visibility = "private"
		}

		projects[i] = Project{
			ID:                repo.ID,
			Name:              repo.Name,
			Path:              repo.Name,
			PathWithNamespace: repo.FullName,
			Description:       repo.Description,
			WebURL:            repo.HTMLURL,
			DefaultBranch:     repo.DefaultBranch,
			Visibility:        visibility,
			LastActivityAt:    repo.PushedAt,
		}
	}

	return projects, nil
}

// GetProjectIssues returns issues for a repository
func (g *GitHubProvider) GetProjectIssues(projectPath string, labels []string, state string) ([]Issue, error) {
	endpoint := fmt.Sprintf("/repos/%s/issues?per_page=100", projectPath)

	// Add state filter
	if state != "" {
		// Convert GitLab states to GitHub states
		switch state {
		case "opened":
			endpoint += "&state=open"
		case "closed":
			endpoint += "&state=closed"
		default:
			endpoint += "&state=" + state
		}
	}

	// Add labels filter
	if len(labels) > 0 {
		endpoint += "&labels=" + url.QueryEscape(strings.Join(labels, ","))
	}

	body, err := g.makeRequest(endpoint)
	if err != nil {
		return nil, err
	}

	var githubIssues []githubIssue
	if err := json.Unmarshal(body, &githubIssues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %v", err)
	}

	issues := make([]Issue, len(githubIssues))
	for i, gi := range githubIssues {
		labelNames := make([]string, len(gi.Labels))
		for j, label := range gi.Labels {
			labelNames[j] = label.Name
		}

		assignee := User{}
		if gi.Assignee != nil {
			assignee = User{
				ID:       gi.Assignee.ID,
				Name:     gi.Assignee.Name,
				Username: gi.Assignee.Login,
			}
		}

		// Convert GitHub state to GitLab format
		state := gi.State
		if state == "open" {
			state = "opened"
		}

		issues[i] = Issue{
			ID:          gi.ID,
			IID:         gi.Number, // GitHub uses "number" as the display ID
			ProjectID:   0,         // GitHub doesn't have project IDs like GitLab
			Title:       gi.Title,
			Description: gi.Body,
			State:       state,
			CreatedAt:   gi.CreatedAt,
			UpdatedAt:   gi.UpdatedAt,
			Labels:      labelNames,
			WebURL:      gi.HTMLURL,
			Author: User{
				ID:       gi.User.ID,
				Name:     gi.User.Name,
				Username: gi.User.Login,
			},
			Assignee: assignee,
		}
	}

	return issues, nil
}

// GetIssue returns a specific issue
func (g *GitHubProvider) GetIssue(projectPath string, issueIID int) (*Issue, error) {
	endpoint := fmt.Sprintf("/repos/%s/issues/%d", projectPath, issueIID)
	body, err := g.makeRequest(endpoint)
	if err != nil {
		return nil, err
	}

	var gi githubIssue
	if err := json.Unmarshal(body, &gi); err != nil {
		return nil, fmt.Errorf("failed to parse issue: %v", err)
	}

	labelNames := make([]string, len(gi.Labels))
	for j, label := range gi.Labels {
		labelNames[j] = label.Name
	}

	assignee := User{}
	if gi.Assignee != nil {
		assignee = User{
			ID:       gi.Assignee.ID,
			Name:     gi.Assignee.Name,
			Username: gi.Assignee.Login,
		}
	}

	// Convert GitHub state to GitLab format
	state := gi.State
	if state == "open" {
		state = "opened"
	}

	return &Issue{
		ID:          gi.ID,
		IID:         gi.Number,
		ProjectID:   0,
		Title:       gi.Title,
		Description: gi.Body,
		State:       state,
		CreatedAt:   gi.CreatedAt,
		UpdatedAt:   gi.UpdatedAt,
		Labels:      labelNames,
		WebURL:      gi.HTMLURL,
		Author: User{
			ID:       gi.User.ID,
			Name:     gi.User.Name,
			Username: gi.User.Login,
		},
		Assignee: assignee,
	}, nil
}

// UpdateIssueLabels updates the labels on an issue
func (g *GitHubProvider) UpdateIssueLabels(projectPath string, issueIID int, labels []string) error {
	endpoint := fmt.Sprintf("/repos/%s/issues/%d", projectPath, issueIID)
	payload := map[string]interface{}{
		"labels": labels,
	}

	_, err := g.makePostRequest(endpoint, payload)
	return err
}

// GetIssueDiscussions returns comments for an issue (GitHub doesn't have discussions, so we return comments)
func (g *GitHubProvider) GetIssueDiscussions(projectPath string, issueIID int) ([]Discussion, error) {
	return g.GetIssueDiscussionsWithContext(context.Background(), projectPath, issueIID)
}

// GetIssueDiscussionsWithContext returns comments for an issue with context
func (g *GitHubProvider) GetIssueDiscussionsWithContext(ctx context.Context, projectPath string, issueIID int) ([]Discussion, error) {
	endpoint := fmt.Sprintf("/repos/%s/issues/%d/comments?per_page=100", projectPath, issueIID)

	body, err := g.makeRequestWithContext(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var githubComments []githubComment
	if err := json.Unmarshal(body, &githubComments); err != nil {
		return nil, fmt.Errorf("failed to parse comments: %v", err)
	}

	// Convert comments to notes
	notes := make([]Note, len(githubComments))
	for i, gc := range githubComments {
		notes[i] = Note{
			ID:        gc.ID,
			Body:      gc.Body,
			CreatedAt: gc.CreatedAt,
			UpdatedAt: gc.UpdatedAt,
			System:    false, // GitHub doesn't distinguish system comments in this API
			Author: User{
				ID:       gc.User.ID,
				Name:     gc.User.Name,
				Username: gc.User.Login,
			},
		}
	}

	// GitHub doesn't have discussions, so we create a single discussion with all comments
	discussion := Discussion{
		ID:    "1", // Single discussion ID
		Notes: notes,
	}

	return []Discussion{discussion}, nil
}

// GetIssueCommentsAfter returns comments after a specific time
func (g *GitHubProvider) GetIssueCommentsAfter(projectPath string, issueIID int, afterTime time.Time) ([]Note, error) {
	return g.GetIssueCommentsAfterWithContext(context.Background(), projectPath, issueIID, afterTime)
}

// GetIssueCommentsAfterWithContext returns comments after a specific time with context
func (g *GitHubProvider) GetIssueCommentsAfterWithContext(ctx context.Context, projectPath string, issueIID int, afterTime time.Time) ([]Note, error) {
	// Get all discussions first
	discussions, err := g.GetIssueDiscussionsWithContext(ctx, projectPath, issueIID)
	if err != nil {
		return nil, err
	}

	var newComments []Note
	for _, discussion := range discussions {
		for _, note := range discussion.Notes {
			// Check for cancellation during processing
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			// Parse created time
			createdAt, err := time.Parse(time.RFC3339, note.CreatedAt)
			if err != nil {
				continue // Skip if we can't parse the time
			}

			// Only include comments after the specified time
			if createdAt.After(afterTime) {
				newComments = append(newComments, note)
			}
		}
	}

	return newComments, nil
}

// GetLatestCommentTime returns the time of the latest comment
func (g *GitHubProvider) GetLatestCommentTime(projectPath string, issueIID int) (*time.Time, error) {
	discussions, err := g.GetIssueDiscussions(projectPath, issueIID)
	if err != nil {
		return nil, err
	}

	var latestTime *time.Time
	for _, discussion := range discussions {
		for _, note := range discussion.Notes {
			// Parse created time
			createdAt, err := time.Parse(time.RFC3339, note.CreatedAt)
			if err != nil {
				continue
			}

			if latestTime == nil || createdAt.After(*latestTime) {
				latestTime = &createdAt
			}
		}
	}

	return latestTime, nil
}

// CreateIssueNote creates a new comment on an issue
func (g *GitHubProvider) CreateIssueNote(projectPath string, issueIID int, body string) (*Note, error) {
	endpoint := fmt.Sprintf("/repos/%s/issues/%d/comments", projectPath, issueIID)
	payload := map[string]string{
		"body": body,
	}

	respBody, err := g.makePostRequest(endpoint, payload)
	if err != nil {
		return nil, err
	}

	var gc githubComment
	if err := json.Unmarshal(respBody, &gc); err != nil {
		return nil, fmt.Errorf("failed to parse comment response: %v", err)
	}

	return &Note{
		ID:        gc.ID,
		Body:      gc.Body,
		CreatedAt: gc.CreatedAt,
		UpdatedAt: gc.UpdatedAt,
		System:    false,
		Author: User{
			ID:       gc.User.ID,
			Name:     gc.User.Name,
			Username: gc.User.Login,
		},
	}, nil
}

// GetProviderType returns the provider type
func (g *GitHubProvider) GetProviderType() ProviderType {
	return GitHub
}

// GetBaseURL returns the base URL
func (g *GitHubProvider) GetBaseURL() string {
	if g.config.URL == "" {
		return "https://github.com"
	}
	// Convert API URL to web URL
	if strings.Contains(g.config.URL, "api.github.com") {
		return "https://github.com"
	}
	return g.config.URL
}

// GetUsername returns the username
func (g *GitHubProvider) GetUsername() string {
	return g.config.Username
}
