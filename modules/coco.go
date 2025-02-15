/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package modules

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/assistant"
	_ "infini.sh/coco/modules/assistant"
	"infini.sh/coco/modules/common"
	_ "infini.sh/coco/modules/connector"
	_ "infini.sh/coco/modules/indexing"
	_ "infini.sh/coco/modules/search"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"time"
)

type Coco struct {
}

func (this *Coco) Setup() {
	orm.MustRegisterSchemaWithIndexName(assistant.Session{}, "session")
	orm.MustRegisterSchemaWithIndexName(common.Document{}, "document")
	orm.MustRegisterSchemaWithIndexName(assistant.ChatMessage{}, "message")

	cocoConfig := common.Config{
		OllamaConfig: common.OllamaConfig{
			Model:         "deepseek-r1:1.5b",
			ContextLength: 131072,
			Keepalive:     "30m",
			Endpoint:      "http://localhost:11434",
		},
		ServerInfo: common.ServerInfo{Version: common.Version{Number: global.Env().GetVersion()}, Updated: time.Now()},
	}

	ok, err := env.ParseConfig("coco", &cocoConfig)
	if ok && err != nil {
		panic(err)
	}
	//update coco's config
	global.Register("APP_CONFIG", &cocoConfig)

	log.Debugf("config: %v", cocoConfig)
}

func (this *Coco) Start() error {
	return nil
}

func (this *Coco) Stop() error {
	return nil
}

func (this *Coco) Name() string {
	return "coco"
}
