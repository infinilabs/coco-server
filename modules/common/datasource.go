/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

type DataSource struct {
	CombinedFullText

	Type string `json:"type,omitempty" elastic_mapping:"type:{type:keyword}"` // Type of the datasource, eg: connector
	Name string `json:"name,omitempty" elastic_mapping:"name:{type:keyword}"` // Display name of this datasource
	Icon string `json:"icon,omitempty" elastic_mapping:"icon:{type:keyword}"` // Display name of this datasource

	Connector ConnectorConfig `json:"connector,omitempty" elastic_mapping:"connector:{type:keyword}"`
	// Whether synchronization is allowed
	SyncEnabled bool `json:"sync_enabled" elastic_mapping:"sync_enabled:{type:keyword}"`
}

type ConnectorConfig struct {
	ConnectorID string      `json:"id,omitempty" elastic_mapping:"id:{type:keyword}"`          // Connector ID for the datasource
	Config      interface{} `json:"config,omitempty" elastic_mapping:"config:{enabled:false}"` // Configs for this Connector
}
