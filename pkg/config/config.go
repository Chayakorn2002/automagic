package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bilbo290/automagic/pkg/provider"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// Provider configuration
	ProviderType     string `envconfig:"PROVIDER_TYPE"`
	ProviderURL      string `envconfig:"PROVIDER_URL"`
	ProviderToken    string `envconfig:"PROVIDER_TOKEN"`
	ProviderUsername string `envconfig:"PROVIDER_USERNAME"`

	// Legacy GitLab configuration (for backward compatibility)
	GitLabURL      string `envconfig:"GITLAB_URL" default:"https://gitlab.com"`
	GitLabToken    string `envconfig:"GITLAB_TOKEN"`
	GitLabUsername string `envconfig:"GITLAB_USERNAME"`

	// GitHub configuration
	GitHubURL      string `envconfig:"GITHUB_URL" default:"https://api.github.com"`
	GitHubToken    string `envconfig:"GITHUB_TOKEN"`
	GitHubUsername string `envconfig:"GITHUB_USERNAME"`

	// Claude configuration
	ClaudeCommand string `envconfig:"CLAUDE_COMMAND" default:"claude"`
	ClaudeFlags   string `envconfig:"CLAUDE_FLAGS" default:"--dangerously-skip-permissions --output-format stream-json --verbose"`

	// Project configuration
	DefaultProjectPath string `envconfig:"DEFAULT_PROJECT_PATH"`

	// Daemon configuration
	DaemonInterval int    `envconfig:"DAEMON_INTERVAL" default:"10"`
	ClaudeLabel    string `envconfig:"CLAUDE_LABEL" default:"claude"`
	ProcessLabel   string `envconfig:"PROCESS_LABEL" default:"picked_up_by_claude"`
	ReviewLabel    string `envconfig:"REVIEW_LABEL" default:"waiting_human_review"`
}

func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
			(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
			value = value[1 : len(value)-1]
		}

		os.Setenv(key, value)
	}

	return scanner.Err()
}

func Load() (*Config, error) {
	var config Config

	// Try to load .env file first
	envFiles := []string{".env", ".env.local"}
	for _, envFile := range envFiles {
		if _, err := os.Stat(envFile); err == nil {
			if err := loadEnvFile(envFile); err != nil {
				fmt.Printf("Warning: failed to load %s: %v\n", envFile, err)
			}
			break
		}
	}

	// Use envconfig to process environment variables
	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %v", err)
	}

	return &config, nil
}

func Validate(config *Config) error {
	if config.ProviderType == "" {
		return fmt.Errorf("provider type is required. Set PROVIDER_TYPE to 'gitlab' or 'github', or set GITLAB_TOKEN/GITHUB_TOKEN for auto-detection")
	}

	if config.ProviderToken == "" {
		return fmt.Errorf("provider token is required. Set PROVIDER_TOKEN or %s_TOKEN environment variable", strings.ToUpper(config.ProviderType))
	}

	if config.ProviderUsername == "" {
		return fmt.Errorf("provider username is required. Set PROVIDER_USERNAME or %s_USERNAME environment variable", strings.ToUpper(config.ProviderType))
	}

	if config.ProviderURL == "" {
		switch config.ProviderType {
		case "github":
			config.ProviderURL = "https://github.com"
		case "gitlab":
			config.ProviderURL = "https://gitlab.com"
		default:
			return fmt.Errorf("invalid provider type: %s", config.ProviderType)
		}
	}

	return nil
}

func SaveProjectSelection(projectPath string) error {
	// Create or update .env file with the selected project
	envFile := ".env"

	// Read existing .env file if it exists
	existingVars := make(map[string]string)
	if file, err := os.Open(envFile); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Remove quotes if present (same logic as loadEnvFile)
				if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
					(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
					value = value[1 : len(value)-1]
				}

				existingVars[key] = value
			}
		}
	}

	// Update the project path
	existingVars["DEFAULT_PROJECT_PATH"] = projectPath

	// Write back to .env file
	file, err := os.Create(envFile)
	if err != nil {
		return fmt.Errorf("failed to create .env file: %v", err)
	}
	defer file.Close()

	// Write comment header
	fmt.Fprintln(file, "# automagic Multi-Provider Automation Configuration")
	fmt.Fprintln(file, "# Edit these values with your provider credentials and preferences")
	fmt.Fprintln(file, "")

	// Write comment sections
	fmt.Fprintln(file, "# Provider Configuration (REQUIRED - choose one)")
	fmt.Fprintln(file, "# Set PROVIDER_TYPE to \"gitlab\" or \"github\", or let it auto-detect from tokens below")
	fmt.Fprintln(file, "")

	// Write generic provider configuration
	fmt.Fprintln(file, "# Option 1: Generic Provider Configuration")
	// Only comment out if no value exists
	providerHasValues := existingVars["PROVIDER_TYPE"] != "" || existingVars["PROVIDER_URL"] != "" || existingVars["PROVIDER_TOKEN"] != "" || existingVars["PROVIDER_USERNAME"] != ""
	writeEnvVarWithComment(file, "PROVIDER_TYPE", existingVars, !providerHasValues)
	writeEnvVarWithComment(file, "PROVIDER_URL", existingVars, !providerHasValues)
	writeEnvVarWithComment(file, "PROVIDER_TOKEN", existingVars, !providerHasValues)
	writeEnvVarWithComment(file, "PROVIDER_USERNAME", existingVars, !providerHasValues)
	fmt.Fprintln(file, "")

	// Write GitLab configuration
	fmt.Fprintln(file, "# Option 2: GitLab Configuration (legacy - will auto-detect as gitlab)")
	writeEnvVar(file, "GITLAB_URL", existingVars)
	writeEnvVar(file, "GITLAB_TOKEN", existingVars)
	writeEnvVar(file, "GITLAB_USERNAME", existingVars)
	fmt.Fprintln(file, "")

	// Write GitHub configuration
	fmt.Fprintln(file, "# Option 3: GitHub Configuration (will auto-detect as github if GITHUB_TOKEN is set)")
	// Only comment out if no value exists
	githubHasValues := existingVars["GITHUB_TOKEN"] != "" || existingVars["GITHUB_URL"] != "" || existingVars["GITHUB_USERNAME"] != ""
	writeEnvVarWithComment(file, "GITHUB_URL", existingVars, !githubHasValues)
	writeEnvVarWithComment(file, "GITHUB_TOKEN", existingVars, !githubHasValues)
	writeEnvVarWithComment(file, "GITHUB_USERNAME", existingVars, !githubHasValues)
	fmt.Fprintln(file, "")

	// Write Claude configuration
	fmt.Fprintln(file, "# Claude Configuration")
	writeEnvVar(file, "CLAUDE_COMMAND", existingVars)
	writeEnvVar(file, "CLAUDE_FLAGS", existingVars)
	fmt.Fprintln(file, "")

	// Write project configuration
	fmt.Fprintln(file, "# Project Configuration (Optional - will be set via interactive mode)")
	writeEnvVar(file, "DEFAULT_PROJECT_PATH", existingVars)
	fmt.Fprintln(file, "")

	// Write daemon configuration
	fmt.Fprintln(file, "# Daemon Configuration (Optional)")
	writeEnvVar(file, "DAEMON_INTERVAL", existingVars)
	writeEnvVar(file, "CLAUDE_LABEL", existingVars)
	writeEnvVar(file, "PROCESS_LABEL", existingVars)
	writeEnvVar(file, "REVIEW_LABEL", existingVars)

	return nil
}

