/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package local_fs

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/api"
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

// Config defines the configuration for the local FS connector.
type Config struct {
	Paths      []string `config:"paths"`
	Extensions []string `config:"extensions"`
}

type Plugin struct {
	api.Handler
	Enabled  bool               `config:"enabled"`
	Queue    *queue.QueueConfig `config:"queue"`
	Interval string             `config:"interval"`
	PageSize int                `config:"page_size"`
}

func (this *Plugin) Setup() {
	ok, err := env.ParseConfig("connector.local_fs", &this)
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	if !this.Enabled {
		return
	}

	if this.PageSize <= 0 {
		this.PageSize = 1000
	}

	if this.Queue == nil {
		this.Queue = &queue.QueueConfig{Name: "indexing_documents"}
	}

	this.Queue = queue.SmartGetOrInitConfig(this.Queue)
}

func (this *Plugin) Start() error {
	if !this.Enabled {
		return nil
	}

	task.RegisterScheduleTask(task.ScheduleTask{
		ID:          util.GetUUID(),
		Group:       "connectors",
		Singleton:   true,
		Interval:    util.GetDurationOrDefault(this.Interval, time.Second*30).String(),
		Description: "indexing local filesystem",
		Task: func(ctx context.Context) {
			connector := common.Connector{}
			connector.ID = "local_fs"
			exists, err := orm.Get(&connector)
			if !exists || err != nil {
				panic(errors.Errorf("local_fs connector not found or error occurred, skipping task:%v", err))
			}

			q := orm.Query{}
			q.Size = this.PageSize
			q.Conds = orm.And(orm.Eq("connector.id", connector.ID), orm.Eq("sync_enabled", true))
			var results []common.DataSource

			err, _ = orm.SearchWithJSONMapper(&results, &q)
			if err != nil {
				log.Errorf("Failed to search for local_fs datasource: %v", err)
				panic(err)
			}

			for _, item := range results {
				toSync, err := connectors.CanDoSync(item)
				if err != nil {
					log.Errorf("error checking sync status with datasource [%s]: %v", item.Name, err)
					continue
				}
				if !toSync {
					continue
				}
				log.Debugf("ID: %s, Name: %s, Other: %s", item.ID, item.Name, util.MustToJSON(item))
				this.scanFolders(&connector, &item)
			}
		},
	})

	return nil
}

func (p *Plugin) scanFolders(connector *common.Connector, datasource *common.DataSource) {
	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		log.Errorf("Failed to create config from data source [%s]: %v", datasource.Name, err)
		return
	}

	var obj Config
	if err := cfg.Unpack(&obj); err != nil {
		log.Errorf("Failed to unpack config for data source [%s]: %v", datasource.Name, err)
		return
	}

	// A map for extensions
	extMap := make(map[string]bool)
	for _, ext := range obj.Extensions {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extMap[strings.ToLower(ext)] = true
	}

	for _, path := range obj.Paths {
		if global.ShuttingDown() {
			break
		}

		log.Debugf("Scanning path: %s for data source: %s", path, datasource.Name)

		err := filepath.WalkDir(path, func(currentPath string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Warnf("Error accessing path %q: %v", currentPath, err)
				return err
			}
			if d.IsDir() {
				return nil
			}

			// Check file extension name
			fileExt := strings.ToLower(filepath.Ext(currentPath))

			// Extension name not matched
			if len(extMap) > 0 && !extMap[fileExt] {
				return nil
			}

			// Skip file while getting info error
			fileInfo, err := d.Info()
			if err != nil {
				log.Warnf("Failed to get file info for %q: %v", currentPath, err)
				return nil
			}

			modTime := fileInfo.ModTime()
			doc := common.Document{
				Source:   common.DataSourceReference{ID: datasource.ID, Type: "connector", Name: datasource.Name},
				Type:     "local_fs",
				Icon:     "default",
				Title:    fileInfo.Name(),
				Category: filepath.Dir(currentPath),
				Content:  "", // skip content
				URL:      currentPath,
				Size:     int(fileInfo.Size()),
			}
			// Unify creation and modification time
			doc.Created = &modTime
			doc.Updated = &modTime
			doc.ID = util.MD5digest(fmt.Sprintf("%s-%s", datasource.ID, currentPath))

			data := util.MustToJSONBytes(doc)
			if err := queue.Push(p.Queue, data); err != nil {
				log.Errorf("Failed to push document to queue for data source [%s]: %v", datasource.Name, err)
			}

			return nil
		})

		if err != nil {
			log.Errorf("Error walking the path %q for data source [%s]: %v\n", path, datasource.Name, err)
		}
	}
}

func (this *Plugin) Stop() error {
	return nil
}

func (this *Plugin) Name() string {
	return "local_fs"
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}
