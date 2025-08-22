/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/cihub/seelog"
)

// SyncState represents the synchronization state for a specific datasource and collection
type SyncState struct {
	DatasourceID   string    `json:"datasource_id"`
	CollectionName string    `json:"collection_name"`
	LastSyncTime   time.Time `json:"last_sync_time"`
	LastModified   time.Time `json:"last_modified"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SyncManager manages the synchronization state for MongoDB collections
type SyncManager struct {
	mu       sync.RWMutex
	states   map[string]*SyncState // key: datasourceID_collectionName
	storageDir string
}

// NewSyncManager creates a new sync manager instance
func NewSyncManager() *SyncManager {
	return &SyncManager{
		states:    make(map[string]*SyncState),
		storageDir: getDefaultSyncStorageDir(),
	}
}

// GetSyncKey generates a unique key for datasource and collection
func (sm *SyncManager) GetSyncKey(datasourceID, collectionName string) string {
	return fmt.Sprintf("%s_%s", datasourceID, collectionName)
}

// GetLastSyncTime retrieves the last sync time for a specific datasource and collection
func (sm *SyncManager) GetLastSyncTime(datasourceID, collectionName string) time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	key := sm.GetSyncKey(datasourceID, collectionName)
	
	// First check in-memory cache
	if state, exists := sm.states[key]; exists {
		return state.LastSyncTime
	}

	// If not in memory, try to load from persistent storage
	state := sm.loadFromStorage(datasourceID, collectionName)
	if state != nil {
		sm.states[key] = state
		return state.LastSyncTime
	}

	return time.Time{} // Return zero time if no sync state found
}

// UpdateLastSyncTime updates the last sync time for a specific datasource and collection
func (sm *SyncManager) UpdateLastSyncTime(datasourceID, collectionName string, syncTime, lastModified time.Time) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := sm.GetSyncKey(datasourceID, collectionName)
	
	state := &SyncState{
		DatasourceID:   datasourceID,
		CollectionName: collectionName,
		LastSyncTime:   syncTime,
		LastModified:   lastModified,
		UpdatedAt:      time.Now(),
	}

	// Update in-memory cache
	sm.states[key] = state

	// Persist to storage
	return sm.saveToStorage(state)
}

// GetLastModifiedTime retrieves the last modified time for a specific datasource and collection
func (sm *SyncManager) GetLastModifiedTime(datasourceID, collectionName string) time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	key := sm.GetSyncKey(datasourceID, collectionName)
	if state, exists := sm.states[key]; exists {
		return state.LastModified
	}

	// Try to load from storage
	state := sm.loadFromStorage(datasourceID, collectionName)
	if state != nil {
		sm.states[key] = state
		return state.LastModified
	}

	return time.Time{}
}

// loadFromStorage loads sync state from persistent storage
func (sm *SyncManager) loadFromStorage(datasourceID, collectionName string) *SyncState {
	key := sm.GetSyncKey(datasourceID, collectionName)
	filename := sanitizeFilename(key) + ".json"
	filepath := filepath.Join(sm.storageDir, filename)

	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, no previous sync
		}
		log.Warnf("[mongodb connector] failed to read sync state file %s: %v", filepath, err)
		return nil
	}

	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Warnf("[mongodb connector] failed to parse sync state file %s: %v", filepath, err)
		return nil
	}

	return &state
}

// saveToStorage saves sync state to persistent storage
func (sm *SyncManager) saveToStorage(state *SyncState) error {
	// Ensure storage directory exists
	if err := os.MkdirAll(sm.storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create sync storage directory: %v", err)
	}

	key := sm.GetSyncKey(state.DatasourceID, state.CollectionName)
	filename := sanitizeFilename(key) + ".json"
	filepath := filepath.Join(sm.storageDir, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sync state: %v", err)
	}

	// Write to file atomically (write to temp file first, then rename)
	tempFile := filepath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp sync state file: %v", err)
	}

	if err := os.Rename(tempFile, filepath); err != nil {
		// Clean up temp file on error
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp sync state file: %v", err)
	}

	return nil
}

// getDefaultSyncStorageDir returns the default directory for storing sync state files
func getDefaultSyncStorageDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".coco", "mongodb", "sync")
}

// sanitizeFilename sanitizes a string to be used as a filename
func sanitizeFilename(name string) string {
	// Replace invalid characters with underscores
	invalid := []rune{'/', '\\', ':', '*', '?', '"', '<', '>', '|'}
	result := []rune(name)
	for i, r := range result {
		for _, inv := range invalid {
			if r == inv {
				result[i] = '_'
				break
			}
		}
	}
	return string(result)
}
