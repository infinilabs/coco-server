package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/cihub/seelog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"infini.sh/coco/modules/common"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

// scanner handles MongoDB collection scanning with incremental sync support
type scanner struct {
	config             *Config
	connector          *common.Connector
	datasource         *common.DataSource
	cursorStateManager *cmn.CursorStateManager
	collectFunc        func(common.Document) error
	client             *mongo.Client
	collection         *mongo.Collection
	sortSpec           bson.D // Query components (cached)
}

// Scan executes the MongoDB collection scan with pagination and incremental sync
func (s *scanner) Scan(ctx *pipeline.Context) error {
	// Initialize MongoDB connection
	if err := s.connect(ctx); err != nil {
		return err
	}
	defer s.disconnect(ctx)

	// Build sort specification
	if err := s.buildSort(); err != nil {
		return fmt.Errorf("failed to build sort specification: %w", err)
	}

	// Load cursor state for incremental sync
	var cursor *cmn.CursorWatermark
	var err error
	if s.config.Incremental.Enabled {
		cursor, err = s.cursorStateManager.LoadWithFallback(ctx, s.config.Incremental)
		if err != nil {
			_ = log.Errorf("[mongodb] failed to load cursor for datasource [%s]: %v", s.datasource.Name, err)
			return fmt.Errorf("failed to load cursor: %w", err)
		}
	}

	if cursor != nil {
		log.Infof("[mongodb] [%s] resuming from cursor: property=%v, tie=%v", s.datasource.Name, cursor.Property, cursor.Tie)
	} else {
		log.Infof("[mongodb] [%s] no cursor found, starting full scan", s.datasource.Name)
	}

	// Execute paginated scan
	return s.executePaginatedScan(ctx, cursor)
}

// connect establishes connection to MongoDB
func (s *scanner) connect(ctx context.Context) error {
	clientOptions := options.Client().ApplyURI(s.config.ConnectionURI)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection with ping
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	s.client = client
	s.collection = client.Database(s.config.Database).Collection(s.config.Collection)

	log.Infof("[mongodb] [%s] connected to MongoDB: %s/%s", s.datasource.Name, s.config.Database, s.config.Collection)
	return nil
}

// disconnect closes the MongoDB connection
func (s *scanner) disconnect(ctx context.Context) {
	if s.client != nil {
		if err := s.client.Disconnect(ctx); err != nil {
			_ = log.Warnf("[mongodb] [%s] error disconnecting: %v", s.datasource.Name, err)
		}
	}
}

// buildSort constructs the BSON sort specification
func (s *scanner) buildSort() error {
	if s.config.Sort != "" {
		// Parse user-provided sort
		var sortMap map[string]int
		if err := json.Unmarshal([]byte(s.config.Sort), &sortMap); err != nil {
			return fmt.Errorf("invalid sort JSON: %w", err)
		}

		s.sortSpec = bson.D{}
		for field, order := range sortMap {
			s.sortSpec = append(s.sortSpec, bson.E{Key: field, Value: order})
		}
		return nil
	}

	// Default sort for incremental sync
	if s.config.IsIncrementalEnabled() {
		s.sortSpec = bson.D{
			{Key: s.config.Incremental.Property, Value: 1},
		}
		if s.config.Incremental.TieBreaker != "" {
			s.sortSpec = append(s.sortSpec, bson.E{Key: s.config.Incremental.TieBreaker, Value: 1})
		}
	}

	return nil
}

// buildQuery constructs the MongoDB query filter with incremental conditions
func (s *scanner) buildQuery(baseCursor *cmn.CursorWatermark) (bson.M, error) {
	var filter bson.M

	// Parse user-provided query
	if s.config.Query != "" {
		if err := json.Unmarshal([]byte(s.config.Query), &filter); err != nil {
			return nil, fmt.Errorf("invalid query JSON: %w", err)
		}
	} else {
		filter = bson.M{}
	}

	// Add incremental filter
	if baseCursor != nil && s.config.IsIncrementalEnabled() {
		incrementalFilter := s.buildIncrementalFilter(baseCursor)

		// Combine with user query using $and
		if len(filter) > 0 {
			filter = bson.M{
				"$and": []bson.M{filter, incrementalFilter},
			}
		} else {
			filter = incrementalFilter
		}
	}

	return filter, nil
}

