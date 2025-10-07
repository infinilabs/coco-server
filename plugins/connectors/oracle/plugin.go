/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package oracle

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	log "github.com/cihub/seelog"
	_ "github.com/sijms/go-ora/v2" // Import the Oracle driver
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	rdbms "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/module"
)

const (
	ConnectorOracle = "oracle"
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
	return ConnectorOracle
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
		log.Infof("[%s connector] received stop signal, cancelling all operations", ConnectorOracle)
		p.cancel()
		p.ctx = nil
		p.cancel = nil
	}
	return nil
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init(fmt.Sprintf("connector.%s", ConnectorOracle), "indexing oracle database", p)
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	p.mu.Lock()
	parentCtx := p.ctx
	p.mu.Unlock()

	if parentCtx == nil {
		_ = log.Warnf("[%s connector] plugin is stopped, skipping scan for datasource [%s]", ConnectorOracle, datasource.Name)
		return
	}

	scanner := &rdbms.Scanner{
		Name:       ConnectorOracle,
		Connector:  connector,
		Datasource: datasource,
		Queue:      p.Queue,
		DriverName: "oracle",
		SqlWithLastModified: func(baseQuery string, lastSyncField string) string {
			// Use :1 as the placeholder for Oracle (go-ora driver uses numbered parameters)
			return fmt.Sprintf(`SELECT * FROM (%s) WHERE %s > :1`, baseQuery, lastSyncField)
		},
		SqlWithPagination: func(baseQuery string, pageSize uint, offset uint) string {
			// Oracle 12c+ requires ORDER BY for OFFSET FETCH.
			if !hasOrderByClause(strings.ToUpper(baseQuery)) {
				// This is a simple check; complex queries might need manual ORDER BY.
				_ = log.Warnf("[%s connector] pagination is enabled but no ORDER BY clause was found in the query for datasource [%s]. Stability is not guaranteed.", ConnectorOracle, datasource.Name)
			}
			return fmt.Sprintf(`%s OFFSET %d ROWS FETCH NEXT %d ROWS ONLY`, baseQuery, offset, pageSize)
		},
	}
	scanner.Scan(parentCtx)
}

// hasOrderByClause checks if the query already contains an ORDER BY clause
func hasOrderByClause(query string) bool {
	// Look for ORDER BY anywhere in the query, not just at the end
	orderByRegex := regexp.MustCompile(`(?i)\bORDER\s+BY\b`)
	return orderByRegex.MatchString(strings.TrimSpace(query))
}
