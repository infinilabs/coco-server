/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/global"
)

func (p *Plugin) scanCollectionWithContext(ctx context.Context, client *mongo.Client, config *Config, collConfig CollectionConfig, datasource *common.DataSource) {
	select {
	case <-ctx.Done():
		log.Debugf("[mongodb connector] context cancelled, stopping scan for collection [%s]", collConfig.Name)
		return
	default:
	}

	if global.ShuttingDown() {
		return
	}

	log.Infof("[mongodb connector] starting scan for collection [%s] in datasource [%s]", collConfig.Name, datasource.Name)

	collection := client.Database(config.Database).Collection(collConfig.Name)

	// Get collection stats for monitoring
	if stats, err := p.getCollectionStats(client, config.Database, collConfig.Name); err == nil {
		log.Debugf("[mongodb connector] collection [%s] stats: %v", collConfig.Name, stats)
	}

	// Build query filter
	filter := p.buildFilter(config, collConfig)

	// Set query options
	findOptions := options.Find()

	// Use page size if pagination is enabled, otherwise use batch size
	if config.Pagination {
		findOptions.SetBatchSize(int32(config.PageSize))
	} else {
		findOptions.SetBatchSize(int32(config.BatchSize))
	}

	// Set projection if fields are specified in collection config and projection is enabled
	// This enables projection pushdown for better performance
	if config.EnableProjection && (collConfig.TitleField != "" || collConfig.ContentField != "" ||
		collConfig.CategoryField != "" || collConfig.TagsField != "" ||
		collConfig.URLField != "" || collConfig.TimestampField != "") {
		projection := bson.D{}

		// Always include _id field for document identification
		projection = append(projection, bson.E{Key: "_id", Value: 1})

		// Add configured fields to projection
		if collConfig.TitleField != "" {
			projection = append(projection, bson.E{Key: collConfig.TitleField, Value: 1})
		}
		if collConfig.ContentField != "" {
			projection = append(projection, bson.E{Key: collConfig.ContentField, Value: 1})
		}
		if collConfig.CategoryField != "" {
			projection = append(projection, bson.E{Key: collConfig.CategoryField, Value: 1})
		}
		if collConfig.TagsField != "" {
			projection = append(projection, bson.E{Key: collConfig.TagsField, Value: 1})
		}
		if collConfig.URLField != "" {
			projection = append(projection, bson.E{Key: collConfig.URLField, Value: 1})
		}
		if collConfig.TimestampField != "" {
			projection = append(projection, bson.E{Key: collConfig.TimestampField, Value: 1})
		}

		// Add any additional fields specified in the filter for proper filtering
		for field := range collConfig.Filter {
			projection = append(projection, bson.E{Key: field, Value: 1})
		}

		findOptions.SetProjection(projection)
	}

	// Optimize query
	p.optimizeQuery(findOptions, collConfig, config)

	// Paginated processing for large datasets
	var skip int64 = 0
	for {
		select {
		case <-ctx.Done():
			log.Debugf("[mongodb connector] context cancelled during scan for collection [%s]", collConfig.Name)
			return
		default:
		}

		if global.ShuttingDown() {
			return
		}

		findOptions.SetSkip(skip)
		findOptions.SetLimit(int64(config.BatchSize))

		cursor, err := collection.Find(ctx, filter, findOptions)
		if err != nil {
			log.Errorf("[mongodb connector] query failed for collection [%s]: %v", collConfig.Name, err)
			return
		}

		documents := p.processCursor(cursor, collConfig, datasource)
		cursor.Close(ctx)

		if len(documents) == 0 {
			break
		}

		// Batch push to queue
		p.pushDocuments(documents)

		skip += int64(len(documents))

		// Memory management
		if skip%10000 == 0 {
			runtime.GC()
		}

		// Update last sync time for incremental sync
		if config.SyncStrategy == "incremental" && config.LastModifiedField != "" {
			// Get the latest timestamp from the current batch
			latestTime := p.getLatestTimestampFromBatch(documents, config.LastModifiedField)
			if !latestTime.IsZero() {
				if err := p.updateLastSyncTime(config, collConfig.Name, latestTime); err != nil {
					log.Warnf("[mongodb connector] failed to update last sync time: %v", err)
				}
			}
		}
	}

	log.Infof("[mongodb connector] finished scanning collection [%s] in datasource [%s]", collConfig.Name, datasource.Name)
}

func (p *Plugin) buildFilter(config *Config, collConfig CollectionConfig) bson.M {
	filter := bson.M{}

	// Copy base filter from collection configuration
	for k, v := range collConfig.Filter {
		filter[k] = v
	}

	// Add timestamp filter for incremental sync
	if config.SyncStrategy == "incremental" && config.LastModifiedField != "" {
		// Check if we have a last sync time stored for this datasource
		// In a real implementation, this would be retrieved from persistent storage
		lastSyncTime := p.getLastSyncTime(config, collConfig.Name)
		if !lastSyncTime.IsZero() {
			filter[config.LastModifiedField] = bson.M{"$gt": lastSyncTime}
		}
	}

	return filter
}

