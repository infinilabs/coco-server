//go:build integration
// +build integration

/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/queue"
)

// TestMongoDBIntegration requires a running MongoDB instance
func TestMongoDBIntegration(t *testing.T) {  
	// Skip if no MongoDB connection string provided  
	mongoURI := os.Getenv("MONGODB_TEST_URI")  
	if mongoURI == "" {  
		t.Skip("MONGODB_TEST_URI not set, skipping integration test")  
	}  
	  
	// Setup test data  
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))  
	if err != nil {  
		t.Fatalf("Failed to connect to MongoDB: %v", err)  
	}  
	defer client.Disconnect(context.Background())  
	  
	// Create test database and collection  
	testDB := "coco_test"  
	testCollection := "test_articles"  
	  
	collection := client.Database(testDB).Collection(testCollection)  
	  
	// Insert test documents  
	testDocs := []interface{}{  
		bson.M{  
			"title":      "Test Article 1",  
			"content":    "This is the content of test article 1",  
			"category":   "Technology",  
			"tags":       []string{"mongodb", "database", "nosql"},  
			"url":        "https://example.com/article1",  
			"updated_at": time.Now(),  
			"status":     "published",  
		},  
		bson.M{  
			"title":      "Test Article 2",  
			"content":    "This is the content of test article 2",  
			"category":   "Programming",  
			"tags":       []string{"go", "golang", "backend"},  
			"url":        "https://example.com/article2",  
			"updated_at": time.Now(),  
			"status":     "published",  
		},  
	}  
	  
	_, err = collection.InsertMany(context.Background(), testDocs)  
	if err != nil {  
		t.Fatalf("Failed to insert test documents: %v", err)  
	}  
	  
	// Clean up after test  
	defer func() {  
		collection.Drop(context.Background())  
	}()  
	  
	// Setup plugin  
	plugin := &Plugin{}  
	plugin.Queue = &queue.QueueConfig{Name: "test_queue"}  
	  
	// Setup test configuration  
	config := &Config{  
		ConnectionURI: mongoURI,  
		Database:      testDB,  
		BatchSize:     10,  
		MaxPoolSize:   5,  
		Timeout:       "10s",  
		Collections: []CollectionConfig{  
			{  
				Name:           testCollection,  
				TitleField:     "title",  
				ContentField:   "content",  
				CategoryField:  "category",  
				TagsField:      "tags",  
				URLField:       "url",  
				TimestampField: "updated_at",  
				Filter: map[string]interface{}{  
					"status": "published",  
				},  
			},  
		},  
	}  
	  
	// Test connection creation  
	mongoClient, err := plugin.createMongoClient(config)  
	if err != nil {  
		t.Fatalf("Failed to create MongoDB client: %v", err)  
	}  
	defer mongoClient.Disconnect(context.Background())  
	  
	// Test health check  
	if err := plugin.healthCheck(mongoClient); err != nil {  
		t.Fatalf("Health check failed: %v", err)  
	}  
	  
	// Test collection stats  
	stats, err := plugin.getCollectionStats(mongoClient, testDB, testCollection)  
	if err != nil {  
		t.Fatalf("Failed to get collection stats: %v", err)  
	}  
	  
	if stats["documentCount"].(int64) != 2 {  
		t.Errorf("Expected 2 documents, got %v", stats["documentCount"])  
	}  
	  
	// Test document scanning  
	testCollection := mongoClient.Database(testDB).Collection(testCollection)  
	filter := plugin.buildFilter(config, config.Collections[0])  
	  
	cursor, err := testCollection.Find(context.Background(), filter)  
	if err != nil {  
		t.Fatalf("Failed to query collection: %v", err)  
	}  
	defer cursor.Close(context.Background())  
	  
	datasource := &common.DataSource{  
		ID:   "test-datasource",  
		Name: "Test MongoDB Integration",  
	}  
	  
	documents := plugin.processCursor(cursor, config.Collections[0], datasource)  
	  
	if len(documents) != 2 {  
		t.Errorf("Expected 2 documents, got %d", len(documents))  
	}  
	  
	// Verify document transformation  
	doc := documents[0]  
	if doc.Title == "" {  
		t.Errorf("Expected non-empty title")  
	}  
	if doc.Content == "" {  
		t.Errorf("Expected non-empty content")  
	}  
	if doc.Category == "" {  
		t.Errorf("Expected non-empty category")  
	}  
	if len(doc.Tags) == 0 {  
		t.Errorf("Expected non-empty tags")  
	}  
	if doc.URL == "" {  
		t.Errorf("Expected non-empty URL")  
	}  
	if doc.Updated == nil {  
		t.Errorf("Expected non-nil updated time")  
	}  
}