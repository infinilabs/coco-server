package neo4j

import (
	"context"
	"fmt"
	"sync"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/module"
)

const ConnectorNeo4j = "neo4j"

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
	return ConnectorNeo4j
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init(fmt.Sprintf("connector.%s", ConnectorNeo4j), "indexing neo4j database", p)
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
		log.Infof("[%s connector] received stop signal, cancelling all operations", ConnectorNeo4j)
		p.cancel()
		p.cancel = nil
		p.ctx = nil
	}
	return nil
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	p.mu.Lock()
	parentCtx := p.ctx
	p.mu.Unlock()

	if parentCtx == nil {
		_ = log.Warnf("[%s connector] plugin is stopped, skipping datasource [%s]", ConnectorNeo4j, datasource.Name)
		return
	}

	worker := &scanner{
		name:       ConnectorNeo4j,
		connector:  connector,
		datasource: datasource,
		queue:      p.Queue,
		stateStore: connectors.NewSyncStateStore(),
	}

	worker.Scan(parentCtx)
}
