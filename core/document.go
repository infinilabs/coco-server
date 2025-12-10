/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"fmt"
	"strings"
	"time"
	"unsafe"
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

	Lang      string      `json:"lang,omitempty" elastic_mapping:"lang:{type:keyword,copy_to:combined_fulltext}"`    // Language code (e.g., "en", "fr")
	Content   string      `json:"content,omitempty" elastic_mapping:"content:{type:text,copy_to:combined_fulltext}"` // Document content for full-text indexing
	Text      []PageText  `json:"text,omitempty" elastic_mapping:"text:{type:nested}"`                               // Document content in text for full-text indexing
	Embedding []Embedding `json:"embedding,omitempty" elastic_mapping:"embedding:{type:nested}"`

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

type PageText struct {
	PageNumber int    `json:"page_number" elastic_mapping:"page_number:{type:integer}"`
	Content    string `json:"content" elastic_mapping:"content:{type:text,analyzer:combined_text_analyzer}"`
}

type Embedding struct {
	ModelProvider      string `json:"model_provider" elastic_mapping:"model_provider:{type:keyword}"`
	Model              string `json:"model" elastic_mapping:"model:{type:keyword}"`
	EmbeddingDimension int32  `json:"embedding_dimension" elastic_mapping:"embedding_dimension:{type:integer}"`

	/*
		A document will be split into chunks. An `EmbeddingXxx` field will be used
		to store these chunks' embeddings.

		Only 1 field will be used, depending on the chosen embedding dimension, see
		the `EmbeddingDimension` field above.

		Having so many `EmbeddingXxx` fields is embarrasing, but we have no choice
		since vector dimension is part of the type information and elastic mapping
		has to be static.

		If you add new fields or remove fields to/from the below list, please update
		variable "SupportedEmbeddingDimensions" as well.
	*/
	Embeddings128  []ChunkEmbedding128  `json:"embeddings128" elastic_mapping:"embeddings128:{type:nested}"`
	Embeddings256  []ChunkEmbedding256  `json:"embeddings256" elastic_mapping:"embeddings256:{type:nested}"`
	Embeddings384  []ChunkEmbedding384  `json:"embeddings384" elastic_mapping:"embeddings384:{type:nested}"`
	Embeddings512  []ChunkEmbedding512  `json:"embeddings512" elastic_mapping:"embeddings512:{type:nested}"`
	Embeddings768  []ChunkEmbedding768  `json:"embeddings768" elastic_mapping:"embeddings768:{type:nested}"`
	Embeddings1024 []ChunkEmbedding1024 `json:"embeddings1024" elastic_mapping:"embeddings1024:{type:nested}"`
	Embeddings1536 []ChunkEmbedding1536 `json:"embeddings1536" elastic_mapping:"embeddings1536:{type:nested}"`
	Embeddings2048 []ChunkEmbedding2048 `json:"embeddings2048" elastic_mapping:"embeddings2048:{type:nested}"`
	Embeddings2560 []ChunkEmbedding2560 `json:"embeddings2560" elastic_mapping:"embeddings2560:{type:nested}"`
	Embeddings4096 []ChunkEmbedding4096 `json:"embeddings4096" elastic_mapping:"embeddings4096:{type:nested}"`
}

// Embedding dimensions supported by us, it should be kept sync with the 
// "EmbeddingXxx" fields of struct Embedding
var SupportedEmbeddingDimensions = []int32{128, 256, 384, 512, 768, 1024, 1536, 2048, 2560, 4096 }

// Set the `EmbeddingsXxx` field using the value provided by `chunkEmbeddings`.
//
// # Panic
//
// Field `EmbeddingDimension` should be set before calling this function, or it
// panics.
func (e *Embedding) SetEmbeddings(chunkEmbeddings []ChunkEmbedding) {
	if e.EmbeddingDimension == 0 {
		panic("Embedding.EmbeddingDimension is not set (value: 0), don't know which field to set")
	}

	// ChunkEmbedding and other ChunkEmbeddingXxx types have the same memory
	// representation so the cast is safe here.
	switch e.EmbeddingDimension {
	case 128:
		e.Embeddings128 = *(*[]ChunkEmbedding128)(unsafe.Pointer(&chunkEmbeddings))
	case 256:
		e.Embeddings256 = *(*[]ChunkEmbedding256)(unsafe.Pointer(&chunkEmbeddings))
	case 384:
		e.Embeddings384 = *(*[]ChunkEmbedding384)(unsafe.Pointer(&chunkEmbeddings))
	case 512:
		e.Embeddings512 = *(*[]ChunkEmbedding512)(unsafe.Pointer(&chunkEmbeddings))
	case 768:
		e.Embeddings768 = *(*[]ChunkEmbedding768)(unsafe.Pointer(&chunkEmbeddings))
	case 1024:
		e.Embeddings1024 = *(*[]ChunkEmbedding1024)(unsafe.Pointer(&chunkEmbeddings))
	case 1536:
		e.Embeddings1536 = *(*[]ChunkEmbedding1536)(unsafe.Pointer(&chunkEmbeddings))
	case 2048:
		e.Embeddings2048 = *(*[]ChunkEmbedding2048)(unsafe.Pointer(&chunkEmbeddings))
	case 2560:
		e.Embeddings2560 = *(*[]ChunkEmbedding2560)(unsafe.Pointer(&chunkEmbeddings))
	case 4096:
		e.Embeddings4096 = *(*[]ChunkEmbedding4096)(unsafe.Pointer(&chunkEmbeddings))
	default:
		panic(fmt.Sprintf("unsupported embedding dimension: %d\n", e.EmbeddingDimension))
	}
}

