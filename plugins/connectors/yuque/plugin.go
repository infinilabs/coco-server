/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package yuque

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	cmn "infini.sh/coco/plugins/connectors/common"
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/pipeline"
)

const YuqueKey = "yuque"

type YuqueConfig struct {
	Token               string `config:"token"`
	IncludePrivateBook  bool   `config:"include_private_book"`
	IncludePrivateDoc   bool   `config:"include_private_doc"`
	IndexingBooks       bool   `config:"indexing_books"`
	SkipIndexingBookToc bool   `config:"skip_indexing_book_toc"`
	IndexingDocs        bool   `config:"indexing_docs"`
	IndexingUsers       bool   `config:"indexing_users"`
	IndexingGroups      bool   `config:"indexing_groups"`
}

type Plugin struct {
	cmn.ConnectorProcessorBase
	SkipInvalidToken bool `config:"skip_invalid_token"`
}

func (this *Plugin) fetch_yuque(pipeCtx *pipeline.Context, connector *common.Connector, datasource *common.DataSource) error {
	if connector == nil || datasource == nil {
		return errors.Error("invalid connector config: connector or datasource is nil")
	}

	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		return errors.Errorf("error creating config from datasource [%s]: %v", datasource.Name, err)
	}

	obj := YuqueConfig{}
	err = cfg.Unpack(&obj)
	if err != nil {
		return errors.Errorf("error unpacking config for datasource [%s]: %v", datasource.Name, err)
	}

	log.Debugf("handle yuque's datasource: %v", obj)
	return this.collect(pipeCtx, connector, datasource, &obj)
}

func init() {
	pipeline.RegisterProcessorPlugin(YuqueKey, New)
}

func New(c *config3.Config) (pipeline.Processor, error) {
	runner := Plugin{SkipInvalidToken: true}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of processor %v, error: %s", YuqueKey, err)
	}

	runner.InitBaseConfig(c)
	return &runner, nil
}

func (processor *Plugin) Name() string {
	return YuqueKey
}

func (processor *Plugin) Process(ctx *pipeline.Context) error {
	connector, datasource := processor.GetBasicInfo(ctx)
	return processor.fetch_yuque(ctx, connector, datasource)
}
