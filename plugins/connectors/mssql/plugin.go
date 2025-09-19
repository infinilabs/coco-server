/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mssql

import (
	"context"
	"fmt"
	"sync"

	log "github.com/cihub/seelog"
	_ "github.com/microsoft/go-mssqldb" // Import the MSSQL driver
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	rdbms "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/module"
)

const (
	ConnectorMSSQL = "mssql"
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
	return ConnectorMSSQL
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
		log.Infof("[%s connector] received stop signal, cancelling all operations", ConnectorMSSQL)
		p.cancel()
		p.ctx = nil
		p.cancel = nil
	}
	return nil
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init(fmt.Sprintf("connector.%s", ConnectorMSSQL), "indexing mssql database", p)
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	p.mu.Lock()
	parentCtx := p.ctx
	p.mu.Unlock()

	if parentCtx == nil {
		_ = log.Warnf("[%s connector] plugin is stopped, skipping scan for datasource [%s]", ConnectorMSSQL, datasource.Name)
		return
	}

	dialect := &SQLServerDialect{}

	scanner := &rdbms.Scanner{
		Name:       ConnectorMSSQL,
		Connector:  connector,
		Datasource: datasource,
		Queue:      p.Queue,
		DriverName: "mssql",
		SqlWithLastModified: func(baseQuery string, lastSyncField string) string {
			return dialect.BuildIncrementalQuery(baseQuery, lastSyncField)
		},
		SqlWithPagination: func(baseQuery string, pageSize uint, offset uint) string {
			return dialect.BuildPaginationQuery(baseQuery, pageSize, offset)
		},
	}
	scanner.Scan(parentCtx)
}
