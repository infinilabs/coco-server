/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"context"
	"time"

	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func (p *Plugin) getOrCreateClient(datasourceID string, config *Config) (*mongo.Client, error) {
	// First check: use read lock to check if connection exists and is valid
	p.mu.RLock()
	if client, exists := p.clients[datasourceID]; exists {
		// Test if the connection is still valid
		if err := client.Ping(context.Background(), readpref.Primary()); err == nil {
			p.mu.RUnlock()
			return client, nil
		}
		p.mu.RUnlock()
	} else {
		p.mu.RUnlock()
	}

	// Acquire write lock to prepare for creating new connection
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Second check: re-check connection status under write lock protection
	// Prevents connection overwrite when multiple goroutines create connections simultaneously
	if client, exists := p.clients[datasourceID]; exists {
		// Test connection again (may have been fixed by another goroutine)
		if err := client.Ping(context.Background(), readpref.Primary()); err == nil {
			return client, nil
		}
		// Connection indeed failed, remove it and disconnect
		delete(p.clients, datasourceID)
		client.Disconnect(context.Background())
	}

	// Create new MongoDB client connection
	client, err := p.createMongoClient(config)
	if err != nil {
		return nil, err
	}

	// Store new connection in the connection pool
	p.clients[datasourceID] = client

	return client, nil
}

func (p *Plugin) createMongoClient(config *Config) (*mongo.Client, error) {
	clientOptions := options.Client()

	// Set connection string
	clientOptions.ApplyURI(config.ConnectionURI)

	// Set authentication database if specified
	if config.AuthDatabase != "" {
		clientOptions.SetAuth(options.Credential{
			AuthSource: config.AuthDatabase,
		})
	}

	// Connection pool configuration
	if config.MaxPoolSize > 0 {
		clientOptions.SetMaxPoolSize(uint64(config.MaxPoolSize))
	}

	// Timeout configuration
	if config.Timeout != "" {
		if timeout, err := time.ParseDuration(config.Timeout); err == nil {
			clientOptions.SetServerSelectionTimeout(timeout)
			clientOptions.SetConnectTimeout(timeout)
		}
	}

	// Configure cluster-specific settings
	switch config.ClusterType {
	case "replica_set":
		// For replica sets, prefer secondary nodes for read operations to distribute load
		clientOptions.SetReadPreference(readpref.SecondaryPreferred())
		// Enable retry writes for replica sets
		clientOptions.SetRetryWrites(true)
		// Set write concern for replica sets
		clientOptions.SetWriteConcern(mongo.WriteConcern{
			W:        "majority",
			J:        true,
			WTimeout: 10 * time.Second,
		})
	case "sharded":
		// For sharded clusters, use primary for writes and nearest for reads
		clientOptions.SetReadPreference(readpref.Nearest())
		// Enable retry writes for sharded clusters
		clientOptions.SetRetryWrites(true)
		// Set write concern for sharded clusters
		clientOptions.SetWriteConcern(mongo.WriteConcern{
			W:        "majority",
			J:        true,
			WTimeout: 10 * time.Second,
		})
	default:
		// For standalone instances, use primary preferred
		clientOptions.SetReadPreference(readpref.PrimaryPreferred())
	}

	return mongo.Connect(context.Background(), clientOptions)
}

func (p *Plugin) healthCheck(client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return client.Ping(ctx, readpref.Primary())
}

func (p *Plugin) handleConnectionError(err error, datasourceID string) {
	// Clean up failed connection
	p.mu.Lock()
	if client, exists := p.clients[datasourceID]; exists {
		client.Disconnect(context.Background())
		delete(p.clients, datasourceID)
	}
	p.mu.Unlock()

	// Log error and wait for retry
	log.Errorf("[mongodb connector] connection error: %v", err)
	time.Sleep(time.Second * 30) // Backoff retry
}
