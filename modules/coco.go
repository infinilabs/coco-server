/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package modules

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	_ "infini.sh/coco/modules/assistant"
	assistantservice "infini.sh/coco/modules/assistant/service"
	_ "infini.sh/coco/modules/attachment"
	"infini.sh/coco/modules/common"
	_ "infini.sh/coco/modules/connector"
	_ "infini.sh/coco/modules/datasource"
	_ "infini.sh/coco/modules/document"
	"infini.sh/coco/modules/integration"
	_ "infini.sh/coco/modules/integration"
	_ "infini.sh/coco/modules/llm"
	_ "infini.sh/coco/modules/skill"
	_ "infini.sh/coco/modules/system"
	_ "infini.sh/coco/modules/wiki"
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
	orm.MustRegisterSchemaWithIndexName(core.Skill{}, "skill"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.WikiWorkspace{}, "wiki-workspace"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.WikiKnowledgeBase{}, "wiki-kb"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.WikiArticle{}, "wiki-article"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.WikiToc{}, "wiki-toc"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.WikiVersion{}, "wiki-version"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.WikiBookmark{}, "wiki-bookmark"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.WikiNotification{}, "wiki-notification"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.WikiEntity{}, "wiki-entity"+suffix)
	orm.MustRegisterSchemaWithIndexName(core.WikiOntologySchema{}, "wiki-ontology-schema"+suffix)
}

func (this *Coco) Start() error {
	integration.InitIntegrationOrigins()
	return nil
}

func (this *Coco) Stop() error {
	cancelled := assistantservice.StopAllMessageReplyTasks()
	if cancelled > 0 {
		log.Infof("cancelled %d inflight assistant tasks during shutdown", cancelled)
	}
	return nil
}

func (this *Coco) Name() string {
	return "coco"
}
