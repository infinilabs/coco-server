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
	Database      string `config:"database"`

	// Collections configuration
	Collections []CollectionConfig `config:"collections"`

	// Pagination configuration
	Pagination bool `config:"pagination"`
	PageSize   int  `config:"page_size"`

	// Last modified field for incremental sync
	LastModifiedField string `config:"last_modified_field"`

	// Performance optimization configuration
	BatchSize   int    `config:"batch_size"`
	Timeout     string `config:"timeout"`
	MaxPoolSize int    `config:"max_pool_size"`

	// Sync strategy
	SyncStrategy string `config:"sync_strategy"`

	// Field mapping configuration
	FieldMapping *FieldMappingConfig `config:"field_mapping"`

	// Advanced query optimization
	EnableProjection bool `config:"enable_projection"` // Enable projection pushdown
	EnableIndexHint  bool `config:"enable_index_hint"` // Enable index hints for better performance
}

type CollectionConfig struct {
	Name           string                 `config:"name"`
	Filter         map[string]interface{} `config:"filter"`
	TitleField     string                 `config:"title_field"`
	ContentField   string                 `config:"content_field"`
	CategoryField  string                 `config:"category_field"`
	TagsField      string                 `config:"tags_field"`
	URLField       string                 `config:"url_field"`
	TimestampField string                 `config:"timestamp_field"`
}

// FieldMappingConfig defines the field mapping configuration
type FieldMappingConfig struct {
	Enabled bool                    `config:"enabled"`
	Mapping map[string]interface{} `config:"mapping"`
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
	if config.PageSize <= 0 {
		config.PageSize = 500
	}
	if config.FieldMapping == nil {
		config.FieldMapping = &FieldMappingConfig{
			Enabled: false,
			Mapping: make(map[string]interface{}),
		}
	}
	
	// Enable advanced optimizations by default for better performance
	if !config.EnableProjection {
		config.EnableProjection = true
	}
	if !config.EnableIndexHint {
		config.EnableIndexHint = true
	}
}

func (p *Plugin) validateConfig(config *Config) error {
	if config.ConnectionURI == "" {
		return fmt.Errorf("connection_uri must be specified")
	}

	if config.Database == "" {
		return fmt.Errorf("database must be specified")
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

	if config.PageSize < 0 {
		return fmt.Errorf("page_size must be positive")
	}

	if config.SyncStrategy != "" && config.SyncStrategy != "full" && config.SyncStrategy != "incremental" {
		return fmt.Errorf("sync_strategy must be 'full' or 'incremental'")
	}

	return nil
}
