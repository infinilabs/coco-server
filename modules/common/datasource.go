/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
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

const DisabledDatasourceIDsKey = "disabled_datasource_ids"

// CacheDisabledDatasourceIDs retrieves all disabled data sources and caches their IDs.
func CacheDisabledDatasourceIDs() error {
	var datasources []DataSource
	q := orm.Query{
		Conds: orm.And(orm.Eq("enabled", false)), // Query for disabled data sources
	}
	err, _ := orm.SearchWithJSONMapper(&datasources, &q)
	if err != nil {
		return err
	}

	// Extract IDs from the retrieved data sources
	datasourceIDs := make([]string, len(datasources))
	for i, ds := range datasources {
		datasourceIDs[i] = ds.ID
	}
	datasourceIDsBytes, err := util.ToJSONBytes(datasourceIDs)
	if err != nil {
		return err
	}

	// Store the disabled datasource IDs in key-value store
	return kv.AddValue(core.DefaultSettingBucketKey, []byte(DisabledDatasourceIDsKey), datasourceIDsBytes)
}

// GetDisabledDatasourceIDs retrieves the list of disabled data source IDs from the cache.
func GetDisabledDatasourceIDs() ([]string, error) {
	// Fetch stored JSON bytes of disabled datasource IDs
	datasourceIDsBytes, err := kv.GetValue(core.DefaultSettingBucketKey, []byte(DisabledDatasourceIDsKey))
	if err != nil {
		return nil, err
	}

	var datasourceIDs []string
	if err := util.FromJSONBytes(datasourceIDsBytes, &datasourceIDs); err != nil {
		return nil, err
	}

	return datasourceIDs, nil
}

// DisableDatasource marks a data source as disabled by adding it to the kv cache.
func DisableDatasource(id string) error {
	disabledDatasourceIDs, err := GetDisabledDatasourceIDs()
	if err != nil {
		return err
	}

	// Check if the ID is already disabled to prevent duplicates
	for _, disabledID := range disabledDatasourceIDs {
		if disabledID == id {
			return nil // Already disabled, no need to update
		}
	}

	// Append the new disabled ID and store the updated list
	disabledDatasourceIDs = append(disabledDatasourceIDs, id)
	disabledDatasourceIDsBytes, err := util.ToJSONBytes(disabledDatasourceIDs)
	if err != nil {
		return err
	}

	return kv.AddValue(core.DefaultSettingBucketKey, []byte(DisabledDatasourceIDsKey), disabledDatasourceIDsBytes)
}

// EnableDatasource removes a data source from the disabled list, marking it as enabled.
func EnableDatasource(id string) error {
	// Retrieve existing disabled data sources
	disabledDatasourceIDs, err := GetDisabledDatasourceIDs()
	if err != nil {
		return err
	}

	// Create a new slice excluding the ID to be enabled
	newDisabledDatasourceIDs := disabledDatasourceIDs[:0] // Reuse existing slice memory
	for _, disabledID := range disabledDatasourceIDs {
		if disabledID != id {
			newDisabledDatasourceIDs = append(newDisabledDatasourceIDs, disabledID)
		}
	}

	disabledDatasourceIDsBytes, err := util.ToJSONBytes(newDisabledDatasourceIDs)
	if err != nil {
		return err
	}

	// Store the updated list in key-value store
	return kv.AddValue(core.DefaultSettingBucketKey, []byte(DisabledDatasourceIDsKey), disabledDatasourceIDsBytes)
}
