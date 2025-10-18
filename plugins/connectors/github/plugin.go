/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package github

import (
	"fmt"

	log "github.com/cihub/seelog"
	_ "github.com/google/go-github/v74/github"
	"infini.sh/coco/modules/common"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
)

const (
	ConnectorGitHub = "github"
)

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorGitHub, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	runner := Plugin{}
	runner.Init(c, &runner)
	return &runner, nil
}

type Plugin struct {
	cmn.ConnectorProcessorBase
}

func (p *Plugin) Name() string {
	return ConnectorGitHub
}

func (p *Plugin) Fetch(ctx *pipeline.Context, connector *common.Connector, datasource *common.DataSource) error {
	cfg := Config{}
	p.MustParseConfig(datasource, &cfg)

	log.Debugf("[%s connector] handling datasource: %v", ConnectorGitHub, cfg)

	if cfg.Token == "" {
		return fmt.Errorf("token is required for datasource [%s]", datasource.Name)
	}

	if cfg.Owner == "" {
		return fmt.Errorf("owner is required for datasource [%s]", datasource.Name)
	}

	client := NewGitHubClient(cfg.Token)

	p.processRepos(ctx, client, &cfg, connector, datasource)

	log.Infof("[%s connector] finished fetching datasource [%s]", ConnectorGitHub, datasource.Name)
	return nil
}
