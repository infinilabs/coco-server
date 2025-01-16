/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package hugo_site

type HugoDocument struct {
	Category    string   `json:"category,omitempty"`    // The main category of the document
	Content     string   `json:"content,omitempty"`     // The content description
	Subcategory string   `json:"subcategory,omitempty"` // The subcategory of the document
	Summary     string   `json:"summary,omitempty"`     // A brief summary
	Tags        []string `json:"tags,omitempty"`        // Tags associated with the document
	Title       string   `json:"title,omitempty"`       // The title of the document
	URL         string   `json:"url,omitempty"`         // The URL for the document reference
	Created     string   `json:"created,omitempty"`
	Updated     string   `json:"updated,omitempty"`
	Lang        string   `json:"lang,omitempty"`
}
