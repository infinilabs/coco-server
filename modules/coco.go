/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package modules

import (
	_ "infini.sh/coco/modules/assistant"
	_ "infini.sh/coco/modules/attachment"
	"infini.sh/coco/modules/common"
	_ "infini.sh/coco/modules/connector"
	_ "infini.sh/coco/modules/datasource"
	_ "infini.sh/coco/modules/document"
	"infini.sh/coco/modules/integration"
	_ "infini.sh/coco/modules/integration"
	_ "infini.sh/coco/modules/llm"
	_ "infini.sh/coco/modules/search"
	_ "infini.sh/coco/modules/system"
	"infini.sh/framework/core/orm"
)

type Coco struct {
}

func (this *Coco) Setup() {
	suffix := common.GetSchemaSuffix()

	orm.MustRegisterSchemaWithIndexName(common.Session{}, "session"+suffix)
	orm.MustRegisterSchemaWithIndexName(common.Document{}, "document"+suffix)
	orm.MustRegisterSchemaWithIndexName(common.ChatMessage{}, "message"+suffix)
	orm.MustRegisterSchemaWithIndexName(common.Attachment{}, "attachment"+suffix)
	orm.MustRegisterSchemaWithIndexName(common.Connector{}, "connector"+suffix)
	orm.MustRegisterSchemaWithIndexName(common.DataSource{}, "datasource"+suffix)
	orm.MustRegisterSchemaWithIndexName(common.Integration{}, "integration"+suffix)
	orm.MustRegisterSchemaWithIndexName(common.ModelProvider{}, "model-provider"+suffix)
	orm.MustRegisterSchemaWithIndexName(common.Assistant{}, "assistant"+suffix)
	orm.MustRegisterSchemaWithIndexName(common.MCPServer{}, "mcp-server"+suffix)
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
