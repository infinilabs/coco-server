/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package dropbox

import (
	"errors"

	"infini.sh/coco/core"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

const NAME = "dropbox"

type Credential struct {
	ClientId     string `config:"client_id" json:"client_id"`
	ClientSecret string `config:"client_secret" json:"client_secret"`
	RedirectUri  string `config:"redirect_url" json:"redirect_url"`
}

type Config struct {
	ClientId     string      `config:"client_id" json:"client_id"`
	ClientSecret string      `config:"client_secret" json:"client_secret"`
	RefreshToken string      `config:"refresh_token" json:"refresh_token"`
	Profile      util.MapStr `config:"profile" json:"profile"`
	Path         string      `config:"path" json:"path"`
}

type Processor struct {
	cmn.ConnectorProcessorBase
}

func init() {
	pipeline.RegisterProcessorPlugin(NAME, New)
	api.HandleUIMethod(api.GET, "/connector/:id/dropbox/connect", connect, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/connector/:id/dropbox/oauth_redirect", oAuthRedirect, api.RequireLogin())
}

func New(c *config.Config) (pipeline.Processor, error) {
	runner := Processor{}
	runner.Init(c, &runner)
	return &runner, nil
}

func (processor *Processor) Name() string {
	return NAME
}

func (processor *Processor) Fetch(pipeCtx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	cfg := Config{}
	processor.MustParseConfig(datasource, &cfg)

	if cfg.ClientId == "" || cfg.ClientSecret == "" {
		return errors.New("client_id and client_secret are required for Dropbox connector")
	}

	log.Tracef("handle dropbox's datasource: %v", cfg)

	// Create Dropbox client
	client := NewDropboxClient(&cfg)

	// Start indexing
	log.Debug("start processing dropbox files")
	processor.startIndexingFiles(pipeCtx, connector, datasource, client)
	log.Debug("finished processing dropbox files")

	return nil
}
