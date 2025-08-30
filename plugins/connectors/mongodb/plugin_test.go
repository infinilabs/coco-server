/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"infini.sh/coco/modules/common"
)

func TestSafeConvertToString(t *testing.T) {
	p := &Plugin{}

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"float", 3.14, "3.140000"},
		{"bool", true, "true"},
		{"nil", nil, ""},
		{"objectid", primitive.NewObjectID(), ""},
		{"array", []interface{}{"a", "b"}, `["a","b"]`},
		{"object", map[string]interface{}{"key": "value"}, `{"key":"value"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.safeConvertToString(tt.input)
			if tt.name == "objectid" {
				// ObjectID will have different values, just check it's not empty
				if result == "" {
					t.Errorf("Expected non-empty ObjectID string")
				}
			} else if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestConvertToStringSlice(t *testing.T) {
	p := &Plugin{}

	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{"string_slice", []string{"a", "b"}, []string{"a", "b"}},
		{"interface_slice", []interface{}{"a", 1, true}, []string{"a", "1", "true"}},
		{"single_string", "hello", []string{"hello"}},
		{"nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.convertToStringSlice(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("Expected %s at index %d, got %s", tt.expected[i], i, v)
				}
			}
		})
	}
}

func TestConvertToTime(t *testing.T) {
	p := &Plugin{}

	now := time.Now()
	timestamp := primitive.NewDateTimeFromTime(now)

	tests := []struct {
		name     string
		input    interface{}
		expected bool // whether result should be non-nil
	}{
		{"time", now, true},
		{"datetime", timestamp, true},
		{"unix_timestamp", now.Unix(), true},
		{"rfc3339_string", now.Format(time.RFC3339), true},
		{"invalid_string", "invalid", false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.convertToTime(tt.input)
			if tt.expected && result == nil {
				t.Errorf("Expected non-nil time")
			} else if !tt.expected && result != nil {
				t.Errorf("Expected nil time")
			}
		})
	}
}

func TestBuildFilter(t *testing.T) {
	p := &Plugin{
		syncManager: NewSyncManager(),
	}

	config := &Config{
		SyncStrategy:      "incremental",
		LastModifiedField: "updated_at",
	}

	collConfig := CollectionConfig{
		Filter: map[string]interface{}{
			"status": "published",
		},
		TimestampField: "updated_at",
	}

	// Create a mock datasource
	datasource := &common.DataSource{
		ID: "test_datasource",
	}

	filter := p.buildFilter(config, collConfig, datasource)

	// Check base filter
	if filter["status"] != "published" {
		t.Errorf("Expected status filter to be preserved")
	}

	// Check timestamp filter - should not exist initially since no sync time is set
	if _, exists := filter["updated_at"]; exists {
		t.Errorf("Expected no timestamp filter initially since no sync time is set")
	}
}

func TestValidateConfig(t *testing.T) {
	p := &Plugin{}

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid_config",
			config: &Config{
				Host:     "localhost",
				Database: "test",
				Collections: []CollectionConfig{
					{Name: "collection1"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing_host_and_uri",
			config: &Config{
				Database: "test",
				Collections: []CollectionConfig{
					{Name: "collection1"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing_database",
			config: &Config{
				Host: "localhost",
				Collections: []CollectionConfig{
					{Name: "collection1"},
				},
			},
			wantErr: true,
		},
		{
			name: "no_collections",
			config: &Config{
				Host:        "localhost",
				Database:    "test",
				Collections: []CollectionConfig{},
			},
			wantErr: true,
		},
		{
			name: "collection_without_name",
			config: &Config{
				Host:     "localhost",
				Database: "test",
				Collections: []CollectionConfig{
					{Name: ""},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransformToDocument(t *testing.T) {
	p := &Plugin{}

	mongoDoc := bson.M{
		"_id":        primitive.NewObjectID(),
		"title":      "Test Article",
		"content":    "This is test content",
		"category":   "Technology",
		"tags":       []interface{}{"mongodb", "database"},
		"url":        "https://example.com/article",
		"updated_at": primitive.NewDateTimeFromTime(time.Now()),
	}

	collConfig := CollectionConfig{
		Name:           "articles",
		TitleField:     "title",
		ContentField:   "content",
		CategoryField:  "category",
		TagsField:      "tags",
		URLField:       "url",
		TimestampField: "updated_at",
	}

	datasource := &common.DataSource{
		Name: "Test MongoDB",
	}

	config := &Config{}
	doc, err := p.transformToDocument(mongoDoc, collConfig, datasource, config)
	if err != nil {
		t.Fatalf("transformToDocument() error = %v", err)
	}

	if doc.Title != "Test Article" {
		t.Errorf("Expected title 'Test Article', got '%s'", doc.Title)
	}

	if doc.Content != "This is test content" {
		t.Errorf("Expected content 'This is test content', got '%s'", doc.Content)
	}

	if doc.Category != "Technology" {
		t.Errorf("Expected category 'Technology', got '%s'", doc.Category)
	}

	if doc.Tags[0] != "mongodb" || doc.Tags[1] != "database" {
		t.Errorf("Expected tags ['mongodb', 'database'], got %v", doc.Tags)
	}

	if doc.URL != "https://example.com/article" {
		t.Errorf("Expected URL 'https://example.com/article', got '%s'", doc.URL)
	}

	if doc.Type != ConnectorMongoDB {
		t.Errorf("Expected type '%s', got '%s'", ConnectorMongoDB, doc.Type)
	}

	if doc.Updated == nil {
		t.Errorf("Expected non-nil Updated time")
	}

	// Check metadata
	if doc.Metadata["mongodb_collection"] != "articles" {
		t.Errorf("Expected collection metadata to be 'articles'")
	}

	if doc.Metadata["mongodb_id"] != mongoDoc["_id"] {
		t.Errorf("Expected mongodb_id metadata to match original _id")
	}
}

func TestBuildConnectionURI(t *testing.T) {
	p := &Plugin{}

	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name: "basic_connection",
			config: &Config{
				ConnectionURI: "mongodb://localhost:27017/testdb",
				Database:      "testdb",
			},
			expected: "mongodb://localhost:27017/testdb",
		},
		{
			name: "with_auth",
			config: &Config{
				Host:     "localhost",
				Port:     27017,
				Username: "user",
				Password: "pass",
				Database: "testdb",
			},
			expected: "mongodb://user:pass@localhost:27017/testdb",
		},
		{
			name: "with_replica_set",
			config: &Config{
				Host:       "localhost",
				Port:       27017,
				Database:   "testdb",
				ReplicaSet: "rs0",
			},
			expected: "mongodb://localhost:27017/testdb?replicaSet=rs0",
		},
		{
			name: "with_auth_database",
			config: &Config{
				Host:         "localhost",
				Port:         27017,
				Database:     "testdb",
				AuthDatabase: "admin",
			},
			expected: "mongodb://localhost:27017/testdb?authSource=admin",
		},
		{
			name: "with_tls",
			config: &Config{
				Host:      "localhost",
				Port:      27017,
				Database:  "testdb",
				EnableTLS: true,
			},
			expected: "mongodb://localhost:27017/testdb?ssl=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.buildConnectionURI(tt.config)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
