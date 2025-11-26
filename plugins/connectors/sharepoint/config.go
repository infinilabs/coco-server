package sharepoint

import (
	"fmt"
	"time"

	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/config"
)

func parseSharePointConfig(datasource *common.DataSource) (*SharePointConfig, error) {
	if datasource.Connector.Config == nil {
		return nil, fmt.Errorf("connector config is nil")
	}

	cfg, err := config.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	sharePointConfig := &SharePointConfig{}
	err = cfg.Unpack(sharePointConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack config: %w", err)
	}

	// 设置默认的重试配置
	if sharePointConfig.RetryConfig.MaxRetries == 0 {
		sharePointConfig.RetryConfig.MaxRetries = 3
	}
	if sharePointConfig.RetryConfig.InitialDelay == 0 {
		sharePointConfig.RetryConfig.InitialDelay = time.Second
	}
	if sharePointConfig.RetryConfig.MaxDelay == 0 {
		sharePointConfig.RetryConfig.MaxDelay = time.Minute
	}
	if sharePointConfig.RetryConfig.BackoffFactor == 0 {
		sharePointConfig.RetryConfig.BackoffFactor = 2.0
	}

	return sharePointConfig, nil
}

func validateSharePointConfig(config *SharePointConfig) error {
	if config.SiteURL == "" {
		return fmt.Errorf("site_url is required")
	}
	if config.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if config.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if config.ClientSecret == "" {
		return fmt.Errorf("client_secret is required")
	}

	return nil
}
