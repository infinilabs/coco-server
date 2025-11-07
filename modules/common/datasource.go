/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"time"
)

type DataSource struct {
	core.CombinedFullText

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

const (
	DatasourcePrimaryCacheKey     = "datasource_primary"
	DeletedDatasourceCacheKey     = "deleted_datasource_ids"
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

func MarkDatasourceNotDeleted(id string) {
	GeneralObjectCache.Delete(DeletedDatasourceCacheKey, id)
}

func MarkDatasourceDeleted(id string) {
	GeneralObjectCache.Set(DeletedDatasourceCacheKey, id, true, time.Duration(6)*time.Hour)
}

func IsDatasourceDeleted(id string) bool {
	deleted := GeneralObjectCache.Get(DeletedDatasourceCacheKey, id)
	if deleted != nil {
		return true
	}
	return false
}

func GetDatasourceConfig(ctx *orm.Context, id string) (*DataSource, error) {
	if IsDatasourceDeleted(id) {
		return nil, errors.Errorf("datasource [%v] has been deleted", id)
	}

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

	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "datasource")
	ctx.Set(orm.SharingCategoryCheckingChildrenEnabled, true)

	exists, err := orm.GetV2(ctx, &obj)
	if err == nil && exists {
		GeneralObjectCache.Set(DatasourceItemsCacheKey, id, &obj, util.GetDurationOrDefault("30m", time.Duration(30)*time.Minute))
		return &obj, nil
	}

	return nil, errors.New("not found")
}
