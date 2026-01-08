/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	log "github.com/cihub/seelog"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config holds S3 connection configuration
type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseSSL          bool
}

// Hardcoded S3 configurations for cover and document storage
var (
	CoverS3Config = S3Config{
		Endpoint:        "192.168.3.181:9101",
		AccessKeyID:     "cloud_s3_accesskey",
		SecretAccessKey: "cloud_s3_secretkey",
		Bucket:          "ai_search_covers",
		UseSSL:          false,
	}

	DocumentS3Config = S3Config{
		Endpoint:        "192.168.3.181:9101",
		AccessKeyID:     "cloud_s3_accesskey",
		SecretAccessKey: "cloud_s3_secretkey",
		Bucket:          "ai_search_documents",
		UseSSL:          false,
	}
)

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
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Detect content type
	contentType := detectContentType(localPath)

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

// uploadBytesToS3 uploads bytes directly to S3 and returns the preview URL
func uploadBytesToS3(ctx context.Context, cfg S3Config, data []byte, objectName, contentType string) (string, error) {
	client, err := newS3Client(cfg)
	if err != nil {
		return "", err
	}

	reader := io.NopCloser(io.NewSectionReader(
		&bytesReaderAt{data: data}, 0, int64(len(data)),
	))

	_, err = client.PutObject(ctx, cfg.Bucket, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload bytes to [%s/%s]: %w", cfg.Bucket, objectName, err)
	}

	schema := "http"
	if cfg.UseSSL {
		schema = "https"
	}
	previewURL := fmt.Sprintf("%s://%s/%s/%s", schema, cfg.Endpoint, cfg.Bucket, objectName)

	log.Debugf("successfully uploaded bytes to [%s], preview URL: %s", objectName, previewURL)
	return previewURL, nil
}

// bytesReaderAt implements io.ReaderAt for []byte
type bytesReaderAt struct {
	data []byte
}

func (b *bytesReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n = copy(p, b.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return
}

// detectContentType returns the MIME type based on file extension
func detectContentType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".md":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

// copyLocalFile copies a local file to the destination path
func copyLocalFile(src, dst string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
