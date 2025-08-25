/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mysql

import (
	"context"
	"fmt"
	"sync"

	log "github.com/cihub/seelog"
	_ "github.com/go-sql-driver/mysql"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	rdbms "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/module"
)

const (
	ConnectorMySQL = "mysql"
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
	return ConnectorMySQL
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
		log.Infof("[%s connector] received stop signal, cancelling all operations", ConnectorMySQL)
		p.cancel()
		p.ctx = nil
		p.cancel = nil
	}
	return nil
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init(fmt.Sprintf("connector.%s", ConnectorMySQL), "indexing postgresql database", p)
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	p.mu.Lock()
	parentCtx := p.ctx
	p.mu.Unlock()

	if parentCtx == nil {
		_ = log.Warnf("[%s connector] plugin is stopped, skipping g for datasource [%s]", ConnectorMySQL, datasource.Name)
		return
	}

	scanner := &rdbms.Scanner{
		Name:       ConnectorMySQL,
		Connector:  connector,
		Datasource: datasource,
		Queue:      p.Queue,
		DriverName: "mysql",
		SqlWithLastModified: func(baseQuery string, lastSyncField string) string {
			return fmt.Sprintf("SELECT * FROM (%s) AS coco_subquery WHERE `%s` > ?", baseQuery, lastSyncField)
		},
		SqlWithPagination: func(baseQuery string, pageSize uint, offset uint) string {
			return fmt.Sprintf(`%s LIMIT %d, %d`, baseQuery, offset, pageSize)
		},
	}
	scanner.Scan(parentCtx)
}
