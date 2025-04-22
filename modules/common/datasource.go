/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

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
