/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package modules

import (
	"infini.sh/coco/core"
	_ "infini.sh/coco/modules/assistant"
	_ "infini.sh/coco/modules/attachment"
	"infini.sh/coco/modules/common"
	_ "infini.sh/coco/modules/connector"
	_ "infini.sh/coco/modules/datasource"
	_ "infini.sh/coco/modules/document"
	"infini.sh/coco/modules/integration"
	_ "infini.sh/coco/modules/integration"
	_ "infini.sh/coco/modules/llm"
	_ "infini.sh/coco/modules/system"
	"infini.sh/framework/core/orm"
)

type Coco struct {
}

func (this *Coco) Setup() {
	suffix := common.GetSchemaSuffix()

	orm.MustRegisterSchemaWithIndexName(core.Session{}, "session"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.Document{}, "document"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.ChatMessage{}, "message"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.Attachment{}, "attachment"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.Connector{}, "connector"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.DataSource{}, "datasource"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.Integration{}, "integration"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.ModelProvider{}, "model-provider"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.Assistant{}, "assistant"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.MCPServer{}, "mcp-server"+suffix)
}

func (this *Coco) Start() error {
	integration.InitIntegrationOrigins()
	return nil
}

func (this *Coco) Stop() error {
	return nil
}

func (this *Coco) Name() string {
	return "coco"
}
