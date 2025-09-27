/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package s3

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/minio/minio-go/v7"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ConnectorS3 = "s3"

type Plugin struct {
	connectors.BasePlugin
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init("connector.s3", "indexing S3 objects", p)
}

func (p *Plugin) Start() error {
	return p.BasePlugin.Start(connectors.DefaultSyncInterval)
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	p.getBucketObjects(connector, datasource)
}

func (p *Plugin) getBucketObjects(connector *common.Connector, datasource *common.DataSource) {
	cfg := Config{}
	err := connectors.ParseConnectorConfigure(connector, datasource, &cfg)
	if err != nil {
		_ = log.Errorf("[%v connector] Parsing connector configuration failed: %v", ConnectorS3, err)
		panic(err)
	}

	log.Debugf("[%v connector] Handling datasource: %v", ConnectorS3, cfg)

	if cfg.Bucket == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		_ = log.Errorf("[%v connector] Missing required configuration for datasource [%s]: bucket, access_key_id, or secret_access_key", ConnectorS3, datasource.Name)
		return
	}

	// A map for extensions
	extMap := make(map[string]bool)
	for _, ext := range cfg.Extensions {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extMap[strings.ToLower(ext)] = true
	}

	// Track all unique folder paths that contain matching objects
	foldersWithMatchingFiles := make(map[string]bool)

	objectVisitor := func(obj minio.ObjectInfo) {
		// Extension name not matched
		if len(extMap) > 0 {
			index := strings.LastIndex(obj.Key, ".")
			if index < 0 || !extMap[strings.ToLower(obj.Key[index:])] {
				return
			}
		}

		// Mark all parent folders as containing matching files
		p.markParentFoldersAsValid(obj.Key, foldersWithMatchingFiles)

		// Create file document using helper
		parentCategoryArray := buildParentCategoryArray(obj.Key)
		title := filepath.Base(obj.Key)
		url := fmt.Sprintf("%s://%s.%s/%s", cfg.Schema(), cfg.Bucket, cfg.Endpoint, obj.Key)
		idSuffix := fmt.Sprintf("%s-%s", cfg.Bucket, obj.Key)

		doc := common.Document{
			Type:    "file",
			Icon:    "file",
			Title:   title,
			Content: "",
			URL:     url,
			Size:    int(obj.Size),
		}
		p.documentWithHierarchy(&doc, parentCategoryArray, datasource, idSuffix)

		// Initialize Metadata if it's nil
		if doc.Metadata == nil {
			doc.Metadata = make(map[string]interface{})
		}

		// Add S3-specific metadata
		for k, v := range obj.Metadata {
			doc.Metadata[k] = v
		}

		for k, v := range obj.UserMetadata {
			doc.Metadata[k] = v
		}

		for k, v := range obj.UserTags {
			doc.Metadata[k] = v
		}

		doc.Owner = &common.UserInfo{
			UserID:   obj.Owner.ID,
			UserName: obj.Owner.DisplayName,
		}

		// S3 not provides creation time
		doc.Created = &obj.LastModified
		doc.Updated = &obj.LastModified

		if global.Env().IsDebug {
			data := util.MustToJSONBytes(doc)
			log.Tracef("[%v connector] Queuing document: %s", ConnectorS3, string(data))
		}

		p.saveDocument(doc, datasource)
	}

	handler, err := NewMinioHandler(cfg)
	if err != nil {
		log.Infof("[%v connector] Failed to init minio client for datasource [%s]: %v", ConnectorS3, datasource.Name, err)
		panic(err)
	}

	// process with each object
	handler.ListObjects(context.TODO(), objectVisitor)

	// Now create folder documents for all folders that contain matching files
	p.createFolderDocuments(foldersWithMatchingFiles, datasource, cfg)

	log.Infof("[%v connector] Finished list objects from bucket [%s] of datasource [%s]. ", ConnectorS3, cfg.Bucket, datasource.Name)
}

