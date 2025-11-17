/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package yuque

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	cmn "infini.sh/coco/plugins/connectors/common"
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
)

const Name = "yuque"

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
}

func (this *Plugin) Fetch(pipeCtx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	config := YuqueConfig{}
	this.MustParseConfig(datasource, &config)

	log.Debugf("handle yuque's datasource: %v", config)
	return this.collect(pipeCtx, connector, datasource, &config)
}

func init() {
	pipeline.RegisterProcessorPlugin(Name, New)
}

func New(c *config3.Config) (pipeline.Processor, error) {
	runner := Plugin{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of processor %v, error: %s", Name, err)
	}

	runner.Init(c, &runner)
	return &runner, nil
}

func (processor *Plugin) Name() string {
	return Name
}
