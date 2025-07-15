package provider

import (
	"context"
	"strings"
	"time"

	"github.com/bilbo290/automagic/pkg/gitlab"
)

// GitLabProvider wraps the existing GitLab client to implement the Provider interface
type GitLabProvider struct {
	client *gitlab.Client
	config *Config
}

// NewGitLabProvider creates a new GitLab provider instance
func NewGitLabProvider(config *Config) *GitLabProvider {
	client := gitlab.NewClient(config.URL, config.Token)
	return &GitLabProvider{
		client: client,
		config: config,
	}
}

// TestConnection tests the connection to GitLab
func (g *GitLabProvider) TestConnection() error {
	return g.client.TestConnection()
}

// GetUserOrganizations returns all groups the user belongs to (GitLab groups = GitHub orgs)
func (g *GitLabProvider) GetUserOrganizations() ([]Organization, error) {
	// For GitLab, we'll use groups as organizations
	// This is a simplified implementation - in a real scenario, you'd need to call GitLab API
	// For now, return empty list since the existing GitLab client doesn't have group support
	return []Organization{}, nil
}

// GetAccessibleProjects returns all accessible projects
func (g *GitLabProvider) GetAccessibleProjects() ([]Project, error) {
	gitlabProjects, err := g.client.GetAccessibleProjects()
	if err != nil {
		return nil, err
	}

	projects := make([]Project, len(gitlabProjects))
	for i, gp := range gitlabProjects {
		projects[i] = Project{
			ID:                gp.ID,
			Name:              gp.Name,
			Path:              gp.Path,
			PathWithNamespace: gp.PathWithNamespace,
			Description:       gp.Description,
			WebURL:            gp.WebURL,
			DefaultBranch:     gp.DefaultBranch,
			Visibility:        gp.Visibility,
			LastActivityAt:    gp.LastActivityAt,
		}
	}
	return projects, nil
}

