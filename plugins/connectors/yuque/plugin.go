package yuque

import (
	"context"
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"time"
)

const YuqueKey = "yuque"

type YuqueConfig struct {
	Token              string `config:"token"`
	IncludePrivateBook bool   `config:"include_private_book"`
	IncludePrivateDoc  bool   `config:"include_private_doc"`
	IndexingBooks      bool   `config:"indexing_books"`
	IndexingDocs       bool   `config:"indexing_docs"`
	IndexingUsers      bool   `config:"indexing_users"`
	IndexingGroups     bool   `config:"indexing_groups"`
}

type Plugin struct {
	api.Handler

	Enabled          bool               `config:"enabled"`
	PageSize         int                `config:"page_size"`
	Interval         string             `config:"interval"`
	Queue            *queue.QueueConfig `config:"queue"`
	SkipInvalidToken bool               `config:"skip_invalid_token"`
}

func (this *Plugin) Setup() {

	ok, err := env.ParseConfig("connector.yuque", &this)
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	if !this.Enabled {
		return
	}

	if this.PageSize <= 0 {
		this.PageSize = 1000
	}

	if this.Queue == nil {
		this.Queue = &queue.QueueConfig{Name: "indexing_documents"}
	}

	this.Queue = queue.SmartGetOrInitConfig(this.Queue)
}

func (this *Plugin) Start() error {
	if this.Enabled {
		task.RegisterScheduleTask(task.ScheduleTask{
			ID:          util.GetUUID(),
			Group:       "connectors",
			Singleton:   true,
			Interval:    util.GetDurationOrDefault(this.Interval, time.Second*30).String(), //connector's task interval
			Description: "indexing yuque docs",
			Task: func(ctx context.Context) {
				connector := common.Connector{}
				connector.ID = YuqueKey
				exists, err := orm.Get(&connector)
				if !exists || err != nil {
					panic("invalid hugo_site connector")
				}

				q := orm.Query{}
				q.Size = this.PageSize
				q.Conds = orm.And(orm.Eq("connector.id", connector.ID))
				var results []common.DataSource

				err, _ = orm.SearchWithJSONMapper(&results, &q)
				if err != nil {
					panic(err)
				}

				for _, item := range results {
					log.Debugf("ID: %s, Name: %s, Other: %s", item.ID, item.Name, util.MustToJSON(item))
					this.fetch_yuque(&connector, &item)
				}
			},
		})
	}

	return nil
}

func (this *Plugin) Stop() error {
	return nil
}

func (this *Plugin) Name() string {
	return YuqueKey
}

func (this *Plugin) fetch_yuque(connector *common.Connector, datasource *common.DataSource) {
	if connector == nil || datasource == nil {
		panic("invalid connector config")
	}

	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		panic(err)
	}

	obj := YuqueConfig{}
	err = cfg.Unpack(&obj)
	if err != nil {
		panic(err)
	}

	log.Debugf("handle hugo_site's datasource: %v", obj)
	this.collect(connector, datasource, &obj)
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}
