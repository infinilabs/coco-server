package mongodb

import (
	"errors"
	"fmt"

	cmn "infini.sh/coco/plugins/connectors/common"
)

// Config represents the MongoDB connector configuration
type Config struct {
	// Connection settings
	ConnectionURI string `config:"connection_uri"` // MongoDB connection string (mongodb://...)
	Database      string `config:"database"`       // Database name
	Collection    string `config:"collection"`     // Collection name

	// Query and filtering
	Query string `config:"query"` // BSON query in JSON format, e.g. {"status": "published"}
	Sort  string `config:"sort"`  // Sort specification in JSON format, e.g. {"updated_at": 1, "_id": 1}

	// Pagination
	Pagination bool `config:"pagination"` // Enable pagination
	PageSize   uint `config:"page_size"`  // Documents per page (default: 500)

	// Incremental sync (reusing common configuration)
	Incremental cmn.IncrementalConfig `config:"incremental"`

	// Field mapping
	FieldMapping cmn.FieldMapping `config:"field_mapping"`
}

const (
	// DefaultPageSize Default values
	DefaultPageSize = 500

	// MaxPageSize Maximum page size to prevent memory issues
	MaxPageSize = 10000
)

// Validate validates the configuration and sets defaults
func (cfg *Config) Validate() error {
	// Required fields
	if cfg.ConnectionURI == "" {
		return errors.New("connection_uri is required")
	}
	if cfg.Database == "" {
		return errors.New("database is required")
	}
	if cfg.Collection == "" {
		return errors.New("collection is required")
	}

	// Validate incremental configuration
	if err := cfg.Incremental.Validate(); err != nil {
		return err
	}

	// Set pagination defaults
	if cfg.Pagination {
		if cfg.PageSize == 0 {
			cfg.PageSize = DefaultPageSize
		}
		if cfg.PageSize > MaxPageSize {
			return fmt.Errorf("page_size cannot exceed %d", MaxPageSize)
		}
	} else {
		// If pagination is disabled, use a reasonable default for memory safety
		if cfg.PageSize == 0 {
			cfg.PageSize = DefaultPageSize
		}
	}

	return nil
}

// IsIncrementalEnabled returns true if incremental sync is configured
func (cfg *Config) IsIncrementalEnabled() bool {
	return cfg.Incremental.IsEnabled()
}

// GetPageSize returns the effective page size
func (cfg *Config) GetPageSize() uint {
	if cfg.PageSize == 0 {
		return DefaultPageSize
	}
	if cfg.PageSize > MaxPageSize {
		return MaxPageSize
	}
	return cfg.PageSize
}

// GetPropertyType returns the normalized property type
func (cfg *Config) GetPropertyType() string {
	return cfg.Incremental.GetPropertyType()
}

// String returns a summary of the configuration (for logging)
func (cfg *Config) String() string {
	return fmt.Sprintf("MongoDB{db=%s, collection=%s, incremental=%v, property=%s}",
		cfg.Database, cfg.Collection, cfg.Incremental.IsEnabled(), cfg.Incremental.Property)
}
