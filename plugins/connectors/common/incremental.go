/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// ModePropertyWatermark is the property-based watermark incremental sync mode
	// This mode tracks changes using a property field (e.g., timestamp, id) with a tie-breaker
	ModePropertyWatermark = "property_watermark"
)

// IncrementalConfig defines the configuration for incremental synchronization
// This is a reusable configuration structure for connectors that support incremental sync
type IncrementalConfig struct {
	// Enabled indicates whether incremental sync is enabled
	Enabled bool `config:"enabled"`

	// Mode specifies the incremental sync mode (currently only "property_watermark" is supported)
	Mode string `config:"mode"`

	// Property is the field name to use as the watermark (e.g., "updated_at", "_id")
	Property string `config:"property"`

	// PropertyType is the data type of the property field ("datetime", "int", "float", "string", "bool")
	PropertyType string `config:"property_type"`

	// TieBreaker is the field used to break ties when multiple records have the same property value
	// This ensures stable ordering and prevents missing records (e.g., "id", "_id", "element_id")
	TieBreaker string `config:"tie_breaker"`

	// ResumeFrom is an optional manual starting point for the first sync
	// Format depends on PropertyType (e.g., RFC3339 for datetime, numeric string for int)
	ResumeFrom string `config:"resume_from"`
}

// Validate validates the incremental configuration
func (inc *IncrementalConfig) Validate() error {
	if !inc.Enabled {
		return nil
	}

	// Default to property_watermark mode
	if inc.Mode == "" {
		inc.Mode = ModePropertyWatermark
	}

	// Only property_watermark mode is supported
	if inc.Mode != ModePropertyWatermark {
		return fmt.Errorf("unsupported incremental mode %q, only %q is supported", inc.Mode, ModePropertyWatermark)
	}

	// Property field is required
	if strings.TrimSpace(inc.Property) == "" {
		return errors.New("incremental.property is required when incremental sync is enabled")
	}

	// Tie-breaker field is required
	if strings.TrimSpace(inc.TieBreaker) == "" {
		return errors.New("incremental.tie_breaker is required when incremental sync is enabled")
	}

	// Normalize property type
	inc.PropertyType = NormalizePropertyType(inc.PropertyType)

	return nil
}

// IsEnabled is a convenience method to check if incremental sync is enabled
func (inc *IncrementalConfig) IsEnabled() bool {
	return inc.Enabled
}

// GetPropertyType returns the normalized property type
func (inc *IncrementalConfig) GetPropertyType() string {
	if inc.PropertyType == "" {
		return "string"
	}
	return inc.PropertyType
}
