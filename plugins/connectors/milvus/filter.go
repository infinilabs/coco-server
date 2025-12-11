package milvus

import (
	"fmt"
	"strings"
	"time"

	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/errors"
)

// buildQueryExpression constructs the Milvus query expression with incremental conditions
func (s *scanner) buildQueryExpression(baseCursor *cmn.CursorWatermark) (string, error) {
	queryExpr := s.config.Filter

	// Add incremental filter
	if baseCursor != nil && s.config.IsIncrementalEnabled() {
		incrementalExpr, err := s.buildIncrementalFilter(baseCursor)
		if err != nil {
			return "", err
		}

		if queryExpr != "" {
			queryExpr = fmt.Sprintf("(%s) and (%s)", queryExpr, incrementalExpr)
		} else {
			queryExpr = incrementalExpr
		}
	}

	return queryExpr, nil
}

// escapeStringForFilter escapes a string value for use in Milvus filter expressions
// to prevent injection attacks while maintaining compatibility with Milvus query syntax
func escapeStringForFilter(s string) (string, error) {
	// Validate: reject strings containing characters that could break filter syntax
	if strings.ContainsAny(s, "\"\\") {
		// Escape backslashes and quotes
		s = strings.ReplaceAll(s, "\\", "\\\\")
		s = strings.ReplaceAll(s, "\"", "\\\"")
	}

	// Additional safety: check for potentially malicious patterns
	if strings.Contains(s, ") or (") || strings.Contains(s, ") and (") {
		return "", fmt.Errorf("string value contains suspicious filter expression patterns: %q", s)
	}

	return fmt.Sprintf("\"%s\"", s), nil
}

// buildIncrementalFilter creates the Milvus query expression for incremental sync
func (s *scanner) buildIncrementalFilter(cursor *cmn.CursorWatermark) (string, error) {
	propertyField := s.config.Incremental.Property
	if propertyField == "" {
		return "", errors.New("incremental property field is not set")
	}

	if cursor.Property == nil {
		return "", errors.New("cursor property value is nil")
	}

	var propertyValStr string
	// Handle different property types for the query expression
	switch s.config.Incremental.PropertyType {
	case "int":
		val, ok := cursor.Property.(int64)
		if !ok {
			return "", fmt.Errorf("invalid int cursor property type: %T, expected int64", cursor.Property)
		}
		propertyValStr = fmt.Sprintf("%d", val)
	case "float":
		val, ok := cursor.Property.(float64)
		if !ok {
			return "", fmt.Errorf("invalid float cursor property type: %T, expected float64", cursor.Property)
		}
		propertyValStr = fmt.Sprintf("%f", val)
	case "datetime":
		val, ok := cursor.Property.(time.Time)
		if !ok {
			return "", fmt.Errorf("invalid datetime cursor property type: %T, expected time.Time", cursor.Property)
		}
		// Milvus stores datetime values as milliseconds since epoch.
		propertyValStr = fmt.Sprintf("%d", val.UnixMilli())
	case "string":
		val, ok := cursor.Property.(string)
		if !ok {
			return "", fmt.Errorf("invalid string cursor property type: %T, expected string", cursor.Property)
		}
		// Escape string to prevent filter injection
		escaped, err := escapeStringForFilter(val)
		if err != nil {
			return "", fmt.Errorf("invalid string value in cursor property: %w", err)
		}
		propertyValStr = escaped
	default:
		// Default to string representation, but may not be valid for all types directly
		propertyValStr = fmt.Sprintf("%v", cursor.Property)
	}

	// If there's a tie-breaker, use OR to handle both cases
	if cursor.Tie != nil && s.config.Incremental.TieBreaker != "" {
		tieField := s.config.Incremental.TieBreaker
		var tieValStr string
		// Assuming tie-breaker is string or int for simplicity, adapt if needed
		switch v := cursor.Tie.(type) {
		case int64:
			tieValStr = fmt.Sprintf("%d", v)
		case string:
			// Escape string tie-breaker to prevent injection
			escaped, err := escapeStringForFilter(v)
			if err != nil {
				return "", fmt.Errorf("invalid string value in cursor tie-breaker: %w", err)
			}
			tieValStr = escaped
		default:
			tieValStr = fmt.Sprintf("%v", v)
		}

		return fmt.Sprintf("(%s > %s) or ((%s == %s) and (%s > %s))",
			propertyField, propertyValStr,
			propertyField, propertyValStr,
			tieField, tieValStr), nil
	}

	// Simple case: just property value comparison
	return fmt.Sprintf("%s > %s", propertyField, propertyValStr), nil
}
