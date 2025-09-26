/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package local_fs

import (
	"fmt"
	"io/fs"
	"os"
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

	// Deduplicate paths to avoid scanning ancestor directories multiple times
	deduplicatedPaths := deduplicatePaths(obj.Paths)
	log.Debugf("[%v connector] Original paths: %v, deduplicated paths: %v", ConnectorLocalFs, obj.Paths, deduplicatedPaths)

	for _, path := range deduplicatedPaths {
		if global.ShuttingDown() {
			break
		}

		log.Debugf("[%v connector] Scanning path: %s for data source: %s", ConnectorLocalFs, path, datasource.Name)

		// First, create a document for the root path itself
		if rootInfo, err := os.Stat(path); err == nil {
			p.createAndSaveDocument(path, path, rootInfo, datasource, extMap)
		} else {
			log.Warnf("[%v connector] Failed to get root path info for %q: %v", ConnectorLocalFs, path, err)
		}

		err := filepath.WalkDir(path, func(currentPath string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Warnf("[%v connector] Error accessing path %q: %v", ConnectorLocalFs, currentPath, err)
				return err
			}

			// Skip the root path since we already processed it
			if currentPath == path {
				return nil
			}

			// Skip file while getting info error
			fileInfo, err := d.Info()
			if err != nil {
				log.Warnf("[%v connector] Failed to get file info for %q: %v", ConnectorLocalFs, currentPath, err)
				return nil
			}

			p.createAndSaveDocument(currentPath, path, fileInfo, datasource, extMap)
			return nil
		})

		if err != nil {
			log.Errorf("[%v connector] Error walking the path %q for data source [%s]: %v\n", ConnectorLocalFs, path, datasource.Name, err)
		}
	}
}

// createAndSaveDocument creates a document from file info and saves it to the queue
func (p *Plugin) createAndSaveDocument(currentPath, basePath string, fileInfo os.FileInfo, datasource *common.DataSource, extMap map[string]bool) {
	// Check file extension name for non-directories
	if !fileInfo.IsDir() {
		fileExt := strings.ToLower(filepath.Ext(currentPath))
		// Extension name not matched
		if len(extMap) > 0 && !extMap[fileExt] {
			return
		}
	}

	modTime := fileInfo.ModTime()
	doc := common.Document{
		Source:   common.DataSourceReference{ID: datasource.ID, Type: "connector", Name: datasource.Name},
		Type:     "local_fs",
		Category: filepath.Dir(currentPath),
		Content:  "", // skip content
		URL:      currentPath,
		Size:     int(fileInfo.Size()),
	}
	doc.System = datasource.System
	if doc.System == nil {
		doc.System = util.MapStr{}
	}

	if fileInfo.IsDir() {
		doc.Icon = "folder"
		doc.Type = "folder"
	} else {
		doc.Icon = "file"
	}

	if currentPath == basePath {
		doc.Title = strings.Trim(basePath, string(filepath.Separator))
	} else {
		doc.Title = fileInfo.Name()
	}

	// Build parent category array from file path
	parentCategoryArray := buildParentCategoryArray(currentPath, basePath)
	if len(parentCategoryArray) > 0 {
		categoryPath := common.GetFullPathForCategories(parentCategoryArray)
		doc.Category = categoryPath
		doc.Categories = parentCategoryArray
		doc.System[common.SystemHierarchyPathKey] = categoryPath
	} else {
		doc.System[common.SystemHierarchyPathKey] = "/"
	}

	// Unify creation and modification time
	doc.Created = &modTime
	doc.Updated = &modTime

	doc.ID = util.MD5digest(fmt.Sprintf("%s-%s", datasource.ID, currentPath))

	data := util.MustToJSONBytes(doc)
	if err := queue.Push(p.Queue, data); err != nil {
		log.Errorf("[%v connector] Failed to push document to queue for data source [%s]: %v", ConnectorLocalFs, datasource.Name, err)
	}
}

// buildParentCategoryArray constructs a hierarchical path array for the file
// based on its path relative to the base scanning path
func buildParentCategoryArray(currentPath, basePath string) []string {
	if currentPath == basePath {
		return nil
	}

	var categories []string

	// Start with the basePath as the first category (trimmed of slashes)
	basePathTrimmed := strings.Trim(basePath, string(filepath.Separator))
	if basePathTrimmed != "" {
		categories = append(categories, basePathTrimmed)
	}

	// Get the relative path from the base path
	relPath, err := filepath.Rel(basePath, currentPath)
	if err != nil {
		_ = log.Warnf("[%v connector] Failed to get relative path for %q from base %q: %v", ConnectorLocalFs, currentPath, basePath, err)
		return categories // Return just the basePath
	}

	// Clean the path and split into components
	relPath = filepath.Clean(relPath)

	// If it's the current directory ".", return just the basePath
	if relPath == "." {
		return categories
	}

	// Split the path into components, filtering out empty ones
	parts := strings.Split(relPath, string(filepath.Separator))
	for _, part := range parts {
		if part != "" && part != "." {
			categories = append(categories, part)
		}
	}

	if len(categories) > 0 {
		return categories[:len(categories)-1]
	}

	return categories
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
