/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

type Connector struct {
	CombinedFullText
	Name        string   `json:"name,omitempty" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"`               // Source of the document (e.g., "github", "google_drive", "dropbox")
	Description string   `json:"description,omitempty" elastic_mapping:"description:{type:keyword,copy_to:combined_fulltext}"` // Source of the document (e.g., "github", "google_drive", "dropbox")
	Category    string   `json:"category,omitempty" elastic_mapping:"category:{type:keyword,copy_to:combined_fulltext}"`       // Primary category of the document (e.g., "report", "article")
	Icon        string   `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"`                                        // Thumbnail image URL, for preview purposes
	Tags        []string `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword,copy_to:combined_fulltext}"`               // Tags or keywords associated with the document, for easier retrieval
	URL         string   `json:"url,omitempty" elastic_mapping:"url:{enabled:false}"`                                          // Direct link to the document, if available

	Assets struct {
		Icons map[string]string //icon_key -> URL
	}
}
