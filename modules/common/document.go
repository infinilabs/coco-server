/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"strings"
	"time"
)

type RichLabel struct {
	Label string `json:"label,omitempty" elastic_mapping:"label:{type:keyword,copy_to:combined_fulltext}"`
	Key   string `json:"key,omitempty" elastic_mapping:"key:{type:keyword}"`
	Icon  string `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"` // Icon Key, need work with datasource's assets to get the icon url
}

type DataSourceReference struct {
	Type string `json:"type,omitempty" elastic_mapping:"type:{type:keyword}"` // ID of the datasource, eg: connector
	Name string `json:"name,omitempty" elastic_mapping:"name:{type:keyword}"` // Name of the datasource (e.g., "My Github", "My Google Drive", "My Dropbox")
	ID   string `json:"id,omitempty" elastic_mapping:"id:{type:keyword}"`     // ID of this the datasource, eg: 8ca2fe8cf5027b0f1b5f932b429e38c3
}

type Document struct {
	CombinedFullText

	Source DataSourceReference `json:"source,omitempty" elastic_mapping:"source:{type:object}"` // Source of the document

	Type string `json:"type,omitempty" elastic_mapping:"type:{type:keyword,copy_to:combined_fulltext}"` // Document type, such as PDF, Docx, etc.

	Category    string `json:"category,omitempty" elastic_mapping:"category:{type:keyword,copy_to:combined_fulltext}"`       // Primary category of the document (e.g., "report", "article")
	Subcategory string `json:"subcategory,omitempty" elastic_mapping:"subcategory:{type:keyword,copy_to:combined_fulltext}"` // Secondary category of the document (e.g., "report", "article")

	//use categories for very complex hierarchy categories
	Categories []string `json:"categories,omitempty" elastic_mapping:"categories:{type:keyword,copy_to:combined_fulltext}"` // Full hierarchy of categories, useful for detailed classification

	//use rich_categories for icon need to display for each category
	RichCategories []RichLabel `json:"rich_categories,omitempty" elastic_mapping:"rich_categories:{type:object}"` // Full hierarchy of categories, useful for detailed classification, with icon decoration

	Title   string `json:"title,omitempty" elastic_mapping:"title:{type:text,copy_to:combined_fulltext,fields:{keyword: {type: keyword}, pinyin: {type: text, analyzer: pinyin_analyzer}}}"` // Document title
	Summary string `json:"summary,omitempty" elastic_mapping:"summary:{type:text,copy_to:combined_fulltext}"`                                                                                // Brief summary or description of the document

	Lang    string `json:"lang,omitempty" elastic_mapping:"lang:{type:keyword,copy_to:combined_fulltext}"`    // Language code (e.g., "en", "fr")
	Content string `json:"content,omitempty" elastic_mapping:"content:{type:text,copy_to:combined_fulltext}"` // Document content for full-text indexing

	Icon      string `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"`           // Icon Key, need work with datasource's assets to get the icon url, if it is a full url, then use it directly
	Thumbnail string `json:"thumbnail,omitempty" elastic_mapping:"thumbnail:{enabled:false}"` // Thumbnail image URL, for preview purposes
	Cover     string `json:"cover,omitempty" elastic_mapping:"cover:{enabled:false}"`         // Cover image URL, if applicable

	Owner *UserInfo `json:"owner,omitempty" elastic_mapping:"owner:{type:object}"` // Document author or owner

	Tags []string `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword,copy_to:combined_fulltext}"` // Tags or keywords associated with the document, for easier retrieval
	URL  string   `json:"url,omitempty" elastic_mapping:"url:{enabled:false}"`                            // Direct link to the document, if available
	Size int      `json:"size,omitempty" elastic_mapping:"size:{type:long}"`                              // File size in bytes, if applicable

	LastUpdatedBy *EditorInfo `json:"last_updated_by,omitempty" elastic_mapping:"last_updated_by:{type:object}"` // Struct containing last update information

}

func (document *Document) GetAllCategories() string {
	// Initialize a slice to hold all category strings
	var allCategories []string

	// Add the primary category if available
	if document.Category != "" {
		allCategories = append(allCategories, document.Category)
	}

	// Add the subcategory if available
	if document.Subcategory != "" {
		allCategories = append(allCategories, document.Subcategory)
	}

	// Add all categories if available
	if len(document.Categories) > 0 {
		allCategories = append(allCategories, document.Categories...)
	}

	// Add rich category labels if available (only the text)
	if len(document.RichCategories) > 0 {
		for _, richCategory := range document.RichCategories {
			// Assuming RichLabel has a `Label` field to hold the category text
			allCategories = append(allCategories, richCategory.Label)
		}
	}

	// Join all the categories with a comma
	return strings.Join(allCategories, ", ")
}

func (document *Document) Cleanup() {
	document.TrimLastDuplicatedCategory()
}

func (document *Document) TrimLastDuplicatedCategory() {
	// Ensure RichCategories is not empty before accessing the last element
	if len(document.RichCategories) > 0 &&
		strings.TrimSpace(document.RichCategories[len(document.RichCategories)-1].Label) == strings.TrimSpace(document.Title) {
		// Remove the last category if it matches the book's title (it's redundant)
		document.RichCategories = document.RichCategories[:len(document.RichCategories)-1]
	}
}

type EditorInfo struct {
	UserInfo  UserInfo   `json:"user,omitempty" elastic_mapping:"user:{type:object}"` // Information about the last user who updated the document
	UpdatedAt *time.Time `json:"timestamp,omitempty"`                                 // Timestamp of the last update
}

// UserInfo represents information about a user in relation to document edits or ownership.
type UserInfo struct {
	UserAvatar string `json:"avatar,omitempty" elastic_mapping:"avatar:{enabled:false}"`                              // Login of the user
	UserName   string `json:"username,omitempty" elastic_mapping:"username:{type:keyword,copy_to:combined_fulltext}"` // Login of the user
	UserID     string `json:"userid,omitempty" elastic_mapping:"userid:{type:keyword,copy_to:combined_fulltext}"`     // Unique identifier for the user
}
