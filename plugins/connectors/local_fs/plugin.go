/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package local_fs

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ConnectorLocalFs = "local_fs"

// Config defines the configuration for the local FS connector.
type Config struct {
	Paths      []string `config:"paths"`
	Extensions []string `config:"extensions"`
}

type Plugin struct {
	connectors.BasePlugin
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init("connector.local_fs", "indexing local filesystem", p)
}

func (p *Plugin) Start() error {
	return p.BasePlugin.Start(connectors.DefaultSyncInterval)
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	p.scanFolders(connector, datasource)
}

func (p *Plugin) scanFolders(connector *common.Connector, datasource *common.DataSource) {
	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		log.Errorf("[%v connector] Failed to create config from data source [%s]: %v", ConnectorLocalFs, datasource.Name, err)
		return
	}

	var obj Config
	if err := cfg.Unpack(&obj); err != nil {
		log.Errorf("[%v connector] Failed to unpack config for data source [%s]: %v", ConnectorLocalFs, datasource.Name, err)
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

		log.Debugf("[%v connector] Scanning path: %s for data source: %s", ConnectorLocalFs, path, datasource.Name)

		err := filepath.WalkDir(path, func(currentPath string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Warnf("[%v connector] Error accessing path %q: %v", ConnectorLocalFs, currentPath, err)
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
				log.Warnf("[%v connector] Failed to get file info for %q: %v", ConnectorLocalFs, currentPath, err)
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
				log.Errorf("[%v connector] Failed to push document to queue for data source [%s]: %v", ConnectorLocalFs, datasource.Name, err)
			}

			return nil
		})

		if err != nil {
			log.Errorf("[%v connector] Error walking the path %q for data source [%s]: %v\n", ConnectorLocalFs, path, datasource.Name, err)
		}
	}
}

func (p *Plugin) Stop() error {
	return nil
}

func (p *Plugin) Name() string {
	return ConnectorLocalFs
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}
