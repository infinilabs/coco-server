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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

func (p *Plugin) processCursor(cursor *mongo.Cursor, collConfig CollectionConfig, datasource *common.DataSource, config *Config) []*common.Document {
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

		doc, err := p.transformToDocument(mongoDoc, &collConfig, datasource, config)
		if err != nil {
			log.Warnf("[mongodb connector] transform document failed: %v", err)
			continue
		}

		documents = append(documents, doc)
		count++
	}

	return documents
}

// transformToDocument transforms a MongoDB document to a common Document
func (p *Plugin) transformToDocument(mongoDoc bson.M, collConfig *CollectionConfig, datasource *common.DataSource, config *Config) (*common.Document, error) {
	doc := &common.Document{}

	// Extract MongoDB ObjectID
	objectID, ok := mongoDoc["_id"].(primitive.ObjectID)
	if !ok {
		// Try to get string ID if ObjectID is not available
		if idStr, ok := mongoDoc["_id"].(string); ok {
			doc.ID = idStr
		} else {
			doc.ID = fmt.Sprintf("%v", mongoDoc["_id"])
		}
	} else {
		doc.ID = objectID.Hex()
	}

	// Set document type
	doc.Type = ConnectorMongoDB

	// Apply field mapping configuration
	p.applyFieldMapping(doc, mongoDoc, config)

	// Store original metadata
	doc.Metadata = make(map[string]interface{})
	doc.Metadata["mongodb_collection"] = collConfig.Name
	doc.Metadata["mongodb_id"] = objectID
	doc.Metadata["raw_document"] = mongoDoc

	return doc, nil
}

// applyFieldMapping applies field mapping configuration to the document
// This function handles all field mappings using the centralized FieldMapping configuration
func (p *Plugin) applyFieldMapping(doc *common.Document, mongoDoc bson.M, config *Config) {
	if config.FieldMapping == nil || !config.FieldMapping.Enabled {
		return
	}

	// Apply standard field mappings
	if config.FieldMapping.TitleField != "" {
		if title, ok := mongoDoc[config.FieldMapping.TitleField]; ok {
			doc.Title = p.safeConvertToString(title)
		}
	}

	if config.FieldMapping.ContentField != "" {
		if content, ok := mongoDoc[config.FieldMapping.ContentField]; ok {
			doc.Content = p.safeConvertToString(content)
		}
	}

	if config.FieldMapping.CategoryField != "" {
		if category, ok := mongoDoc[config.FieldMapping.CategoryField]; ok {
			doc.Category = p.safeConvertToString(category)
		}
	}

	// Handle tags
	if config.FieldMapping.TagsField != "" {
		if tags, ok := mongoDoc[config.FieldMapping.TagsField]; ok {
			doc.Tags = p.convertToStringSlice(tags)
		}
	}

	// Handle URL
	if config.FieldMapping.URLField != "" {
		if url, ok := mongoDoc[config.FieldMapping.URLField]; ok {
			doc.URL = p.safeConvertToString(url)
		}
	}

	// Handle timestamp
	if config.FieldMapping.TimestampField != "" {
		if timestamp, ok := mongoDoc[config.FieldMapping.TimestampField]; ok {
			if t := p.convertToTime(timestamp); t != nil {
				doc.Updated = t
			}
		}
	}

	// Apply custom field mappings from the mapping configuration
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
