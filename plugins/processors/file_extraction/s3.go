/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// Utilies needed to interact with S3
//
// * We download S3 files to lcoal disk to process them
// * We upload documents and their covers to S3 so that we can preview them
//   using the preview URL.

package file_extraction

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	log "github.com/cihub/seelog"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config holds S3 connection configuration
type S3Config struct {
	Endpoint        string `config:"endpoint"`
	AccessKeyID     string `config:"access_key_id"`
	SecretAccessKey string `config:"secret_access_key"`
	Bucket          string `config:"bucket"`
	UseSSL          bool   `config:"use_ssl"`
}

// newS3Client creates a new minio client with the given configuration
func newS3Client(cfg S3Config) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}
	return client, nil
}

// downloadFromS3 downloads a file from S3 to a local path
func downloadFromS3(ctx context.Context, cfg S3Config, objectKey, localPath string) error {
	client, err := newS3Client(cfg)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for download: %w", err)
	}

	err = client.FGetObject(ctx, cfg.Bucket, objectKey, localPath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to download object [%s] from bucket [%s]: %w", objectKey, cfg.Bucket, err)
	}

	log.Debugf("successfully downloaded [%s/%s] to [%s]", cfg.Bucket, objectKey, localPath)
	return nil
}

// uploadToS3 uploads a local file to S3 and returns the preview URL
func uploadToS3(ctx context.Context, cfg S3Config, localPath, objectName string) (string, error) {
	client, err := newS3Client(cfg)
	if err != nil {
		return "", err
	}

	// Open the file
	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for upload: %w", err)
	}
	defer DeferClose(file)

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Detect MIME type
	contentType := detectMimeType(localPath)

	// Upload
	_, err = client.PutObject(ctx, cfg.Bucket, objectName, file, fileInfo.Size(), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object [%s] to bucket [%s]: %w", objectName, cfg.Bucket, err)
	}

	// Build preview URL
	schema := "http"
	if cfg.UseSSL {
		schema = "https"
	}
	previewURL := fmt.Sprintf("%s://%s/%s/%s", schema, cfg.Endpoint, cfg.Bucket, objectName)

	log.Debugf("successfully uploaded [%s] to [%s], preview URL: %s", localPath, objectName, previewURL)
	return previewURL, nil
}

// detectMimeType returns the MIME type based on file extension.
//
// Fall back to "application/octet-stream" if unknown.
func detectMimeType(path string) string {
	ext := filepath.Ext(path)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}

// copyLocalFile copies a local file to the destination path
//
// TODO(SteveLauC): check if framework provides such a funciton, if so, reuse it.
func copyLocalFile(src, dst string) error {
	// Ensure parent directory exists
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

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
