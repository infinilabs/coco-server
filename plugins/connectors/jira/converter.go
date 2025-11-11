package jira

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andygrunwald/go-jira"
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
)

// transformToDocument converts a Jira issue into a Document
func transformToDocument(issue *jira.Issue, ds *common.DataSource, config *Config, includeComments bool) (*common.Document, error) {
	if issue == nil || issue.Fields == nil {
		return nil, fmt.Errorf("invalid issue: nil issue or fields")
	}

	// Build content from description and optionally comments
	var contentParts []string

	// Extract description
	if issue.Fields.Description != "" {
		descText := extractTextFromField(issue.Fields.Description)
		if descText != "" {
			contentParts = append(contentParts, descText)
		}
	}

	// Include comments if configured
	if includeComments && issue.Fields.Comments != nil && issue.Fields.Comments.Comments != nil {
		for _, comment := range issue.Fields.Comments.Comments {
			if comment.Body != "" {
				commentText := extractTextFromField(comment.Body)
				if commentText != "" {
					authorName := "Unknown"
					if comment.Author.DisplayName != "" {
						authorName = comment.Author.DisplayName
					}
					contentParts = append(contentParts, fmt.Sprintf("[Comment by %s]: %s", authorName, commentText))
				}
			}
		}
	}

	// Combine all content parts
	content := strings.Join(contentParts, "\n\n")

	// Build issue URL - preserve the path from endpoint (e.g., /jira for Apache Jira)
	var issueURL string
	baseURL, err := url.Parse(config.Endpoint)
	if err == nil {
		basePath := strings.TrimSuffix(baseURL.Path, "/")
		issueURL = fmt.Sprintf("%s://%s%s/browse/%s", baseURL.Scheme, baseURL.Host, basePath, issue.Key)
	} else {
		issueURL = fmt.Sprintf("%s/browse/%s", config.Endpoint, issue.Key)
	}

	// Build categories for hierarchy: [project_name]
	var categories []string
	if issue.Fields.Project.Name != "" {
		categories = []string{issue.Fields.Project.Name}
	}

	// Generate ID suffix from issue key
	idSuffix := fmt.Sprintf("issue-%s", issue.Key)

	// Create document with hierarchy
	doc := connectors.CreateDocumentWithHierarchy(
		"jira_issue",
		"default",
		issue.Fields.Summary,
		issueURL,
		len(content),
		categories,
		ds,
		idSuffix,
	)

	// Set content
	doc.Content = content

	// Set timestamps
	createdTime := time.Time(issue.Fields.Created)
	if !createdTime.IsZero() {
		doc.Created = &createdTime
	}
	updatedTime := time.Time(issue.Fields.Updated)
	if !updatedTime.IsZero() {
		doc.Updated = &updatedTime
	}

	// Set owner (reporter)
	if issue.Fields.Reporter != nil {
		doc.Owner = &common.UserInfo{
			UserName: issue.Fields.Reporter.DisplayName,
			UserID:   issue.Fields.Reporter.AccountID,
		}
	}

	// Set tags from labels
	if len(issue.Fields.Labels) > 0 {
		doc.Tags = issue.Fields.Labels
	}

	// Save searchable metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["issue_key"] = issue.Key
	doc.Metadata["issue_id"] = issue.ID
	if issue.Fields.Project.Key != "" {
		doc.Metadata["project_key"] = issue.Fields.Project.Key
		doc.Metadata["project_name"] = issue.Fields.Project.Name
	}

	// Save the whole issue object to Payload (not indexed, but stored)
	if doc.Payload == nil {
		doc.Payload = make(map[string]interface{})
	}
	doc.Payload["issue"] = issue

	log.Debugf("[jira] [%s] transformed issue %s to document (content size: %d bytes)", ds.Name, issue.Key, doc.Size)

	return &doc, nil
}

// transformAttachmentToDocument converts a Jira attachment into a Document
func transformAttachmentToDocument(issue *jira.Issue, attachment *jira.Attachment, ds *common.DataSource, config *Config) (*common.Document, error) {
	if attachment == nil {
		return nil, fmt.Errorf("invalid attachment: nil")
	}

	// Build categories for hierarchy: [project_name]
	var categories []string
	if issue.Fields.Project.Name != "" {
		categories = []string{issue.Fields.Project.Name}
	}

	// Generate ID suffix from attachment ID
	idSuffix := fmt.Sprintf("attachment-%s-%s", issue.Key, attachment.ID)

	// Create document with hierarchy
	doc := connectors.CreateDocumentWithHierarchy(
		"jira_attachment",
		"default",
		fmt.Sprintf("[Attachment] %s", attachment.Filename),
		attachment.Content,
		0,
		categories,
		ds,
		idSuffix,
	)

	// Set timestamps - attachment.Created is a string in format "2006-01-02T15:04:05.000-0700"
	if attachment.Created != "" {
		createdTime, err := time.Parse("2006-01-02T15:04:05.000-0700", attachment.Created)
		if err == nil && !createdTime.IsZero() {
			doc.Created = &createdTime
		}
	}

	// Set owner (attachment author)
	if attachment.Author != nil {
		doc.Owner = &common.UserInfo{
			UserName: attachment.Author.DisplayName,
			UserID:   attachment.Author.AccountID,
		}
	}

	// Save searchable metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["issue_key"] = issue.Key
	doc.Metadata["issue_id"] = issue.ID
	doc.Metadata["project_key"] = issue.Fields.Project.Key
	doc.Metadata["project_name"] = issue.Fields.Project.Name

	// Save the whole attachment object to Payload
	if doc.Payload == nil {
		doc.Payload = make(map[string]interface{})
	}
	doc.Payload["attachment"] = attachment

	log.Debugf("[jira] [%s] transformed attachment %s (%s) to document", ds.Name, attachment.Filename, attachment.ID)

	return &doc, nil
}

// extractTextFromField extracts plain text from a Jira field (string or ADF object)
func extractTextFromField(field interface{}) string {
	switch v := field.(type) {
	case string:
		// Simple string - may contain HTML, need to clean it
		return htmlToText(v)
	case map[string]interface{}:
		// ADF (Atlassian Document Format) object - extract text recursively
		return extractTextFromADF(v)
	default:
		return ""
	}
}

// extractTextFromADF extracts plain text from an Atlassian Document Format object
func extractTextFromADF(adf map[string]interface{}) string {
	var parts []string

	// Check if this is a content node with text
	if text, ok := adf["text"].(string); ok && text != "" {
		parts = append(parts, text)
	}

	// Recursively process content array
	if content, ok := adf["content"].([]interface{}); ok {
		for _, item := range content {
			if node, ok := item.(map[string]interface{}); ok {
				text := extractTextFromADF(node)
				if text != "" {
					parts = append(parts, text)
				}
			}
		}
	}

	return strings.Join(parts, " ")
}

// htmlToText converts HTML content to plain text
func htmlToText(html string) string {
	if html == "" {
		return ""
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		_ = log.Warnf("[jira] failed to parse HTML: %v", err)
		// Fallback: return HTML as-is
		return html
	}

	// Extract text content
	text := doc.Text()

	// Clean up whitespace
	text = strings.TrimSpace(text)
	text = strings.Join(strings.Fields(text), " ")

	return text
}
