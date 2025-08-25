/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package github

// Config defines the configuration for the GitHub connector.
type Config struct {
	Token             string   `config:"token"`
	Owner             string   `config:"owner"`
	Repos             []string `config:"repos"`
	IndexIssues       bool     `config:"index_issues"`
	IndexPullRequests bool     `config:"index_pull_requests"`
}
