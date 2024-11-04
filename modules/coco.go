/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package modules

import (
	"infini.sh/coco/modules/assistant"
	"infini.sh/framework/core/orm"
)

type Coco struct {
	
}

func (this *Coco) Setup() {
	err := orm.RegisterSchemaWithIndexName(assistant.Session{}, "session")
	if err != nil {
		panic(err)
	}
	err = orm.RegisterSchemaWithIndexName(assistant.ChatMessage{}, "message")
	if err != nil {
		panic(err)
	}





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
