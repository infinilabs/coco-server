/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"time"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
)

const StreamableHTTP = "streamable_http"
const SSE = "sse"
const Stdio = "stdio"

const (
	MCPServerCachePrimary       = "mcp_server"
	MCPServerItemCacheKey       = "mcp_server_item"
	EnabledMCPServerIDsCacheKey = "enabled_mcp_server_ids"
)

func GetMPCServer(id string) (*core.MCPServer, error) {
	item := GeneralObjectCache.Get(MCPServerItemCacheKey, id)
	var server *core.MCPServer
	if item != nil && !item.Expired() {
		var ok bool
		if server, ok = item.Value().(*core.MCPServer); ok {
			return server, nil
		}
	}

	server = &core.MCPServer{}
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
	var server []core.MCPServer
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
