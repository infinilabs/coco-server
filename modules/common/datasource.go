/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/framework/core/orm"
	ccache "infini.sh/framework/lib/cache"
	"time"
)

type DataSource struct {
	CombinedFullText

	Type string `json:"type,omitempty" elastic_mapping:"type:{type:keyword,copy_to:combined_fulltext}"` // Type of the datasource, eg: connector
	Name string `json:"name,omitempty" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"` // Display name of this datasource
	Icon string `json:"icon,omitempty" elastic_mapping:"icon:{type:keyword}"`                           // Display name of this datasource

	Connector ConnectorConfig `json:"connector,omitempty" elastic_mapping:"connector:{type:object}"` // Connector configuration
	// Whether synchronization is allowed
	SyncEnabled bool `json:"sync_enabled" elastic_mapping:"sync_enabled:{type:keyword}"`
	Enabled     bool `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`
}

type ConnectorConfig struct {
	ConnectorID string      `json:"id,omitempty" elastic_mapping:"id:{type:keyword}"`          // Connector ID for the datasource
	Config      interface{} `json:"config,omitempty" elastic_mapping:"config:{enabled:false}"` // Configs for this Connector
}

const (
	DatasourceCachePrimary        = "datasource"
	DisabledDatasourceIDsCacheKey = "disabled_ids"
)

var DisabledDatasourceIDsCache = ccache.Layered(ccache.Configure().MaxSize(10000).ItemsToPrune(100))

// GetDisabledDatasourceIDs retrieves the list of disabled data source IDs from the cache.
func GetDisabledDatasourceIDs() ([]string, error) {
	item := DisabledDatasourceIDsCache.Get(DatasourceCachePrimary, DisabledDatasourceIDsCacheKey)
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
	DisabledDatasourceIDsCache.Set(DatasourceCachePrimary, DisabledDatasourceIDsCacheKey, datasourceIDs, time.Duration(30)*time.Minute)
	return datasourceIDs, nil

}
