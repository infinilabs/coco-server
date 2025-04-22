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

import (
	"infini.sh/framework/core/orm"
	ccache "infini.sh/framework/lib/cache"
	"time"
)

const (
	generalCache                  = "general_object_cache"
	DisabledDatasourceIDsCacheKey = "disabled_datasource_ids"
	EnabledDatasourceIDsCacheKey  = "enabled_datasource_ids"
	EnabledMCPServerIDsCacheKey   = "enabled_mcp_server_ids"

	AssistantCachePrimary = "assistant"
)

var GeneralObjectCache = ccache.Layered(ccache.Configure().MaxSize(10000).ItemsToPrune(100))

func ClearDatasourceCache() {
	GeneralObjectCache.Delete(generalCache, DisabledDatasourceIDsCacheKey)
	GeneralObjectCache.Delete(generalCache, EnabledDatasourceIDsCacheKey)
}

// GetDisabledDatasourceIDs retrieves the list of disabled data source IDs from the cache.
func GetDisabledDatasourceIDs() ([]string, error) {
	item := GeneralObjectCache.Get(generalCache, DisabledDatasourceIDsCacheKey)
	var datasourceIDs []string
	if item != nil && !item.Expired() {
		var ok bool
		if datasourceIDs, ok = item.Value().([]string); ok {
			return datasourceIDs, nil
		}
	}
	// Cache is empty, read from database and cache the IDs
	var datasources []DataSource
	q := orm.Query{
		Conds: orm.And(orm.Eq("enabled", false)), // Query for disabled data sources
	}
	err, _ := orm.SearchWithJSONMapper(&datasources, &q)
	if err != nil {
		return nil, err
	}

	// Extract IDs from the retrieved data sources
	datasourceIDs = make([]string, len(datasources))
	for i, ds := range datasources {
		datasourceIDs[i] = ds.ID
	}
	GeneralObjectCache.Set(generalCache, DisabledDatasourceIDsCacheKey, datasourceIDs, time.Duration(30)*time.Minute)
	return datasourceIDs, nil

}

func GetAllEnabledDatasourceIDs() ([]string, error) {
	item := GeneralObjectCache.Get(generalCache, EnabledDatasourceIDsCacheKey)
	var datasourceIDs []string
	if item != nil && !item.Expired() {
		var ok bool
		if datasourceIDs, ok = item.Value().([]string); ok {
			return datasourceIDs, nil
		}
	}
	// Cache is empty, read from database and cache the IDs
	var datasources []DataSource
	q := orm.Query{
		Conds: orm.And(orm.Eq("enabled", true)),
	}
	err, _ := orm.SearchWithJSONMapper(&datasources, &q)
	if err != nil {
		return nil, err
	}

	// Extract IDs from the retrieved data sources
	datasourceIDs = make([]string, len(datasources))
	for i, ds := range datasources {
		datasourceIDs[i] = ds.ID
	}
	GeneralObjectCache.Set(generalCache, EnabledDatasourceIDsCacheKey, datasourceIDs, time.Duration(30)*time.Minute)
	return datasourceIDs, nil

}

func ClearMCPServerCache() {
	GeneralObjectCache.Delete(generalCache, EnabledMCPServerIDsCacheKey)
}

func GetAllEnabledMCPServerIDs() ([]string, error) {
	item := GeneralObjectCache.Get(generalCache, EnabledMCPServerIDsCacheKey)
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
	GeneralObjectCache.Set(generalCache, EnabledMCPServerIDsCacheKey, idArray, time.Duration(30)*time.Minute)
	return idArray, nil

}