// buildParentCategoryArray constructs a hierarchical path array for the S3 object
// based on its key, excluding the bucket name
func buildParentCategoryArray(objectKey string) []string {
	if objectKey == "" {
		return nil
	}

	var categories []string

	// Clean the object key path and split into components
	objectKey = filepath.Clean(objectKey)

	// Use forward slash for S3 keys (always use / regardless of OS)
	objectKey = strings.ReplaceAll(objectKey, "\\", "/")

	// Split the path into components, filtering out empty ones
	parts := strings.Split(objectKey, "/")
	for _, part := range parts {
		if part != "" && part != "." {
			categories = append(categories, part)
		}
	}

	// Return all parts except the last one (the file name)
	if len(categories) > 1 {
		return categories[:len(categories)-1]
	}

	return nil // Return nil if there are no parent folders
}

// markParentFoldersAsValid marks all parent folders of an S3 object as containing matching files
func (p *Plugin) markParentFoldersAsValid(objectKey string, foldersWithMatchingFiles map[string]bool) {
	if objectKey == "" {
		return
	}

	// Clean the object key path
	objectKey = filepath.Clean(objectKey)
	objectKey = strings.ReplaceAll(objectKey, "\\", "/")

	// Split into path components
	parts := strings.Split(objectKey, "/")

	// Build each folder path and mark it as valid
	currentPath := ""
	for _, part := range parts[:len(parts)-1] { // Exclude the filename
		if part != "" && part != "." {
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = currentPath + "/" + part
			}
			foldersWithMatchingFiles[currentPath] = true
		}
	}
}

// createFolderDocuments creates document entries for all folders that contain matching files
func (p *Plugin) createFolderDocuments(foldersWithMatchingFiles map[string]bool, datasource *common.DataSource, cfg Config) {
	for folderPath := range foldersWithMatchingFiles {
		if global.ShuttingDown() {
			return
		}
		p.saveFolder(folderPath, datasource, cfg)
	}
}

// documentWithHierarchy creates a document with proper hierarchy settings
func (p *Plugin) documentWithHierarchy(doc *common.Document, parentCategoryArray []string, datasource *common.DataSource, idSuffix string) {
	doc.Source = common.DataSourceReference{
		ID:   datasource.ID,
		Type: "connector",
		Name: datasource.Name,
	}
	doc.System = datasource.System
	if doc.System == nil {
		doc.System = util.MapStr{}
	}

	// Set hierarchy information
	if len(parentCategoryArray) > 0 {
		categoryPath := common.GetFullPathForCategories(parentCategoryArray)
		doc.Category = categoryPath
		doc.Categories = parentCategoryArray
		doc.System[common.SystemHierarchyPathKey] = categoryPath
	} else {
		// This is a top-level item, set parent_path to '/'
		doc.System[common.SystemHierarchyPathKey] = "/"
		doc.Category = "/"
	}
	doc.ID = util.MD5digest(fmt.Sprintf("%s-%s", datasource.ID, idSuffix))
}

// saveDocument pushes a document to the queue
func (p *Plugin) saveDocument(doc common.Document, datasource *common.DataSource) {
	data := util.MustToJSONBytes(doc)
	if err := queue.Push(p.Queue, data); err != nil {
		_ = log.Errorf("[%v connector] Failed to push document to queue for datasource [%s]: %v", ConnectorS3, datasource.Name, err)
	}
}

// saveFolder creates and saves a document for a folder
func (p *Plugin) saveFolder(folderPath string, datasource *common.DataSource, cfg Config) {
	folderName := filepath.Base(folderPath)
	parentCategoryArray := buildParentCategoryArray(folderPath)
	url := fmt.Sprintf("%s://%s.%s/%s/", cfg.Schema(), cfg.Bucket, cfg.Endpoint, folderPath)
	idSuffix := fmt.Sprintf("%s-folder-%s", cfg.Bucket, folderPath)

	doc := common.Document{
		Type:    "folder",
		Icon:    "folder",
		Title:   folderName,
		Content: "",
		URL:     url,
		Size:    0,
	}
	p.documentWithHierarchy(&doc, parentCategoryArray, datasource, idSuffix)

	p.saveDocument(doc, datasource)
}

func (p *Plugin) Stop() error {
	return nil
}

func (p *Plugin) Name() string {
	return ConnectorS3
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}