func (p *Plugin) optimizeQuery(findOptions *options.FindOptions, collConfig CollectionConfig, config *Config) {
	// Set read concern level
	findOptions.SetReadConcern(readconcern.Local())

	// If there's a timestamp field and index hints are enabled, suggest using related index
	if config.EnableIndexHint && collConfig.TimestampField != "" {
		findOptions.SetHint(bson.D{{Key: collConfig.TimestampField, Value: 1}})
	}
}

func (p *Plugin) getCollectionStats(client *mongo.Client, database, collection string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := client.Database(database)
	coll := db.Collection(collection)

	// Get collection stats
	var result bson.M
	err := db.RunCommand(ctx, bson.D{
		{Key: "collStats", Value: collection},
	}).Decode(&result)

	if err != nil {
		return nil, err
	}

	// Get document count
	count, err := coll.CountDocuments(ctx, bson.D{})
	if err != nil {
		log.Warnf("[mongodb connector] failed to get document count: %v", err)
	} else {
		result["documentCount"] = count
	}

	return result, nil
}

// getLatestTimestampFromBatch finds the latest timestamp from a batch of documents
func (p *Plugin) getLatestTimestampFromBatch(documents []*common.Document, timestampField string) time.Time {
	var latestTime time.Time

	for _, doc := range documents {
		if doc.Updated != nil && !doc.Updated.IsZero() {
			if latestTime.IsZero() || doc.Updated.After(latestTime) {
				latestTime = *doc.Updated
			}
		}
	}

	return latestTime
}

// getLastSyncTime retrieves the last sync time for a specific collection
// Uses file-based storage for persistence across restarts
func (p *Plugin) getLastSyncTime(config *Config, collectionName string) time.Time {
	// Create a unique key for this datasource and collection
	syncKey := fmt.Sprintf("%s_%s_%s", config.ConnectionURI, config.Database, collectionName)

	// Get the sync time from persistent storage
	syncTime, err := p.getSyncTimeFromStorage(syncKey)
	if err != nil {
		log.Warnf("[mongodb connector] failed to get last sync time for %s: %v", syncKey, err)
		return time.Time{} // Return zero time on error
	}

	return syncTime
}

// getSyncTimeFromStorage retrieves the last sync time from file storage
func (p *Plugin) getSyncTimeFromStorage(syncKey string) (time.Time, error) {
	// Create sync storage directory if it doesn't exist
	syncDir := p.getSyncStorageDir()
	if err := os.MkdirAll(syncDir, 0755); err != nil {
		return time.Time{}, fmt.Errorf("failed to create sync storage directory: %v", err)
	}

	// Create filename from sync key (sanitize for filesystem)
	filename := p.sanitizeFilename(syncKey) + ".json"
	filepath := filepath.Join(syncDir, filename)

	// Read the sync time file
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return zero time (no previous sync)
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to read sync time file: %v", err)
	}

	// Parse the JSON data
	var syncData struct {
		LastSyncTime time.Time `json:"last_sync_time"`
		UpdatedAt    time.Time `json:"updated_at"`
	}

	if err := json.Unmarshal(data, &syncData); err != nil {
		return time.Time{}, fmt.Errorf("failed to parse sync time data: %v", err)
	}

	return syncData.LastSyncTime, nil
}

// updateLastSyncTime updates the last sync time for a specific collection
func (p *Plugin) updateLastSyncTime(config *Config, collectionName string, syncTime time.Time) error {
	// Create a unique key for this datasource and collection
	syncKey := fmt.Sprintf("%s_%s_%s", config.ConnectionURI, config.Database, collectionName)

	// Update the sync time in persistent storage
	return p.updateSyncTimeInStorage(syncKey, syncTime)
}

// updateSyncTimeInStorage saves the last sync time to file storage
func (p *Plugin) updateSyncTimeInStorage(syncKey string, syncTime time.Time) error {
	// Create sync storage directory if it doesn't exist
	syncDir := p.getSyncStorageDir()
	if err := os.MkdirAll(syncDir, 0755); err != nil {
		return fmt.Errorf("failed to create sync storage directory: %v", err)
	}

	// Create filename from sync key (sanitize for filesystem)
	filename := p.sanitizeFilename(syncKey) + ".json"
	filepath := filepath.Join(syncDir, filename)

	// Prepare the sync data
	syncData := struct {
		LastSyncTime time.Time `json:"last_sync_time"`
		UpdatedAt    time.Time `json:"updated_at"`
	}{
		LastSyncTime: syncTime,
		UpdatedAt:    time.Now(),
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(syncData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sync time data: %v", err)
	}

	// Write to file atomically (write to temp file first, then rename)
	tempFile := filepath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp sync time file: %v", err)
	}

	if err := os.Rename(tempFile, filepath); err != nil {
		// Clean up temp file on error
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp sync time file: %v", err)
	}

	return nil
}

// getSyncStorageDir returns the directory for storing sync time files
func (p *Plugin) getSyncStorageDir() string {
	// Use a subdirectory in the current working directory
	// In production, you might want to use a configurable path
	return filepath.Join(".", "sync_storage", "mongodb")
}

// sanitizeFilename converts a sync key to a safe filename
func (p *Plugin) sanitizeFilename(syncKey string) string {
	// Replace unsafe characters with underscores
	// This is a simple approach - in production you might want more sophisticated sanitization
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := syncKey

	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Limit length to avoid filesystem issues
	if len(result) > 200 {
		result = result[:200]
	}

	return result
}
