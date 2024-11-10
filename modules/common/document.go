/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/framework/core/orm"
	"time"
)

type Document struct {
	orm.ORMObjectBase                  // Embedding ORM base for persistence-related fields
	Source        string            `json:"source,omitempty"`     // Source of the document (e.g., "github", "google_drive", "dropbox")
	Category      string            `json:"category,omitempty"`   // Primary category of the document (e.g., "report", "article")
	Categories    []string          `json:"categories,omitempty"` // Full hierarchy of categories, useful for detailed classification
	Cover         string            `json:"cover,omitempty"`      // Cover image URL, if applicable
	Title         string            `json:"title,omitempty"`      // Document title
	Summary       string            `json:"summary,omitempty"`    // Brief summary or description of the document
	Type          string            `json:"type,omitempty"`       // Document type, such as PDF, Docx, etc.
	Lang          string            `json:"lang,omitempty"`       // Language code (e.g., "en", "fr")
	Content       string            `json:"content,omitempty"`    // Document content for full-text indexing
	Thumbnail     string            `json:"thumbnail,omitempty"`  // Thumbnail image URL, for preview purposes
	Owner         string            `json:"owner,omitempty"`      // Document author or owner
	Tags          []string          `json:"tags,omitempty"`       // Tags or keywords associated with the document, for easier retrieval
	URL           string            `json:"url,omitempty"`        // Direct link to the document, if available
	Size          int               `json:"size,omitempty"`       // File size in bytes, if applicable
	Metadata      map[string]string `json:"metadata,omitempty"`   // Additional source-specific metadata (e.g., file version, permissions)
	LastUpdatedBy struct {
		UserInfo  UserInfo   `json:"user,omitempty"`      // Information about the last user who updated the document
		UpdatedAt *time.Time `json:"timestamp,omitempty"` // Timestamp of the last update
	} `json:"last_updated_by,omitempty"`                    // Struct containing last update information
}

// UserInfo represents information about a user in relation to document edits or ownership.
type UserInfo struct {
	UserName string `json:"username,omitempty"` // Username of the user
	UserID   string `json:"userid,omitempty"`   // Unique identifier for the user
}
