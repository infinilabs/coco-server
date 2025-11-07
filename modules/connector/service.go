/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connector

import (
	"errors"
	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	ccache "infini.sh/framework/lib/cache"
	"time"
)

var connectorCacheKey = "Datasource"
var configCache = ccache.Layered(ccache.Configure().MaxSize(10000).ItemsToPrune(100))

func GetConnectorConfig(id string) (*core.Connector, error) {
	v := configCache.Get(connectorCacheKey, id)
	if v != nil {
		if !v.Expired() {
			x, ok := v.Value().(*core.Connector)
			if ok && x != nil {
				return x, nil
			}
		}
	}

	obj := core.Connector{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if err == nil && exists {
		configCache.Set(connectorCacheKey, id, &obj, util.GetDurationOrDefault("30m", time.Duration(30)*time.Minute))
		return &obj, nil
	}

	return nil, errors.New("not found")
}
