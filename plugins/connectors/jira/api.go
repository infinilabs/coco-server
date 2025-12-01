package jira

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	sdk "github.com/andygrunwald/go-jira"
	log "github.com/cihub/seelog"
)

// Client JiraClient wraps the go-jira client with custom functionality
type Client struct {
	client         *sdk.Client
	endpoint       string
	datasourceName string
}

// BearerAuthTransport implements http.RoundTripper for Bearer token authentication
type BearerAuthTransport struct {
	Token     string
	Transport http.RoundTripper
}

// RoundTrip adds the Bearer token to each request
func (t *BearerAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.Token)
	return t.Transport.RoundTrip(req)
}

// NewJiraClient creates a new Jira client with optional authentication
func NewJiraClient(endpoint, username, token, datasourceName string) (*Client, error) {
	// Parse and validate endpoint
	baseURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	var jiraClient *sdk.Client

	// Setup authentication if credentials are provided
	if username != "" && token != "" {
		// Use Basic Auth with username + password for Jira Cloud
		tp := sdk.BasicAuthTransport{
			Username: username,
			Password: token,
		}
		jiraClient, err = sdk.NewClient(tp.Client(), baseURL.String())
	} else if token != "" {
		// Use Bearer token authentication for Personal Access Tokens (Jira Server/DC)
		bearerTransport := &BearerAuthTransport{
			Token:     token,
			Transport: http.DefaultTransport,
		}
		httpClient := &http.Client{Transport: bearerTransport}
		jiraClient, err = sdk.NewClient(httpClient, baseURL.String())
	} else {
		// Anonymous access (for public Jira instances)
		jiraClient, err = sdk.NewClient(http.DefaultClient, baseURL.String())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	authType := "anonymous"
	if username != "" && token != "" {
		authType = "basic"
	} else if token != "" {
		authType = "bearer"
	}
	log.Debugf("[jira] [%s] created client for endpoint: %s (auth: %s)", datasourceName, endpoint, authType)

	return &Client{
		client:         jiraClient,
		endpoint:       endpoint,
		datasourceName: datasourceName,
	}, nil
}

// SearchIssues searches for issues using JQL and returns paginated results
func (c *Client) SearchIssues(ctx context.Context, jql string, startAt, maxResults int) ([]sdk.Issue, int, error) {
	log.Debugf("[jira] [%s] searching issues: jql=%s, startAt=%d, maxResults=%d", c.datasourceName, jql, startAt, maxResults)

	// Specify fields to retrieve
	fields := []string{
		"summary",
		"description",
		"issuetype",
		"project",
		"status",
		"priority",
		"reporter",
		"assignee",
		"created",
		"updated",
		"labels",
		"resolution",
		"comment",
		"attachment",
	}

	// Setup search options
	searchOptions := &sdk.SearchOptions{
		StartAt:    startAt,
		MaxResults: maxResults,
		Fields:     fields,
		Expand:     "renderedFields", // Get rendered HTML for description
	}

	// Execute search
	issues, response, err := c.client.Issue.SearchWithContext(ctx, jql, searchOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search issues: %w", err)
	}

	// Get total count from response
	total := response.Total
	if total == 0 && len(issues) > 0 {
		// Fallback: if Total is not set, use length of issues
		total = len(issues)
	}

	log.Debugf("[jira] [%s] found %d issues (total: %d)", c.datasourceName, len(issues), total)

	return issues, total, nil
}

// GetComments retrieves all comments for a specific issue
func (c *Client) GetComments(ctx context.Context, issueKey string) ([]*sdk.Comment, error) {
	log.Debugf("[jira] [%s] fetching comments for issue: %s", c.datasourceName, issueKey)

	// Get issue with comments expanded
	issue, _, err := c.client.Issue.GetWithContext(ctx, issueKey, &sdk.GetQueryOptions{
		Expand: "renderedFields",
		Fields: "comment",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get issue comments: %w", err)
	}

	if issue.Fields == nil || issue.Fields.Comments == nil {
		log.Debugf("[jira] [%s] no comments found for issue: %s", c.datasourceName, issueKey)
		return nil, nil
	}

	log.Debugf("[jira] [%s] found %d comments for issue: %s", c.datasourceName, len(issue.Fields.Comments.Comments), issueKey)

	return issue.Fields.Comments.Comments, nil
}

// GetAttachments retrieves attachment metadata for a specific issue
// Note: This returns metadata only, not the actual file content
func (c *Client) GetAttachments(ctx context.Context, issueKey string) ([]*sdk.Attachment, error) {
	log.Debugf("[jira] [%s] fetching attachments for issue: %s", c.datasourceName, issueKey)

	// Get issue with attachments
	issue, _, err := c.client.Issue.GetWithContext(ctx, issueKey, &sdk.GetQueryOptions{
		Fields: "attachment",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get issue attachments: %w", err)
	}

	if issue.Fields == nil || issue.Fields.Attachments == nil {
		log.Debugf("[jira] [%s] no attachments found for issue: %s", c.datasourceName, issueKey)
		return nil, nil
	}

	log.Debugf("[jira] [%s] found %d attachments for issue: %s", c.datasourceName, len(issue.Fields.Attachments), issueKey)

	return issue.Fields.Attachments, nil
}

// TestConnection tests the connection to Jira and validates credentials
func (c *Client) TestConnection(ctx context.Context) error {
	log.Debugf("[jira] [%s] testing connection to: %s", c.datasourceName, c.endpoint)

	// Try to get myself (current user) as a connection test
	_, _, err := c.client.User.GetSelfWithContext(ctx)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	log.Debugf("[jira] [%s] connection test successful", c.datasourceName)
	return nil
}

// GetProject retrieves project information by key
func (c *Client) GetProject(ctx context.Context, projectKey string) (*sdk.Project, error) {
	log.Debugf("[jira] [%s] fetching project: %s", c.datasourceName, projectKey)

	project, _, err := c.client.Project.GetWithContext(ctx, projectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	log.Debugf("[jira] [%s] project found: %s (%s)", c.datasourceName, project.Name, project.Key)
	return project, nil
}
