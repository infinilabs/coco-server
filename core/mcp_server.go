/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

type MCPServer struct {
	CombinedFullText
	Name        string   `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext,fields:{text: {type: text}, pinyin: {type: text, analyzer: pinyin_analyzer}}}"`
	Description string   `json:"description,omitempty" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"`
	Icon        string   `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"`                // Display name of this datasource
	Type        string   `json:"type" elastic_mapping:"type:{type:keyword,copy_to:combined_fulltext}"` // possible values: "sse", "stdio", "streamable_http"
	Category    string   `json:"category,omitempty" elastic_mapping:"category:{type:keyword,copy_to:combined_fulltext}"`
	Tags        []string `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword}"`

	Config  interface{} `json:"config,omitempty" elastic_mapping:"config:{enabled:false}"`
	Enabled bool        `json:"enabled" elastic_mapping:"enabled:{type:boolean}"` // Whether the connector is enabled or not
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
