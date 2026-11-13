/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package core

import "infini.sh/framework/core/orm"

// MCPEndpoint (W7/P5): a named bearer token an external agent (Cursor,
// Claude, CI) presents against the MCP server. The token plaintext is
// shown exactly once at creation/rotation; only its SHA-256 is stored.
// Scope fields (datasource allowlist, tool groups) are persisted now and
// enforced as the per-tool narrowing lands — the token itself, its
// rotation and its rate limit are live immediately.
type MCPEndpoint struct {
	orm.ORMObjectBase

	Name               string   `json:"name" elastic_mapping:"name:{type:keyword}"`
	Enabled            bool     `json:"enabled" elastic_mapping:"enabled:{type:boolean}"`
	TokenHash          string   `json:"token_hash,omitempty" elastic_mapping:"token_hash:{type:keyword}"`
	DatasourceIDs      []string `json:"datasource_ids,omitempty" elastic_mapping:"datasource_ids:{type:keyword}"` // empty = all
	ToolGroups         []string `json:"tool_groups,omitempty" elastic_mapping:"tool_groups:{type:keyword}"`       // read/ask/wiki/write; empty = all; write off unless explicit
	RateLimitPerMinute int      `json:"rate_limit_per_minute,omitempty" elastic_mapping:"rate_limit_per_minute:{type:integer}"`
	LastUsedAt         int64    `json:"last_used_at,omitempty" elastic_mapping:"last_used_at:{type:long}"`
}
