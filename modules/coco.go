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
	_ "infini.sh/coco/modules/llm"
	_ "infini.sh/coco/modules/search"
	_ "infini.sh/coco/modules/system"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"time"
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
	orm.MustRegisterSchemaWithIndexName(common.ModelProvider{}, "model-provider")
	orm.MustRegisterSchemaWithIndexName(common.Assistant{}, "assistant")

	cocoConfig := common.Config{
		LLMConfig: &common.LLMConfig{
			Type:                "deepseek",
			DefaultModel:        "deepseek-r1",
			IntentAnalysisModel: "tongyi-intent-detect-v3",
			PickingDocModel:     "deepseek-r1-distill-qwen-32b",
			AnsweringModel:      "deepseek-r1",
			ContextLength:       131072,
			Keepalive:           "30m",
			Endpoint:            "https://dashscope.aliyuncs.com/compatible-mode/v1",
		},
		ServerInfo: &common.ServerInfo{Version: common.Version{Number: global.Env().GetVersion()}, Updated: time.Now()},
	}

	ok, err := env.ParseConfig("coco", &cocoConfig)
	if ok && err != nil {
		panic(err)
	}

	//update coco's config
	global.Register("APP_CONFIG", &cocoConfig)

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
