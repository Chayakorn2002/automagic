package provider

import (
	"fmt"
	"strings"
)

// ProviderFactory implements the Factory interface
type ProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// CreateProvider creates a provider instance based on configuration
func (f *ProviderFactory) CreateProvider(config *Config) (Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	switch config.Type {
	case GitLab:
		return NewGitLabProvider(config), nil
	case GitHub:
		return NewGitHubProvider(config), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

// CreateProviderFromString creates a provider based on a string type
func (f *ProviderFactory) CreateProviderFromString(providerType, url, token, username string) (Provider, error) {
	var pType ProviderType

	switch strings.ToLower(providerType) {
	case "gitlab":
		pType = GitLab
	case "github":
		pType = GitHub
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}

	config := &Config{
		Type:     pType,
		URL:      url,
		Token:    token,
		Username: username,
	}

	return f.CreateProvider(config)
}

// GetSupportedProviders returns a list of supported provider types
func (f *ProviderFactory) GetSupportedProviders() []ProviderType {
	return []ProviderType{GitLab, GitHub}
}

// DetectProviderFromURL attempts to detect the provider type from a URL
func (f *ProviderFactory) DetectProviderFromURL(url string) ProviderType {
	url = strings.ToLower(url)

	if strings.Contains(url, "gitlab") {
		return GitLab
	}
	if strings.Contains(url, "github") {
		return GitHub
	}

	// Default to GitLab for backward compatibility
	return GitLab
}