// GetAccessibleProjectsWithOptions returns projects with filtering options
func (g *GitLabProvider) GetAccessibleProjectsWithOptions(options ProjectListOptions) ([]Project, error) {
	// Get all accessible projects first
	allProjects, err := g.GetAccessibleProjects()
	if err != nil {
		return nil, err
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

	// Apply organization filtering (for GitLab, this would be group filtering)
	if len(options.Organizations) > 0 {
		filteredProjects := make([]Project, 0)
		for _, project := range allProjects {
			for _, org := range options.Organizations {
				// Check if project path starts with the organization/group name
				if strings.HasPrefix(project.PathWithNamespace, org+"/") {
					filteredProjects = append(filteredProjects, project)
					break
				}
			}
		}
		allProjects = filteredProjects
	}

	return allProjects, nil
}

// GetProject returns a specific project
func (g *GitLabProvider) GetProject(projectID string) (*Project, error) {
	gitlabProject, err := g.client.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	return &Project{
		ID:                gitlabProject.ID,
		Name:              gitlabProject.Name,
		Path:              gitlabProject.Path,
		PathWithNamespace: gitlabProject.PathWithNamespace,
		Description:       gitlabProject.Description,
		WebURL:            gitlabProject.WebURL,
		DefaultBranch:     gitlabProject.DefaultBranch,
		Visibility:        gitlabProject.Visibility,
		LastActivityAt:    gitlabProject.LastActivityAt,
	}, nil
}

// SearchProjects searches for projects by query
func (g *GitLabProvider) SearchProjects(query string) ([]Project, error) {
	gitlabProjects, err := g.client.SearchProjects(query)
	if err != nil {
		return nil, err
	}

	projects := make([]Project, len(gitlabProjects))
	for i, gp := range gitlabProjects {
		projects[i] = Project{
			ID:                gp.ID,
			Name:              gp.Name,
			Path:              gp.Path,
			PathWithNamespace: gp.PathWithNamespace,
			Description:       gp.Description,
			WebURL:            gp.WebURL,
			DefaultBranch:     gp.DefaultBranch,
			Visibility:        gp.Visibility,
			LastActivityAt:    gp.LastActivityAt,
		}
	}
	return projects, nil
}

// GetProjectIssues returns issues for a project
func (g *GitLabProvider) GetProjectIssues(projectPath string, labels []string, state string) ([]Issue, error) {
	gitlabIssues, err := g.client.GetProjectIssues(projectPath, labels, state)
	if err != nil {
		return nil, err
	}

	issues := make([]Issue, len(gitlabIssues))
	for i, gi := range gitlabIssues {
		issues[i] = Issue{
			ID:          gi.ID,
			IID:         gi.IID,
			ProjectID:   gi.ProjectID,
			Title:       gi.Title,
			Description: gi.Description,
			State:       gi.State,
			CreatedAt:   gi.CreatedAt,
			UpdatedAt:   gi.UpdatedAt,
			Labels:      gi.Labels,
			WebURL:      gi.WebURL,
			Author: User{
				ID:       gi.Author.ID,
				Name:     gi.Author.Name,
				Username: gi.Author.Username,
			},
			Assignee: User{
				ID:       gi.Assignee.ID,
				Name:     gi.Assignee.Name,
				Username: gi.Assignee.Username,
			},
		}
	}
	return issues, nil
}

// GetIssue returns a specific issue
func (g *GitLabProvider) GetIssue(projectPath string, issueIID int) (*Issue, error) {
	gitlabIssue, err := g.client.GetIssue(projectPath, issueIID)
	if err != nil {
		return nil, err
	}

	return &Issue{
		ID:          gitlabIssue.ID,
		IID:         gitlabIssue.IID,
		ProjectID:   gitlabIssue.ProjectID,
		Title:       gitlabIssue.Title,
		Description: gitlabIssue.Description,
		State:       gitlabIssue.State,
		CreatedAt:   gitlabIssue.CreatedAt,
		UpdatedAt:   gitlabIssue.UpdatedAt,
		Labels:      gitlabIssue.Labels,
		WebURL:      gitlabIssue.WebURL,
		Author: User{
			ID:       gitlabIssue.Author.ID,
			Name:     gitlabIssue.Author.Name,
			Username: gitlabIssue.Author.Username,
		},
		Assignee: User{
			ID:       gitlabIssue.Assignee.ID,
			Name:     gitlabIssue.Assignee.Name,
			Username: gitlabIssue.Assignee.Username,
		},
	}, nil
}

// UpdateIssueLabels updates the labels on an issue
func (g *GitLabProvider) UpdateIssueLabels(projectPath string, issueIID int, labels []string) error {
	return g.client.UpdateIssueLabels(projectPath, issueIID, labels)
}

// GetIssueDiscussions returns discussions for an issue
func (g *GitLabProvider) GetIssueDiscussions(projectPath string, issueIID int) ([]Discussion, error) {
	gitlabDiscussions, err := g.client.GetIssueDiscussions(projectPath, issueIID)
	if err != nil {
		return nil, err
	}

	discussions := make([]Discussion, len(gitlabDiscussions))
	for i, gd := range gitlabDiscussions {
		notes := make([]Note, len(gd.Notes))
		for j, gn := range gd.Notes {
			notes[j] = Note{
				ID:        gn.ID,
				Body:      gn.Body,
				CreatedAt: gn.CreatedAt,
				UpdatedAt: gn.UpdatedAt,
				System:    gn.System,
				Author: User{
					ID:       gn.Author.ID,
					Name:     gn.Author.Name,
					Username: gn.Author.Username,
				},
			}
		}
		discussions[i] = Discussion{
			ID:    gd.ID,
			Notes: notes,
		}
	}
	return discussions, nil
}

// GetIssueDiscussionsWithContext returns discussions for an issue with context
func (g *GitLabProvider) GetIssueDiscussionsWithContext(ctx context.Context, projectPath string, issueIID int) ([]Discussion, error) {
	gitlabDiscussions, err := g.client.GetIssueDiscussionsWithContext(ctx, projectPath, issueIID)
	if err != nil {
		return nil, err
	}

	discussions := make([]Discussion, len(gitlabDiscussions))
	for i, gd := range gitlabDiscussions {
		notes := make([]Note, len(gd.Notes))
		for j, gn := range gd.Notes {
			notes[j] = Note{
				ID:        gn.ID,
				Body:      gn.Body,
				CreatedAt: gn.CreatedAt,
				UpdatedAt: gn.UpdatedAt,
				System:    gn.System,
				Author: User{
					ID:       gn.Author.ID,
					Name:     gn.Author.Name,
					Username: gn.Author.Username,
				},
			}
		}
		discussions[i] = Discussion{
			ID:    gd.ID,
			Notes: notes,
		}
	}
	return discussions, nil
}

// GetIssueCommentsAfter returns comments after a specific time
func (g *GitLabProvider) GetIssueCommentsAfter(projectPath string, issueIID int, afterTime time.Time) ([]Note, error) {
	gitlabNotes, err := g.client.GetIssueCommentsAfter(projectPath, issueIID, afterTime)
	if err != nil {
		return nil, err
	}

	notes := make([]Note, len(gitlabNotes))
	for i, gn := range gitlabNotes {
		notes[i] = Note{
			ID:        gn.ID,
			Body:      gn.Body,
			CreatedAt: gn.CreatedAt,
			UpdatedAt: gn.UpdatedAt,
			System:    gn.System,
			Author: User{
				ID:       gn.Author.ID,
				Name:     gn.Author.Name,
				Username: gn.Author.Username,
			},
		}
	}
	return notes, nil
}

// GetIssueCommentsAfterWithContext returns comments after a specific time with context
func (g *GitLabProvider) GetIssueCommentsAfterWithContext(ctx context.Context, projectPath string, issueIID int, afterTime time.Time) ([]Note, error) {
	gitlabNotes, err := g.client.GetIssueCommentsAfterWithContext(ctx, projectPath, issueIID, afterTime)
	if err != nil {
		return nil, err
	}

	notes := make([]Note, len(gitlabNotes))
	for i, gn := range gitlabNotes {
		notes[i] = Note{
			ID:        gn.ID,
			Body:      gn.Body,
			CreatedAt: gn.CreatedAt,
			UpdatedAt: gn.UpdatedAt,
			System:    gn.System,
			Author: User{
				ID:       gn.Author.ID,
				Name:     gn.Author.Name,
				Username: gn.Author.Username,
			},
		}
	}
	return notes, nil
}

// GetLatestCommentTime returns the time of the latest comment
func (g *GitLabProvider) GetLatestCommentTime(projectPath string, issueIID int) (*time.Time, error) {
	return g.client.GetLatestCommentTime(projectPath, issueIID)
}

// CreateIssueNote creates a new comment on an issue
func (g *GitLabProvider) CreateIssueNote(projectPath string, issueIID int, body string) (*Note, error) {
	gitlabNote, err := g.client.CreateIssueNote(projectPath, issueIID, body)
	if err != nil {
		return nil, err
	}

	return &Note{
		ID:        gitlabNote.ID,
		Body:      gitlabNote.Body,
		CreatedAt: gitlabNote.CreatedAt,
		UpdatedAt: gitlabNote.UpdatedAt,
		System:    gitlabNote.System,
		Author: User{
			ID:       gitlabNote.Author.ID,
			Name:     gitlabNote.Author.Name,
			Username: gitlabNote.Author.Username,
		},
	}, nil
}

// GetProviderType returns the provider type
func (g *GitLabProvider) GetProviderType() ProviderType {
	return GitLab
}

// GetBaseURL returns the base URL
func (g *GitLabProvider) GetBaseURL() string {
	return g.config.URL
}

// GetUsername returns the username
func (g *GitLabProvider) GetUsername() string {
	return g.config.Username
}