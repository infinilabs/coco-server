/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package gitea

import (
	"context"
	"fmt"
	"sync"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/module"
)

const (
	ConnectorGitea = "gitea"
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
	return ConnectorGitea
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
		log.Infof("[%s connector] received stop signal, cancelling all operations", ConnectorGitea)
		p.cancel()
		p.ctx = nil
		p.cancel = nil
	}
	return nil
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init(fmt.Sprintf("connector.%s", ConnectorGitea), "indexing gitea repositories", p)
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	p.mu.Lock()
	parentCtx := p.ctx
	p.mu.Unlock()

	if parentCtx == nil {
		_ = log.Warnf("[%s connector] plugin is stopped, skipping scan for datasource [%s]", ConnectorGitea, datasource.Name)
		return
	}

	cfg := Config{}
	if err := connectors.ParseConnectorConfigure(connector, datasource, &cfg); err != nil {
		_ = log.Errorf("[%s connector] parsing connector configuration failed for datasource [%s]: %v", ConnectorGitea, datasource.Name, err)
		return
	}

	if cfg.Token == "" {
		_ = log.Errorf("[%s connector] token is required for datasource [%s]", ConnectorGitea, datasource.Name)
		return
	}

	if cfg.Owner == "" {
		_ = log.Errorf("[%s connector] owner is required for datasource [%s]", ConnectorGitea, datasource.Name)
		return
	}

	client, err := NewGiteaClient(cfg.BaseURL, cfg.Token)
	if err != nil {
		_ = log.Errorf("[%s connector] failed to create gitea client for datasource [%s]: %v", ConnectorGitea, datasource.Name, err)
		return
	}

	scanCtx, scanCancel := context.WithCancel(parentCtx)
	defer scanCancel()

	p.processRepos(scanCtx, client, &cfg, datasource)

	log.Infof("[%s connector] finished scanning datasource [%s]", ConnectorGitea, datasource.Name)
}
