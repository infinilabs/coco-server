package mongodb

import (
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				ConnectionURI: "mongodb://localhost:27017/test",
				Database:      "test",
				Collections: []CollectionConfig{
					{
						Name: "users",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing connection_uri",
			config: &Config{
				Database: "test",
				Collections: []CollectionConfig{
					{
						Name: "users",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing database",
			config: &Config{
				ConnectionURI: "mongodb://localhost:27017/test",
				Collections: []CollectionConfig{
					{
						Name: "users",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing collections",
			config: &Config{
				ConnectionURI: "mongodb://localhost:27017/test",
				Database:      "test",
				Collections:   []CollectionConfig{},
			},
			wantErr: true,
		},
		{
			name: "collection without name",
			config: &Config{
				ConnectionURI: "mongodb://localhost:27017/test",
				Database:      "test",
				Collections: []CollectionConfig{
					{
						Name: "",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid batch_size",
			config: &Config{
				ConnectionURI: "mongodb://localhost:27017/test",
				Database:      "test",
				Collections: []CollectionConfig{
					{
						Name: "users",
					},
				},
				BatchSize: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid max_pool_size",
			config: &Config{
				ConnectionURI: "mongodb://localhost:27017/test",
				Database:      "test",
				Collections: []CollectionConfig{
					{
						Name: "users",
					},
				},
				MaxPoolSize: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid page_size",
			config: &Config{
				ConnectionURI: "mongodb://localhost:27017/test",
				Database:      "test",
				Collections: []CollectionConfig{
					{
						Name: "users",
					},
				},
				PageSize: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid sync_strategy",
			config: &Config{
				ConnectionURI: "mongodb://localhost:27017/test",
				Database:      "test",
				Collections: []CollectionConfig{
					{
						Name: "users",
					},
				},
				SyncStrategy: "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid sync_strategy full",
			config: &Config{
				ConnectionURI: "mongodb://localhost:27017/test",
				Database:      "test",
				Collections: []CollectionConfig{
					{
						Name: "users",
					},
				},
				SyncStrategy: "full",
			},
			wantErr: false,
		},
		{
			name: "valid sync_strategy incremental",
			config: &Config{
				ConnectionURI: "mongodb://localhost:27017/test",
				Database:      "test",
				Collections: []CollectionConfig{
					{
						Name: "users",
					},
				},
				SyncStrategy: "incremental",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &Plugin{}
			err := plugin.validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetDefaultConfig(t *testing.T) {
	plugin := &Plugin{}
	config := &Config{}

	plugin.setDefaultConfig(config)

	// Check default values
	if config.BatchSize != 1000 {
		t.Errorf("expected BatchSize to be 1000, got %d", config.BatchSize)
	}

	if config.MaxPoolSize != 10 {
		t.Errorf("expected MaxPoolSize to be 10, got %d", config.MaxPoolSize)
	}

	if config.Timeout != "30s" {
		t.Errorf("expected Timeout to be '30s', got %s", config.Timeout)
	}

	if config.SyncStrategy != "full" {
		t.Errorf("expected SyncStrategy to be 'full', got %s", config.SyncStrategy)
	}

	if config.PageSize != 500 {
		t.Errorf("expected PageSize to be 500, got %d", config.PageSize)
	}

	if config.AuthDatabase != "admin" {
		t.Errorf("expected AuthDatabase to be 'admin', got %s", config.AuthDatabase)
	}

	if config.ClusterType != "standalone" {
		t.Errorf("expected ClusterType to be 'standalone', got %s", config.ClusterType)
	}

	if config.FieldMapping == nil {
		t.Error("expected FieldMapping to be initialized")
	}

	if !config.FieldMapping.Enabled {
		t.Error("expected FieldMapping.Enabled to be false by default")
	}

	if !config.EnableProjection {
		t.Error("expected EnableProjection to be true by default")
	}

	if !config.EnableIndexHint {
		t.Error("expected EnableIndexHint to be true by default")
	}
}

func TestCollectionConfig(t *testing.T) {
	config := CollectionConfig{
		Name:           "users",
		Filter:         map[string]interface{}{"status": "active"},
		TitleField:     "name",
		ContentField:   "bio",
		CategoryField:  "role",
		TagsField:      "skills",
		URLField:       "profile_url",
		TimestampField: "updated_at",
	}

	if config.Name != "users" {
		t.Errorf("expected Name to be 'users', got %s", config.Name)
	}

	if config.Filter["status"] != "active" {
		t.Errorf("expected Filter['status'] to be 'active', got %v", config.Filter["status"])
	}

	if config.TitleField != "name" {
		t.Errorf("expected TitleField to be 'name', got %s", config.TitleField)
	}

	if config.ContentField != "bio" {
		t.Errorf("expected ContentField to be 'bio', got %s", config.ContentField)
	}

	if config.CategoryField != "role" {
		t.Errorf("expected CategoryField to be 'role', got %s", config.CategoryField)
	}

	if config.TagsField != "skills" {
		t.Errorf("expected TagsField to be 'skills', got %s", config.TagsField)
	}

	if config.URLField != "profile_url" {
		t.Errorf("expected URLField to be 'profile_url', got %s", config.URLField)
	}

	if config.TimestampField != "updated_at" {
		t.Errorf("expected TimestampField to be 'updated_at', got %s", config.TimestampField)
	}
}

func TestFieldMappingConfig(t *testing.T) {
	config := FieldMappingConfig{
		Enabled: true,
		Mapping: map[string]interface{}{
			"id":      "user_id",
			"title":   "user_name",
			"content": "user_bio",
		},
	}

	if !config.Enabled {
		t.Error("expected Enabled to be true")
	}

	if config.Mapping["id"] != "user_id" {
		t.Errorf("expected Mapping['id'] to be 'user_id', got %v", config.Mapping["id"])
	}

	if config.Mapping["title"] != "user_name" {
		t.Errorf("expected Mapping['title'] to be 'user_name', got %v", config.Mapping["title"])
	}

	if config.Mapping["content"] != "user_bio" {
		t.Errorf("expected Mapping['content'] to be 'user_bio', got %v", config.Mapping["content"])
	}
}

func TestConfigWithPagination(t *testing.T) {
	config := &Config{
		ConnectionURI: "mongodb://localhost:27017/test",
		Database:      "test",
		Collections: []CollectionConfig{
			{
				Name: "users",
			},
		},
		Pagination: true,
		PageSize:   100,
	}

	plugin := &Plugin{}
	err := plugin.validateConfig(config)
	if err != nil {
		t.Errorf("validateConfig() error = %v", err)
	}

	if !config.Pagination {
		t.Error("expected Pagination to be true")
	}

	if config.PageSize != 100 {
		t.Errorf("expected PageSize to be 100, got %d", config.PageSize)
	}
}

func TestConfigWithLastModifiedField(t *testing.T) {
	config := &Config{
		ConnectionURI:     "mongodb://localhost:27017/test",
		Database:          "test",
		LastModifiedField: "updated_at",
		Collections: []CollectionConfig{
			{
				Name: "users",
			},
		},
	}

	plugin := &Plugin{}
	err := plugin.validateConfig(config)
	if err != nil {
		t.Errorf("validateConfig() error = %v", err)
	}

	if config.LastModifiedField != "updated_at" {
		t.Errorf("expected LastModifiedField to be 'updated_at', got %s", config.LastModifiedField)
	}
}

func TestConfigWithAuthDatabase(t *testing.T) {
	config := &Config{
		ConnectionURI: "mongodb://user:pass@localhost:27017/test",
		Database:      "test",
		AuthDatabase:  "admin",
		Collections: []CollectionConfig{
			{
				Name: "users",
			},
		},
	}

	plugin := &Plugin{}
	err := plugin.validateConfig(config)
	if err != nil {
		t.Errorf("validateConfig() error = %v", err)
	}

	if config.AuthDatabase != "admin" {
		t.Errorf("expected AuthDatabase to be 'admin', got %s", config.AuthDatabase)
	}
}

func TestConfigWithClusterType(t *testing.T) {
	tests := []struct {
		name        string
		clusterType string
		wantErr     bool
	}{
		{"standalone", "standalone", false},
		{"replica_set", "replica_set", false},
		{"sharded", "sharded", false},
		{"invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ConnectionURI: "mongodb://localhost:27017/test",
				Database:      "test",
				ClusterType:   tt.clusterType,
				Collections: []CollectionConfig{
					{
						Name: "users",
					},
				},
			}

			plugin := &Plugin{}
			err := plugin.validateConfig(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAdvancedConfigOptions(t *testing.T) {
	config := &Config{
		ConnectionURI: "mongodb://localhost:27017/test",
		Database:      "test",
		Collections: []CollectionConfig{
			{
				Name: "users",
			},
		},
		EnableProjection: false,
		EnableIndexHint:  false,
	}

	plugin := &Plugin{}
	plugin.setDefaultConfig(config)

	// Test that advanced options are enabled by default
	if !config.EnableProjection {
		t.Error("expected EnableProjection to be enabled by default")
	}

	if !config.EnableIndexHint {
		t.Error("expected EnableIndexHint to be enabled by default")
	}

	// Test with explicit values
	config.EnableProjection = false
	config.EnableIndexHint = false
	plugin.setDefaultConfig(config)

	if config.EnableProjection {
		t.Error("expected EnableProjection to respect explicit false value")
	}

	if config.EnableIndexHint {
		t.Error("expected EnableIndexHint to respect explicit false value")
	}
}
