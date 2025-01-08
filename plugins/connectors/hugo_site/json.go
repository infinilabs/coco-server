/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package hugo_site

type HugoDocument struct {
	Category    string   `json:"category"`    // The main category of the document
	Content     string   `json:"content"`     // The content description
	Subcategory string   `json:"subcategory"` // The subcategory of the document
	Summary     string   `json:"summary"`     // A brief summary
	Tags        []string `json:"tags"`        // Tags associated with the document
	Title       string   `json:"title"`       // The title of the document
	URL         string   `json:"url"`         // The URL for the document reference
}

