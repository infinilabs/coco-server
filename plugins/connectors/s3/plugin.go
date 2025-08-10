/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package s3

import (
	"context"
	"fmt"
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
		log.Errorf("[%v connector] Parsing connector configuration failed: %v", ConnectorS3, err)
		panic(err)
	}

	log.Debugf("[%v connector] Handling datasource: %v", ConnectorS3, cfg)

	if cfg.Bucket == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		log.Errorf("[%v connector] Missing required configuration for datasource [%s]: bucket, access_key_id, or secret_access_key", ConnectorS3, datasource.Name)
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

	objectVisitor := func(obj minio.ObjectInfo) {
		// Extension name not matched
		if len(extMap) > 0 {
			index := strings.LastIndex(obj.Key, ".")
			if index < 0 || !extMap[strings.ToLower(obj.Key[index:])] {
				return
			}
		}
		doc := common.Document{
			Source: common.DataSourceReference{
				ID:   datasource.ID,
				Type: "connector",
				Name: datasource.Name,
			},
			Type:     ConnectorS3,
			Icon:     "default",
			Title:    obj.Key,
			Content:  "",
			Category: cfg.Bucket,
			URL:      fmt.Sprintf("%s://%s.%s/%s", cfg.Schema(), cfg.Bucket, cfg.Endpoint, obj.Key),
			Size:     int(obj.Size),
		}

		doc.System = datasource.System

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
		doc.ID = util.MD5digest(fmt.Sprintf("%s-%s-%s", datasource.ID, cfg.Bucket, obj.Key))

		data := util.MustToJSONBytes(doc)
		if global.Env().IsDebug {
			log.Tracef("[%v connector] Queuing document: %s", ConnectorS3, string(data))
		}

		if err := queue.Push(p.Queue, data); err != nil {
			log.Errorf("[%v connector] Failed to push document to queue for datasource [%s]: %v", ConnectorS3, datasource.Name, err)
			panic(err)
		}
	}

	handler, err := NewMinioHandler(cfg)
	if err != nil {
		log.Infof("[%v connector] Failed to init minio client for datasource [%s]: %v", ConnectorS3, datasource.Name, err)
		panic(err)
	}

	// process with each object
	handler.ListObjects(context.TODO(), objectVisitor)

	log.Infof("[%v connector] Finished list objects from bucket [%s] of datasource [%s]. ", ConnectorS3, cfg.Bucket, datasource.Name)
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
