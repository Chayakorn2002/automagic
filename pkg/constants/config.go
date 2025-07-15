package constants

const (
	ConfigTemplate = `# automagic Multi-Provider Automation Configuration
# Edit these values with your provider credentials and preferences

# Provider Configuration (REQUIRED - choose one)
# Set PROVIDER_TYPE to "gitlab" or "github", or let it auto-detect from tokens below

# Option 1: Generic Provider Configuration
# PROVIDER_TYPE=gitlab
# PROVIDER_URL=https://gitlab.com
# PROVIDER_TOKEN=your-token-here
# PROVIDER_USERNAME=your-username

# Option 2: GitLab Configuration (legacy - will auto-detect as gitlab)
# GITLAB_URL=https://gitlab.com
# GITLAB_TOKEN=glpat-your-token-here
# GITLAB_USERNAME=your-gitlab-username

# Option 3: GitHub Configuration (will auto-detect as github if GITHUB_TOKEN is set)
# GITHUB_URL=https://api.github.com
# GITHUB_TOKEN=ghp_your-token-here
# GITHUB_USERNAME=your-github-username

# Claude Configuration
CLAUDE_COMMAND=claude
CLAUDE_FLAGS="--dangerously-skip-permissions --output-format stream-json --verbose"

# Project Configuration (Optional - will be set via interactive mode)
DEFAULT_PROJECT_PATH=

# Daemon Configuration (Optional)
DAEMON_INTERVAL=10
CLAUDE_LABEL=claude
PROCESS_LABEL=picked_up_by_claude
REVIEW_LABEL=waiting_human_review
`
)
