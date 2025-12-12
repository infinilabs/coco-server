package milvus

import (
	"context"
	"fmt"
	"time"

	log "github.com/cihub/seelog"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"

	"infini.sh/coco/core"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
)

// scanner handles Milvus collection scanning with incremental sync support
type scanner struct {
	config             *Config
	connector          *core.Connector
	datasource         *core.DataSource
	cursorStateManager *cmn.CursorStateManager
	collectFunc        func(core.Document) error
	milvusClient       client.Client
	collectionInfo     *entity.Collection
	pkField            string // Primary key field name
}

// Scan executes the Milvus collection scan with pagination and incremental sync
func (s *scanner) Scan(ctx *pipeline.Context) error {
	// Initialize Milvus connection
	if err := s.connect(ctx); err != nil {
		return err
	}
	defer s.disconnect()

	// Describe collection to get schema and primary key field
	if err := s.describeCollection(ctx); err != nil {
		return err
	}

	collectionLoaded, loadErr := s.loadCollection(ctx)
	if loadErr != nil {
		return loadErr
	}
	defer func() {
		if collectionLoaded {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.releaseCollection(releaseCtx); err != nil {
				_ = log.Warnf("[%s] [%s] failed to release collection %q: %v", ConnectorName, s.datasource.Name, s.config.Collection, err)
			}
		}
	}()

	// Load cursor state for incremental sync
	var cursor *cmn.CursorWatermark
	var err error
	if s.config.Incremental.Enabled {
		cursor, err = s.cursorStateManager.LoadWithFallback(ctx, s.config.Incremental)
		if err != nil {
			_ = log.Errorf("[%s] failed to load cursor for datasource [%s]: %v", ConnectorName, s.datasource.Name, err)
			return fmt.Errorf("failed to load cursor: %w", err)
		}

		// Warn if tie-breaker is not configured
		if s.config.Incremental.TieBreaker == "" {
			_ = log.Warnf("[%s] [%s] incremental sync enabled without tie_breaker - cursor may not advance correctly with duplicate timestamps",
				ConnectorName, s.datasource.Name)
		}
	}

	if cursor != nil {
		log.Infof("[%s] [%s] resuming from cursor: property=%v, tie=%v", ConnectorName, s.datasource.Name, cursor.Property, cursor.Tie)
	} else {
		log.Infof("[%s] [%s] no cursor found, starting full scan", ConnectorName, s.datasource.Name)
	}

	// Execute paginated scan
	return s.executePaginatedScan(ctx, cursor)
}

// connect establishes connection to Milvus
func (s *scanner) connect(ctx context.Context) error {
	cfg := client.Config{
		Address:  s.config.Address,
		Username: s.config.Username,
		Password: s.config.Password,
		DBName:   s.config.DBName,
	}

	c, err := client.NewClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to Milvus at %s: %w", s.config.Address, err)
	}
	s.milvusClient = c

	log.Infof("[%s] [%s] connected to Milvus: %s", ConnectorName, s.datasource.Name, s.config.Address)
	return nil
}

// disconnect closes the Milvus connection
func (s *scanner) disconnect() {
	if s.milvusClient != nil {
		_ = s.milvusClient.Close()
	}
}

// loadCollection ensures the Milvus collection is loaded before querying.
// returns true if this call initiated a load, false if already loaded.
func (s *scanner) loadCollection(ctx context.Context) (bool, error) {
	if progress, err := s.milvusClient.GetLoadingProgress(ctx, s.config.Collection, nil); err == nil && progress == 100 {
		log.Infof("[%s] [%s] collection %q already loaded", ConnectorName, s.datasource.Name, s.config.Collection)
		return false, nil
	}
	if err := s.milvusClient.LoadCollection(ctx, s.config.Collection, false); err != nil {
		return false, fmt.Errorf("failed to load collection %q: %w", s.config.Collection, err)
	}
	log.Infof("[%s] [%s] collection %q loaded", ConnectorName, s.datasource.Name, s.config.Collection)
	return true, nil
}

func (s *scanner) releaseCollection(ctx context.Context) error {
	if err := s.milvusClient.ReleaseCollection(ctx, s.config.Collection); err != nil {
		return err
	}
	log.Infof("[%s] [%s] collection %q released", ConnectorName, s.datasource.Name, s.config.Collection)
	return nil
}

