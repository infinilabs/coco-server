/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	orm2 "infini.sh/framework/plugins/enterprise/security/orm"
	"time"
)

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
	var datasources []core.DataSource
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
	var datasources []core.DataSource
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

func GetDatasourceConfig(ctx *orm.Context, id string) (*core.DataSource, error) {
	if IsDatasourceDeleted(id) {
		return nil, errors.Errorf("datasource [%v] has been deleted", id)
	}

	v := GeneralObjectCache.Get(DatasourceItemsCacheKey, id)
	if v != nil {
		if !v.Expired() {
			x, ok := v.Value().(*core.DataSource)
			if ok && x != nil {
				return x, nil
			}
		}
	}

	obj := core.DataSource{}
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

func GetUserDatasource(ctx *orm.Context) []string {
	orm.WithModel(ctx, &core.DataSource{})
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "datasource")
	ctx.Set(orm.SharingCategoryCheckingChildrenEnabled, true)
	builder := orm.NewQuery()
	builder.Size(1000)

	docs := []core.DataSource{}

	_, _ = elastic.SearchV2WithResultItemMapper(ctx, &docs, builder, nil)
	ids := []string{}
	for _, v := range docs {
		ids = append(ids, v.ID)
	}
	return ids
}

func GetUsersOwnDatasource(userID string) []string {
	ctx := orm.NewContext()
	ctx.DirectReadAccess()
	orm.WithModel(ctx, &core.DataSource{})
	builder := orm.NewQuery()
	builder.Must(orm.TermQuery(orm2.SystemOwnerQueryField, userID))
	builder.Size(1000)

	docs := []core.DataSource{}

	_, _ = elastic.SearchV2WithResultItemMapper(ctx, &docs, builder, nil)
	ids := []string{}
	for _, v := range docs {
		ids = append(ids, v.ID)
	}
	return ids
}
