/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"errors"
	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
)

const (
	IntegrationTypeEmbedded = "embedded"
	IntegrationTypeFloating = "floating"
)

func InternalGetIntegration(id string) (*core.Integration, error) {
	obj := core.Integration{}
	obj.ID = id
	ctx := orm.NewContext()
	ctx.DirectReadAccess()
	exists, err := orm.GetV2(ctx, &obj)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("integration not found")
	}
	return &obj, nil
}

// GetDatasourceByIntegration returns the datasource IDs that the integration is allowed to access
func GetDatasourceByIntegration(integrationID string) ([]string, bool, error) {
	var items = []core.Integration{}
	q := orm.Query{
		Size:  1,
		Conds: orm.And(orm.Eq("id", integrationID), orm.Eq("enabled", true)),
	}
	err, _ := orm.SearchWithJSONMapper(&items, &q)
	if err != nil {
		return nil, false, err
	}
	if len(items) == 0 {
		return nil, false, nil
	}
	var ret = make([]string, 0, len(items))
	for _, item := range items {
		for _, datasourceID := range item.EnabledModule.Search.Datasource {
			if datasourceID == "*" {
				return nil, true, nil
			}
			ret = append(ret, datasourceID)
		}
	}
	return ret, false, nil
}

func GetDatasourceByIntegration1(integration *core.Integration) ([]string, bool, error) {
	ret := []string{}
	for _, datasourceID := range integration.EnabledModule.Search.Datasource {
		if datasourceID == "*" {
			return nil, true, nil
		}
		ret = append(ret, datasourceID)
	}
	return ret, false, nil
}