// Range of this chunk.
//
// A chunk contains roughly the same amount of tokens, say 8192 tokens. And
// thus, a chunk can span many pages if these pages are small, or it is only
// part of a page if it is big.
//
// In the later case, `Start` and `End` will be in format "<page num>-<sub-page num>"
// that "<sub-page num>" specifies the part of that page.
type ChunkRange struct {
	// Start page of this chunk.
	Start int `json:"start" elastic_mapping:"start:{type:integer}"`
	// End page of this chuhk. This is **inclusive**.
	End int `json:"end" elastic_mapping:"end:{type:integer}"`
}

// A `ChunkEmbedding` definition without any tag information.
//
// It should have the same memory representation as other `ChunkEmbeddingXxx`
// variants.
type ChunkEmbedding struct {
	Range     ChunkRange
	Embedding []float32
}

type ChunkEmbedding128 struct {
	Range     ChunkRange `json:"page_range" elastic_mapping:"page_range:{type:object}"`
	Embedding []float32  `json:"embedding" elastic_mapping:"embedding:{type:knn_dense_float_vector,knn:{dims:128}}"`
}

type ChunkEmbedding256 struct {
	Range     ChunkRange `json:"page_range" elastic_mapping:"page_range:{type:object}"`
	Embedding []float32  `json:"embedding" elastic_mapping:"embedding:{type:knn_dense_float_vector,knn:{dims:256}}"`
}

type ChunkEmbedding384 struct {
	Range     ChunkRange `json:"page_range" elastic_mapping:"page_range:{type:object}"`
	Embedding []float32  `json:"embedding" elastic_mapping:"embedding:{type:knn_dense_float_vector,knn:{dims:384}}"`
}

type ChunkEmbedding512 struct {
	Range     ChunkRange `json:"page_range" elastic_mapping:"page_range:{type:object}"`
	Embedding []float32  `json:"embedding" elastic_mapping:"embedding:{type:knn_dense_float_vector,knn:{dims:512}}"`
}

type ChunkEmbedding768 struct {
	Range     ChunkRange `json:"page_range" elastic_mapping:"page_range:{type:object}"`
	Embedding []float32  `json:"embedding" elastic_mapping:"embedding:{type:knn_dense_float_vector,knn:{dims:768}}"`
}

type ChunkEmbedding1024 struct {
	Range     ChunkRange `json:"page_range" elastic_mapping:"page_range:{type:object}"`
	Embedding []float32  `json:"embedding" elastic_mapping:"embedding:{type:knn_dense_float_vector,knn:{dims:1024}}"`
}

type ChunkEmbedding1536 struct {
	Range     ChunkRange `json:"page_range" elastic_mapping:"page_range:{type:object}"`
	Embedding []float32  `json:"embedding" elastic_mapping:"embedding:{type:knn_dense_float_vector,knn:{dims:1536}}"`
}

type ChunkEmbedding2048 struct {
	Range     ChunkRange `json:"page_range" elastic_mapping:"page_range:{type:object}"`
	Embedding []float32  `json:"embedding" elastic_mapping:"embedding:{type:knn_dense_float_vector,knn:{dims:2048}}"`
}

type ChunkEmbedding2560 struct {
	Range     ChunkRange `json:"page_range" elastic_mapping:"page_range:{type:object}"`
	Embedding []float32  `json:"embedding" elastic_mapping:"embedding:{type:knn_dense_float_vector,knn:{dims:2560}}"`
}

type ChunkEmbedding4096 struct {
	Range     ChunkRange `json:"page_range" elastic_mapping:"page_range:{type:object}"`
	Embedding []float32  `json:"embedding" elastic_mapping:"embedding:{type:knn_dense_float_vector,knn:{dims:4096}}"`
}
