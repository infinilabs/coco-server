package milvus

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"

	"infini.sh/coco/core"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

// processDocuments transforms and collects documents, optionally tracking cursor position
func (s *scanner) processDocuments(ctx *pipeline.Context, docs []map[string]interface{}) (*cmn.CursorWatermark, error) {
	var lastCursor *cmn.CursorWatermark
	successCount := 0
	transformErrors := 0
	collectErrors := 0
	cursorErrors := 0

	for _, doc := range docs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Transform map to core.Document
		document, err := s.mapToDocument(doc)
		if err != nil {
			transformErrors++
			_ = log.Warnf("[%s] [%s] failed to transform document: %v", ConnectorName, s.datasource.Name, err)
			continue
		}

		// Collect document through pipeline
		if s.collectFunc != nil {
			if err := s.collectFunc(*document); err != nil {
				collectErrors++
				_ = log.Warnf("[%s] [%s] failed to collect document: %v", ConnectorName, s.datasource.Name, err)
				// Continue to try to extract cursor even if collection failed
			} else {
				successCount++
			}
		}

		// Extract cursor from last document if tracking enabled
		if s.config.IsIncrementalEnabled() {
			cursor, err := s.extractCursor(doc)
			if err != nil {
				cursorErrors++
				_ = log.Warnf("[%s] [%s] failed to extract cursor from document: %v", ConnectorName, s.datasource.Name, err)
			} else if cursor != nil {
				// Update lastCursor only if it's nil or the current cursor is "greater"
				if lastCursor == nil || cmn.CompareCursors(cursor, lastCursor, s.config.Incremental.PropertyType) > 0 {
					lastCursor = cursor
				}
			}
		}
	}

	// Check if incremental sync failed completely
	if s.config.IsIncrementalEnabled() && lastCursor == nil {
		if cursorErrors == len(docs) {
			return nil, fmt.Errorf("all %d documents missing incremental property %q - check schema configuration and ensure output_fields includes this field",
				len(docs), s.config.Incremental.Property)
		}
		if cursorErrors > 0 {
			_ = log.Warnf("[%s] [%s] %d/%d documents missing cursor fields - incremental sync may be incomplete",
				ConnectorName, s.datasource.Name, cursorErrors, len(docs))
		}
	}

	// Check document processing failure rate
	totalErrors := transformErrors + collectErrors
	if totalErrors > 0 {
		failureRate := float64(totalErrors) / float64(len(docs))
		if failureRate > 0.5 {
			return nil, fmt.Errorf("high document processing failure rate: %d/%d failed (%.1f%%) - "+
				"transform_errors=%d collect_errors=%d",
				totalErrors, len(docs), failureRate*100, transformErrors, collectErrors)
		}
		_ = log.Warnf("[%s] [%s] processed %d/%d documents successfully (%d transform errors, %d collect errors)",
			ConnectorName, s.datasource.Name, successCount, len(docs), transformErrors, collectErrors)
	}

	return lastCursor, nil
}

// columnsToRows converts Milvus columnar results to a slice of row-based maps
func (s *scanner) columnsToRows(res []entity.Column) ([]map[string]interface{}, error) {
	if len(res) == 0 {
		return nil, nil
	}

	// All columns should have the same number of rows
	numRows := res[0].Len()
	rows := make([]map[string]interface{}, numRows)

	for _, col := range res {
		fieldName := col.Name()
		for rowIdx := 0; rowIdx < numRows; rowIdx++ {
			if rows[rowIdx] == nil {
				rows[rowIdx] = make(map[string]interface{})
			}
			val, err := col.Get(rowIdx)
			if err != nil {
				return nil, fmt.Errorf("failed to get value from column %q at row %d: %w", fieldName, rowIdx, err)
			}
			rows[rowIdx][fieldName] = val
		}
	}
	return rows, nil
}

// extractCursor extracts cursor values from a Milvus document (map)
func (s *scanner) extractCursor(doc map[string]interface{}) (*cmn.CursorWatermark, error) {
	// Extract property value
	propertyValue, ok := doc[s.config.Incremental.Property]
	if !ok {
		return nil, fmt.Errorf("incremental property field %q not found in document", s.config.Incremental.Property)
	}

	// Extract tie-breaker value if configured
	var tieValue interface{}
	if s.config.Incremental.TieBreaker != "" {
		tieValue, ok = doc[s.config.Incremental.TieBreaker]
		if !ok {
			return nil, fmt.Errorf("tie-breaker field %q not found in document", s.config.Incremental.TieBreaker)
		}
	}

	// Use the common cursor serializer
	return s.cursorStateManager.Serializer.FromValue(propertyValue, tieValue)
}

// mapToDocument transforms a map[string]interface{} to a core.Document
func (s *scanner) mapToDocument(data map[string]interface{}) (*core.Document, error) {
	doc := &core.Document{
		Source: core.DataSourceReference{
			ID:   s.datasource.ID,
			Type: "connector",
			Name: s.datasource.Name,
		},
	}
	doc.System = s.datasource.System
	doc.Payload = data // Milvus fields directly become payload

	// If field mapping is enabled, apply transformations
	if s.config.FieldMapping.Enabled && s.config.FieldMapping.Mapping != nil {
		transformer := cmn.Transformer{Payload: data, Visited: make(map[string]bool)}
		transformer.Transform(doc, s.config.FieldMapping.Mapping)
	}

	// Ensure ID is set. If not mapped, use PK from Milvus
	if doc.ID == "" {
		if pkVal, ok := data[s.pkField]; ok {
			doc.ID = fmt.Sprintf("%v", pkVal)
		} else {
			return nil, fmt.Errorf("primary key field %q not found in Milvus document for ID generation", s.pkField)
		}
	}

	// Prepend datasource ID and hash if configured
	if s.config.FieldMapping.Enabled { // Only apply if field mapping is enabled for consistency with other connectors
		doc.ID = fmt.Sprintf("%s-%s", s.datasource.ID, doc.ID)
		if s.config.FieldMapping.IDHashable() {
			doc.ID = util.MD5digest(doc.ID)
		}
	}

	return doc, nil
}
