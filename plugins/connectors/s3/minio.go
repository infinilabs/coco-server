/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package s3

import (
	"context"
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"infini.sh/framework/core/global"
)

// Config defined S3 configuration
type Config struct {
	AccessKeyID     string   `config:"access_key_id"`
	SecretAccessKey string   `config:"secret_access_key"`
	Bucket          string   `config:"bucket"`
	Endpoint        string   `config:"endpoint"`
	UseSSL          bool     `config:"use_ssl"`
	Prefix          string   `config:"prefix"`
	Extensions      []string `config:"extensions"`
}

func (c Config) Schema() string {
	if c.UseSSL {
		return "https"
	}
	return "http"
}

func (c Config) String() string {
	return fmt.Sprintf("Bucket: %s, Endpoint: %s, AccessKeyID: %s, UseSSL: %v", c.Bucket, c.Endpoint, c.AccessKeyID, c.UseSSL)
}

type MinioHandler struct {
	Config
	Client *minio.Client
}

func NewMinioHandler(config Config) (*MinioHandler, error) {
	minioHandler := MinioHandler{Config: config}

	// Initialize minio client object.
	minioClient, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})

	if err != nil {
		log.Errorf("[s3 config] Failed to create minio client [%s]: %v", ConnectorS3, config, err)
		return nil, err
	}
	minioHandler.Client = minioClient
	return &minioHandler, nil
}

func (r *MinioHandler) ListObjects(ctx context.Context, visitor func(minio.ObjectInfo)) {
	log.Infof("[s3] Scanning bucket [%v]", r.Bucket)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	objectCh := r.Client.ListObjects(ctx, r.Bucket, minio.ListObjectsOptions{
		Prefix:    r.Prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if global.ShuttingDown() {
			log.Info("[s3] Shutdown signal received, stopping list objects.")
			break
		}

		if object.Err != nil {
			log.Errorf("[s3] Failed to list objects for bucket [%s]: %v", r.Bucket, object.Err)
			return
		}

		// process each object
		visitor(object)
	}
}