// buildIncrementalFilter creates the query filter for incremental sync
func (s *scanner) buildIncrementalFilter(cursor *cmn.CursorWatermark) bson.M {
	propertyField := s.config.Incremental.Property

	// Use raw type information if available for precise BSON conversion
	var propertyRawType string
	if cursor.Stored != nil {
		propertyRawType = cursor.Stored.Property.RawType
	}
	propertyValue := s.convertToBSONValueWithType(cursor.Property, propertyRawType)

	// If there's a tie-breaker, use $or to handle both cases
	if cursor.Tie != nil && s.config.Incremental.TieBreaker != "" {
		var tieRawType string
		if cursor.Stored != nil && cursor.Stored.Tie != nil {
			tieRawType = cursor.Stored.Tie.RawType
		}
		tieValue := s.convertToBSONValueWithType(cursor.Tie, tieRawType)

		return bson.M{
			"$or": []bson.M{
				// Property is greater than cursor
				{propertyField: bson.M{"$gt": propertyValue}},
				// Property equals cursor AND tie-breaker is greater
				{
					propertyField:                   propertyValue,
					s.config.Incremental.TieBreaker: bson.M{"$gt": tieValue},
				},
			},
		}
	}

	// Simple case: just property value comparison
	return bson.M{
		propertyField: bson.M{"$gt": propertyValue},
	}
}

// convertToBSONValue converts Go types to proper BSON types for queries
// Uses cursor's RawType information when available for accurate type conversion
func (s *scanner) convertToBSONValue(value interface{}) interface{} {
	switch v := value.(type) {
	case time.Time:
		return primitive.NewDateTimeFromTime(v)
	case *time.Time:
		if v != nil {
			return primitive.NewDateTimeFromTime(*v)
		}
		return nil
	case string:
		// Try to parse as ObjectID hex string (24 character hex)
		if len(v) == 24 {
			if oid, err := primitive.ObjectIDFromHex(v); err == nil {
				return oid
			}
		}
		return value
	default:
		return value
	}
}

// convertToBSONValueWithType converts a value to BSON type using raw type hint
func (s *scanner) convertToBSONValueWithType(value interface{}, rawType string) interface{} {
	// If we have raw type information, use it for precise conversion
	if rawType != "" {
		switch rawType {
		case "ObjectId": // MongoDB uses ObjectId() not ObjectID()
			if str, ok := value.(string); ok && len(str) == 24 {
				if oid, err := primitive.ObjectIDFromHex(str); err == nil {
					return oid
				}
			}
		case "DateTime":
			if t, ok := value.(time.Time); ok {
				return primitive.NewDateTimeFromTime(t)
			}
		}
	}

	// Fall back to standard conversion
	return s.convertToBSONValue(value)
}

