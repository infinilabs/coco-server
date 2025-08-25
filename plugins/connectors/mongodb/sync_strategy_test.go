/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"testing"
)

func TestFullSyncStrategy(t *testing.T) {
	strategy := &FullSyncStrategy{}

	config := &Config{}
	collConfig := CollectionConfig{
		Filter: map[string]interface{}{
			"status": "published",
		},
	}
	datasourceID := "test_datasource"
	syncManager := &SyncManager{}

	// Test filter building
	filter := strategy.BuildFilter(config, collConfig, datasourceID, syncManager)

	// Should preserve base filter
	if filter["status"] != "published" {
		t.Errorf("Expected status filter to be preserved, got %v", filter["status"])
	}

	// Should not have timestamp filtering
	if _, exists := filter["updated_at"]; exists {
		t.Errorf("Expected no timestamp filtering for full sync strategy")
	}

	// Test strategy properties
	if strategy.ShouldUpdateSyncTime() {
		t.Error("Expected full sync strategy to not update sync time")
	}

	if strategy.GetStrategyName() != "full" {
		t.Errorf("Expected strategy name to be 'full', got %s", strategy.GetStrategyName())
	}
}

func TestIncrementalSyncStrategy(t *testing.T) {
	strategy := &IncrementalSyncStrategy{}

	config := &Config{
		LastModifiedField: "updated_at",
	}
	collConfig := CollectionConfig{
		Filter: map[string]interface{}{
			"status": "published",
		},
	}
	datasourceID := "test_datasource"
	syncManager := &SyncManager{}

	// Test filter building
	filter := strategy.BuildFilter(config, collConfig, datasourceID, syncManager)

	// Should preserve base filter
	if filter["status"] != "published" {
		t.Errorf("Expected status filter to be preserved, got %v", filter["status"])
	}

	// Should not have timestamp filtering initially (no previous sync time)
	if _, exists := filter["updated_at"]; exists {
		t.Errorf("Expected no timestamp filtering initially for incremental sync strategy")
	}

	// Test strategy properties
	if !strategy.ShouldUpdateSyncTime() {
		t.Error("Expected incremental sync strategy to update sync time")
	}

	if strategy.GetStrategyName() != "incremental" {
		t.Errorf("Expected strategy name to be 'incremental', got %s", strategy.GetStrategyName())
	}
}

func TestSyncStrategyFactory(t *testing.T) {
	factory := &SyncStrategyFactory{}

	// Test full strategy creation
	fullStrategy := factory.CreateStrategy("full")
	if fullStrategy.GetStrategyName() != "full" {
		t.Errorf("Expected full strategy, got %s", fullStrategy.GetStrategyName())
	}

	// Test incremental strategy creation
	incStrategy := factory.CreateStrategy("incremental")
	if incStrategy.GetStrategyName() != "incremental" {
		t.Errorf("Expected incremental strategy, got %s", incStrategy.GetStrategyName())
	}

	// Test default strategy creation
	defaultStrategy := factory.CreateStrategy("")
	if defaultStrategy.GetStrategyName() != "full" {
		t.Errorf("Expected default strategy to be full, got %s", defaultStrategy.GetStrategyName())
	}

	// Test invalid strategy creation
	invalidStrategy := factory.CreateStrategy("invalid")
	if invalidStrategy.GetStrategyName() != "full" {
		t.Errorf("Expected invalid strategy to default to full, got %s", invalidStrategy.GetStrategyName())
	}

	// Test strategy name display
	if factory.GetStrategyName("full") != "full" {
		t.Errorf("Expected strategy name 'full', got %s", factory.GetStrategyName("full"))
	}

	if factory.GetStrategyName("incremental") != "incremental" {
		t.Errorf("Expected strategy name 'incremental', got %s", factory.GetStrategyName("incremental"))
	}

	if factory.GetStrategyName("") != "full (default)" {
		t.Errorf("Expected strategy name 'full (default)', got %s", factory.GetStrategyName(""))
	}
}
