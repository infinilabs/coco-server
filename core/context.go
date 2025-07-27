// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Console is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
	"strings"
)

const Secret = "coco"

var secretKey string

func GetSecret() (string, error) {
	if secretKey != "" {
		return secretKey, nil
	}

	exists, err := kv.ExistsKey("Coco", []byte(Secret))
	if err != nil {
		return "", fmt.Errorf("failed to check if secret key exists: %w", err)
	}

	if !exists {
		key := util.GetUUID()
		err = kv.AddValue("Coco", []byte(Secret), []byte(key))
		if err != nil {
			return "", fmt.Errorf("failed to store new secret key: %w", err)
		}
		secretKey = key
	} else {
		v, err := kv.GetValue("Coco", []byte(Secret))
		if err != nil {
			return "", fmt.Errorf("failed to retrieve secret key: %w", err)
		}
		if len(v) > 0 {
			secretKey = string(v)
		}
	}

	if secretKey == "" {
		return "", fmt.Errorf("secret key is empty or invalid")
	}

	return secretKey, nil
}

// Maximum allowed size for query DSL input (1MB)
const maxQueryDSLSize = 1024 * 1024

// validateQueryDSLInput validates the input queryDsl for security and structure
func validateQueryDSLInput(queryDsl []byte) error {
	// Check input size limits
	if len(queryDsl) > maxQueryDSLSize {
		return errors.New("query DSL exceeds maximum allowed size of 1MB")
	}

	// Allow empty query (will be handled as match_all)
	if len(queryDsl) == 0 {
		return nil
	}

	// Validate that it's valid JSON
	var temp interface{}
	if err := json.Unmarshal(queryDsl, &temp); err != nil {
		return fmt.Errorf("invalid JSON structure: %w", err)
	}

	// Convert to string for basic content validation
	queryStr := string(queryDsl)

	// Basic security checks - prevent potential script injection
	dangerousPatterns := []string{
		"<script", "</script", "javascript:", "eval(", "expression(",
		"document.", "window.", "alert(", "confirm(", "prompt(",
	}

	queryStrLower := strings.ToLower(queryStr)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(queryStrLower, pattern) {
			return fmt.Errorf("query contains potentially dangerous content: %s", pattern)
		}
	}

	return nil
}

// validateFilterInput validates the filter parameter for security and structure
func validateFilterInput(filter util.MapStr) error {
	if filter == nil {
		return errors.New("filter cannot be nil")
	}

	// Convert to JSON and back to validate structure
	filterBytes, err := json.Marshal(filter)
	if err != nil {
		return fmt.Errorf("filter contains invalid data structure: %w", err)
	}

	// Check filter size (should be reasonable)
	if len(filterBytes) > 10*1024 { // 10KB limit for filters
		return errors.New("filter exceeds maximum allowed size of 10KB")
	}

	// Validate filter contains valid Elasticsearch query components
	validFilterKeys := map[string]bool{
		"term": true, "terms": true, "range": true, "exists": true,
		"bool": true, "match": true, "match_phrase": true, "wildcard": true,
		"prefix": true, "regexp": true, "fuzzy": true, "ids": true,
	}

	// Check if filter contains at least one valid query type
	hasValidKey := false
	for key := range filter {
		if validFilterKeys[key] {
			hasValidKey = true
			break
		}
	}

	if !hasValidKey {
		return errors.New("filter must contain at least one valid Elasticsearch query clause")
	}

	return nil
}

// isValidElasticsearchQuery performs basic validation of Elasticsearch query structure
func isValidElasticsearchQuery(mapObj util.MapStr) error {
	// Check for valid top-level Elasticsearch query structure
	validTopLevelKeys := map[string]bool{
		"query": true, "size": true, "from": true, "sort": true,
		"_source": true, "aggs": true, "aggregations": true,
		"highlight": true, "track_scores": true, "track_total_hits": true,
		"version": true, "timeout": true, "terminate_after": true,
	}

	// Allow empty query (defaults to match_all)
	if len(mapObj) == 0 {
		return nil
	}

	// Validate top-level keys
	for key := range mapObj {
		if !validTopLevelKeys[key] && !strings.HasPrefix(key, "_") {
			return fmt.Errorf("invalid top-level query key: %s", key)
		}
	}

	// If query field exists, validate its structure
	if queryField, exists := mapObj["query"]; exists {
		if queryMap, ok := queryField.(map[string]interface{}); ok {
			return validateQueryClause(queryMap)
		}
		return errors.New("query field must be an object")
	}

	return nil
}

// validateQueryClause validates individual query clauses
func validateQueryClause(query map[string]interface{}) error {
	validQueryTypes := map[string]bool{
		"match": true, "match_all": true, "match_phrase": true, "match_phrase_prefix": true,
		"multi_match": true, "term": true, "terms": true, "range": true, "exists": true,
		"bool": true, "wildcard": true, "prefix": true, "regexp": true, "fuzzy": true,
		"ids": true, "constant_score": true, "dis_max": true, "function_score": true,
		"boosting": true, "nested": true, "has_child": true, "has_parent": true,
	}

	if len(query) == 0 {
		return errors.New("query clause cannot be empty")
	}

	for queryType := range query {
		if !validQueryTypes[queryType] {
			return fmt.Errorf("invalid query type: %s", queryType)
		}
	}

	return nil
}

func RewriteQueryWithFilter(queryDsl []byte, filter util.MapStr) ([]byte, error) {
	// Validate inputs
	if err := validateQueryDSLInput(queryDsl); err != nil {
		return nil, fmt.Errorf("invalid query DSL input: %w", err)
	}

	if err := validateFilterInput(filter); err != nil {
		return nil, fmt.Errorf("invalid filter input: %w", err)
	}

	// Parse query DSL with proper error handling
	mapObj := util.MapStr{}
	if len(queryDsl) > 0 {
		err := util.FromJSONBytes(queryDsl, &mapObj)
		if err != nil {
			return nil, fmt.Errorf("failed to parse query DSL: %w", err)
		}
	}

	// Validate the parsed query structure
	if err := isValidElasticsearchQuery(mapObj); err != nil {
		return nil, fmt.Errorf("invalid Elasticsearch query structure: %w", err)
	}

	// Build filter query
	must := []util.MapStr{filter}
	filterQ := util.MapStr{
		"bool": util.MapStr{
			"must": must,
		},
	}

	// Apply filter to existing query or create new query
	if queryField, exists := mapObj["query"]; exists {
		// Safely handle existing query with type assertion
		if queryMap, ok := queryField.(map[string]interface{}); ok {
			newQuery := util.MapStr{
				"bool": util.MapStr{
					"filter": filterQ,
					"must":   []interface{}{queryMap},
				},
			}
			mapObj["query"] = newQuery
		} else {
			return nil, errors.New("existing query field has invalid structure")
		}
	} else {
		// Create new query with just the filter
		mapObj["query"] = util.MapStr{
			"bool": util.MapStr{
				"filter": filterQ,
			},
		}
	}

	// Convert back to JSON with error handling
	queryDsl, err := json.Marshal(mapObj)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize modified query: %w", err)
	}

	return queryDsl, nil
}
