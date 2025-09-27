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

		p.scanPath(path, datasource, extMap)
	}
}

// scanPath performs a single DFS traversal to collect files and determine which folders to save
func (p *Plugin) scanPath(basePath string, datasource *common.DataSource, extMap map[string]bool) {
	// Track which folders contain matching files and collect folder info
	foldersWithMatchingFiles := make(map[string]bool)
	folderInfos := make(map[string]os.FileInfo)

	// Single pass: collect files and folder information
	err := filepath.WalkDir(basePath, func(currentPath string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Warnf("[%v connector] Error accessing path %q: %v", ConnectorLocalFs, currentPath, err)
			return err
		}

		fileInfo, err := d.Info()
		if err != nil {
			log.Warnf("[%v connector] Failed to get file info for %q: %v", ConnectorLocalFs, currentPath, err)
			return nil
		}

		if d.IsDir() {
			// Store folder info for later processing
			folderInfos[currentPath] = fileInfo
		} else {
			// Process file
			fileExt := strings.ToLower(filepath.Ext(currentPath))
			// If no extension filter or file matches filter
			if len(extMap) == 0 || extMap[fileExt] {
				// Mark all parent directories as containing matching files
				p.markParentFoldersAsValid(currentPath, basePath, foldersWithMatchingFiles)

				// Save the file immediately
				p.saveDocument(currentPath, basePath, fileInfo, datasource)
			}
		}

		return nil
	})

	if err != nil {
		log.Errorf("[%v connector] Error walking the path %q for data source [%s]: %v\n", ConnectorLocalFs, basePath, datasource.Name, err)
		return
	}

	// Now process folders: save only those that contain matching files
	for folderPath, folderInfo := range folderInfos {
		if foldersWithMatchingFiles[folderPath] {
			p.saveDocument(folderPath, basePath, folderInfo, datasource)
		} else {
			log.Debugf("[%v connector] Skipping empty folder: %s (no files with matching extensions)", ConnectorLocalFs, folderPath)
		}
	}
}

// markParentFoldersAsValid marks all parent folders of a file as containing matching files
func (p *Plugin) markParentFoldersAsValid(filePath, basePath string, foldersWithMatchingFiles map[string]bool) {
	currentDir := filepath.Dir(filePath)

	for currentDir != basePath && currentDir != "." && currentDir != "/" {
		foldersWithMatchingFiles[currentDir] = true
		currentDir = filepath.Dir(currentDir)
	}

	// Also mark the base path
	foldersWithMatchingFiles[basePath] = true
}

// saveDocument saves a document directly without additional folder content checking
func (p *Plugin) saveDocument(currentPath, basePath string, fileInfo os.FileInfo, datasource *common.DataSource) {
	modTime := fileInfo.ModTime()
	doc := common.Document{
		Source:   common.DataSourceReference{ID: datasource.ID, Type: "connector", Name: datasource.Name},
		Type:     "file",
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
