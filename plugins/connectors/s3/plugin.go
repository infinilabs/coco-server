/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package s3

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"infini.sh/coco/core"

	log "github.com/cihub/seelog"
	"github.com/minio/minio-go/v7"
	"infini.sh/coco/plugins/connectors"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

const ConnectorS3 = "s3"

type Plugin struct {
	cmn.ConnectorProcessorBase
}

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorS3, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	runner := Plugin{}
	runner.Init(c, &runner)
	return &runner, nil
}

func (p *Plugin) Name() string {
	return ConnectorS3
}

func (p *Plugin) Fetch(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	cfg := Config{}
	p.MustParseConfig(datasource, &cfg)

	log.Debugf("[%s connector] handling datasource: %v", ConnectorS3, cfg)

	if cfg.Bucket == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return fmt.Errorf("missing required configuration for datasource [%s]: bucket, access_key_id, or secret_access_key", datasource.Name)
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
		url := fmt.Sprintf("%s://%s/%s/%s", cfg.Schema(), cfg.Endpoint, cfg.Bucket, obj.Key)
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

		doc.Metadata["url_is_raw_content"] = true

		doc.Owner = &core.UserInfo{
			UserID:   obj.Owner.ID,
			UserName: obj.Owner.DisplayName,
		}

		// S3 not provides creation time
		doc.Created = &obj.LastModified
		doc.Updated = &obj.LastModified

		if global.Env().IsDebug {
			data := util.MustToJSONBytes(doc)
			log.Tracef("[%s connector] Queuing document: %s", ConnectorS3, string(data))
		}

		p.Collect(ctx, connector, datasource, doc)
	}

	handler, err := NewMinioHandler(cfg)
	if err != nil {
		return fmt.Errorf("failed to init minio client for datasource [%s]: %v", datasource.Name, err)
	}

	// process with each object
	handler.ListObjects(context.TODO(), objectVisitor)

	// Now create folder documents for all folders that contain matching files
	p.createFolderDocuments(ctx, foldersWithMatchingFiles, connector, datasource, cfg)

	log.Infof("[%s connector] finished fetching datasource [%s]", ConnectorS3, datasource.Name)
	return nil
}

// createFolderDocuments creates document entries for all folders that contain matching files
func (p *Plugin) createFolderDocuments(ctx *pipeline.Context, foldersWithMatchingFiles map[string]bool, connector *core.Connector, datasource *core.DataSource, cfg Config) {
	var docs []core.Document
	for folderPath := range foldersWithMatchingFiles {
		if global.ShuttingDown() {
			log.Info("[s3 connector] Shutdown signal received, stopping folder creation.")
			break
		}
		folderName := filepath.Base(folderPath)
		parentCategoryArray := connectors.BuildParentCategoryArray(folderPath)
		url := fmt.Sprintf("%s://%s.%s/%s/", cfg.Schema(), cfg.Bucket, cfg.Endpoint, folderPath)
		idSuffix := fmt.Sprintf("%s-folder-%s", cfg.Bucket, folderPath)

		doc := connectors.CreateDocumentWithHierarchy(connectors.TypeFolder, connectors.IconFolder, folderName, url, 0,
			parentCategoryArray, datasource, idSuffix)

		docs = append(docs, doc)
	}

	if len(docs) > 0 {
		p.BatchCollect(ctx, connector, datasource, docs)
	}
}
