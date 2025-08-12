/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"fmt"
	"time"
)

// Config defines the configuration for the MongoDB connector
type Config struct {
	// Connection configuration
	ConnectionURI string `config:"connection_uri"`
	Host          string `config:"host"`
	Port          int    `config:"port"`
	Username      string `config:"username"`
	Password      string `config:"password"`
	Database      string `config:"database"`
	AuthDatabase  string `config:"auth_database"`

	// Replica set and sharding configuration
	ReplicaSet     string `config:"replica_set"`
	ReadPreference string `config:"read_preference"`

	// TLS/SSL configuration
	EnableTLS   bool   `config:"enable_tls"`
	TLSCAFile   string `config:"tls_ca_file"`
	TLSCertFile string `config:"tls_cert_file"`
	TLSKeyFile  string `config:"tls_key_file"`
	TLSInsecure bool   `config:"tls_insecure"`

	// Data filtering configuration
	Collections []CollectionConfig `config:"collections"`

	// Performance optimization configuration
	BatchSize   int    `config:"batch_size"`
	Timeout     string `config:"timeout"`
	MaxPoolSize int    `config:"max_pool_size"`

	// Sync strategy
	SyncStrategy   string    `config:"sync_strategy"`
	TimestampField string    `config:"timestamp_field"`
	LastSyncTime   time.Time `config:"last_sync_time"`
}

type CollectionConfig struct {
	Name           string                 `config:"name"`
	Filter         map[string]interface{} `config:"filter"`
	Fields         []string               `config:"fields"`
	TitleField     string                 `config:"title_field"`
	ContentField   string                 `config:"content_field"`
	CategoryField  string                 `config:"category_field"`
	TagsField      string                 `config:"tags_field"`
	URLField       string                 `config:"url_field"`
	TimestampField string                 `config:"timestamp_field"`
}

func (p *Plugin) setDefaultConfig(config *Config) {
	if config.BatchSize <= 0 {
		config.BatchSize = 1000
	}
	if config.MaxPoolSize <= 0 {
		config.MaxPoolSize = 10
	}
	if config.Timeout == "" {
		config.Timeout = "30s"
	}
	if config.SyncStrategy == "" {
		config.SyncStrategy = "full"
	}
}

func (p *Plugin) validateConfig(config *Config) error {
	if config.ConnectionURI == "" {
		if config.Host == "" {
			return fmt.Errorf("either connection_uri or host must be specified")
		}
		if config.Database == "" {
			return fmt.Errorf("database must be specified")
		}
	}

	if len(config.Collections) == 0 {
		return fmt.Errorf("at least one collection must be configured")
	}

	for i, coll := range config.Collections {
		if coll.Name == "" {
			return fmt.Errorf("collection[%d].name is required", i)
		}
	}

	if config.BatchSize < 0 {
		return fmt.Errorf("batch_size must be positive")
	}

	if config.MaxPoolSize < 0 {
		return fmt.Errorf("max_pool_size must be positive")
	}

	if config.SyncStrategy != "" && config.SyncStrategy != "full" && config.SyncStrategy != "incremental" {
		return fmt.Errorf("sync_strategy must be 'full' or 'incremental'")
	}

	return nil
}
