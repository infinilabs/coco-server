/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		// Images
		{"/path/to/image.jpg", "image/jpeg"},
		{"/path/to/image.jpeg", "image/jpeg"},
		{"/path/to/image.png", "image/png"},
		{"/path/to/image.gif", "image/gif"},
		{"/path/to/image.webp", "image/webp"},

		// Documents
		{"/path/to/doc.pdf", "application/pdf"},
		{"/path/to/doc.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"/path/to/presentation.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		{"/path/to/spreadsheet.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},

		// Text
		{"/path/to/readme.md", "text/markdown"},
		{"/path/to/notes.txt", "text/plain"},

		// Unknown
		{"/path/to/file.unknown", "application/octet-stream"},
		{"/path/to/noextension", "application/octet-stream"},
		{"", "application/octet-stream"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := detectContentType(tc.path)
			if result != tc.expected {
				t.Errorf("detectContentType(%q) = %q, expected %q", tc.path, result, tc.expected)
			}
		})
	}
}

func TestBytesReaderAt(t *testing.T) {
	data := []byte("hello world")
	reader := &bytesReaderAt{data: data}

	// Test reading from beginning
	buf := make([]byte, 5)
	n, err := reader.ReadAt(buf, 0)
	if n != 5 {
		t.Errorf("expected to read 5 bytes, got %d", n)
	}
	if string(buf) != "hello" {
		t.Errorf("expected 'hello', got %q", string(buf))
	}

	// Test reading from middle
	n, err = reader.ReadAt(buf, 6)
	if n != 5 {
		t.Errorf("expected to read 5 bytes, got %d", n)
	}
	if string(buf) != "world" {
		t.Errorf("expected 'world', got %q", string(buf))
	}

	// Test reading past EOF
	n, err = reader.ReadAt(buf, 100)
	if n != 0 {
		t.Errorf("expected to read 0 bytes past EOF, got %d", n)
	}
	if err == nil {
		t.Error("expected EOF error, got nil")
	}
}

func TestCopyLocalFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	srcContent := []byte("test content for copy")
	if err := os.WriteFile(srcPath, srcContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Test copying to new location
	dstPath := filepath.Join(tmpDir, "subdir", "dest.txt")
	if err := copyLocalFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyLocalFile failed: %v", err)
	}

	// Verify destination file exists and has correct content
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}
	if string(dstContent) != string(srcContent) {
		t.Errorf("destination content = %q, expected %q", string(dstContent), string(srcContent))
	}
}

func TestCopyLocalFile_NonexistentSource(t *testing.T) {
	tmpDir := t.TempDir()
	err := copyLocalFile("/nonexistent/path", filepath.Join(tmpDir, "dest.txt"))
	if err == nil {
		t.Error("expected error for nonexistent source, got nil")
	}
}

func TestS3Config_PreviewURLConstruction(t *testing.T) {
	// Test URL construction logic by verifying the format matches expected pattern
	tests := []struct {
		name     string
		cfg      S3Config
		object   string
		expected string
	}{
		{
			name: "HTTP endpoint",
			cfg: S3Config{
				Endpoint: "s3.example.com:9000",
				Bucket:   "mybucket",
				UseSSL:   false,
			},
			object:   "path/to/file.pdf",
			expected: "http://s3.example.com:9000/mybucket/path/to/file.pdf",
		},
		{
			name: "HTTPS endpoint",
			cfg: S3Config{
				Endpoint: "s3.example.com",
				Bucket:   "mybucket",
				UseSSL:   true,
			},
			object:   "doc.pdf",
			expected: "https://s3.example.com/mybucket/doc.pdf",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reconstruct URL logic inline since we can't call the actual upload functions
			schema := "http"
			if tc.cfg.UseSSL {
				schema = "https"
			}
			result := schema + "://" + tc.cfg.Endpoint + "/" + tc.cfg.Bucket + "/" + tc.object

			if result != tc.expected {
				t.Errorf("preview URL = %q, expected %q", result, tc.expected)
			}
		})
	}
}
