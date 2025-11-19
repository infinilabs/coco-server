/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package gitlab

// Config defines the configuration for the GitLab connector.
type Config struct {
	BaseURL            string   `config:"base_url"`
	Token              string   `config:"token"`
	Owner              string   `config:"owner"`
	Repos              []string `config:"repos"`
	IndexIssues        bool     `config:"index_issues"`
	IndexMergeRequests bool     `config:"index_merge_requests"`
	IndexWikis         bool     `config:"index_wikis"`
	IndexSnippets      bool     `config:"index_snippets"`
	HttpClient         string   `json:"http_client" config:"http_client"`
}
