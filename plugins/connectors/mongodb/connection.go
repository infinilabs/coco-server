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
	p.mu.RLock()
	if client, exists := p.clients[datasourceID]; exists {
		p.mu.RUnlock()
		// Test connection
		if err := client.Ping(context.Background(), readpref.Primary()); err == nil {
			return client, nil
		}
		// Connection failed, remove it
		p.mu.Lock()
		delete(p.clients, datasourceID)
		client.Disconnect(context.Background())
		p.mu.Unlock()
	} else {
		p.mu.RUnlock()
	}

	// Create new client
	client, err := p.createMongoClient(config)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.clients[datasourceID] = client
	p.mu.Unlock()

	return client, nil
}

func (p *Plugin) createMongoClient(config *Config) (*mongo.Client, error) {
	clientOptions := options.Client()

	// Set connection string
	clientOptions.ApplyURI(config.ConnectionURI)

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

	// Set default read preference for better performance
	clientOptions.SetReadPreference(readpref.PrimaryPreferred())

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
