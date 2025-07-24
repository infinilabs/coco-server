/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connectors

import (
	"context"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

const DefaultSyncInterval = time.Second * 30

// Scannable defines the interface of the connector scanning logic.
// Each concrete connector plugin must implement this interface.
type Scannable interface {
	Scan(connector *common.Connector, datasource *common.DataSource)
	Name() string
}

// BasePlugin provides common structure and lifecycle management for connector plugins.
type BasePlugin struct {
	api.Handler
	Enabled     bool               `config:"enabled"`
	Queue       *queue.QueueConfig `config:"queue"`
	Interval    string             `config:"interval"`
	PageSize    int                `config:"page_size"`
	Description string
	worker      Scannable
}

// Init Initializes base plugin configuration from environment & configuration files.
func (p *BasePlugin) Init(configKey string, description string, worker Scannable) {
	ok, err := env.ParseConfig(configKey, p)
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	if !p.Enabled {
		return
	}

	if p.PageSize <= 0 {
		p.PageSize = 1000
	}

	if p.Queue == nil {
		p.Queue = &queue.QueueConfig{Name: "indexing_documents"}
	}

	p.Queue = queue.SmartGetOrInitConfig(p.Queue)
	p.Description = description
	p.worker = worker
}

// Start register connector data source task
func (p *BasePlugin) Start(defaultInterval time.Duration) error {
	if !p.Enabled {
		return nil
	}

	task.RegisterScheduleTask(task.ScheduleTask{
		ID:          util.GetUUID(),
		Group:       "connectors",
		Singleton:   true,
		Interval:    util.GetDurationOrDefault(p.Interval, defaultInterval).String(),
		Description: p.Description,
		Task: func(ctx context.Context) {
			connector := common.Connector{}
			connector.ID = p.worker.Name()
			exists, err := orm.Get(&connector)
			if !exists || err != nil {
				log.Errorf("[%v connector] Connector not found or error occurred, skipping task:%v", connector.ID, err)
				return
			}

			q := orm.Query{}
			q.Size = p.PageSize
			q.Conds = orm.And(orm.Eq("connector.id", connector.ID), orm.Eq("sync_enabled", true))
			var results []common.DataSource

			err, _ = orm.SearchWithJSONMapper(&results, &q)
			if err != nil {
				log.Errorf("[%v connector] Failed to search data sources: %v", connector.ID, err)
				return
			}

			for _, item := range results {
				toSync, err := CanDoSync(item)
				if err != nil {
					log.Errorf("[%v connector] Error checking sync status of data source [%s]: %v", connector.ID, item.Name, err)
					continue
				}

				if !toSync {
					continue
				}
				if global.Env().IsDebug {
					log.Debugf("[%v connector] Start syncing data source: ID: [%s], Name: [%s], Other: %s", connector.ID, item.ID, item.Name, util.MustToJSON(item))
				}
				p.worker.Scan(&connector, &item)
			}
		},
	})

	return nil
}
