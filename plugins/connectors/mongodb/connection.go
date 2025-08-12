/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	log "github.com/cihub/seelog"
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

	// Set connection string or detailed configuration
	if config.ConnectionURI != "" {
		clientOptions.ApplyURI(config.ConnectionURI)
	} else {
		uri := p.buildConnectionURI(config)
		clientOptions.ApplyURI(uri)
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

	// TLS configuration
	if config.EnableTLS {
		tlsConfig := p.buildTLSConfig(config)
		clientOptions.SetTLSConfig(tlsConfig)
	}

	// Read preference setting
	if config.ReadPreference != "" {
		readPref := p.buildReadPreference(config.ReadPreference)
		clientOptions.SetReadPreference(readPref)
	}

	return mongo.Connect(context.Background(), clientOptions)
}

func (p *Plugin) buildConnectionURI(config *Config) string {
	var uri strings.Builder
	uri.WriteString("mongodb://")

	// Authentication
	if config.Username != "" {
		uri.WriteString(config.Username)
		if config.Password != "" {
			uri.WriteString(":")
			uri.WriteString(config.Password)
		}
		uri.WriteString("@")
	}

	// Host and port
	host := config.Host
	if host == "" {
		host = "localhost"
	}
	port := config.Port
	if port == 0 {
		port = 27017
	}
	uri.WriteString(fmt.Sprintf("%s:%d", host, port))

	// Database
	if config.Database != "" {
		uri.WriteString("/")
		uri.WriteString(config.Database)
	}

	// Query parameters
	var params []string
	if config.AuthDatabase != "" {
		params = append(params, "authSource="+config.AuthDatabase)
	}
	if config.ReplicaSet != "" {
		params = append(params, "replicaSet="+config.ReplicaSet)
	}
	if config.EnableTLS {
		params = append(params, "ssl=true")
		if config.TLSInsecure {
			params = append(params, "sslInsecure=true")
		}
	}

	if len(params) > 0 {
		uri.WriteString("?")
		uri.WriteString(strings.Join(params, "&"))
	}

	return uri.String()
}

func (p *Plugin) buildTLSConfig(config *Config) *tls.Config {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLSInsecure,
	}

	// Add certificate files if provided
	// Implementation would depend on specific TLS requirements

	return tlsConfig
}

func (p *Plugin) buildReadPreference(preference string) *readpref.ReadPref {
	switch strings.ToLower(preference) {
	case "primary":
		return readpref.Primary()
	case "secondary":
		return readpref.Secondary()
	case "nearest":
		return readpref.Nearest()
	case "primarypreferred":
		return readpref.PrimaryPreferred()
	case "secondarypreferred":
		return readpref.SecondaryPreferred()
	default:
		return readpref.Primary()
	}
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
