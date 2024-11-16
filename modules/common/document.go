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
	Source        string            `json:"source,omitempty" elastic_mapping:"source:{type:keyword,copy_to:search_text}"`     // Source of the document (e.g., "github", "google_drive", "dropbox")
	Category      string            `json:"category,omitempty" elastic_mapping:"category:{type:keyword,copy_to:search_text}"`   // Primary category of the document (e.g., "report", "article")
	Categories    []string          `json:"categories,omitempty" elastic_mapping:"categories:{type:keyword,copy_to:search_text}"` // Full hierarchy of categories, useful for detailed classification
	Cover         string            `json:"cover,omitempty" elastic_mapping:"cover:{enabled:false}"`      // Cover image URL, if applicable
	Title         string            `json:"title,omitempty" elastic_mapping:"title:{type:keyword,copy_to:search_text}"`      // Document title
	Summary       string            `json:"summary,omitempty" elastic_mapping:"summary:{type:text}"`    // Brief summary or description of the document
	Type          string            `json:"type,omitempty" elastic_mapping:"type:{type:keyword,copy_to:search_text}"`       // Document type, such as PDF, Docx, etc.
	Lang          string            `json:"lang,omitempty" elastic_mapping:"lang:{type:keyword,copy_to:search_text}"`       // Language code (e.g., "en", "fr")
	Content       string            `json:"content,omitempty" elastic_mapping:"content:{type:text}"`    // Document content for full-text indexing
	Thumbnail     string            `json:"thumbnail,omitempty" elastic_mapping:"thumbnail:{enabled:false}"`  // Thumbnail image URL, for preview purposes
	Owner         string            `json:"owner,omitempty" elastic_mapping:"owner:{type:keyword,copy_to:search_text}"`      // Document author or owner
	Tags          []string          `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword,copy_to:search_text}"`       // Tags or keywords associated with the document, for easier retrieval
	URL           string            `json:"url,omitempty" elastic_mapping:"url:{enabled:false}"`        // Direct link to the document, if available
	Size          int               `json:"size,omitempty" elastic_mapping:"size:{type:long}"`       // File size in bytes, if applicable
	Metadata      map[string]string `json:"metadata,omitempty"`   // Additional source-specific metadata (e.g., file version, permissions)
	LastUpdatedBy struct {
		UserInfo  UserInfo   `json:"user,omitempty" elastic_mapping:"user:{type:object}"`      // Information about the last user who updated the document
		UpdatedAt *time.Time `json:"timestamp,omitempty"` // Timestamp of the last update
	} `json:"last_updated_by,omitempty" elastic_mapping:"last_updated_by:{type:object}"`                    // Struct containing last update information


	//
	SearchText string `json:"-" elastic_mapping:"search_text:{type:text,index_prefixes:{},index_phrases:true, analyzer:suggest_text_search }"`
}

// UserInfo represents information about a user in relation to document edits or ownership.
type UserInfo struct {
	UserName string `json:"username,omitempty" elastic_mapping:"username:{type:keyword,copy_to:search_text}"` // Username of the user
	UserID   string `json:"userid,omitempty" elastic_mapping:"userid:{type:keyword,copy_to:search_text}"`   // Unique identifier for the user
}
