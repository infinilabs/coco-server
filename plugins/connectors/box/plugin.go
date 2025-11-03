/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package box

import (
	"fmt"

	"infini.sh/coco/core"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"

	log "github.com/cihub/seelog"
)

const NAME = "box"

type Config struct {
	// Account type: "box_free" or "box_enterprise"
	IsEnterprise string `config:"is_enterprise" json:"is_enterprise"`

	// OAuth credentials (required for both account types)
	ClientID     string `config:"client_id" json:"client_id"`
	ClientSecret string `config:"client_secret" json:"client_secret"`

	// For Box Free Account only
	RefreshToken string `config:"refresh_token" json:"refresh_token"`

	// For Box Enterprise Account only
	EnterpriseID string `config:"enterprise_id" json:"enterprise_id"`

	// Optional settings
	ConcurrentDownloads int `config:"concurrent_downloads" json:"concurrent_downloads"`
}

type Processor struct {
	cmn.ConnectorProcessorBase
}

func init() {
	pipeline.RegisterProcessorPlugin(NAME, New)

	// Register OAuth routes for Box Free Account
	api.HandleUIMethod(api.GET, "/connector/:id/box/connect", connect, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/connector/:id/box/oauth_redirect", oAuthRedirect, api.RequireLogin())
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

	log.Debugf("[%s connector] handling datasource: %s", NAME, datasource.Name)

	// Validate required configuration
	if cfg.ClientID == "" {
		return fmt.Errorf("client_id is required for Box connector")
	}
	if cfg.ClientSecret == "" {
		return fmt.Errorf("client_secret is required for Box connector")
	}

	// Validate account type specific configuration
	if cfg.IsEnterprise == AccountTypeFree {
		if cfg.RefreshToken == "" {
			return fmt.Errorf("refresh_token is required for Box Free Account")
		}
	} else if cfg.IsEnterprise == AccountTypeEnterprise {
		if cfg.EnterpriseID == "" {
			return fmt.Errorf("enterprise_id is required for Box Enterprise Account")
		}
	} else {
		// Default to free account if not specified
		cfg.IsEnterprise = AccountTypeFree
		if cfg.RefreshToken == "" {
			return fmt.Errorf("refresh_token is required for Box Free Account")
		}
	}

	// Try to get client from cache first
	client, cached := GetCachedClient(datasource.ID)

	if cached {
		log.Debugf("[%s connector] Using cached client for datasource: %s", NAME, datasource.ID)
	} else {
		// No cached client, create a new one
		log.Debugf("[%s connector] No cached client found, creating new client for datasource: %s", NAME, datasource.ID)
		client = NewBoxClient(&cfg)

		// Test connection for new client
		log.Debugf("[%s connector] testing connection...", NAME)
		if err := client.Ping(); err != nil {
			return fmt.Errorf("failed to connect to Box: %v", err)
		}
		log.Debugf("[%s connector] connection test successful", NAME)

		// Cache the new client for future use
		CacheClient(datasource.ID, client)
	}

	// Start processing files
	log.Debugf("[%s connector] start processing box files for datasource: %s", NAME, datasource.Name)

	if cfg.IsEnterprise == AccountTypeEnterprise {
		// Enterprise account: fetch files for all users
		log.Infof("[%s connector] Fetching data from Box's Enterprise Account", NAME)

		users, err := client.GetUsers()
		if err != nil {
			return fmt.Errorf("failed to get enterprise users: %v", err)
		}

		log.Infof("[%s connector] Found %d users in enterprise", NAME, len(users))

		for _, user := range users {
			log.Debugf("[%s connector] Processing files for user: %s (%s)", NAME, user.Name, user.ID)
			processor.startIndexingFilesForUser(pipeCtx, connector, datasource, client, user.ID, user.Name)
		}
	} else {
		// Free account: fetch files for current authenticated user
		log.Infof("[%s connector] Fetching data from Box's Free Account", NAME)
		processor.startIndexingFiles(pipeCtx, connector, datasource, client)
	}

	log.Infof("[%s connector] finished fetching datasource [%s]", NAME, datasource.Name)

	return nil
}
