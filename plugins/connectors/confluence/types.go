/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package confluence

// User defines user information
type User struct {
	Type        string `json:"type"`
	Username    string `json:"username"`
	UserKey     string `json:"userKey"`
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
}

// Results array represent search results
type Results struct {
	ID      string  `json:"id,omitempty"`
	Type    string  `json:"type,omitempty"`
	Status  string  `json:"status,omitempty"`
	Content Content `json:"content"`
	Excerpt string  `json:"excerpt,omitempty"`
	Title   string  `json:"title,omitempty"`
	URL     string  `json:"url,omitempty"`
}

// Content specifies content properties
type Content struct {
	ID         string      `json:"id,omitempty"`
	Type       string      `json:"type"`
	Status     string      `json:"status,omitempty"`
	Title      string      `json:"title"`
	Ancestors  []Ancestor  `json:"ancestors,omitempty"`
	Body       Body        `json:"body"`
	Version    *Version    `json:"version,omitempty"`
	Space      *Space      `json:"space"`
	History    *History    `json:"history,omitempty"`
	Links      *Links      `json:"_links,omitempty"`
	Extensions *Extensions `json:"extensions,omitempty"`
}

// Body represents the storage information
type Body struct {
	Storage Storage  `json:"storage"`
	View    *Storage `json:"view,omitempty"`
}

// Storage represents the storage information
type Storage struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// Version represents the content version number that used for updating content
type Version struct {
	Number    int    `json:"number"`
	MinorEdit bool   `json:"minorEdit"`
	Message   string `json:"message,omitempty"`
	By        *User  `json:"by,omitempty"`
	When      string `json:"when,omitempty"`
}

// Space represents the space information. A page has a space.
type Space struct {
	ID     int    `json:"id,omitempty"`
	Key    string `json:"key,omitempty"`
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
	Status string `json:"status,omitempty"`
}

// Links represents link information
type Links struct {
	Base     string `json:"base"`
	TinyUI   string `json:"tinyui"`
	WebUI    string `json:"webui"`
	Download string `json:"download"`
}

// Ancestor represents ancestors to create sub-pages
type Ancestor struct {
	ID string `json:"id"`
}

// History contains object history information
type History struct {
	LastUpdated LastUpdated `json:"lastUpdated"`
	Latest      bool        `json:"latest"`
	CreatedBy   User        `json:"createdBy"`
	CreatedDate string      `json:"createdDate"`
}

// LastUpdated  contains information about the last update
type LastUpdated struct {
	By           User   `json:"by"`
	When         string `json:"when"`
	FriendlyWhen string `json:"friendlyWhen"`
	Message      string `json:"message"`
	Number       int    `json:"number"`
	MinorEdit    bool   `json:"minorEdit"`
	SyncRev      string `json:"syncRev"`
	ConfRev      string `json:"confRev"`
}

// Labels is the label container type
type Labels struct {
	Labels []Label `json:"results"`
	Start  int     `json:"start,omitempty"`
	Limit  int     `json:"limit,omitempty"`
	Size   int     `json:"size,omitempty"`
}

// Label contains label information
type Label struct {
	Prefix string `json:"prefix"`
	Name   string `json:"name"`
	ID     string `json:"id,omitempty"`
	Label  string `json:"label,omitempty"`
}

// SearchLinks parsing out the _links section to allow paging etc.
type SearchLinks struct {
	Base    string `json:"base,omitempty"`
	Context string `json:"content,omitempty"`
	Next    string `json:"next,omitempty"`
	Self    string `json:"self,omitempty"`
}

type Extensions struct {
	MediaType string `json:"mediaType,omitempty"`
	FileSize  int    `json:"fileSize,omitempty"`
}

// SearchContentRequest represents query parameters that are used in the Confluence content search API
// Link: https://developer.atlassian.com/cloud/confluence/rest/v1/api-group-content/#api-wiki-rest-api-content-search-get
// Link: https://developer.atlassian.com/cloud/confluence/advanced-searching-using-cql/
type SearchContentRequest struct {
	CQL        string
	CQLContext string
	Limit      int
	Start      int
	Expand     []string
	// Excerpt               string
	// IncludeArchivedSpaces string
}

// SearchContentResponse represents get content result
type SearchContentResponse struct {
	Results []Content    `json:"results"`
	Start   int          `json:"start,omitempty"`
	Limit   int          `json:"limit,omitempty"`
	Size    int          `json:"size,omitempty"`
	Links   *SearchLinks `json:"_links,omitempty"`
}

func (s *SearchContentResponse) Next() string {
	if s.Links == nil {
		return ""
	}
	return s.Links.Next
}
