/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"context"
	"sync"

	log "github.com/cihub/seelog"
	"go.mongodb.org/mongo-driver/mongo"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
)

const ConnectorMongoDB = "mongodb"

type Plugin struct {
	connectors.BasePlugin
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	clients     map[string]*mongo.Client
	syncManager *SyncManager
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}

func (p *Plugin) Name() string {
	return ConnectorMongoDB
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init("connector.mongodb", "indexing mongodb documents", p)
}

func (p *Plugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.clients = make(map[string]*mongo.Client)
	p.syncManager = NewSyncManager()
	return p.BasePlugin.Start(connectors.DefaultSyncInterval)
}

func (p *Plugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
	}

	// Clean up all connections
	for _, client := range p.clients {
		if client != nil {
			client.Disconnect(context.Background())
		}
	}
	p.clients = nil

	return nil
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	// Get the parent context
	p.mu.RLock()
	parentCtx := p.ctx
	p.mu.RUnlock()

	// Check if the plugin has been stopped
	if parentCtx == nil {
		log.Warnf("[mongodb connector] plugin is stopped, skipping scan for datasource [%s]", datasource.Name)
		return
	}

	config := &Config{}
	err := connectors.ParseConnectorConfigure(connector, datasource, config)
	if err != nil {
		log.Errorf("[mongodb connector] parsing configuration failed: %v", err)
		return
	}

	// Validate configuration
	if err := p.validateConfig(config); err != nil {
		log.Errorf("[mongodb connector] invalid configuration for datasource [%s]: %v", datasource.Name, err)
		return
	}

	// Set default values
	p.setDefaultConfig(config)

	log.Debugf("[mongodb connector] handling datasource: %v", config)

	client, err := p.getOrCreateClient(datasource.ID, config)
	if err != nil {
		log.Errorf("[mongodb connector] failed to create client for datasource [%s]: %v", datasource.Name, err)
		p.handleConnectionError(err, datasource.ID)
		return
	}

	// Health check
	if err := p.healthCheck(client); err != nil {
		log.Errorf("[mongodb connector] health check failed for datasource [%s]: %v", datasource.Name, err)
		p.handleConnectionError(err, datasource.ID)
		return
	}

	// Simple sequential scanning for each collection
	// Since the connector is already wrapped in a background task, we use simple implementation
	for _, collConfig := range config.Collections {
		if global.ShuttingDown() {
			log.Debugf("[mongodb connector] shutting down, stopping scan for collection [%s]", collConfig.Name)
			break
		}

		// Check if context is cancelled
		select {
		case <-parentCtx.Done():
			log.Debugf("[mongodb connector] context cancelled, stopping scan for collection [%s]", collConfig.Name)
			return
		default:
		}

		log.Debugf("[mongodb connector] scanning collection [%s]", collConfig.Name)

		// Execute collection scanning
		if err := p.scanCollectionWithContext(parentCtx, client, config, collConfig, datasource); err != nil {
			log.Errorf("[mongodb connector] failed to scan collection [%s]: %v", collConfig.Name, err)
			// Continue with next collection instead of failing completely
			continue
		}

		log.Debugf("[mongodb connector] successfully scanned collection [%s]", collConfig.Name)
	}

	log.Infof("[mongodb connector] finished scanning datasource [%s]", datasource.Name)
}