func writeEnvVar(file *os.File, key string, vars map[string]string) {
	if value, exists := vars[key]; exists && value != "" {
		// Quote the value if it contains spaces
		if strings.Contains(value, " ") {
			fmt.Fprintf(file, "%s=\"%s\"\n", key, value)
		} else {
			fmt.Fprintf(file, "%s=%s\n", key, value)
		}
	}
}

func writeEnvVarWithComment(file *os.File, key string, vars map[string]string, commented bool) {
	if value, exists := vars[key]; exists && value != "" {
		// Quote the value if it contains spaces
		prefix := ""
		if commented {
			prefix = "# "
		}
		if strings.Contains(value, " ") {
			fmt.Fprintf(file, "%s%s=\"%s\"\n", prefix, key, value)
		} else {
			fmt.Fprintf(file, "%s%s=%s\n", prefix, key, value)
		}
	} else if commented {
		// Write commented placeholder
		fmt.Fprintf(file, "# %s=your-value-here\n", key)
	}
}

// PrintConfig prints the current configuration for debugging
func PrintConfig(config *Config) {
	fmt.Println("Current Configuration:")

	// Print provider configuration
	fmt.Printf("  Provider Type: %s\n", config.ProviderType)
	fmt.Printf("  Provider URL: %s\n", config.ProviderURL)
	fmt.Printf("  Provider Username: %s\n", config.ProviderUsername)
	fmt.Printf("  Provider Token: %s\n", maskToken(config.ProviderToken))

	// Print legacy configurations if available
	if config.GitLabToken != "" {
		fmt.Printf("  GitLab URL: %s\n", config.GitLabURL)
		fmt.Printf("  GitLab Username: %s\n", config.GitLabUsername)
		fmt.Printf("  GitLab Token: %s\n", maskToken(config.GitLabToken))
	}
	if config.GitHubToken != "" {
		fmt.Printf("  GitHub URL: %s\n", config.GitHubURL)
		fmt.Printf("  GitHub Username: %s\n", config.GitHubUsername)
		fmt.Printf("  GitHub Token: %s\n", maskToken(config.GitHubToken))
	}

	fmt.Printf("  Claude Command: %s\n", config.ClaudeCommand)
	fmt.Printf("  Claude Flags: %s\n", config.ClaudeFlags)
	fmt.Printf("  Default Project: %s\n", config.DefaultProjectPath)
	fmt.Printf("  Daemon Interval: %d seconds\n", config.DaemonInterval)
	fmt.Printf("  Labels: %s → %s → %s\n",
		config.ClaudeLabel,
		config.ProcessLabel,
		config.ReviewLabel)
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "***" + token[len(token)-4:]
}

// GetProviderConfig creates a provider.Config from the main config
func (c *Config) GetProviderConfig() *provider.Config {
	var providerType provider.ProviderType
	switch strings.ToLower(c.ProviderType) {
	case "github":
		providerType = provider.GitHub
	case "gitlab":
		providerType = provider.GitLab
	}

	return &provider.Config{
		Type:     providerType,
		URL:      c.ProviderURL,
		Token:    c.ProviderToken,
		Username: c.ProviderUsername,
	}
}

// IsGitHub returns true if the configured provider is GitHub
func (c *Config) IsGitHub() bool {
	return strings.ToLower(c.ProviderType) == "github"
}

// IsGitLab returns true if the configured provider is GitLab
func (c *Config) IsGitLab() bool {
	return strings.ToLower(c.ProviderType) == "gitlab"
}
