package feishu

import (
	"context"
	"time"

	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

const (
	ConnectorLark = "lark"
)

type LarkPlugin struct {
	Plugin
}

func (this *LarkPlugin) Setup() {
	// Set plugin type first
	this.SetPluginType(PluginTypeLark)

	ok, err := env.ParseConfig("connector.lark", &this.Plugin)
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	if !this.Enabled {
		return
	}
	if this.PageSize <= 0 {
		this.PageSize = 100
	}
	if this.Queue == nil {
		this.Queue = &queue.QueueConfig{Name: "indexing_documents"}
	}
	this.Queue = queue.SmartGetOrInitConfig(this.Queue)

	// Set default OAuth configuration if not provided
	if this.OAuthConfig == nil {
		apiConfig := this.GetAPIConfig()
		this.OAuthConfig = &OAuthConfig{
			AuthURL:     apiConfig.AuthURL,
			TokenURL:    apiConfig.TokenURL,
			RedirectURI: "/connector/lark/oauth_redirect", // Will be dynamically built from request
		}
	}

	// Register OAuth routes
	log.Debugf("[lark connector] Attempting to register OAuth routes...")
	api.HandleUIMethod(api.GET, "/connector/lark/connect", this.connect, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/connector/lark/oauth_redirect", this.oAuthRedirect, api.RequireLogin())
	log.Infof("[lark connector] OAuth routes registered successfully")
}

func (this *LarkPlugin) Start() error {
	if this.Enabled {
		task.RegisterScheduleTask(task.ScheduleTask{
			ID:          util.GetUUID(),
			Group:       "connectors",
			Singleton:   true,
			Interval:    util.GetDurationOrDefault(this.Interval, time.Second*30).String(),
			Description: "indexing lark cloud documents",
			Task: func(ctx context.Context) {
				connector := common.Connector{}
				connector.ID = ConnectorLark
				exists, err := orm.Get(&connector)
				if !exists {
					log.Debugf("Connector %s not found", connector.ID)
					return
				}
				if err != nil {
					panic(errors.Errorf("invalid %s connector:%v", connector.ID, err))
				}

				q := orm.Query{}
				q.Size = this.PageSize
				q.Conds = orm.And(orm.Eq("connector.id", connector.ID), orm.Eq("sync_enabled", true))
				var results []common.DataSource
				err, _ = orm.SearchWithJSONMapper(&results, &q)
				if err != nil {
					panic(err)
				}

				for _, item := range results {
					toSync, err := connectors.CanDoSync(item)
					if err != nil {
						_ = log.Errorf("error checking syncable with datasource [%s]: %v", item.Name, err)
						continue
					}
					if !toSync {
						continue
					}
					log.Debugf("fetch lark cloud docs: ID: %s, Name: %s", item.ID, item.Name)
					this.fetchCloudDocs(&connector, &item)
				}
			},
		})
	}
	return nil
}

func (this *LarkPlugin) Stop() error {
	return nil
}

func (this *LarkPlugin) Name() string {
	return ConnectorLark
}

func init() {
	module.RegisterUserPlugin(&LarkPlugin{})
}
