/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"fmt"
	"strings"
	"time"
)

type RichLabel struct {
	Label string `json:"label,omitempty" elastic_mapping:"label:{type:keyword,copy_to:combined_fulltext}"`
	Key   string `json:"key,omitempty" elastic_mapping:"key:{type:keyword}"`
	Icon  string `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"` // Icon Key, need work with datasource's assets to get the icon url
}

type DataSourceReference struct {
	Type string `json:"type,omitempty" elastic_mapping:"type:{type:keyword}"`  // ID of the datasource, eg: connector
	Name string `json:"name,omitempty" elastic_mapping:"name:{type:keyword}"`  // Name of the datasource (e.g., "My Github", "My Google Drive", "My Dropbox")
	ID   string `json:"id,omitempty" elastic_mapping:"id:{type:keyword}"`      // ID of this the datasource, eg: 8ca2fe8cf5027b0f1b5f932b429e38c3
	Icon string `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"` // Icon Key, need work with datasource's assets to get the icon url, if it is a full url, then use it directly
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

	Lang                   string          `json:"lang,omitempty" elastic_mapping:"lang:{type:keyword,copy_to:combined_fulltext}"`    // Language code (e.g., "en", "fr")
	Content                string          `json:"content,omitempty" elastic_mapping:"content:{type:text,copy_to:combined_fulltext}"` // Document content for full-text indexing
	Chunks                 []DocumentChunk `json:"document_chunk,omitempty" elastic_mapping:"document_chunk:{type:nested}"`
	ChunksWithImageContent []DocumentChunk `json:"document_chunk_with_image_content,omitempty" elastic_mapping:"document_chunk_with_image_content:{type:nested}"`
	Images                 []PageImages    `json:"images,omitempty" elastic_mapping:"images:{type:nested}"` // Images appeared in the document, grouped by page

	Icon      string `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"`           // Icon Key, need work with datasource's assets to get the icon url, if it is a full url, then use it directly
	Thumbnail string `json:"thumbnail,omitempty" elastic_mapping:"thumbnail:{enabled:false}"` // Thumbnail image URL, for preview purposes
	Cover     string `json:"cover,omitempty" elastic_mapping:"cover:{enabled:false}"`         // Cover image URL, if applicable

	Owner *UserInfo `json:"owner,omitempty" elastic_mapping:"owner:{type:object}"` // Document author or owner

	Tags []string `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword,copy_to:combined_fulltext}"` // Tags or keywords associated with the document, for easier retrieval
	URL  string   `json:"url,omitempty" elastic_mapping:"url:{enabled:false}"`                            // Direct link to the document, if available
	Size int      `json:"size,omitempty" elastic_mapping:"size:{type:long}"`                              // File size in bytes, if applicable

	LastUpdatedBy *EditorInfo `json:"last_updated_by,omitempty" elastic_mapping:"last_updated_by:{type:object}"` // Struct containing last update information
	Disabled      bool        `json:"disabled,omitempty" elastic_mapping:"disabled:{type:boolean}"`              // Whether the document is disabled or not
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
	UserInfo  *UserInfo  `json:"user,omitempty" elastic_mapping:"user:{type:object}"` // Information about the last user who updated the document
	UpdatedAt *time.Time `json:"timestamp,omitempty"`                                 // Timestamp of the last update
}

// UserInfo represents information about a user in relation to document edits or ownership.
type UserInfo struct {
	UserAvatar string `json:"avatar,omitempty" elastic_mapping:"avatar:{enabled:false}"`                              // Login of the user
	UserName   string `json:"username,omitempty" elastic_mapping:"username:{type:keyword,copy_to:combined_fulltext}"` // Login of the user
	UserID     string `json:"userid,omitempty" elastic_mapping:"userid:{type:keyword,copy_to:combined_fulltext}"`     // Unique identifier for the user
}

type DocumentChunk struct {
	Range     ChunkRange `json:"range" elastic_mapping:"range:{type:object}"`
	Text      string     `json:"text" elastic_mapping:"text:{type:text}"`
	Embedding Embedding  `json:"embedding" elastic_mapping:"embedding:{type:object}"`
}

