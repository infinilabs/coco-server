/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package modules

import (
	"infini.sh/coco/modules/assistant"
	_ "infini.sh/coco/modules/assistant"
	_ "infini.sh/coco/modules/attachment"
	"infini.sh/coco/modules/common"
	_ "infini.sh/coco/modules/connector"
	_ "infini.sh/coco/modules/datasource"
	_ "infini.sh/coco/modules/document"
	"infini.sh/coco/modules/integration"
	_ "infini.sh/coco/modules/integration"
	_ "infini.sh/coco/modules/search"
	_ "infini.sh/coco/modules/system"
	"infini.sh/framework/core/orm"
)

type Coco struct {
}

func (this *Coco) Setup() {
	orm.MustRegisterSchemaWithIndexName(common.Session{}, "session")
	orm.MustRegisterSchemaWithIndexName(common.Document{}, "document")
	orm.MustRegisterSchemaWithIndexName(assistant.ChatMessage{}, "message")
	orm.MustRegisterSchemaWithIndexName(common.Attachment{}, "attachment")
	orm.MustRegisterSchemaWithIndexName(common.Connector{}, "connector")
	orm.MustRegisterSchemaWithIndexName(common.DataSource{}, "datasource")
	orm.MustRegisterSchemaWithIndexName(common.Integration{}, "integration")
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
