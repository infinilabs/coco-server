/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"context"
	"runtime"
	"time"

	log "github.com/cihub/seelog"
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
	findOptions.SetBatchSize(int32(config.BatchSize))

	// Set projection if fields are specified
	if len(collConfig.Fields) > 0 {
		projection := bson.D{}
		for _, field := range collConfig.Fields {
			projection = append(projection, bson.E{Key: field, Value: 1})
		}
		findOptions.SetProjection(projection)
	}

	// Optimize query
	p.optimizeQuery(findOptions, collConfig)

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
	}

	log.Infof("[mongodb connector] finished scanning collection [%s] in datasource [%s]", collConfig.Name, datasource.Name)
}

func (p *Plugin) buildFilter(config *Config, collConfig CollectionConfig) bson.M {
	filter := bson.M{}

	// Copy base filter
	for k, v := range collConfig.Filter {
		filter[k] = v
	}

	// Add timestamp filter for incremental sync
	if config.SyncStrategy == "incremental" && collConfig.TimestampField != "" {
		if !config.LastSyncTime.IsZero() {
			filter[collConfig.TimestampField] = bson.M{"$gt": config.LastSyncTime}
		}
	}

	return filter
}

func (p *Plugin) optimizeQuery(findOptions *options.FindOptions, collConfig CollectionConfig) {
	// Set read concern level
	findOptions.SetReadConcern(readconcern.Local())

	// If there's a timestamp field, suggest using related index
	if collConfig.TimestampField != "" {
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
