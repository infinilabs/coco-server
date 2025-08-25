/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package github

import (
	"context"
	"fmt"
	"sync"

	log "github.com/cihub/seelog"
	_ "github.com/google/go-github/v74/github"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/module"
)

const (
	ConnectorGitHub = "github"
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
	return ConnectorGitHub
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
		log.Infof("[%s connector] received stop signal, cancelling all operations", ConnectorGitHub)
		p.cancel()
		p.ctx = nil
		p.cancel = nil
	}
	return nil
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init(fmt.Sprintf("connector.%s", ConnectorGitHub), "indexing github repositories", p)
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	p.mu.Lock()
	parentCtx := p.ctx
	p.mu.Unlock()

	if parentCtx == nil {
		_ = log.Warnf("[%s connector] plugin is stopped, skipping scan for datasource [%s]", ConnectorGitHub, datasource.Name)
		return
	}

	cfg := Config{}
	if err := connectors.ParseConnectorConfigure(connector, datasource, &cfg); err != nil {
		_ = log.Errorf("[%s connector] parsing connector configuration failed for datasource [%s]: %v", ConnectorGitHub, datasource.Name, err)
		return
	}

	if cfg.Token == "" {
		_ = log.Errorf("[%s connector] token is required for datasource [%s]", ConnectorGitHub, datasource.Name)
		return
	}

	if cfg.Owner == "" {
		_ = log.Errorf("[%s connector] owner is required for datasource [%s]", ConnectorGitHub, datasource.Name)
		return
	}

	client := NewGitHubClient(cfg.Token)

	scanCtx, scanCancel := context.WithCancel(parentCtx)
	defer scanCancel()

	p.processRepos(scanCtx, client, &cfg, datasource)

	log.Infof("[%s connector] finished scanning datasource [%s]", ConnectorGitHub, datasource.Name)
}
