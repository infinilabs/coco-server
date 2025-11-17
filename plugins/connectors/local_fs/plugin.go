/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package local_fs

import (
	"fmt"
	"infini.sh/coco/core"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

const ConnectorLocalFs = "local_fs"

// Config defines the configuration for the local FS connector.
type Config struct {
	Paths      []string `config:"paths"`
	Extensions []string `config:"extensions"`
}

type Plugin struct {
	cmn.ConnectorProcessorBase
}

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorLocalFs, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	runner := Plugin{}
	runner.Init(c, &runner)
	return &runner, nil
}

func (p *Plugin) Name() string {
	return ConnectorLocalFs
}

func (p *Plugin) Fetch(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	cfg := Config{}
	p.MustParseConfig(datasource, &cfg)

	log.Debugf("[%s connector] handling datasource: %v", ConnectorLocalFs, cfg)

	// A map for extensions
	extMap := make(map[string]bool)
	for _, ext := range cfg.Extensions {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extMap[strings.ToLower(ext)] = true
	}

	// Deduplicate paths to avoid scanning ancestor directories multiple times
	deduplicatedPaths := deduplicatePaths(cfg.Paths)
	log.Debugf("[%s connector] Original paths: %v, deduplicated paths: %v", ConnectorLocalFs, cfg.Paths, deduplicatedPaths)

	for _, path := range deduplicatedPaths {
		if global.ShuttingDown() {
			break
		}

		log.Debugf("[%s connector] Scanning path: %s for data source: %s", ConnectorLocalFs, path, datasource.Name)

		p.scanPath(ctx, path, connector, datasource, extMap)
	}

	log.Infof("[%s connector] finished fetching datasource [%s]", ConnectorLocalFs, datasource.Name)
	return nil
}

// scanPath performs a single DFS traversal to collect files and determine which folders to save
func (p *Plugin) scanPath(ctx *pipeline.Context, basePath string, connector *core.Connector, datasource *core.DataSource, extMap map[string]bool) {
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
				p.saveDocument(ctx, currentPath, basePath, fileInfo, connector, datasource)
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
			p.saveDocument(ctx, folderPath, basePath, folderInfo, connector, datasource)
		} else {
			log.Debugf("[%s connector] Skipping empty folder: %s (no files with matching extensions)", ConnectorLocalFs, folderPath)
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
func (p *Plugin) saveDocument(ctx *pipeline.Context, currentPath, basePath string, fileInfo os.FileInfo, connector *core.Connector, datasource *core.DataSource) {
	modTime := fileInfo.ModTime()
	doc := core.Document{
		Source:   core.DataSourceReference{ID: datasource.ID, Type: "connector", Name: datasource.Name},
		Type:     connectors.TypeFile,
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
		doc.Icon = connectors.IconFolder
		doc.Type = connectors.TypeFolder
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

	p.Collect(ctx, connector, datasource, doc)
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
