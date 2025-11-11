package jira

import (
	"fmt"
)

// Config holds the configuration for the Jira connector
type Config struct {
	Endpoint         string `config:"endpoint"`          // Jira instance URL (required)
	ProjectKey       string `config:"project_key"`       // Project key to index (required)
	Username         string `config:"username"`          // Username for authentication (optional)
	Token            string `config:"token"`             // Password (when username present) or Personal Access Token (optional)
	IndexComments    bool   `config:"index_comments"`    // Whether to index issue comments (default: false)
	IndexAttachments bool   `config:"index_attachments"` // Whether to index attachments (default: false)
}

// Constants for pagination
const (
	DefaultPageSize = 100  // Default number of issues to fetch per page
	MaxPageSize     = 1000 // Maximum allowed by Jira API
)

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate required fields
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	if c.ProjectKey == "" {
		return fmt.Errorf("project_key is required")
	}

	// Authentication validation:
	// - If username is provided, token (password) must also be provided (for Basic Auth)
	// - Token alone is valid (for Bearer token auth with Personal Access Tokens)
	if c.Username != "" && c.Token == "" {
		return fmt.Errorf("token is required when username is provided")
	}

	return nil
}

// IsAuthConfigured returns true if authentication credentials are configured
func (c *Config) IsAuthConfigured() bool {
	// Auth is configured if we have either:
	// - Username + Token (password for Basic Auth in Jira Cloud)
	// - Token only (Personal Access Token for Bearer auth in Jira Server/DC)
	return c.Token != ""
}
