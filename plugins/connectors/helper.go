/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connectors

import (
	"context"
	stderrors "errors"
	"fmt"
	"sync"
	"time"

	"infini.sh/coco/modules/common"
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
)

type DatasourceSyncState struct {
	LastSyncTime time.Time `json:"last_sync_time,omitempty"`
}

var (
	datasourceSyncState = make(map[string]DatasourceSyncState) // datasource id => state
)

var locker = sync.RWMutex{}

func CanDoSync(datasource common.DataSource) (bool, error) {
	locker.Lock()
	defer locker.Unlock()

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

// ParseConnectorConfigure parse connector data source config
func ParseConnectorConfigure(connector *common.Connector, datasource *common.DataSource, config interface{}) error {
	if connector == nil || datasource == nil {
		return errors.New("invalid connector or datasource config")
	}

	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		return errors.Wrapf(err, "Create config from datasource [%s] failed", datasource.Name)
	}

	err = cfg.Unpack(config)
	if err != nil {
		return errors.Wrapf(err, "Unpack config for datasource [%s] failed", datasource.Name)
	}
	return nil
}

const ContextDone errors.ErrorCode = 3

type Error struct {
	error
	code errors.ErrorCode
	msg  string
}

func (c Error) Error() string          { return fmt.Sprintf("%s: %v", c.msg, c.Cause()) }
func (c Error) Cause() error           { return c.error }
func (c Error) Code() errors.ErrorCode { return c.code }

// CheckContextDone checks if the context has been canceled or has timed out.
// It returns a wrapped custom error if the context is done, otherwise returns nil.
func CheckContextDone(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return Error{ctx.Err(), ContextDone, "context canceled"}
	default:
		return nil
	}
}

func ResolveCode(err error) errors.ErrorCode {
	var e Error
	if stderrors.As(err, &e) {
		return e.Code()
	}
	return errors.Default
}