// describeCollection fetches the schema of the target collection
func (s *scanner) describeCollection(ctx context.Context) error {
	coll, err := s.milvusClient.DescribeCollection(ctx, s.config.Collection)
	if err != nil {
		return fmt.Errorf("failed to describe collection %q: %w", s.config.Collection, err)
	}
	s.collectionInfo = coll

	// Identify the primary key field
	for _, fieldSchema := range coll.Schema.Fields {
		if fieldSchema.PrimaryKey {
			s.pkField = fieldSchema.Name
			break
		}
	}
	if s.pkField == "" {
		return fmt.Errorf("collection %q does not have a primary key field", s.config.Collection)
	}

	// Map of schema field name to *entity.Field for O(1) lookup
	schemaFields := make(map[string]*entity.Field, len(coll.Schema.Fields))
	for _, f := range coll.Schema.Fields {
		schemaFields[f.Name] = f
	}

	fieldSet := make(map[string]struct{})

	// If output_fields are not specified, use all scalar fields + primary key + incremental columns
	if len(s.config.OutputFields) == 0 {
		for _, fieldSchema := range coll.Schema.Fields {
			if fieldSchema.PrimaryKey {
				continue
			}
			if fieldSchema.DataType == entity.FieldTypeFloatVector || fieldSchema.DataType == entity.FieldTypeBinaryVector {
				continue
			}
			fieldSet[fieldSchema.Name] = struct{}{}
			s.config.OutputFields = append(s.config.OutputFields, fieldSchema.Name)
		}
		log.Infof("[%s] [%s] using auto-detected output fields before required columns: %v", ConnectorName, s.datasource.Name, s.config.OutputFields)
	} else {
		for _, outputField := range s.config.OutputFields {
			if _, exists := schemaFields[outputField]; !exists {
				return fmt.Errorf("configured output_field %q not found in collection %q schema", outputField, s.config.Collection)
			}
			fieldSet[outputField] = struct{}{}
		}
	}

	if err := s.ensureRequiredOutputFields(schemaFields, fieldSet); err != nil {
		return err
	}

	log.Infof("[%s] [%s] collection %q described, primary key: %q, output fields: %v", ConnectorName, s.datasource.Name, s.config.Collection, s.pkField, s.config.OutputFields)
	return nil
}

// ensureRequiredOutputFields guarantees that primary key, incremental property, and tie-breaker fields are part of output_fields.
func (s *scanner) ensureRequiredOutputFields(schemaFields map[string]*entity.Field, fieldSet map[string]struct{}) error {
	required := []string{s.pkField}
	if s.config.IsIncrementalEnabled() {
		required = append(required, s.config.Incremental.Property)
		if tie := s.config.Incremental.TieBreaker; tie != "" {
			required = append(required, tie)

			// Validate tie-breaker field characteristics
			if tieField, exists := schemaFields[tie]; exists {
				if !tieField.PrimaryKey && !tieField.AutoID {
					_ = log.Warnf("[%s] [%s] tie_breaker field %q is not primary key or auto-ID - "+
						"ensure it provides uniqueness to prevent cursor stagnation. "+
						"Consider using a field with high cardinality or the primary key as tie_breaker.",
						ConnectorName, s.datasource.Name, tie)
				}
			}
		} else {
			// No tie-breaker configured - warn about potential issues
			_ = log.Warnf("[%s] [%s] incremental sync configured without tie_breaker - "+
				"cursor may stagnate if multiple documents share the same %q value. "+
				"Configure tie_breaker to primary key or unique field for reliable incremental sync.",
				ConnectorName, s.datasource.Name, s.config.Incremental.Property)
		}
	}

	for _, fieldName := range required {
		if fieldName == "" {
			continue
		}
		if _, exists := schemaFields[fieldName]; !exists {
			return fmt.Errorf("required field %q not found in collection %q schema", fieldName, s.config.Collection)
		}
		if _, exists := fieldSet[fieldName]; !exists {
			fieldSet[fieldName] = struct{}{}
			s.config.OutputFields = append(s.config.OutputFields, fieldName)
		}
	}
	return nil
}

