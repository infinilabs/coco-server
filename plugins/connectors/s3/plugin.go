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
		connectors.MarkParentFoldersAsValid(obj.Key, foldersWithMatchingFiles)

		// Create file document using helper
		parentCategoryArray := connectors.BuildParentCategoryArray(obj.Key)
		title := filepath.Base(obj.Key)
		url := fmt.Sprintf("%s://%s.%s/%s", cfg.Schema(), cfg.Bucket, cfg.Endpoint, obj.Key)
		idSuffix := fmt.Sprintf("%s-%s", cfg.Bucket, obj.Key)

		doc := connectors.CreateDocumentWithHierarchy(connectors.TypeFile, connectors.TypeFile, title, url, int(obj.Size),
			parentCategoryArray, datasource, idSuffix)

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

// createFolderDocuments creates document entries for all folders that contain matching files
func (p *Plugin) createFolderDocuments(foldersWithMatchingFiles map[string]bool, datasource *common.DataSource, cfg Config) {
	for folderPath := range foldersWithMatchingFiles {
		if global.ShuttingDown() {
			return
		}
		p.saveFolder(folderPath, datasource, cfg)
	}
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
	parentCategoryArray := connectors.BuildParentCategoryArray(folderPath)
	url := fmt.Sprintf("%s://%s.%s/%s/", cfg.Schema(), cfg.Bucket, cfg.Endpoint, folderPath)
	idSuffix := fmt.Sprintf("%s-folder-%s", cfg.Bucket, folderPath)

	doc := connectors.CreateDocumentWithHierarchy(connectors.TypeFolder, connectors.IconFolder, folderName, url, 0,
		parentCategoryArray, datasource, idSuffix)

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
