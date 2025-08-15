package mongodb

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
	err := plugin.updateLastSyncTime(config, collectionName, testTime)
	if err != nil {
		t.Fatalf("Failed to update last sync time: %v", err)
	}
	
	// Test getting last sync time
	retrievedTime := plugin.getLastSyncTime(config, collectionName)
	if !retrievedTime.Equal(testTime) {
		t.Errorf("Retrieved time %v does not match stored time %v", retrievedTime, testTime)
	}
}

func TestSyncTimeStorageNonExistent(t *testing.T) {
	// Create a temporary test directory
	testDir := t.TempDir()
	
	// Create a test plugin instance
	plugin := &Plugin{}
	
	// Test retrieving non-existent sync time
	syncKey := "non_existent_key"
	retrievedTime, err := plugin.getSyncTimeFromStorage(syncKey)
	if err != nil {
		t.Fatalf("Failed to retrieve non-existent sync time: %v", err)
	}
	
	if !retrievedTime.IsZero() {
		t.Errorf("Expected zero time for non-existent key, got %v", retrievedTime)
	}
}

func TestSyncTimeStorageInvalidData(t *testing.T) {
	// Create a temporary test directory
	testDir := t.TempDir()
	
	// Create a test plugin instance
	plugin := &Plugin{}
	
	// Create a sync storage directory
	syncDir := filepath.Join(testDir, "sync_storage", "mongodb")
	if err := os.MkdirAll(syncDir, 0755); err != nil {
		t.Fatalf("Failed to create sync storage directory: %v", err)
	}
	
	// Create an invalid JSON file
	invalidFile := filepath.Join(syncDir, "invalid.json")
	invalidData := []byte(`{"invalid": "json"`)
	if err := os.WriteFile(invalidFile, invalidData, 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}
	
	// Test retrieving from invalid file
	syncKey := "invalid"
	_, err := plugin.getSyncTimeFromStorage(syncKey)
	if err == nil {
		t.Error("Expected error when reading invalid JSON, got none")
	}
}

func TestSanitizeFilename(t *testing.T) {
	plugin := &Plugin{}
	
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "mongodb://localhost:27017/testdb",
			expected: "mongodb___localhost_27017_testdb",
		},
		{
			input:    "mongodb://user:pass@localhost:27017/testdb?authSource=admin",
			expected: "mongodb___user_pass_localhost_27017_testdb_authSource_admin",
		},
		{
			input:    "mongodb://localhost:27017/testdb/collection",
			expected: "mongodb___localhost_27017_testdb_collection",
		},
		{
			input:    "mongodb://localhost:27017/testdb\\collection",
			expected: "mongodb___localhost_27017_testdb_collection",
		},
	}
	
	for _, tt := range tests {
		result := plugin.sanitizeFilename(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
		}
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
