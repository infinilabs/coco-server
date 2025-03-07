/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connectors

import (
	"infini.sh/coco/modules/common"
	"time"
)

type DatasourceSyncState struct {
	LastSyncTime time.Time `json:"last_sync_time,omitempty"`
}

var (
	datasourceSyncState = make(map[string]DatasourceSyncState) // datasource id => state
)

func CanDoSync(datasource common.DataSource) (bool, error) {
	datasourceID := datasource.ID
	var strInterval = "1h"
	if v, ok := datasource.Connector.Config.(map[string]interface{}); ok {
		if interval, ok := v["interval"].(string); ok {
			strInterval = interval
		}
	}
	now := time.Now()
	state, ok := datasourceSyncState[datasourceID]
	if !ok {
		datasourceSyncState[datasourceID] = DatasourceSyncState{LastSyncTime: now}
		return true, nil
	}
	intervalDuration, err := time.ParseDuration(strInterval)
	if err != nil {
		return false, err
	}
	toSync := now.After(state.LastSyncTime.Add(intervalDuration))
	if toSync {
		state.LastSyncTime = now
		datasourceSyncState[datasourceID] = state
	}
	return toSync, nil
}
