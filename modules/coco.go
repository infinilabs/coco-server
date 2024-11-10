/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package modules

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/assistant"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
)

type Coco struct {
	
}

func (this *Coco) Setup() {
	orm.MustRegisterSchemaWithIndexName(assistant.Session{}, "session")
	orm.MustRegisterSchemaWithIndexName(common.Document{}, "document")
	orm.MustRegisterSchemaWithIndexName(assistant.ChatMessage{}, "message")

	cocoConfig := common.Config{
		OllamaConfig: common.OllamaConfig{
			Model:     "llama3.2:1b",
			Keepalive: "30m",
			Endpoint:  "http://localhost:11434",
		},
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
