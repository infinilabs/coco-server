/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connector

import (
	"errors"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	ccache "infini.sh/framework/lib/cache"
	"time"
)

var connectorCacheKey = "Datasource"
var configCache = ccache.Layered(ccache.Configure().MaxSize(10000).ItemsToPrune(100))

func GetConnectorConfig(id string) (*common.Connector, error) {
	v := configCache.Get(connectorCacheKey, id)
	if v != nil {
		if !v.Expired() {
			x, ok := v.Value().(*common.Connector)
			if ok && x != nil {
				return x, nil
			}
		}
	}

	obj := common.Connector{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if err == nil && exists {
		configCache.Set(connectorCacheKey, id, &obj, util.GetDurationOrDefault("30m", time.Duration(30)*time.Minute))
		return &obj, nil
	}

	return nil, errors.New("not found")
}
