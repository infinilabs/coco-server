/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"fmt"
)

// Config defines the configuration for the MongoDB connector
type Config struct {
	// Connection configuration
	ConnectionURI string `config:"connection_uri"`
	Database      string `config:"database"`
	AuthDatabase  string `config:"auth_database"` // Authentication database (e.g., "admin")
	ClusterType   string `config:"cluster_type"`  // Cluster type: "standalone", "replica_set", "sharded"

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

	// Field mapping configuration - This handles all field mappings
	FieldMapping *FieldMappingConfig `config:"field_mapping"`

	// Advanced query optimization
	EnableProjection bool `config:"enable_projection"` // Enable projection pushdown
	EnableIndexHint  bool `config:"enable_index_hint"` // Enable index hints for better performance
}

// CollectionConfig defines collection-specific configuration
// Field mapping is now handled by the global FieldMapping configuration
type CollectionConfig struct {
	Name   string                 `config:"name"`   // Collection name
	Filter map[string]interface{} `config:"filter"` // MongoDB query filter for this collection
}

// FieldMappingConfig defines the field mapping configuration
// This replaces the individual field configurations in CollectionConfig
type FieldMappingConfig struct {
	Enabled bool                   `config:"enabled"`
	Mapping map[string]interface{} `config:"mapping"`

	// Standard field mappings for common document fields
	TitleField     string `config:"title_field"`     // MongoDB field name for document title
	ContentField   string `config:"content_field"`   // MongoDB field name for document content
	CategoryField  string `config:"category_field"`  // MongoDB field name for document category
	TagsField      string `config:"tags_field"`      // MongoDB field name for document tags
	URLField       string `config:"url_field"`       // MongoDB field name for document URL
	TimestampField string `config:"timestamp_field"` // MongoDB field name for document timestamp
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
	if config.AuthDatabase == "" {
		config.AuthDatabase = "admin" // Default to admin database for authentication
	}
	if config.ClusterType == "" {
		config.ClusterType = "standalone" // Default to standalone MongoDB instance
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

	if config.ClusterType != "" && config.ClusterType != "standalone" && config.ClusterType != "replica_set" && config.ClusterType != "sharded" {
		return fmt.Errorf("cluster_type must be 'standalone', 'replica_set', or 'sharded'")
	}

	return nil
}
