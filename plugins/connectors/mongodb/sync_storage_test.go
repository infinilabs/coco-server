package mongodb

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"infini.sh/coco/modules/common"
)

func TestSyncTimeStorage(t *testing.T) {
	// Create a temporary test directory
	testDir := t.TempDir()

	// Create a test plugin instance
	plugin := &Plugin{}

	// Test data
	syncKey := "test_mongodb_localhost_27017_testdb_testcollection"
	testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Test storing sync time
	err := plugin.updateSyncTimeInStorage(syncKey, testTime)
	if err != nil {
		t.Fatalf("Failed to store sync time: %v", err)
	}

	// Test retrieving sync time
	retrievedTime, err := plugin.getSyncTimeFromStorage(syncKey)
	if err != nil {
		t.Fatalf("Failed to retrieve sync time: %v", err)
	}

	if !retrievedTime.Equal(testTime) {
		t.Errorf("Retrieved time %v does not match stored time %v", retrievedTime, testTime)
	}

	// Test updating sync time
	newTime := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	err = plugin.updateSyncTimeInStorage(syncKey, newTime)
	if err != nil {
		t.Fatalf("Failed to update sync time: %v", err)
	}

	// Verify the update
	updatedTime, err := plugin.getSyncTimeFromStorage(syncKey)
	if err != nil {
		t.Fatalf("Failed to retrieve updated sync time: %v", err)
	}

	if !updatedTime.Equal(newTime) {
		t.Errorf("Updated time %v does not match expected time %v", updatedTime, newTime)
	}
}

func TestSyncTimeStorageWithConfig(t *testing.T) {
	// Create a temporary test directory
	testDir := t.TempDir()

	// Create a test plugin instance
	plugin := &Plugin{}

	// Test configuration
	config := &Config{
		ConnectionURI: "mongodb://localhost:27017",
		Database:      "testdb",
	}
	collectionName := "testcollection"
	testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Test updating last sync time
	err := plugin.syncManager.UpdateLastSyncTime(datasourceID, collectionName, testTime, testTime)
	if err != nil {
		t.Fatalf("Failed to update last sync time: %v", err)
	}

	// Test getting last sync time
	retrievedTime := plugin.syncManager.GetLastSyncTime(datasourceID, collectionName)
	if !retrievedTime.Equal(testTime) {
		t.Errorf("Retrieved time %v does not match stored time %v", retrievedTime, testTime)
	}
}

func TestSyncTimeStorageNonExistent(t *testing.T) {
	// Create a test plugin instance with sync manager
	plugin := &Plugin{
		syncManager: NewSyncManager(),
	}

	// Test retrieving non-existent sync time
	datasourceID := "test_datasource"
	collectionName := "test_collection"
	retrievedTime := plugin.syncManager.GetLastSyncTime(datasourceID, collectionName)

	if !retrievedTime.IsZero() {
		t.Errorf("Expected zero time for non-existent key, got %v", retrievedTime)
	}
}

func TestSyncTimeStorageInvalidData(t *testing.T) {
	// Create a test plugin instance with sync manager
	plugin := &Plugin{
		syncManager: NewSyncManager(),
	}

	// Test retrieving from non-existent datasource/collection
	datasourceID := "invalid_datasource"
	collectionName := "invalid_collection"
	retrievedTime := plugin.syncManager.GetLastSyncTime(datasourceID, collectionName)

	if !retrievedTime.IsZero() {
		t.Errorf("Expected zero time for invalid datasource/collection, got %v", retrievedTime)
	}
}

func TestGetLatestTimestampFromBatch(t *testing.T) {
	plugin := &Plugin{}

	// Create test documents with different timestamps
	doc1 := &common.Document{
		Updated: &time.Time{},
	}
	doc1.Updated = &time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	doc2 := &common.Document{
		Updated: &time.Time{},
	}
	doc2.Updated = &time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

	doc3 := &common.Document{
		Updated: &time.Time{},
	}
	doc3.Updated = &time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC)

	documents := []*common.Document{doc1, doc2, doc3}

	// Test getting latest timestamp
	latestTime := plugin.getLatestTimestampFromBatch(documents, "updated_at")
	expectedTime := time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC)

	if !latestTime.Equal(expectedTime) {
		t.Errorf("Expected latest time %v, got %v", expectedTime, latestTime)
	}
}

func TestGetLatestTimestampFromBatchWithNil(t *testing.T) {
	plugin := &Plugin{}

	// Create test documents with some nil timestamps
	doc1 := &common.Document{
		Updated: nil,
	}

	doc2 := &common.Document{
		Updated: &time.Time{},
	}
	doc2.Updated = &time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

	documents := []*common.Document{doc1, doc2}

	// Test getting latest timestamp
	latestTime := plugin.getLatestTimestampFromBatch(documents, "updated_at")
	expectedTime := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

	if !latestTime.Equal(expectedTime) {
		t.Errorf("Expected latest time %v, got %v", expectedTime, latestTime)
	}
}

func TestGetLatestTimestampFromBatchEmpty(t *testing.T) {
	plugin := &Plugin{}

	// Test with empty documents slice
	documents := []*common.Document{}

	latestTime := plugin.getLatestTimestampFromBatch(documents, "updated_at")

	if !latestTime.IsZero() {
		t.Errorf("Expected zero time for empty documents, got %v", latestTime)
	}
}
