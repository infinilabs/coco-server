/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package yuque

import "time"

type Book struct {
	ID               int64     `json:"id"`
	Type             string    `json:"type"` // Document type (e.g., Book, Design, Sheet, Resource)
	Slug             string    `json:"slug"`
	Name             string    `json:"name"`
	UserID           int64     `json:"user_id"`
	Description      string    `json:"description"`
	CreatorID        int64     `json:"creator_id"`
	Public           int       `json:"public"`
	ItemsCount       int       `json:"items_count"`
	LikesCount       int       `json:"likes_count"`
	WatchesCount     int       `json:"watches_count"`
	ContentUpdatedAt time.Time `json:"content_updated_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	User             User      `json:"user"`
	Namespace        string    `json:"namespace"`
}

type BookDetail struct {
	ID               int64     `json:"id"`                 // Knowledge base ID
	Type             string    `json:"type"`               // Type (Book: Document, Design: Gallery, Sheet: Spreadsheet, Resource: Asset)
	Slug             string    `json:"slug"`               // Path or URL slug
	Name             string    `json:"name"`               // Name of the knowledge base
	UserID           int64     `json:"user_id"`            // ID of the owning user or team
	Description      string    `json:"description"`        // Description or summary of the knowledge base
	TocYML           string    `json:"toc_yml"`            // Table of Contents in YAML format
	CreatorID        int64     `json:"creator_id"`         // ID of the creator
	Public           int       `json:"public"`             // Visibility (0: Private, 1: Public, 2: Internal)
	ItemsCount       int       `json:"items_count"`        // Number of documents
	LikesCount       int       `json:"likes_count"`        // Number of likes
	WatchesCount     int       `json:"watches_count"`      // Number of subscriptions
	ContentUpdatedAt time.Time `json:"content_updated_at"` // Last update time of the META data (ISO 8601 format)
	CreatedAt        time.Time `json:"created_at"`         // Creation time (ISO 8601 format)
	UpdatedAt        time.Time `json:"updated_at"`         // Last update time (ISO 8601 format)
	User             User      `json:"user"`               // Associated user information
	Namespace        string    `json:"namespace"`          // Full path or namespace
}

type BookToc struct {
	UUID        string      `json:"uuid"`                  // Unique ID of the node
	Type        string      `json:"type"`                  // Node type (DOC: document, LINK: external link, TITLE: group)
	Title       string      `json:"title"`                 // Name of the node
	URL         string      `json:"url"`                   // URL of the node
	Slug        string      `json:"slug"`                  // Deprecated: URL slug of the node
	ID          interface{} `json:"id,omitempty"`          // Deprecated: Document ID
	DocID       interface{} `json:"doc_id,omitempty"`      // Document ID
	Level       int         `json:"level,omitempty"`       // Level of the node in the hierarchy
	Depth       int         `json:"depth,omitempty"`       // Deprecated: Depth of the node in the hierarchy
	OpenWindow  int         `json:"open_window,omitempty"` // Whether to open in a new window (0: current page, 1: new window)
	Visible     int         `json:"visible,omitempty"`     // Visibility of the node (0: not visible, 1: visible)
	PrevUUID    string      `json:"prev_uuid"`             // UUID of the previous sibling node at the same level
	SiblingUUID string      `json:"sibling_uuid"`          // UUID of the next sibling node at the same level
	ChildUUID   string      `json:"child_uuid"`            // UUID of the first child node
	ParentUUID  string      `json:"parent_uuid"`           // UUID of the parent node
}
