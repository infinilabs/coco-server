/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package yuque

import "time"

// Tag represents detailed information about a tag.
type Tag struct {
	ID        int64     `json:"id"`         // Tag ID
	Title     string    `json:"title"`      // Tag name
	DocID     int64     `json:"doc_id"`     // Document ID associated with the tag
	BookID    int64     `json:"book_id"`    // Knowledge base ID associated with the tag
	UserID    int64     `json:"user_id"`    // Creator's user ID
	CreatedAt time.Time `json:"created_at"` // Creation timestamp (ISO 8601 format)
	UpdatedAt time.Time `json:"updated_at"` // Last update timestamp (ISO 8601 format)
}