// executePaginatedScan performs paginated scanning with optional incremental sync
func (s *scanner) executePaginatedScan(ctx *pipeline.Context, baseCursor *cmn.CursorWatermark) error {
	if s.config.IsIncrementalEnabled() {
		return s.incrementalScan(ctx, baseCursor)
	}
	return s.fullScan(ctx)
}

// fullScan performs a traditional paginated scan without incremental semantics.
func (s *scanner) fullScan(ctx *pipeline.Context) error {
	pageSize := int64(s.config.GetPageSize())
	offset := int64(0)
	pageNum := 0
	totalDocs := 0
	baseCursor := (*cmn.CursorWatermark)(nil)

	consistencyLevel := entity.ClStrong

	if global.Env().IsDebug {
		log.Debugf("[%s] [%s] starting full-sync scan - collection: %s, page_size: %d",
			ConnectorName, s.datasource.Name, s.config.Collection, s.config.GetPageSize())
	}

	for {
		if global.ShuttingDown() {
			return errors.New("shutting down")
		}

		select {
		case <-ctx.Done():
			return errors.New("context deadline exceeded")
		default:
		}

		pageNum++

		queryExpr, err := s.buildQueryExpression(baseCursor)
		if err != nil {
			return fmt.Errorf("failed to build query expression: %w", err)
		}

		log.Debugf("[%s] [%s] executing full page %d - filter: %q, output_fields: %v, offset: %d, limit: %d",
			ConnectorName, s.datasource.Name, pageNum, queryExpr, s.config.OutputFields, offset, pageSize)

		queryOptions := []client.SearchQueryOptionFunc{
			client.WithLimit(pageSize),
			client.WithSearchQueryConsistencyLevel(consistencyLevel),
		}
		queryOptions = append(queryOptions, client.WithOffset(offset))

		res, err := s.milvusClient.Query(ctx,
			s.config.Collection,
			nil,
			queryExpr,
			s.config.OutputFields,
			queryOptions...,
		)
		if err != nil {
			return fmt.Errorf("failed to execute Milvus query for page %d: %w", pageNum, err)
		}

		rows, err := s.columnsToRows(res)
		if err != nil {
			return fmt.Errorf("failed to convert Milvus columns to rows: %w", err)
		}

		if len(rows) == 0 {
			log.Infof("[%s] [%s] full-sync scan complete - no more documents", ConnectorName, s.datasource.Name)
			break
		}

		totalDocs += len(rows)
		log.Infof("[%s] [%s] processing full page %d: %d documents", ConnectorName, s.datasource.Name, pageNum, len(rows))

		if _, err := s.processDocuments(ctx, rows); err != nil {
			return err
		}

		offset += int64(len(rows))
		if len(rows) < int(pageSize) {
			log.Infof("[%s] [%s] full-sync scan complete - last page processed", ConnectorName, s.datasource.Name)
			break
		}
	}

	log.Infof("[%s] [%s] full-sync scan completed: %d pages, %d documents processed", ConnectorName, s.datasource.Name, pageNum, totalDocs)
	return nil
}

