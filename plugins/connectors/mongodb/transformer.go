/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"context"
	"fmt"

	log "github.com/cihub/seelog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

func (p *Plugin) processCursor(cursor *mongo.Cursor, collConfig CollectionConfig, datasource *common.DataSource) []*common.Document {
	var documents []*common.Document
	count := 0
	maxBatchSize := 1000 // Prevent memory overflow

	// Pre-allocate slice with capacity to reduce memory allocations
	documents = make([]*common.Document, 0, maxBatchSize)

	for cursor.Next(context.Background()) && count < maxBatchSize {
		if global.ShuttingDown() {
			break
		}

		var mongoDoc bson.M
		if err := cursor.Decode(&mongoDoc); err != nil {
			log.Warnf("[mongodb connector] decode document failed: %v", err)
			continue
		}

		doc, err := p.transformToDocument(mongoDoc, collConfig, datasource, config)
		if err != nil {
			log.Warnf("[mongodb connector] transform document failed: %v", err)
			continue
		}

		documents = append(documents, doc)
		count++
	}

	return documents
}

func (p *Plugin) transformToDocument(mongoDoc bson.M, collConfig CollectionConfig, datasource *common.DataSource, config *Config) (*common.Document, error) {
	doc := &common.Document{
		Source: common.DataSourceReference{
			ID:   datasource.ID,
			Type: "connector",
			Name: datasource.Name,
		},
		Type: ConnectorMongoDB,
		Icon: "default",
	}

	doc.System = datasource.System

	// Generate unique ID
	objectID := mongoDoc["_id"]
	doc.ID = util.MD5digest(fmt.Sprintf("%s-%s-%v", datasource.ID, collConfig.Name, objectID))

	// Field mapping using collection-specific fields
	if collConfig.TitleField != "" {
		if title, ok := mongoDoc[collConfig.TitleField]; ok {
			doc.Title = p.safeConvertToString(title)
		}
	}

	if collConfig.ContentField != "" {
		if content, ok := mongoDoc[collConfig.ContentField]; ok {
			doc.Content = p.safeConvertToString(content)
		}
	}

	if collConfig.CategoryField != "" {
		if category, ok := mongoDoc[collConfig.CategoryField]; ok {
			doc.Category = p.safeConvertToString(category)
		}
	}

	// Handle tags
	if collConfig.TagsField != "" {
		if tags, ok := mongoDoc[collConfig.TagsField]; ok {
			doc.Tags = p.convertToStringSlice(tags)
		}
	}

	// Handle URL
	if collConfig.URLField != "" {
		if url, ok := mongoDoc[collConfig.URLField]; ok {
			doc.URL = p.safeConvertToString(url)
		}
	}

	// Handle timestamp
	if collConfig.TimestampField != "" {
		if timestamp, ok := mongoDoc[collConfig.TimestampField]; ok {
			if t := p.convertToTime(timestamp); t != nil {
				doc.Updated = t
			}
		}
	}

	// Store original metadata
	doc.Metadata = make(map[string]interface{})
	doc.Metadata["mongodb_collection"] = collConfig.Name
	doc.Metadata["mongodb_id"] = objectID
	doc.Metadata["raw_document"] = mongoDoc

	// Apply global field mapping if enabled
	p.applyGlobalFieldMapping(doc, mongoDoc, config)

	return doc, nil
}

// applyGlobalFieldMapping applies global field mapping configuration to the document
// This function can be used when global field mapping is enabled in the config
func (p *Plugin) applyGlobalFieldMapping(doc *common.Document, mongoDoc bson.M, config *Config) {
	if config.FieldMapping != nil && config.FieldMapping.Enabled {
		// Apply global field mappings if configured
		for targetField, sourceField := range config.FieldMapping.Mapping {
			if sourceFieldStr, ok := sourceField.(string); ok {
				if value, exists := mongoDoc[sourceFieldStr]; exists {
					switch targetField {
					case "id":
						// Handle ID field specially
						doc.ID = p.safeConvertToString(value)
					case "title":
						doc.Title = p.safeConvertToString(value)
					case "content":
						doc.Content = p.safeConvertToString(value)
					case "category":
						doc.Category = p.safeConvertToString(value)
					case "tags":
						doc.Tags = p.convertToStringSlice(value)
					case "url":
						doc.URL = p.safeConvertToString(value)
					case "metadata":
						// Handle metadata fields
						if doc.Metadata == nil {
							doc.Metadata = make(map[string]interface{})
						}
						doc.Metadata[sourceFieldStr] = value
					}
				}
			}
		}
	}
}

func (p *Plugin) pushDocuments(documents []*common.Document) {
	for _, doc := range documents {
		if global.ShuttingDown() {
			return
		}

		data := util.MustToJSONBytes(doc)
		if err := queue.Push(p.Queue, data); err != nil {
			log.Errorf("[mongodb connector] failed to push document to queue: %v", err)
			continue
		}
	}
}
