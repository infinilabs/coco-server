/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package box

import (
	"fmt"
	"strings"
	"time"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

type FolderNode struct {
	ID              string
	Name            string
	ParentID        string
	Processed       bool
	FullPath        string
	FullPathArray   []string
	ParentPathArray []string
	ModifiedTime    string
	CreatedTime     string
	Size            int64
	Type            string
	URL             string
	UserID          string // For enterprise accounts with multiple users
}

func (processor *Processor) startIndexingFiles(
	ctx *pipeline.Context,
	connector *core.Connector,
	datasource *core.DataSource,
	client *BoxClient,
) {
	processor.startIndexingFilesForUser(ctx, connector, datasource, client, "", "")
}

func (processor *Processor) startIndexingFilesForUser(
	ctx *pipeline.Context,
	connector *core.Connector,
	datasource *core.DataSource,
	client *BoxClient,
	userID, userName string,
) {
	// Get root folder ID (0 for Box)
	rootFolderID := "0"

	// For enterprise accounts with multiple users, create a user-specific root category
	// For free accounts, use standard root "/"
	var rootPath string
	var pathArray []string

	if userID != "" {
		// Enterprise account: create user-specific hierarchy with user name as top category
		// This ensures different users' files are properly separated in the hierarchy
		rootPath = "/"
		if userName != "" {
			pathArray = []string{userName}
		} else {
			pathArray = []string{userID}
		}
	} else {
		// Free account: standard root without user prefix
		rootPath = "/"
		pathArray = []string{}
	}

	rootFolder := &FolderNode{
		ID:              rootFolderID,
		Name:            "",
		ParentID:        "",
		Processed:       false,
		FullPath:        rootPath,
		FullPathArray:   pathArray,
		ParentPathArray: []string{},
		ModifiedTime:    time.Now().Format(time.RFC3339),
		CreatedTime:     time.Now().Format(time.RFC3339),
		Size:            0,
		Type:            "folder",
		URL:             "https://app.box.com",
		UserID:          userID,
	}

	// Process files recursively starting from root
	processor.processFolderRecursively(ctx, connector, datasource, client, rootFolder)
}

func (processor *Processor) processFolderRecursively(
	ctx *pipeline.Context,
	connector *core.Connector,
	datasource *core.DataSource,
	client *BoxClient,
	folder *FolderNode,
) {
	log.Debugf("Processing folder: %s (ID: %s)", folder.Name, folder.ID)

	// Skip creating document for root folder (/)
	if folder.Name != "" {
		// Create folder directory document
		folderDoc := common.CreateHierarchyPathFolderDoc(
			datasource,
			folder.ID,
			folder.Name,
			folder.ParentPathArray,
		)
		folderDoc.URL = folder.URL
		folderDoc.Metadata = util.MapStr{
			"folder_type": "folder",
			"folder_id":   folder.ID,
			"platform":    "box",
			"size":        folder.Size,
			"created_at":  folder.CreatedTime,
			"modified_at": folder.ModifiedTime,
		}

		// Collect folder document
		processor.Collect(ctx, connector, datasource, folderDoc)
	}

	// Get folder items
	offset := 0
	limit := DefaultPageSize

	for {
		// Pass userID for enterprise accounts (as-user header)
		items, err := client.GetFolderItems(folder.ID, offset, limit, folder.UserID)
		if err != nil {
			log.Errorf("Failed to get folder items for %s: %v", folder.ID, err)
			break
		}

		// Process each item
		for _, item := range items.Entries {
			processor.processItem(ctx, connector, datasource, item, client, folder)
		}

		// Check if we have more items
		if offset+limit >= items.TotalCount {
			break
		}
		offset += limit
	}
}

func (processor *Processor) processItem(
	ctx *pipeline.Context,
	connector *core.Connector,
	datasource *core.DataSource,
	item *BoxFile,
	client *BoxClient,
	parentFolder *FolderNode,
) {
	if item.Type == FileTypeFolder {
		// Compute child folder path
		var childPath string
		if parentFolder.FullPath == "" || parentFolder.FullPath == "/" {
			childPath = "/" + item.Name
		} else {
			childPath = parentFolder.FullPath + "/" + item.Name
		}

		// Process folder
		childFolder := &FolderNode{
			ID:              item.ID,
			Name:            item.Name,
			ParentID:        parentFolder.ID,
			Processed:       false,
			FullPath:        childPath,
			FullPathArray:   append(parentFolder.FullPathArray, item.Name),
			ParentPathArray: parentFolder.FullPathArray,
			ModifiedTime:    item.ModifiedAt.Format(time.RFC3339),
			CreatedTime:     item.CreatedAt.Format(time.RFC3339),
			Size:            item.Size,
			Type:            "folder",
			URL:             item.URL,
			UserID:          parentFolder.UserID, // Propagate UserID
		}

		// Recursively process the folder
		processor.processFolderRecursively(ctx, connector, datasource, client, childFolder)
	} else if item.Type == FileTypeFile {
		// Process file
		processor.processFile(ctx, connector, datasource, item, parentFolder)
	}
}

func (processor *Processor) processFile(
	ctx *pipeline.Context,
	connector *core.Connector,
	datasource *core.DataSource,
	file *BoxFile,
	parentFolder *FolderNode,
) {
	// Create document
	doc := core.Document{
		Source: core.DataSourceReference{
			ID:   datasource.ID,
			Type: "connector",
			Name: datasource.Name,
		},
	}

	doc.System = datasource.System
	doc.Title = file.Name
	doc.Type = file.Type
	doc.Icon = getIconTypeFromExtension(file.Extension)
	doc.URL = file.URL

	// Set hierarchy path
	doc.Category = common.GetFullPathForCategories(parentFolder.FullPathArray)
	doc.Categories = parentFolder.FullPathArray

	if doc.System == nil {
		doc.System = util.MapStr{}
	}
	doc.System[common.SystemHierarchyPathKey] = doc.Category

	// Set timestamps
	created := file.CreatedAt
	modified := file.ModifiedAt
	doc.Created = &created
	doc.Updated = &modified

	// Set metadata
	doc.Metadata = util.MapStr{
		"file_id":       file.ID,
		"file_type":     file.Type,
		"size":          file.Size,
		"description":   file.Description,
		"item_status":   file.ItemStatus,
		"sequence_id":   file.SequenceID,
		"etag":          file.ETag,
		"platform":      "box",
		"created_by":    file.CreatedBy,
		"modified_by":   file.ModifiedBy,
		"owned_by":      file.OwnedBy,
		"parent":        file.Parent,
		"download_url":  file.DownloadURL,
		"thumbnail_url": file.ThumbnailURL,
		"shared_link":   file.SharedLink,
	}

	// Add user_id for enterprise accounts
	if parentFolder.UserID != "" {
		doc.Metadata["user_id"] = parentFolder.UserID
	}
	doc.Payload = map[string]interface{}{
		"file_id":   file.ID,
		"file_name": file.Name,
		"file_type": file.Type,
		"size":      file.Size,
	}

	// Generate document ID
	// For enterprise accounts, include userID to avoid conflicts between users
	if parentFolder.UserID != "" {
		doc.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", datasource.ID, parentFolder.UserID, file.ID))
	} else {
		doc.ID = util.MD5digest(fmt.Sprintf("%v-%v", datasource.ID, file.ID))
	}

	// Collect document
	// Note: File content extraction is handled by the coco-server framework's
	// document processing pipeline, not in the connector itself
	processor.Collect(ctx, connector, datasource, doc)
}

// getIconTypeFromExtension returns the icon type based on file extension (without dot)
func getIconTypeFromExtension(ext string) string {
	ext = strings.ToLower(ext)

	// Map extensions to Box icon types
	switch ext {
	// PDF
	case "pdf":
		return "pdf"

	// Microsoft Office - Word
	case "doc", "docx", "docm", "dot", "dotx", "dotm":
		return "docx"

	// Microsoft Office - Excel
	case "xls", "xlsx", "xlsm", "xlsb", "xlt", "xltx", "xltm":
		return "excel-spreadsheet"

	// Microsoft Office - PowerPoint
	case "ppt", "pptx", "pptm", "pot", "potx", "potm", "pps", "ppsx", "ppsm":
		return "powerpoint-presentation"

	// Apple iWork - Pages
	case "pages":
		return "pages"

	// Apple iWork - Numbers
	case "numbers":
		return "numbers"

	// Apple iWork - Keynote
	case "keynote", "key":
		return "keynote"

	// Google Docs (if saved with these extensions)
	case "gdoc":
		return "google-docs"

	// Google Sheets
	case "gsheet":
		return "google-sheets"

	// Google Slides
	case "gslides":
		return "google-slides"

	// Box specific formats
	case "boxnote":
		return "boxnote"

	// Box Canvas
	case "boxcanvas":
		return "boxcanvas"

	// Bookmark
	case "url", "webloc", "website":
		return "bookmark"

	// Default
	default:
		return "default"
	}
}
