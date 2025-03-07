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
			Endpoint:     google.Endpoint,
		}
	}

	if this.oAuthConfig == nil {
		this.oAuthConfig = &oauth2.Config{}
	}
	this.oAuthConfig.Scopes = []string{
		"https://www.googleapis.com/auth/drive.metadata.readonly", // Access Drive metadata
		"https://www.googleapis.com/auth/userinfo.email",          // Access the user's profile information
		"https://www.googleapis.com/auth/userinfo.profile",        // Access the user's profile information
	}

	api.HandleUIMethod(api.GET, "/connector/google_drive/connect", this.connect, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/connector/google_drive/reset", this.reset, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/connector/google_drive/oauth_redirect", this.oAuthRedirect, api.RequireLogin())

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
				if this.oAuthConfig.ClientID == "" {
					log.Debugf("skipping google_drive connector task since empty client_id")
					return
				}
				connector := common.Connector{}
				connector.ID = "google_drive"
				exists, err := orm.Get(&connector)
				if !exists || err != nil {
					panic("invalid hugo_site connector")
				}

				q := orm.Query{}
				q.Size = this.PageSize //TODO
				q.Conds = orm.And(orm.Eq("connector.id", connector.ID), orm.Eq("sync_enabled", true))
				var results []common.DataSource
				err, _ = orm.SearchWithJSONMapper(&results, &q)
				if err != nil {
					panic(err)
				}

				log.Infof("total %v google_drives pending to fetch", len(results))

				for _, item := range results {
					if global.ShuttingDown() {
						break
					}

					log.Infof("fetch google_drive: ID: %s, Name: %s", item.ID, item.Name)
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
	AccessToken  string      `config:"access_token" json:"access_token"`
	RefreshToken string      `config:"refresh_token" json:"refresh_token"`
	TokenExpiry  string      `config:"token_expiry" json:"token_expiry"`
	Profile      util.MapStr `config:"profile" json:"profile"`
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

	if datasourceCfg.AccessToken != "" && datasourceCfg.RefreshToken != "" {
		// Initialize the token using AccessToken and RefreshToken
		tok := oauth2.Token{
			AccessToken:  datasourceCfg.AccessToken,
			RefreshToken: datasourceCfg.RefreshToken,
			Expiry:       parseExpiry(datasourceCfg.TokenExpiry),
		}

		//TODO: Define tenantID and userID, possibly based on your context
		var tenantID = "test"
		var userID = "test"

		// Check if the token is valid
		if !tok.Valid() {
			// Check if SkipInvalidToken is false, which means token must be valid
			if !this.SkipInvalidToken && !tok.Valid() {
				panic("token is invalid")
			}

			// If the token is invalid, attempt to refresh it
			log.Warnf("Token is invalid or expired, attempting to refresh...")

			// Refresh the token using the refresh token
			if tok.RefreshToken != "" {
				log.Debug("Attempting to refresh the token")
				tokenSource := this.oAuthConfig.TokenSource(context.Background(), &tok)
				refreshedToken, err := tokenSource.Token()
				if err != nil {
					log.Errorf("Failed to refresh token: %v", err)
					panic("Failed to refresh token")
				}

				// Save the refreshed token
				datasourceCfg.AccessToken = refreshedToken.AccessToken
				datasourceCfg.RefreshToken = refreshedToken.RefreshToken
				datasourceCfg.TokenExpiry = refreshedToken.Expiry.Format(time.RFC3339) // Format using RFC3339

				datasource.Connector.Config = datasourceCfg

				log.Debugf("updating datasource with new refresh token: %v", datasource.ID)

				// Optionally, save the new tokens in your store (e.g., database or config)
				err = orm.Update(nil, datasource)
				if err != nil {
					log.Errorf("Failed to save updated datasource configuration: %v", err)
					panic("Failed to save updated configuration")
				}

				log.Debug("Token refreshed successfully")
			} else {
				log.Warnf("No refresh token available, unable to refresh token.")
			}

		} else {
			// Token is valid, proceed with indexing files
			log.Debug("start processing google drive files")
			this.startIndexingFiles(connector, datasource, tenantID, userID, &tok)
			log.Debug("finished processing google drive files")
		}
	}

}

// Helper function to parse token expiry time
func parseExpiry(expiryStr string) time.Time {
	// Parse the expiry time string into a time.Time object
	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		log.Errorf("Failed to parse token expiry: %v", err)
		return time.Time{} // Return zero value time on error
	}
	return expiry
}

func init() {
	module.RegisterUserPlugin(&Plugin{SkipInvalidToken: true, PageSize: 100})
}