// A `Embedding` stores a chunk's embedding.
//
// Only 1 field will be used, depending on the chosen embedding dimension, see
// the `Dimension` field above.
//
// Having so many `EmbeddingXxx` fields is embarrasing, but we have no choice
// since vector dimension is part of the type information and elastic mapping
// has to be static.
//
// If you add or remove fields, please update variable "SupportedEmbeddingDimensions"
// as well.
type Embedding struct {
	Embedding128  []float32 `json:"embedding128,omitempty" elastic_mapping:"embedding128:{type:knn_dense_float_vector,knn:{dims:128,model:lsh,similarity:cosine,L:99,k:1}}"`
	Embedding256  []float32 `json:"embedding256,omitempty" elastic_mapping:"embedding256:{type:knn_dense_float_vector,knn:{dims:256,model:lsh,similarity:cosine,L:99,k:1}}"`
	Embedding384  []float32 `json:"embedding384,omitempty" elastic_mapping:"embedding384:{type:knn_dense_float_vector,knn:{dims:384,model:lsh,similarity:cosine,L:99,k:1}}"`
	Embedding512  []float32 `json:"embedding512,omitempty" elastic_mapping:"embedding512:{type:knn_dense_float_vector,knn:{dims:512,model:lsh,similarity:cosine,L:99,k:1}}"`
	Embedding768  []float32 `json:"embedding768,omitempty" elastic_mapping:"embedding768:{type:knn_dense_float_vector,knn:{dims:768,model:lsh,similarity:cosine,L:99,k:1}}"`
	Embedding1024 []float32 `json:"embedding1024,omitempty" elastic_mapping:"embedding1024:{type:knn_dense_float_vector,knn:{dims:1024,model:lsh,similarity:cosine,L:99,k:1}}"`
	Embedding1536 []float32 `json:"embedding1536,omitempty" elastic_mapping:"embedding1536:{type:knn_dense_float_vector,knn:{dims:1536,model:lsh,similarity:cosine,L:99,k:1}}"`
	Embedding2048 []float32 `json:"embedding2048,omitempty" elastic_mapping:"embedding2048:{type:knn_dense_float_vector,knn:{dims:2048,model:lsh,similarity:cosine,L:99,k:1}}"`
	Embedding2560 []float32 `json:"embedding2560,omitempty" elastic_mapping:"embedding2560:{type:knn_dense_float_vector,knn:{dims:2560,model:lsh,similarity:cosine,L:99,k:1}}"`
	Embedding4096 []float32 `json:"embedding4096,omitempty" elastic_mapping:"embedding4096:{type:knn_dense_float_vector,knn:{dims:4096,model:lsh,similarity:cosine,L:99,k:1}}"`
}

// Set the actual value of this "Embedding"
func (e *Embedding) SetValue(embedding []float32) {
	dimension := len(embedding)
	switch dimension {
	case 128:
		e.Embedding128 = embedding
	case 256:
		e.Embedding256 = embedding
	case 384:
		e.Embedding384 = embedding
	case 512:
		e.Embedding512 = embedding
	case 768:
		e.Embedding768 = embedding
	case 1024:
		e.Embedding1024 = embedding
	case 1536:
		e.Embedding1536 = embedding
	case 2048:
		e.Embedding2048 = embedding
	case 2560:
		e.Embedding2560 = embedding
	case 4096:
		e.Embedding4096 = embedding
	default:
		panic(fmt.Sprintf("embedding's dimension is invalid, we accept %v", SupportedEmbeddingDimensions))
	}
}

// Embedding dimensions supported by us, it should be kept sync with the
// "EmbeddingXxx" fields of struct Embedding
var SupportedEmbeddingDimensions = []int32{128, 256, 384, 512, 768, 1024, 1536, 2048, 2560, 4096}

// Range of a chunk.
//
// A chunk contains roughly the same amount of tokens, say 8192 tokens. And
// thus, a chunk can span many pages if these pages are small, or it is only
// part of a page if the page is big.
type ChunkRange struct {
	// Start page of this chunk.
	Start int `json:"start" elastic_mapping:"start:{type:integer}"`
	// End page of this chunk. This is **inclusive**.
	End int `json:"end" elastic_mapping:"end:{type:integer}"`
}

// Helper struct to store images per page
type PageImages struct {
	Page      int      `json:"page" elastic_mapping:"page:{type:integer}"`
	Filenames []string `json:"filenames" elastic_mapping:"filenames:{type:keyword,copy_to:combined_fulltext}"`
}
