/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

// registered connectors
type Connector struct {
	CombinedFullText
	Name        string   `json:"name,omitempty" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"`            // Source of the document (e.g., "github", "google_drive", "dropbox")
	Description string   `json:"description,omitempty" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"` // Source of the document (e.g., "github", "google_drive", "dropbox")
	Category    string   `json:"category,omitempty" elastic_mapping:"category:{type:keyword,copy_to:combined_fulltext}"`    // Primary category of the document (e.g., "report", "article")
	Icon        string   `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"`                                     // Thumbnail image URL, for preview purposes
	Tags        []string `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword,copy_to:combined_fulltext}"`            // Tags or keywords associated with the document, for easier retrieval
	URL         string   `json:"url,omitempty" elastic_mapping:"url:{enabled:false}"`                                       // Direct link to the document, if available

	Assets struct {
		Icons map[string]string `json:"icons,omitempty" elastic_mapping:"icons:{enabled:false}"` //icon_key -> URL
	} `json:"assets,omitempty" elastic_mapping:"assets:{enabled:false}"`
	Builtin bool                   `json:"builtin" elastic_mapping:"builtin:{type:boolean}"`          // Whether the connector is built-in or user-defined
	Config  map[string]interface{} `json:"config,omitempty" elastic_mapping:"config:{enabled:false}"` // Connector-specific configuration settings
}