// incrementalScan performs cursor-based pagination with dynamic filter updates.
// CRITICAL: Unlike full-sync mode, incremental mode:
// 1. Rebuilds the filter expression on each iteration using the current cursor
// 2. Always uses offset=0 (cursor handles positioning via filter)
// 3. Updates currentCursor after each batch to advance the watermark
//
// This pattern prevents data loss that can occur with static filters + offset pagination.
// See MongoDB (scanner.go:295-305) and Neo4j (scanner.go:163-186) connectors for reference.
func (s *scanner) incrementalScan(ctx *pipeline.Context, baseCursor *cmn.CursorWatermark) error {
	pageSize := int64(s.config.GetPageSize())
	pageNum := 0
	totalDocs := 0
	var latestCursor *cmn.CursorWatermark
	cursorSaved := false
	currentCursor := baseCursor // Track current cursor position

	consistencyLevel := entity.ClStrong

	log.Infof("[%s] [%s] starting incremental scan - collection: %s, page_size: %d",
		ConnectorName, s.datasource.Name, s.config.Collection, pageSize)

	for {
		if global.ShuttingDown() {
			return errors.New("shutting down")
		}

		select {
		case <-ctx.Done():
			return errors.New("context deadline exceeded")
		default:
		}

		pageNum++

		// Rebuild filter with current cursor position (cursor-based pagination)
		queryExpr, err := s.buildQueryExpression(currentCursor)
		if err != nil {
			return fmt.Errorf("failed to build incremental query expression: %w", err)
		}

		// Debug logging for cursor progression
		if global.Env().IsDebug && currentCursor != nil {
			log.Debugf("[%s] [%s] incremental page %d - cursor: property=%v, tie=%v, filter: %q",
				ConnectorName, s.datasource.Name, pageNum, currentCursor.Property, currentCursor.Tie, queryExpr)
		}

		// NOTE: Milvus Query API does not support ORDER BY as of v2.4.x
		// Relying on tie-breaker for stable ordering when duplicate property values exist
		queryOptions := []client.SearchQueryOptionFunc{
			client.WithOffset(0), // Always 0 - cursor handles positioning
			client.WithLimit(pageSize),
			client.WithSearchQueryConsistencyLevel(consistencyLevel),
		}

		res, err := s.milvusClient.Query(ctx,
			s.config.Collection,
			nil,
			queryExpr,
			s.config.OutputFields,
			queryOptions...,
		)
		if err != nil {
			return fmt.Errorf("failed to execute incremental query for page %d: %w", pageNum, err)
		}

		rows, err := s.columnsToRows(res)
		if err != nil {
			return fmt.Errorf("failed to convert incremental columns to rows: %w", err)
		}

		if len(rows) == 0 {
			log.Infof("[%s] [%s] incremental scan complete - no more documents", ConnectorName, s.datasource.Name)
			break
		}

		totalDocs += len(rows)
		log.Infof("[%s] [%s] processing incremental page %d: %d documents", ConnectorName, s.datasource.Name, pageNum, len(rows))

		lastCursor, err := s.processDocuments(ctx, rows)
		if err != nil {
			return err
		}
		if lastCursor == nil {
			return fmt.Errorf("incremental sync enabled but failed to extract cursor values (property=%q tie=%q)",
				s.config.Incremental.Property, s.config.Incremental.TieBreaker)
		}

		// Detect cursor stagnation (infinite loop protection)
		if pageNum > 1 && currentCursor != nil {
			diff := cmn.CompareCursors(lastCursor, currentCursor, s.config.Incremental.PropertyType)
			if diff == 0 {
				_ = log.Errorf("[%s] [%s] cursor stagnation detected on page %d - stopping scan to prevent infinite loop. "+
					"This indicates duplicate property values without valid tie_breaker, or Milvus result ordering instability. "+
					"Configure a unique tie_breaker field to resolve this issue.",
					ConnectorName, s.datasource.Name, pageNum)
				return fmt.Errorf("cursor stagnation detected after page %d: incremental scan cannot proceed without unique tie_breaker - "+
					"property=%q value=%v did not advance", pageNum, s.config.Incremental.Property, lastCursor.Property)
			}
		}

		// Update current cursor for next iteration
		currentCursor = lastCursor

		if latestCursor == nil || cmn.CompareCursors(lastCursor, latestCursor, s.config.Incremental.PropertyType) > 0 {
			latestCursor = lastCursor
			if err := s.cursorStateManager.Save(ctx, s.config.Incremental.Property, latestCursor); err != nil {
				return fmt.Errorf("failed to persist incremental cursor: %w", err)
			}
			cursorSaved = true
		}

		if len(rows) < int(pageSize) {
			log.Infof("[%s] [%s] incremental scan complete - last page processed", ConnectorName, s.datasource.Name)
			break
		}
	}

	if latestCursor != nil && !cursorSaved {
		if err := s.cursorStateManager.Save(ctx, s.config.Incremental.Property, latestCursor); err != nil {
			return fmt.Errorf("failed to persist incremental cursor: %w", err)
		}
	}

	log.Infof("[%s] [%s] incremental scan completed: %d pages, %d documents processed", ConnectorName, s.datasource.Name, pageNum, totalDocs)
	return nil
}
