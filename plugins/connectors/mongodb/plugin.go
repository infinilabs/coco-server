/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mongodb

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"go.mongodb.org/mongo-driver/mongo"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/task"
)

const ConnectorMongoDB = "mongodb"

// TaskStatus represents task execution status
type TaskStatus struct {
	TaskID      string    `json:"task_id"`      // Task ID
	Collection  string    `json:"collection"`   // Collection name
	Status      string    `json:"status"`       // Task status: running, completed, failed, cancelled
	Error       error     `json:"error"`        // Error information (if any)
	CompletedAt time.Time `json:"completed_at"` // Completion time
}

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

	scanCtx, scanCancel := context.WithCancel(parentCtx)
	defer scanCancel()

	// Use framework task scheduling to replace goroutine and sync.WaitGroup
	// Create concurrent scanning tasks for each collection, organized by task group
	taskGroup := "mongodb_scan_" + datasource.ID
	var taskIDs []string

	// Task status monitoring channel
	taskStatusChan := make(chan TaskStatus, len(config.Collections))
	totalTasks := len(config.Collections)

	// Start task status monitoring goroutine, use channel to synchronize task completion status
	go p.monitorTaskStatus(taskGroup, totalTasks, taskStatusChan)

	// Create scanning tasks for all collections
	for _, collConfig := range config.Collections {
		if global.ShuttingDown() {
			break
		}

		// Create concurrent scanning task for each collection
		// Generate a unique task identifier for this collection scan
		uniqueTaskID := fmt.Sprintf("%s_%s_%d", taskGroup, collConfig.Name, time.Now().UnixNano())

		taskID := task.RunWithinGroup(taskGroup, func(ctx context.Context) error {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				log.Debugf("[mongodb connector] task cancelled for collection [%s]", collConfig.Name)
				return ctx.Err()
			default:
			}

			// Execute collection scanning
			err := p.scanCollectionWithContext(scanCtx, client, config, collConfig, datasource)

			// Send task completion status
			// Use unique task identifier to avoid conflicts
			select {
			case taskStatusChan <- TaskStatus{
				TaskID:      uniqueTaskID, // Use unique task identifier
				Collection:  collConfig.Name,
				Status:      "completed",
				Error:       err,
				CompletedAt: time.Now(),
			}:
			default:
				log.Warnf("[mongodb connector] task status channel full, status for collection [%s] not sent", collConfig.Name)
			}

			return err
		})

		if taskID != "" {
			taskIDs = append(taskIDs, taskID)
		}
	}

	// Wait for all tasks to complete or timeout
	if len(taskIDs) > 0 {
		log.Debugf("[mongodb connector] launched %d collection scan tasks in group [%s]", len(taskIDs), taskGroup)

		// Wait for tasks to complete or timeout
		timeout := time.After(30 * time.Minute) // 30 minutes timeout

		// Wait for all tasks to complete
		completedCount := 0
		for completedCount < totalTasks {
			select {
			case <-timeout:
				log.Warnf("[mongodb connector] timeout waiting for tasks to complete, completed: %d/%d", completedCount, totalTasks)
				return
			case status := <-taskStatusChan:
				completedCount++
				if status.Error != nil {
					log.Warnf("[mongodb connector] task for collection [%s] completed with error: %v", status.Collection, status.Error)
				} else {
					log.Debugf("[mongodb connector] task for collection [%s] completed successfully (%d/%d)", status.Collection, completedCount, totalTasks)
				}
			case <-scanCtx.Done():
				log.Debugf("[mongodb connector] scan context cancelled, stopping task monitoring")
				return
			}
		}

		log.Infof("[mongodb connector] all %d collection scan tasks completed successfully", totalTasks)
	}

	log.Infof("[mongodb connector] finished scanning datasource [%s]", datasource.Name)
}

// monitorTaskStatus monitors task execution status
func (p *Plugin) monitorTaskStatus(taskGroup string, totalTasks int, statusChan <-chan TaskStatus) {
	log.Debugf("[mongodb connector] starting task status monitoring for group [%s], total tasks: %d", taskGroup, totalTasks)

	completedTasks := 0
	failedTasks := 0
	startTime := time.Now()

	// Create task status mapping
	taskStatusMap := make(map[string]*TaskStatus)

	for status := range statusChan {
		// Update task status
		taskStatusMap[status.TaskID] = &status

		if status.Status == "completed" {
			completedTasks++
			if status.Error != nil {
				failedTasks++
				log.Warnf("[mongodb connector] task [%s] for collection [%s] completed with error: %v",
					status.TaskID, status.Collection, status.Error)
			} else {
				log.Debugf("[mongodb connector] task [%s] for collection [%s] completed successfully",
					status.TaskID, status.Collection)
			}
		}

		// Record progress
		progress := float64(completedTasks) / float64(totalTasks) * 100
		log.Debugf("[mongodb connector] task progress: %d/%d (%.1f%%) completed, %d failed",
			completedTasks, totalTasks, progress, failedTasks)

		// Check if all tasks are completed
		if completedTasks >= totalTasks {
			duration := time.Since(startTime)
			log.Infof("[mongodb connector] all tasks in group [%s] completed in %v, success: %d, failed: %d",
				taskGroup, duration, completedTasks-failedTasks, failedTasks)
			break
		}
	}

	// Generate task execution report
	p.generateTaskReport(taskGroup, taskStatusMap, totalTasks, startTime)
}

// generateTaskReport generates task execution report
func (p *Plugin) generateTaskReport(taskGroup string, taskStatusMap map[string]*TaskStatus, totalTasks int, startTime time.Time) {
	duration := time.Since(startTime)
	successCount := 0
	failedCount := 0

	for _, status := range taskStatusMap {
		if status.Error != nil {
			failedCount++
		} else {
			successCount++
		}
	}

	// Record detailed execution report
	log.Infof("[mongodb connector] task group [%s] execution report:", taskGroup)
	log.Infof("[mongodb connector]   - Total tasks: %d", totalTasks)
	log.Infof("[mongodb connector]   - Successful: %d", successCount)
	log.Infof("[mongodb connector]   - Failed: %d", failedCount)
	log.Infof("[mongodb connector]   - Duration: %v", duration)
	log.Infof("[mongodb connector]   - Average time per task: %v", duration/time.Duration(totalTasks))

	// If there are failed tasks, record detailed information
	if failedCount > 0 {
		log.Warnf("[mongodb connector] failed tasks details:")
		for _, status := range taskStatusMap {
			if status.Error != nil {
				log.Warnf("[mongodb connector]   - Collection [%s]: %v", status.Collection, status.Error)
			}
		}
	}
}
