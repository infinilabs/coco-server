/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
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
	DatasourcePrimaryCacheKey     = "datasource_primary"
	DisabledDatasourceIDsCacheKey = "disabled_datasource_ids"
	EnabledDatasourceIDsCacheKey  = "enabled_datasource_ids"
	DatasourceItemsCacheKey       = "datasource_items"
)

func ClearDatasourceCache(id string) {
	GeneralObjectCache.Delete(DatasourceItemsCacheKey, id)
}

func ClearDatasourcesCache() {
	GeneralObjectCache.Delete(DatasourcePrimaryCacheKey, DisabledDatasourceIDsCacheKey)
	GeneralObjectCache.Delete(DatasourcePrimaryCacheKey, EnabledDatasourceIDsCacheKey)
}

// GetDisabledDatasourceIDs retrieves the list of disabled data source IDs from the cache.
func GetDisabledDatasourceIDs() ([]string, error) {
	item := GeneralObjectCache.Get(DatasourcePrimaryCacheKey, DisabledDatasourceIDsCacheKey)
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
	GeneralObjectCache.Set(DatasourcePrimaryCacheKey, DisabledDatasourceIDsCacheKey, datasourceIDs, time.Duration(30)*time.Minute)
	return datasourceIDs, nil

}

func GetAllEnabledDatasourceIDs() ([]string, error) {
	item := GeneralObjectCache.Get(DatasourcePrimaryCacheKey, EnabledDatasourceIDsCacheKey)
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
	GeneralObjectCache.Set(DatasourcePrimaryCacheKey, EnabledDatasourceIDsCacheKey, datasourceIDs, time.Duration(30)*time.Minute)
	return datasourceIDs, nil

}

func GetDatasourceConfig(id string) (*DataSource, error) {
	v := GeneralObjectCache.Get(DatasourceItemsCacheKey, id)
	if v != nil {
		if !v.Expired() {
			x, ok := v.Value().(*DataSource)
			if ok && x != nil {
				return x, nil
			}
		}
	}

	obj := DataSource{}
	obj.ID = id
	exists, err := orm.Get(&obj)
	if err == nil && exists {
		GeneralObjectCache.Set(DatasourceItemsCacheKey, id, &obj, util.GetDurationOrDefault("30m", time.Duration(30)*time.Minute))
		return &obj, nil
	}

	return nil, errors.New("not found")
}