// executePaginatedScan performs paginated scanning with optional incremental sync
// Supports two modes:
// - Incremental: cursor-based pagination with dynamic filter updates (skip=0, filter changes)
// - Full-sync: offset-based pagination (skip increases, filter constant)
func (s *scanner) executePaginatedScan(ctx *pipeline.Context, baseCursor *cmn.CursorWatermark) error {
	pageSize := int64(s.config.GetPageSize())
	pageNum := 0
	totalDocs := 0

	// Incremental mode: track current cursor position
	// Full-sync mode: currentCursor stays nil
	currentCursor := baseCursor
	isIncremental := s.config.IsIncrementalEnabled()

	scanMode := "full-sync"
	if isIncremental {
		scanMode = "incremental"
	}

	if global.Env().IsDebug {
		// Log initial scan info
		sortJSON, _ := json.Marshal(s.sortSpec)
		log.Debugf("[mongodb] [%s] starting %s scan - database: %s, collection: %s, sort: %s, page_size: %d",
			s.datasource.Name, scanMode, s.config.Database, s.config.Collection, string(sortJSON), s.config.GetPageSize())
	}

	// Build initial filter (for full-sync, this is the only filter we need)
	filter, err := s.buildQuery(currentCursor)
	if err != nil {
		return err
	}

	// Log initial filter for full-sync mode
	if global.Env().IsDebug && !isIncremental {
		filterJSON, _ := json.Marshal(filter)
		log.Infof("[mongodb] [%s] full-sync scan filter: %s", s.datasource.Name, string(filterJSON))
	}

	// Pagination loop
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

		// Calculate skip offset
		// Incremental: skip=0 (filter handles positioning)
		// full-sync: skip increases each iteration
		var skip int64
		if isIncremental {
			skip = 0
			// Rebuild filter with current cursor for incremental mode
			filter, err = s.buildQuery(currentCursor)
			if err != nil {
				return err
			}

			if global.Env().IsDebug {
				// Log query details for each incremental page
				filterJSON, _ := json.Marshal(filter)
				log.Debugf("[mongodb] [%s] executing incremental page %d - filter: %s, limit: %d",
					s.datasource.Name, pageNum, string(filterJSON), pageSize)
			}
		} else {
			skip = int64(pageNum-1) * pageSize
			log.Debugf("[mongodb] [%s] executing full-sync page %d - skip: %d, limit: %d",
				s.datasource.Name, pageNum, skip, pageSize)
		}

		// Execute query
		docs, err := s.executePage(ctx, filter, skip, pageSize)
		if err != nil {
			return fmt.Errorf("failed to execute page %d: %w", pageNum, err)
		}

		// Check if we've reached the end
		if len(docs) == 0 {
			log.Infof("[mongodb] [%s] %s scan complete - no more documents", s.datasource.Name, scanMode)
			break
		}

		totalDocs += len(docs)
		log.Infof("[mongodb] [%s] processing page %d: %d documents", s.datasource.Name, pageNum, len(docs))

		// Process documents
		// Incremental mode: track cursor from last document
		// Full-sync mode: no cursor tracking
		lastCursor, err := s.processDocuments(ctx, docs, isIncremental)
		if err != nil {
			return err
		}

		// Save cursor for incremental mode
		if isIncremental && lastCursor != nil {
			currentCursor = lastCursor
			if err := s.cursorStateManager.Save(ctx, s.config.Incremental.Property, currentCursor); err != nil {
				_ = log.Warnf("[mongodb] [%s] failed to save cursor state: %v", s.datasource.Name, err)
			}
		}

		// If we got fewer documents than page size, we've reached the end
		if len(docs) < int(pageSize) {
			log.Infof("[mongodb] [%s] %s scan complete - last page processed", s.datasource.Name, scanMode)
			break
		}
	}

	log.Infof("[mongodb] [%s] %s scan completed: %d pages, %d documents processed", s.datasource.Name, scanMode, pageNum, totalDocs)
	return nil
}

// processDocuments transforms and collects documents, optionally tracking cursor position
func (s *scanner) processDocuments(ctx *pipeline.Context, docs []bson.M, trackCursor bool) (*cmn.CursorWatermark, error) {
	var lastCursor *cmn.CursorWatermark

	for _, doc := range docs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Transform BSON to Document
		document, err := s.bsonToDocument(doc)
		if err != nil {
			_ = log.Warnf("[mongodb] [%s] failed to transform document: %v", s.datasource.Name, err)
			continue
		}

		// Collect document through pipeline
		if s.collectFunc != nil {
			if err := s.collectFunc(*document); err != nil {
				_ = log.Warnf("[mongodb] [%s] failed to collect document: %v", s.datasource.Name, err)
			}
		}

		// Extract cursor from last document if tracking enabled
		if trackCursor {
			cursor, err := s.extractCursor(doc)
			if err != nil {
				_ = log.Warnf("[mongodb] [%s] failed to extract cursor: %v", s.datasource.Name, err)
			} else {
				lastCursor = cursor
			}
		}
	}

	return lastCursor, nil
}

// executePage executes a single page query
func (s *scanner) executePage(ctx context.Context, filter bson.M, skip, limit int64) ([]bson.M, error) {
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit)

	if len(s.sortSpec) > 0 {
		opts.SetSort(s.sortSpec)
	}

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// bsonToDocument transforms a BSON document to a common.Document
func (s *scanner) bsonToDocument(bsonDoc bson.M) (*common.Document, error) {
	cfg := s.config
	doc, err := bsonToDocument(bsonDoc, cfg, s.datasource)
	if err != nil {
		return nil, err
	}

	if cfg.FieldMapping.Enabled && doc.ID != "" {
		doc.ID = fmt.Sprintf("%s-%s", s.datasource.ID, doc.ID)
		if cfg.FieldMapping.IDHashable() {
			doc.ID = util.MD5digest(doc.ID)
		}
	}
	return doc, nil
}
