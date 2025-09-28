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
	TypeOrg    = "org"
	TypeRepo   = "repository"
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

// Git-specific hierarchy helper functions

type GitFolder struct {
	Icon  string
	Title string
}

var (
	gitFolders = map[string]GitFolder{
		"issue":         {Icon: "issue", Title: "Issues"},
		"pull_request":  {Icon: "pull_request", Title: "Pull Requests"},
		"merge_request": {Icon: "merge_request", Title: "Merge Requests"},
		"wiki":          {Icon: "wiki", Title: "Wikis"},
		"snippet":       {Icon: "snippet", Title: "Snippets"},
	}
)

func resolveGitFolder(typeName string) GitFolder {
	if info, ok := gitFolders[typeName]; ok {
		return info
	}

	return GitFolder{Icon: typeName, Title: typeName}
}

// BuildGitRepositoryCategories returns categories for repository level
func BuildGitRepositoryCategories(owner, repo string) []string {
	return []string{owner, repo}
}

// BuildGitContentCategories returns categories for content type level (issues, pull_requests, etc.)
func BuildGitContentCategories(owner, repo string) []string {
	return []string{owner, repo}
}

// BuildGitItemCategories returns categories for individual content items
func BuildGitItemCategories(owner, repo, contentType string) []string {
	return []string{owner, repo, resolveGitFolder(contentType).Title}
}

// GitFolderTracker tracks folder hierarchies for git providers
type GitFolderTracker struct {
	Organizations map[string]bool // owner name -> tracked
	Repositories  map[string]bool // "owner/repo" -> tracked
	ContentTypes  map[string]bool // "owner/repo/content_type" -> tracked
}

// NewGitFolderTracker creates a new folder tracker for git hierarchy
func NewGitFolderTracker() *GitFolderTracker {
	return &GitFolderTracker{
		Organizations: make(map[string]bool),
		Repositories:  make(map[string]bool),
		ContentTypes:  make(map[string]bool),
	}
}

// TrackGitFolders tracks git folder hierarchy levels
func (tracker *GitFolderTracker) TrackGitFolders(owner, repo string, contentTypes []string) {
	// Track organization/user level
	tracker.Organizations[owner] = true

	// Track repository level
	repoKey := fmt.Sprintf("%s/%s", owner, repo)
	tracker.Repositories[repoKey] = true

	// Track content type levels
	for _, contentType := range contentTypes {
		contentKey := fmt.Sprintf("%s/%s/%s", owner, repo, contentType)
		tracker.ContentTypes[contentKey] = true
	}
}

// CreateGitFolderDocuments creates folder documents for all tracked git hierarchy levels
func (tracker *GitFolderTracker) CreateGitFolderDocuments(datasource *common.DataSource, pushFunc func(doc common.Document)) {
	// Create organization/user folder documents (Level 1)
	for owner := range tracker.Organizations {
		idSuffix := fmt.Sprintf("git-folder-%s", owner)
		doc := CreateDocumentWithHierarchy(TypeFolder, TypeOrg, owner, "", 0, nil, datasource, idSuffix)
		pushFunc(doc)
	}

	// Create repository folder documents (Level 2)
	for repoKey := range tracker.Repositories {
		parts := strings.Split(repoKey, "/")
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]
		idSuffix := fmt.Sprintf("git-folder-%s-%s", owner, repo)

		doc := CreateDocumentWithHierarchy(TypeFolder, TypeRepo, repo, "", 0, []string{owner}, datasource, idSuffix)
		pushFunc(doc)
	}

	// Create content type folder documents (Level 3B)
	for contentKey := range tracker.ContentTypes {
		parts := strings.Split(contentKey, "/")
		if len(parts) != 3 {
			continue
		}
		owner, repo, contentType := parts[0], parts[1], parts[2]

		categories := BuildGitContentCategories(owner, repo)
		idSuffix := fmt.Sprintf("git-folder-%s-%s-%s", owner, repo, contentType)

		info := resolveGitFolder(contentType)
		doc := CreateDocumentWithHierarchy(TypeFolder, info.Icon, info.Title, "", 0, categories, datasource, idSuffix)

		pushFunc(doc)
	}
}
