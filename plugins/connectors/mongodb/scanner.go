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
	filter := p.buildFilter(config, collConfig, datasource)

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
				// Update sync time using sync manager with datasource ID and collection name
				if err := p.syncManager.UpdateLastSyncTime(datasource.ID, collConfig.Name, latestTime, latestTime); err != nil {
					log.Warnf("[mongodb connector] failed to update last sync time: %v", err)
				}
			}
		}
	}

	log.Infof("[mongodb connector] finished scanning collection [%s] in datasource [%s]", collConfig.Name, datasource.Name)
}

func (p *Plugin) buildFilter(config *Config, collConfig CollectionConfig, datasource *common.DataSource) bson.M {
	filter := bson.M{}

	// Copy base filter from collection configuration
	for k, v := range collConfig.Filter {
		filter[k] = v
	}

	// Add timestamp filter for incremental sync
	if config.SyncStrategy == "incremental" && config.LastModifiedField != "" {
		// Get last sync time from sync manager using datasource ID and collection name
		lastSyncTime := p.syncManager.GetLastSyncTime(datasource.ID, collConfig.Name)
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
