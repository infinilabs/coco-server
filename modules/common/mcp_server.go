// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package common

type MCPServer struct {
	CombinedFullText
	Name        string      `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"`
	Type        string      `json:"type" elastic_mapping:"type:{type:keyword,copy_to:combined_fulltext}"` // possible values: "sse", "stdio", "streamable_http"
	Description string      `json:"description,omitempty" elastic_mapping:"description:{type:keyword,copy_to:combined_fulltext}"`
	Category    string      `json:"category,omitempty" elastic_mapping:"category:{type:keyword,copy_to:combined_fulltext}"` // possible values: "sse", "stdio"
	Config      interface{} `json:"config,omitempty" elastic_mapping:"config:{enabled:false}"`
	Enabled     bool        `json:"enabled" elastic_mapping:"enabled:{type:boolean}"` // Whether the connector is enabled or not
}

type SSEConfig struct {
	URL string `json:"url" elastic_mapping:"url:{type:keyword}"`
}

// StdioConfig is a struct for the standard input/output configuration
type StdioConfig struct {
	Command string            `json:"command"`        // command to run, possible values: npx, uvx
	Args    []string          `json:"args,omitempty"` // arguments to pass to the command
	Env     map[string]string `json:"env,omitempty"`  // environment variables
}

type StreamableHttpConfig struct {
	URL string `json:"url" elastic_mapping:"url:{type:keyword}"`
}
