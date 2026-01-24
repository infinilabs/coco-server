/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package utils

import (
	"fmt"

	log "github.com/cihub/seelog"
	"golang.org/x/text/language"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/datasource"
)

// GetDatasource returns the datasource for the given document.
func GetDatasource(doc *core.Document) (*core.DataSource, error) {
	if doc.Source.ID == "" {
		return nil, fmt.Errorf("document has no datasource ID")
	}

	datasources, err := datasource.GetDatasourceByID([]string{doc.Source.ID})
	if err != nil {
		return nil, fmt.Errorf("failed to get datasource: %w", err)
	}
	if len(datasources) != 1 {
		return nil, fmt.Errorf("expected exactly 1 datasource, got %d", len(datasources))
	}

	return &datasources[0], nil
}

// GetConnectorID returns the connector ID for the given document.
func GetConnectorID(doc *core.Document) (string, error) {
	ds, err := GetDatasource(doc)
	if err != nil {
		return "", err
	}

	if ds.Connector.ConnectorID == "" {
		return "", fmt.Errorf("datasource has no connector ID")
	}

	return ds.Connector.ConnectorID, nil
}

// ValidateAndNormalizeLLMLang validates and normalizes a language tag for LLM generation.
// Defaults to "en-US" if empty or invalid.
// Returns the normalized BCP 47 language tag (e.g., "en-US", "zh-CN").
func ValidateAndNormalizeLLMLang(processorName, lang string) string {
	// Default to English if not set
	if lang == "" {
		return "en-US"
	}

	// Validate and normalize language tag
	tag, err := language.Parse(lang)
	if err != nil {
		log.Warnf("Processor [%s]: invalid llm_generation_lang %q, falling back to en-US: %v", processorName, lang, err)
		return "en-US"
	}

	// Normalize to BCP 47 format (e.g., "en_US" -> "en-US")
	return tag.String()
}
