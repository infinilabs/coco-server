/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package yuque

import "time"

// Document represents detailed information about a document.
type Document struct {
	ID               int64     `json:"id"`                 // Document ID
	Type             string    `json:"type"`               // Document type (Doc: Regular document, Sheet: Spreadsheet, Thread: Topic, Board: Gallery, Table: Data table)
	Slug             string    `json:"slug"`               // Path or URL slug
	Title            string    `json:"title"`              // Title of the document
	Description      string    `json:"description"`        // Summary or abstract of the document
	Cover            string    `json:"cover"`              // Cover image URL
	UserID           int64     `json:"user_id"`            // ID of the owning user or team
	BookID           int64     `json:"book_id"`            // ID of the associated knowledge base
	LastEditorID     int64     `json:"last_editor_id"`     // ID of the last editor
	Public           int       `json:"public"`             // Visibility (0: Private, 1: Public, 2: Internal)
	Status           int       `json:"status"`             // Status (0: Draft, 1: Published)
	LikesCount       int       `json:"likes_count"`        // Number of likes
	ReadCount        int       `json:"read_count"`         // Number of reads
	Hits             int       `json:"hits,omitempty"`     // Deprecated: Number of reads (optional, requires `optional_properties=hits`)
	CommentsCount    int       `json:"comments_count"`     // Number of comments
	WordCount        int       `json:"word_count"`         // Word count of the content
	CreatedAt        time.Time `json:"created_at"`         // Creation timestamp (ISO 8601 format)
	UpdatedAt        time.Time `json:"updated_at"`         // Last update timestamp (ISO 8601 format)
	ContentUpdatedAt time.Time `json:"content_updated_at"` // Content last update timestamp (ISO 8601 format)
	PublishedAt      time.Time `json:"published_at"`       // Publish timestamp (ISO 8601 format)
	FirstPublishedAt time.Time `json:"first_published_at"` // First publish timestamp (ISO 8601 format)
	Book             *Book     `json:"book"`               // Associated knowledge base information
	User             *User     `json:"user"`               // Owning user information
	LastEditor       *User     `json:"last_editor"`        // Last editor information
	LatestVersionID  int64     `json:"latest_version_id"`  // Latest published version ID (optional, requires `optional_properties=latest_version_id`)
	Tags             []Tag     `json:"tags"`               // Associated tags
}

type DocumentDetail struct {
	ID               int64     `json:"id"`                 // Document ID
	Type             string    `json:"type"`               // Document type (e.g., Doc, Sheet, Thread, Board, Table)
	Slug             string    `json:"slug"`               // Path or slug
	Title            string    `json:"title"`              // Title of the document
	Description      string    `json:"description"`        // Summary or description
	Cover            string    `json:"cover"`              // Cover image URL
	UserID           int64     `json:"user_id"`            // ID of the owning user/team
	BookID           int64     `json:"book_id"`            // Knowledge base ID
	LastEditorID     int64     `json:"last_editor_id"`     // Last editor's user ID
	Format           string    `json:"format"`             // Content format (markdown, lake, html, lakesheet)
	BodyDraft        string    `json:"body_draft"`         // Content draft
	Body             string    `json:"body"`               // Raw content
	BodySheet        string    `json:"body_sheet"`         // Content for sheet-type documents (in JSON format)
	BodyTable        string    `json:"body_table"`         // Content for table-type documents (in JSON format)
	BodyHTML         string    `json:"body_html"`          // Content in HTML format
	BodyLake         string    `json:"body_lake"`          // Content in Yuque Lake format
	Public           int       `json:"public"`             // Visibility (0: private, 1: public, 2: enterprise-wide public)
	Status           int       `json:"status"`             // Status (0: draft, 1: published)
	LikesCount       int       `json:"likes_count"`        // Number of likes
	ReadCount        int       `json:"read_count"`         // Number of reads
	Hits             int       `json:"hits"`               // Deprecated: Number of reads
	CommentsCount    int       `json:"comments_count"`     // Number of comments
	WordCount        int       `json:"word_count"`         // Word count of the content
	CreatedAt        time.Time `json:"created_at"`         // Creation timestamp (ISO 8601 format)
	UpdatedAt        time.Time `json:"updated_at"`         // Last update timestamp (ISO 8601 format)
	ContentUpdatedAt time.Time `json:"content_updated_at"` // Content update timestamp (ISO 8601 format)
	PublishedAt      time.Time `json:"published_at"`       // Publication timestamp (ISO 8601 format)
	FirstPublishedAt time.Time `json:"first_published_at"` // First publication timestamp (ISO 8601 format)
	Book             *Book     `json:"book"`               // Associated book object
	User             *User     `json:"user"`               // Associated user object
	Creator          *User     `json:"creator"`            // Creator's user object
	Tags             []Tag     `json:"tags"`               // Associated tags
	LatestVersionID  int64     `json:"latest_version_id"`  // ID of the latest published version
}
