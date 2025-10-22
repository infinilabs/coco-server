/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package hugo_site

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

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

type Config struct {
	Urls []string `json:"urls" config:"urls"`
}

// ParseTimestamp safely parses a timestamp string into a *time.Time.
// Returns nil if parsing fails.
func ParseTimestamp(timestamp string) *time.Time {
	layout := time.RFC3339 // ISO 8601 format
	parsedTime, err := time.Parse(layout, timestamp)
	if err != nil {
		return nil
	}
	return &parsedTime
}

// Function to construct the full URL using only the domain from the seed URL
func getFullURL(seedURL, relativePath string) (string, error) {
	// Parse the seed URL
	parsedURL, err := url.Parse(seedURL)
	if err != nil {
		return "", fmt.Errorf("invalid seed URL: %w", err)
	}

	// Extract the domain (scheme and host)
	domain := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// Remove any leading "/" from relativePath to avoid duplication
	relativePath = strings.TrimPrefix(relativePath, "/")

	// Combine the domain with the relative path
	fullURL := fmt.Sprintf("%s/%s", domain, relativePath)

	return fullURL, nil
}
