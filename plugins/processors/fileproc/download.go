/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package fileproc

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/connectors/local_fs"
	"infini.sh/coco/plugins/connectors/s3"
	utils "infini.sh/coco/plugins/processors"
)

// S3Config holds S3 connection configuration extracted from a datasource connector.
type S3Config struct {
	Endpoint        string `config:"endpoint"`
	AccessKeyID     string `config:"access_key_id"`
	SecretAccessKey string `config:"secret_access_key"`
	Bucket          string `config:"bucket"`
	UseSSL          bool   `config:"use_ssl"`
}

// DownloadToLocal downloads (or copies) doc's file to tempDir and returns its
// local path.  processorName is used only for log messages.
func DownloadToLocal(ctx context.Context, doc *core.Document, connectorID, tempDir string) (string, error) {
	fileName := filepath.Base(doc.URL)
	if fileName == "" || fileName == "." {
		fileName = doc.ID + filepath.Ext(doc.URL)
	}
	localPath := filepath.Join(tempDir, fileName)

	switch connectorID {
	case s3.ConnectorS3:
		return downloadFromS3Connector(ctx, doc, localPath)
	case local_fs.ConnectorLocalFs:
		if err := CopyLocalFile(doc.URL, localPath); err != nil {
			return "", fmt.Errorf("failed to copy local file: %w", err)
		}
		return localPath, nil
	default:
		return "", fmt.Errorf("unsupported connector: %s", connectorID)
	}
}

func downloadFromS3Connector(ctx context.Context, doc *core.Document, localPath string) (string, error) {
	ds, err := utils.GetDatasource(doc)
	if err != nil {
		return "", fmt.Errorf("failed to get datasource: %w", err)
	}

	connectorConfig, ok := ds.Connector.Config.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid connector config type")
	}

	cfg := S3Config{
		Endpoint:        getStringFromMap(connectorConfig, "endpoint"),
		AccessKeyID:     getStringFromMap(connectorConfig, "access_key_id"),
		SecretAccessKey: getStringFromMap(connectorConfig, "secret_access_key"),
		Bucket:          getStringFromMap(connectorConfig, "bucket"),
		UseSSL:          getBoolFromMap(connectorConfig, "use_ssl"),
	}

	if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.Bucket == "" {
		return "", fmt.Errorf("incomplete S3 configuration for datasource [%s]: %+v", ds.ID, cfg)
	}

	objectKey, err := parseS3ObjectKey(doc.URL, cfg.Bucket, cfg.Endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to parse S3 object key from URL: %w", err)
	}

	if err := downloadFromS3(ctx, &cfg, objectKey, localPath); err != nil {
		return "", err
	}
	return localPath, nil
}

func parseS3ObjectKey(url, bucket, endpoint string) (string, error) {
	for _, scheme := range []string{"http", "https", "s3"} {
		prefix := fmt.Sprintf("%s://%s.%s/", scheme, bucket, endpoint)
		if strings.HasPrefix(url, prefix) {
			return strings.TrimPrefix(url, prefix), nil
		}
	}
	// Fallback: http(s)://{endpoint}/{bucket}/{key}
	for _, scheme := range []string{"http", "https"} {
		prefix := fmt.Sprintf("%s://%s/%s/", scheme, endpoint, bucket)
		if strings.HasPrefix(url, prefix) {
			return strings.TrimPrefix(url, prefix), nil
		}
	}
	return "", fmt.Errorf("unable to parse object key from URL: %s", url)
}

func downloadFromS3(ctx context.Context, cfg *S3Config, objectKey, localPath string) error {
	client, err := newS3Client(cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for download: %w", err)
	}

	if err := client.FGetObject(ctx, cfg.Bucket, objectKey, localPath, minio.GetObjectOptions{}); err != nil {
		return fmt.Errorf("failed to download object [%s] from bucket [%s]: %w", objectKey, cfg.Bucket, err)
	}

	log.Debugf("successfully downloaded [%s/%s] to [%s]", cfg.Bucket, objectKey, localPath)
	return nil
}

func newS3Client(cfg *S3Config) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}
	return client, nil
}

// CopyLocalFile copies src to dst, creating parent directories as needed.
func CopyLocalFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer DeferClose(srcFile)

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer DeferClose(dstFile)

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	return nil
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
