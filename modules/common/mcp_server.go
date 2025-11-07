/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"time"
)

type MCPServer struct {
	core.CombinedFullText
	Name        string   `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"`
	Description string   `json:"description,omitempty" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"`
	Icon        string   `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"`                // Display name of this datasource
	Type        string   `json:"type" elastic_mapping:"type:{type:keyword,copy_to:combined_fulltext}"` // possible values: "sse", "stdio", "streamable_http"
	Category    string   `json:"category,omitempty" elastic_mapping:"category:{type:keyword,copy_to:combined_fulltext}"`
	Tags        []string `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword}"`

	Config  interface{} `json:"config,omitempty" elastic_mapping:"config:{enabled:false}"`
	Enabled bool        `json:"enabled" elastic_mapping:"enabled:{type:boolean}"` // Whether the connector is enabled or not
}

const StreamableHTTP = "streamable_http"
const SSE = "sse"
const Stdio = "stdio"

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

const (
	MCPServerCachePrimary       = "mcp_server"
	MCPServerItemCacheKey       = "mcp_server_item"
	EnabledMCPServerIDsCacheKey = "enabled_mcp_server_ids"
)

func GetMPCServer(id string) (*MCPServer, error) {
	item := GeneralObjectCache.Get(MCPServerItemCacheKey, id)
	var server *MCPServer
	if item != nil && !item.Expired() {
		var ok bool
		if server, ok = item.Value().(*MCPServer); ok {
			return server, nil
		}
	}

	server = &MCPServer{}
	server.ID = id
	_, err := orm.Get(server)
	if err != nil {
		return nil, err
	}
	// Cache the provider object
	GeneralObjectCache.Set(MCPServerItemCacheKey, id, server, time.Duration(30)*time.Minute)
	return server, nil
}

func ClearMCPServerCache() {
	GeneralObjectCache.Delete(MCPServerCachePrimary, EnabledMCPServerIDsCacheKey)
	GeneralObjectCache.DeleteAll(MCPServerItemCacheKey)
	GeneralObjectCache.DeleteAll(AssistantCachePrimary)
}

func GetAllEnabledMCPServerIDs() ([]string, error) {
	item := GeneralObjectCache.Get(MCPServerCachePrimary, EnabledMCPServerIDsCacheKey)
	var idArray []string
	if item != nil && !item.Expired() {
		var ok bool
		if idArray, ok = item.Value().([]string); ok {
			return idArray, nil
		}
	}
	// Cache is empty, read from database and cache the IDs
	var server []MCPServer
	q := orm.Query{
		Conds: orm.And(orm.Eq("enabled", true)),
	}
	err, _ := orm.SearchWithJSONMapper(&server, &q)
	if err != nil {
		return nil, err
	}

	// Extract IDs from the retrieved data sources
	idArray = make([]string, len(server))
	for i, ds := range server {
		idArray[i] = ds.ID
	}
	GeneralObjectCache.Set(MCPServerCachePrimary, EnabledMCPServerIDsCacheKey, idArray, time.Duration(30)*time.Minute)
	return idArray, nil

}
