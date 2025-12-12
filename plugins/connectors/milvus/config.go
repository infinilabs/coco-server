package milvus

import (
	"errors"
	"fmt"
	"strings"

	cmn "infini.sh/coco/plugins/connectors/common"
)

const (
	DefaultPageSize = 1000
	MaxPageSize     = 10000
)

// Config represents the Milvus connector configuration
type Config struct {
	// Connection settings
	Address    string `config:"address"`    // Milvus service address (e.g., "localhost:19530")
	Username   string `config:"username"`   // Optional: Username for Milvus connection
	Password   string `config:"password"`   // Optional: Password for Milvus connection
	DBName     string `config:"db_name"`    // Optional: Database name (Milvus 2.2.0+)
	Collection string `config:"collection"` // Required: Name of the Milvus collection

	// Query settings
	OutputFields []string `config:"output_fields"` // Fields to retrieve (scalar fields, primary key). Default to all scalar fields.
	Filter       string   `config:"filter"`        // Scalar filtering expression (e.g., "age > 10 and name like \"abc%\"")

	// Pagination
	PageSize uint `config:"page_size"` // Documents per page (default: 1000)

	// Incremental sync (reusing common configuration)
	Incremental cmn.IncrementalConfig `config:"incremental"`

	// Field mapping
	FieldMapping cmn.FieldMapping `config:"field_mapping"`
}

// Validate validates the configuration and sets defaults
func (cfg *Config) Validate() error {
	// Required fields
	if strings.TrimSpace(cfg.Address) == "" {
		return errors.New("address is required")
	}
	if strings.TrimSpace(cfg.Collection) == "" {
		return errors.New("collection is required")
	}

	// Validate incremental configuration
	if err := cfg.Incremental.Validate(); err != nil {
		return err
	}

	// Set pagination defaults
	if cfg.PageSize == 0 {
		cfg.PageSize = DefaultPageSize
	}
	if cfg.PageSize > MaxPageSize {
		return fmt.Errorf("page_size cannot exceed %d", MaxPageSize)
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

// String returns a summary of the configuration (for logging)
func (cfg *Config) String() string {
	return fmt.Sprintf("Milvus{address=%s, db=%s, collection=%s, incremental=%v, property=%s}",
		cfg.Address, cfg.DBName, cfg.Collection, cfg.Incremental.IsEnabled(), cfg.Incremental.Property)
}
