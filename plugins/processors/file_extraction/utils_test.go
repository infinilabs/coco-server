/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"os"
	"path/filepath"
	"testing"
)

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
