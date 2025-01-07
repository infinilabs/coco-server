/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"time"
)

type RichLabel struct {
	Label string `json:"label,omitempty" elastic_mapping:"label:{type:keyword,copy_to:combined_fulltext}"`
	Icon  string `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"` // Icon Key, need work with datasource's assets to get the icon url
}

type Document struct {
	CombinedFullText

	Source string `json:"source,omitempty" elastic_mapping:"source:{type:keyword,copy_to:combined_fulltext}"` // Source of the document (e.g., "github", "google_drive", "dropbox")
	Type   string `json:"type,omitempty" elastic_mapping:"type:{type:keyword,copy_to:combined_fulltext}"`     // Document type, such as PDF, Docx, etc.

	Category       string      `json:"category,omitempty" elastic_mapping:"category:{type:keyword,copy_to:combined_fulltext}"`               // Primary category of the document (e.g., "report", "article")
	Categories     []string    `json:"categories,omitempty" elastic_mapping:"categories:{type:keyword,copy_to:combined_fulltext}"`           // Full hierarchy of categories, useful for detailed classification
	RichCategories []RichLabel `json:"rich_categories,omitempty" elastic_mapping:"rich_categories:{type:object}"` // Full hierarchy of categories, useful for detailed classification, with icon decoration

	Title   string `json:"title,omitempty" elastic_mapping:"title:{type:text,copy_to:combined_fulltext,fields:{keyword: {type: keyword}, pinyin: {type: text, analyzer: pinyin_analyzer}}}"` // Document title
	Summary string `json:"summary,omitempty" elastic_mapping:"summary:{type:text,copy_to:combined_fulltext}"`                                                                                // Brief summary or description of the document

	Lang    string `json:"lang,omitempty" elastic_mapping:"lang:{type:keyword,copy_to:combined_fulltext}"`    // Language code (e.g., "en", "fr")
	Content string `json:"content,omitempty" elastic_mapping:"content:{type:text,copy_to:combined_fulltext}"` // Document content for full-text indexing

	Icon      string `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"`           // Icon Key, need work with datasource's assets to get the icon url
	Thumbnail string `json:"thumbnail,omitempty" elastic_mapping:"thumbnail:{enabled:false}"` // Thumbnail image URL, for preview purposes
	Cover     string `json:"cover,omitempty" elastic_mapping:"cover:{enabled:false}"`         // Cover image URL, if applicable

	Owner *UserInfo `json:"owner,omitempty" elastic_mapping:"owner:{type:object}"` // Document author or owner

	Tags []string `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword,copy_to:combined_fulltext}"` // Tags or keywords associated with the document, for easier retrieval
	URL  string   `json:"url,omitempty" elastic_mapping:"url:{enabled:false}"`                            // Direct link to the document, if available
	Size int      `json:"size,omitempty" elastic_mapping:"size:{type:long}"`                              // File size in bytes, if applicable

	LastUpdatedBy *EditorInfo `json:"last_updated_by,omitempty" elastic_mapping:"last_updated_by:{type:object}"` // Struct containing last update information

}

type EditorInfo struct {
	UserInfo  UserInfo   `json:"user,omitempty" elastic_mapping:"user:{type:object}"` // Information about the last user who updated the document
	UpdatedAt *time.Time `json:"timestamp,omitempty"`                                 // Timestamp of the last update
}

// UserInfo represents information about a user in relation to document edits or ownership.
type UserInfo struct {
	UserAvatar string `json:"avatar,omitempty" elastic_mapping:"avatar:{enabled:false}"`                              // Username of the user
	UserName   string `json:"username,omitempty" elastic_mapping:"username:{type:keyword,copy_to:combined_fulltext}"` // Username of the user
	UserID     string `json:"userid,omitempty" elastic_mapping:"userid:{type:keyword,copy_to:combined_fulltext}"`     // Unique identifier for the user
}
