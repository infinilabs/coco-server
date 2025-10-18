/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package gitlab

import (
	"fmt"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
)

const (
	ConnectorGitLab = "gitlab"
)

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorGitLab, New)
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
	return ConnectorGitLab
}

func (p *Plugin) Fetch(ctx *pipeline.Context, connector *common.Connector, datasource *common.DataSource) error {
	cfg := Config{}
	p.MustParseConfig(datasource, &cfg)

	log.Debugf("[%s connector] handling datasource: %v", ConnectorGitLab, cfg)

	if cfg.Token == "" {
		return fmt.Errorf("token is required for datasource [%s]", datasource.Name)
	}

	if cfg.Owner == "" {
		return fmt.Errorf("owner is required for datasource [%s]", datasource.Name)
	}

	client, err := NewGitLabClient(cfg.Token, cfg.BaseURL)
	if err != nil {
		return fmt.Errorf("failed to create gitlab client for datasource [%s]: %v", datasource.Name, err)
	}

	p.processProjects(ctx, client, &cfg, connector, datasource)

	log.Infof("[%s connector] finished fetching datasource [%s]", ConnectorGitLab, datasource.Name)
	return nil
}
