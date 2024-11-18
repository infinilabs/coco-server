/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package google_drive

import (
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/queue"
)

type Plugin struct {
	Enabled     bool   `config:"enabled"`
	Credentials string `config:"credentials"`
	Token       string `config:"token"`
	Queue  *queue.QueueConfig `config:"queue"`
}

func (this *Plugin) Setup() {
	ok, err := env.ParseConfig("connector.google_drive", &this)
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}
	if this.Queue==nil{
		this.Queue=&queue.QueueConfig{Name: "indexing_documents"}
	}

}

func (this *Plugin) Start() error {

	if this.Enabled {
		startIndexingFiles(this.Credentials, this.Token,this.Queue)
	}

	return nil
}

func (this *Plugin) Stop() error {
	return nil
}

func (this *Plugin) Name() string {
	return "google_drive"
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}
