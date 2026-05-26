/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// Package fileproc is a shared helper package that provides common
// file-processing utilities for the file pipeline processors:
// file_metadata, document_cover, text_extraction, and face_extraction.
package fileproc

import (
	"io"
	"path/filepath"
	"strings"

	log "github.com/cihub/seelog"
)

// DeferClose is a utility used to check the return value from Close in a
// defer statement.
func DeferClose(c io.Closer) {
	if err := c.Close(); err != nil {
		log.Errorf("Close() failed with error: %s", err)
	}
}

// Escape replaces each character in charsToEscape with its backslash-escaped
// version.  Backslash `\` itself is always escaped.
func Escape(input string, charsToEscape []rune) string {
	escapeMap := make(map[rune]bool, len(charsToEscape)+1)
	for _, r := range charsToEscape {
		escapeMap[r] = true
	}
	escapeMap['\\'] = true

	var builder strings.Builder
	for _, c := range input {
		if escapeMap[c] {
			builder.WriteRune('\\')
		}
		builder.WriteRune(c)
	}
	return builder.String()
}

// ContentTypeFromURL derives a logical content-type string from the file
// extension of url (which may be an S3 URL, a local path, or any string ending
// with a filename).  Returns one of: "image", "pdf", "pptx", "docx",
// "xlsx", "markdown", or "" for unrecognised types.
//
// This allows processors to determine document type independently without
// relying on metadata written by a previous pipeline stage.
func ContentTypeFromURL(url string) string {
	ext := strings.ToLower(filepath.Ext(url))
	switch ext {
	case ".jpg", ".jpeg", ".jfif", ".png", ".gif", ".webp", ".bmp", ".tiff", ".tif":
		return "image"
	case ".pdf":
		return "pdf"
	case ".pptx", ".ppt", ".pptm":
		return "pptx"
	case ".docx", ".doc":
		return "docx"
	case ".xlsx", ".xls":
		return "xlsx"
	case ".md":
		return "markdown"
	default:
		return ""
	}
}

// IsImage returns true if the file's extension indicates an image.
// Combines the extension sets previously spread across isImage / isImageFile.
func IsImage(pathOrName string) bool {
	ext := strings.ToLower(filepath.Ext(pathOrName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp", ".jfif":
		return true
	}
	return false
}
