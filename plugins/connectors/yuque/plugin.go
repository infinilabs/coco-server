package yuque

import (
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/queue"
	"net/http"
)

const YuqueKey = "yuque"

type Config struct {
	Enabled bool `config:"enabled"`

	Token              string             `config:"token"` //TODO move to db
	Interval           string             `config:"interval"`
	Queue              *queue.QueueConfig `config:"queue"`
	IncludePrivateBook bool               `config:"include_private_book"`
	IncludePrivateDoc  bool               `config:"include_private_doc"`
	IndexingBooks      bool               `config:"indexing_books"`
	IndexingDocs       bool               `config:"indexing_docs"`
	IndexingUsers      bool               `config:"indexing_users"`
	IndexingGroups     bool               `config:"indexing_groups"`
}

type Plugin struct {
	api.Handler

	cfg Config
}

func (this *Plugin) Setup() {
	this.cfg = Config{
		Enabled:            false,
		Interval:           "10s",
		IncludePrivateDoc:  false,
		IncludePrivateBook: false,
		IndexingBooks:      true,
		IndexingDocs:       true,
		IndexingUsers:      true,
		Queue:              &queue.QueueConfig{Name: "indexing_documents"},
	}

	ok, err := env.ParseConfig("connector.yuque", &this.cfg)
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	if !this.cfg.Enabled {
		return
	}

	if this.cfg.Queue == nil {
		this.cfg.Queue = queue.GetOrInitConfig("indexing_documents")
	} else {
		queueCfg := queue.SmartGetOrInitConfig(this.cfg.Queue)
		this.cfg.Queue = queueCfg
	}

	if this.cfg.Token == "" {
		panic("invalid token")
	}

	api.HandleAPIMethod(api.GET, "/connector/yuque/connect", this.connect)

	//api.HandleAPIMethod(api.POST, "/connector/yuque/reset", this.reset)
	//api.HandleAPIMethod(api.GET, "/connector/yuque/oauth_redirect", this.oAuthRedirect)

}

func (this *Plugin) Start() error {
	//
	//if this.cfg.Enabled {
	//	//get all accounts which enabled google drive connector
	//
	//	task.RegisterScheduleTask(task.ScheduleTask{
	//		ID:          util.GetUUID(),
	//		Group:       "connectors",
	//		Singleton:   true,
	//		Interval:    util.GetDurationOrDefault(this.Interval, time.Second*30).String(),
	//		Description: "indexing google drive files",
	//		Task: func(ctx context.Context) {
	//
	//			log.Tracef("entering task, indexing google drive files")
	//
	//			//TODO
	//			var tenantID = "test"
	//			var userID = "test"
	//
	//			exists, tok, err := this.getToken(tenantID, userID)
	//			if err != nil {
	//				panic(err)
	//			}
	//
	//			if !exists {
	//				return
	//			}
	//
	//			if !tok.Valid() {
	//				//continue //TODO
	//				if !this.SkipInvalidToken && !tok.Valid() {
	//					panic("token is invalid")
	//				}
	//				log.Warnf("skip invalid token: %v", tok)
	//			} else {
	//				log.Debug("start processing google drive files")
	//				this.startIndexingFiles(tenantID, userID, tok)
	//				log.Debug("finished process google drive files")
	//			}
	//		},
	//	})
	//
	//}

	return nil
}

func (this *Plugin) Stop() error {
	return nil
}

func (this *Plugin) Name() string {
	return YuqueKey
}

func (this *Plugin) connect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	this.collect()
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}
