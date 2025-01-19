/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package google_drive

import (
	"context"
	log "github.com/cihub/seelog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
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
	"os"
	"time"
)

type Plugin struct {
	api.Handler

	Enabled          bool               `config:"enabled"`
	Interval         string             `config:"interval"`
	PageSize         int                `config:"page_size"`
	CredentialFile   string             `config:"credential_file"`
	Credential       *Credential        `config:"credential"`
	SkipInvalidToken bool               `config:"skip_invalid_token"`
	Queue            *queue.QueueConfig `config:"queue"`
	oAuthConfig      *oauth2.Config
}

type Credential struct {
	ClientId                string   `config:"client_id" json:"client_id"`
	ProjectId               string   `config:"project_id" json:"project_id"`
	AuthUri                 string   `config:"auth_uri" json:"auth_uri"`
	TokenUri                string   `config:"token_uri" json:"token_uri"`
	AuthProviderX509CertUrl string   `config:"auth_provider_x509_cert_url" json:"auth_provider_x509_cert_url"`
	ClientSecret            string   `config:"client_secret" json:"client_secret"`
	RedirectUri             string   `config:"redirect_uris" json:"redirect_uris"`
	JavascriptOrigins       []string `config:"javascript_origins" json:"javascript_origins"`
}

func (this *Plugin) Setup() {
	ok, err := env.ParseConfig("connector.google_drive", &this)
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	if !this.Enabled {
		return
	}

	if this.Queue == nil {
		this.Queue = &queue.QueueConfig{Name: "indexing_documents"}
	}

	this.Queue = queue.SmartGetOrInitConfig(this.Queue)

	if this.CredentialFile != "" {
		b, err := os.ReadFile(this.CredentialFile)
		if err != nil {
			panic(err)
		}

		// Load credentials
		this.oAuthConfig, err = google.ConfigFromJSON(b, drive.DriveMetadataReadonlyScope)
		if err != nil {
			panic(err)
		}
	} else if this.Credential != nil {

		if this.Credential.ClientId == "" || this.Credential.ClientSecret == "" || len(this.Credential.RedirectUri) == 0 {
			panic("Missing Google OAuth credentials")
		}

		this.oAuthConfig = &oauth2.Config{
			ClientID:     this.Credential.ClientId,
			ClientSecret: this.Credential.ClientSecret,
			RedirectURL:  this.Credential.RedirectUri,
			Scopes:       []string{"https://www.googleapis.com/auth/drive.metadata.readonly"},
			Endpoint:     google.Endpoint,
		}
	} else {
		panic("Missing Google OAuth credentials")
	}

	api.HandleUIMethod(api.GET, "/connector/google_drive/connect", this.connect)
	api.HandleUIMethod(api.POST, "/connector/google_drive/reset", this.reset)
	api.HandleUIMethod(api.GET, "/connector/google_drive/oauth_redirect", this.oAuthRedirect)

}

func (this *Plugin) Start() error {

	if this.Enabled {
		task.RegisterScheduleTask(task.ScheduleTask{
			ID:          util.GetUUID(),
			Group:       "connectors",
			Singleton:   true,
			Interval:    util.GetDurationOrDefault(this.Interval, time.Second*30).String(), //connector's task interval
			Description: "indexing google drive files",
			Task: func(ctx context.Context) {
				connector := common.Connector{}
				connector.ID = "google_drive"
				exists, err := orm.Get(&connector)
				if !exists || err != nil {
					panic("invalid hugo_site connector")
				}

				q := orm.Query{}
				q.Size = this.PageSize //TODO
				q.Conds = orm.And(orm.Eq("connector.id", connector.ID))
				var results []common.DataSource

				err, _ = orm.SearchWithJSONMapper(&results, &q)
				if err != nil {
					panic(err)
				}

				for _, item := range results {
					log.Debugf("ID: %s, Name: %s, Other: %s", item.ID, item.Name, util.MustToJSON(item))
					this.fetch_google_drive(&connector, &item)
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
	return "google_drive"
}

type Config struct {
	Token string `config:"token"`
}

func (this *Plugin) fetch_google_drive(connector *common.Connector, datasource *common.DataSource) {
	if connector == nil || datasource == nil {
		panic("invalid connector config")
	}

	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		panic(err)
	}

	datasourceCfg := Config{}
	err = cfg.Unpack(&datasourceCfg)
	if err != nil {
		panic(err)
	}

	log.Tracef("handle google_drive's datasource: %v", datasourceCfg)

	if datasourceCfg.Token != "" {
		tok := oauth2.Token{}
		err = util.FromJSONBytes([]byte(datasourceCfg.Token), &tok)
		if err != nil {
			panic(err)
		}

		//TODO
		var tenantID = "test"
		var userID = "test"

		if !tok.Valid() {
			//continue //TODO
			if !this.SkipInvalidToken && !tok.Valid() {
				panic("token is invalid")
			}
			//TODO, update the datasource's config, avoid access before fix
			log.Warnf("skip invalid google_drive token: %v", tok)
		} else {
			log.Debug("start processing google drive files")
			this.startIndexingFiles(tenantID, userID, datasource.ID, &tok)
			log.Debug("finished process google drive files")
		}
	}

}

func init() {
	module.RegisterUserPlugin(&Plugin{SkipInvalidToken: true, PageSize: 100})
}
