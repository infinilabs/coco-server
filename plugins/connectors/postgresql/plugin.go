/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package postgresql

import (
	"context"
	"fmt"
	"sync"

	log "github.com/cihub/seelog"
	_ "github.com/lib/pq"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	rdbms "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/module"
)

const (
	ConnectorPostgreSQL = "postgresql"
)

func init() {
	module.RegisterUserPlugin(&Plugin{})
}

type Plugin struct {
	connectors.BasePlugin
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

func (p *Plugin) Name() string {
	return ConnectorPostgreSQL
}

func (p *Plugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return p.BasePlugin.Start(connectors.DefaultSyncInterval)
}

func (p *Plugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cancel != nil {
		log.Infof("[%s connector] received stop signal, cancelling all operations", ConnectorPostgreSQL)
		p.cancel()
		p.ctx = nil
		p.cancel = nil
	}
	return nil
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init(fmt.Sprintf("connector.%s", ConnectorPostgreSQL), "indexing postgresql database", p)
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	p.mu.Lock()
	parentCtx := p.ctx
	p.mu.Unlock()

	if parentCtx == nil {
		_ = log.Warnf("[%s connector] plugin is stopped, skipping g for datasource [%s]", ConnectorPostgreSQL, datasource.Name)
		return
	}

	scanner := &rdbms.Scanner{
		Name:       ConnectorPostgreSQL,
		Connector:  connector,
		Datasource: datasource,
		Queue:      p.Queue,
		DriverName: "postgres",
		SqlWithLastModified: func(baseQuery string, lastSyncField string) string {
			return fmt.Sprintf(`SELECT * FROM (%s) AS coco_subquery WHERE "%s" > $1`, baseQuery, lastSyncField)
		},
		SqlWithPagination: func(baseQuery string, pageSize uint, offset uint) string {
			return fmt.Sprintf(`%s LIMIT %d OFFSET %d`, baseQuery, pageSize, offset)
		},
	}
	scanner.Scan(parentCtx)
}
