/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "infini.sh/framework/core/pipeline"

type DataSource struct {
	CombinedFullText

	Type        string `json:"type,omitempty" elastic_mapping:"type:{type:keyword,copy_to:combined_fulltext}"` // Type of the datasource, eg: connector
	Name        string `json:"name,omitempty" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"` // Display name of this datasource
	Description string `json:"description,omitempty" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"`
	Icon        string `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"` // Display name of this datasource

	Category string   `json:"category,omitempty" elastic_mapping:"category:{type:keyword}"`
	Tags     []string `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword}"`

	Connector ConnectorConfig `json:"connector,omitempty" elastic_mapping:"connector:{type:object}"` // Connector configuration

	// Whether synchronization is allowed
	SyncConfig SyncConfig `json:"sync" elastic_mapping:"sync:{type:object}"`
	Enabled    bool       `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`

	// Enrichment pipeline
	EnrichmentPipeline *pipeline.PipelineConfigV2 `json:"enrichment_pipeline" elastic_mapping:"enrichment_pipeline:{type:object}"` //if the pipeline is enabled, pass each batch messages to this pipeline for enrichment

	WebhookConfig WebhookConfig `json:"webhook,omitempty" elastic_mapping:"webhook:{type:object}"`

	//OAuthConfig OAuthConfig `json:"oauth_config,omitempty" elastic_mapping:"oauth_config:{type:object}"`
}

type WebhookConfig struct {
	Enabled bool `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`
}

type OAuthConfig struct {
	Enabled bool `json:"enabled,omitempty" elastic_mapping:"enabled:{type:keyword}"`
	Expired bool `json:"expired" elastic_mapping:"expired:{type:object}"`
}

type SyncConfig struct {
	Enabled  bool   `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`
	Strategy string `json:"strategy" elastic_mapping:"strategy:{type:keyword}"`
	Interval string `json:"interval" elastic_mapping:"interval:{type:keyword}"`
	PageSize int    `json:"page_size" config:"page_size"`
}

type ConnectorConfig struct {
	ConnectorID string      `json:"id,omitempty" elastic_mapping:"id:{type:keyword}"`          // Connector ID for the datasource
	Config      interface{} `json:"config,omitempty" elastic_mapping:"config:{enabled:false}"` // Configs for this Connector, also pass to connector's processor
}
