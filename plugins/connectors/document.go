/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connectors

import (
	"fmt"
	"path/filepath"
	"strings"

	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/util"
)

const (
	TypeFolder = "folder"
	TypeFile   = "file"
)

// MarkParentFoldersAsValid marks all parent folders of a file path as containing matching files.
// This is a common utility function used by connectors that need to track folder hierarchies.
func MarkParentFoldersAsValid(filePath string, foldersWithMatchingFiles map[string]bool) {
	if filePath == "" {
		return
	}

	// First convert all backslashes to forward slashes for consistent handling
	filePath = strings.ReplaceAll(filePath, "\\", "/")

	// Clean the file path
	filePath = filepath.Clean(filePath)
	filePath = filepath.ToSlash(filePath)

	// Split into path components
	parts := strings.Split(filePath, "/")

	// Build each folder path and mark it as valid
	currentPath := ""
	for _, part := range parts[:len(parts)-1] { // Exclude the filename
		if part != "" && part != "." {
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = currentPath + "/" + part
			}
			foldersWithMatchingFiles[currentPath] = true
		}
	}
}

// BuildParentCategoryArray constructs a hierarchical path array for a file path,
// returning only the parent folder names (excluding the filename).
// This is a common utility function used by connectors to build category hierarchies.
func BuildParentCategoryArray(filePath string) []string {
	if filePath == "" {
		return nil
	}

	var categories []string

	// First convert all backslashes to forward slashes for consistent handling
	filePath = strings.ReplaceAll(filePath, "\\", "/")

	// Clean the file path and ensure it uses forward slashes
	filePath = filepath.Clean(filePath)
	filePath = filepath.ToSlash(filePath)

	// Split the path into components, filtering out empty ones
	parts := strings.Split(filePath, "/")
	for _, part := range parts {
		if part != "" && part != "." {
			categories = append(categories, part)
		}
	}

	// Return all parts except the last one (the file name)
	if len(categories) > 1 {
		return categories[:len(categories)-1]
	}

	return nil // Return nil if there are no parent folders
}

// SetDocumentHierarchy sets the hierarchy information on a document based on the parent category array.
// This is a common utility function used by connectors to establish document hierarchy relationships.
func SetDocumentHierarchy(doc *common.Document, parentCategoryArray []string) {
	if len(parentCategoryArray) > 0 {
		categoryPath := common.GetFullPathForCategories(parentCategoryArray)
		doc.Category = categoryPath
		doc.Categories = parentCategoryArray
		doc.System[common.SystemHierarchyPathKey] = categoryPath
	} else {
		// This is a top-level item, set parent_path to '/'
		doc.System[common.SystemHierarchyPathKey] = "/"
		doc.Category = "/"
	}
}

// CreateDocumentWithHierarchy creates a document with proper hierarchy settings and basic metadata.
// This is a common utility function used by connectors to create documents with consistent structure.
func CreateDocumentWithHierarchy(docType, icon, title, url string, size int,
	parentCategoryArray []string, datasource *common.DataSource, idSuffix string) common.Document {

	doc := common.Document{
		Source: common.DataSourceReference{
			ID:   datasource.ID,
			Type: "connector",
			Name: datasource.Name,
		},
		Type:    docType,
		Icon:    icon,
		Title:   title,
		Content: "",
		URL:     url,
		Size:    size,
	}

	doc.System = datasource.System
	if doc.System == nil {
		doc.System = util.MapStr{}
	}

	// Set hierarchy information using the common helper
	SetDocumentHierarchy(&doc, parentCategoryArray)

	doc.ID = util.MD5digest(fmt.Sprintf("%s-%s", datasource.ID, idSuffix))

	return doc
}
