/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"go.mongodb.org/mongo-driver/bson"
	"time"

	log "github.com/cihub/seelog"
)

// SyncStrategy defines the interface for different synchronization strategies
type SyncStrategy interface {
	BuildFilter(config *Config, collConfig CollectionConfig, datasourceID string, syncManager *SyncManager) bson.M
	ShouldUpdateSyncTime() bool
	GetStrategyName() string
}

// FullSyncStrategy implements full synchronization strategy
type FullSyncStrategy struct{}

func (f *FullSyncStrategy) BuildFilter(config *Config, collConfig CollectionConfig, datasourceID string, syncManager *SyncManager) bson.M {
	filter := bson.M{}

	// Copy base filter from collection configuration
	for k, v := range collConfig.Filter {
		filter[k] = v
	}

	// Full sync strategy - no timestamp filtering, process all documents
	log.Debugf("[mongodb connector] full sync strategy for collection [%s] - processing all documents", collConfig.Name)
	return filter
}

func (f *FullSyncStrategy) ShouldUpdateSyncTime() bool {
	// Full sync doesn't need to track sync time
	return false
}

func (f *FullSyncStrategy) GetStrategyName() string {
	return "full"
}

// IncrementalSyncStrategy implements incremental synchronization strategy
type IncrementalSyncStrategy struct{}

func (i *IncrementalSyncStrategy) BuildFilter(config *Config, collConfig CollectionConfig, datasourceID string, syncManager *SyncManager) bson.M {
	filter := bson.M{}

	// Copy base filter from collection configuration
	for k, v := range collConfig.Filter {
		filter[k] = v
	}

	// Add timestamp filter for incremental sync
	if config.LastModifiedField != "" {
		// Get last sync time from sync manager using datasource ID and collection name
		lastSyncTime := syncManager.GetLastSyncTime(datasourceID, collConfig.Name)
		if !lastSyncTime.IsZero() {
			filter[config.LastModifiedField] = bson.M{"$gt": lastSyncTime}
			log.Debugf("[mongodb connector] incremental sync for collection [%s] - filtering documents newer than %v", 
				collConfig.Name, lastSyncTime)
		} else {
			log.Debugf("[mongodb connector] incremental sync for collection [%s] - no previous sync time, processing all documents", 
				collConfig.Name)
		}
	} else {
		log.Warnf("[mongodb connector] incremental sync strategy specified but LastModifiedField not configured for collection [%s]", 
			collConfig.Name)
	}

	return filter
}

func (i *IncrementalSyncStrategy) ShouldUpdateSyncTime() bool {
	// Incremental sync needs to track sync time
	return true
}

func (i *IncrementalSyncStrategy) GetStrategyName() string {
	return "incremental"
}

// SyncStrategyFactory creates sync strategy instances
type SyncStrategyFactory struct{}

// CreateStrategy creates a sync strategy based on the configuration
func (f *SyncStrategyFactory) CreateStrategy(strategyName string) SyncStrategy {
	switch strategyName {
	case "incremental":
		return &IncrementalSyncStrategy{}
	case "full":
		fallthrough
	default:
		return &FullSyncStrategy{}
	}
}

// GetStrategyName returns the display name for logging purposes
func (f *SyncStrategyFactory) GetStrategyName(strategyName string) string {
	switch strategyName {
	case "incremental":
		return "incremental"
	case "full":
		return "full"
	default:
		return "full (default)"
	}
}
