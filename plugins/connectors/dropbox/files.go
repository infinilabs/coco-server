/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package dropbox

import (
	"path/filepath"
	"strings"
	"time"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

type ListFolderArg struct {
	Path      string `json:"path"`
	Recursive bool   `json:"recursive"`
	Limit     int    `json:"limit"`
}

type ListFolderContinueArg struct {
	Cursor string `json:"cursor"`
}

type ListFolderResult struct {
	Entries []Metadata `json:"entries"`
	Cursor  string     `json:"cursor"`
	HasMore bool       `json:"has_more"`
}

type Metadata struct {
	Tag            string `json:".tag"`
	Id             string `json:"id"`
	Name           string `json:"name"`
	PathDisplay    string `json:"path_display"`
	PathLower      string `json:"path_lower"`
	ClientModified string `json:"client_modified"`
	ServerModified string `json:"server_modified"`
	Rev            string `json:"rev"`
	Size           uint64 `json:"size"`
	ContentHash    string `json:"content_hash"`
}

func (processor *Processor) startIndexingFiles(pipeCtx *pipeline.Context, connector *core.Connector, datasource *core.DataSource, client *DropboxClient) {
	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				log.Error("error on indexing dropbox files,", r)
			}
		}
	}()

	batchNumber := util.GetUUID()

	path := ""
	var cfgMap map[string]interface{}
	if m, ok := datasource.Connector.Config.(util.MapStr); ok {
		cfgMap = m
	} else if m, ok := datasource.Connector.Config.(map[string]interface{}); ok {
		cfgMap = m
	}

	if cfgMap != nil && cfgMap["path"] != nil {
		if p, ok := cfgMap["path"].(string); ok {
			path = p
		}
	}

	// Dropbox API requires empty string for root, not "/"
	if path == "/" {
		path = ""
	}

	processor.listFolder(pipeCtx, connector, datasource, client, path, batchNumber)
}

func (processor *Processor) listFolder(pipeCtx *pipeline.Context, connector *core.Connector, datasource *core.DataSource, client *DropboxClient, path string, batchNumber string) {

	arg := ListFolderArg{
		Path:      path,
		Recursive: true,
		Limit:     100, // Default limit
	}

	hasMore := true
	cursor := ""

	for hasMore {
		if global.ShuttingDown() {
			break
		}

		var result *ListFolderResult
		var err error

		if cursor != "" {
			result, err = client.ListFolderContinue(cursor)
		} else {
			result, err = client.ListFolder(arg)
		}

		if err != nil {
			log.Errorf("Failed to list folder: %v", err)
			break
		}

		for _, entry := range result.Entries {
			if global.ShuttingDown() {
				return
			}
			processor.processEntry(pipeCtx, connector, datasource, client, entry, batchNumber)
		}

		hasMore = result.HasMore
		cursor = result.Cursor
	}
}

func (processor *Processor) processEntry(pipeCtx *pipeline.Context, connector *core.Connector, datasource *core.DataSource, client *DropboxClient, entry Metadata, batchNumber string) {
	// Skip .DS_Store
	if entry.Name == ".DS_Store" {
		return
	}

	isFolder := entry.Tag == "folder"

	var createdAt, updatedAt *time.Time
	if entry.ClientModified != "" {
		parsedTime, err := time.Parse(time.RFC3339, entry.ClientModified)
		if err == nil {
			createdAt = &parsedTime
		}
	}
	if entry.ServerModified != "" {
		parsedTime, err := time.Parse(time.RFC3339, entry.ServerModified)
		if err == nil {
			updatedAt = &parsedTime
		}
	}

	// Determine type
	docType := "file"
	if isFolder {
		docType = "folder"
	}

	document := core.Document{
		Source: core.DataSourceReference{
			ID:   datasource.ID,
			Name: datasource.Name,
			Type: "connector",
		},
		Title: entry.Name,
		Type:  docType,
		Size:  int(entry.Size),
		Icon:  processor.getIcon(entry.Name, isFolder),
	}

	document.System = datasource.System
	document.ID = common.GetDocID(datasource.ID, entry.Id)
	document.Created = createdAt
	document.Updated = updatedAt

	// Hierarchy
	// path_display looks like "/folder/file.txt"
	if entry.PathDisplay != "" {
		dir := filepath.Dir(entry.PathDisplay)
		if dir != "." && dir != "/" {
			// Clean up path to be list of categories
			// "/a/b" -> ["a", "b"]
			cleanPath := strings.TrimPrefix(dir, "/")
			parts := strings.Split(cleanPath, "/")
			if len(parts) > 0 && parts[0] != "" {
				document.Categories = parts
				document.Category = common.GetFullPathForCategories(parts)
				if document.System == nil {
					document.System = util.MapStr{}
				}
				document.System[common.SystemHierarchyPathKey] = document.Category
			}
		}
	}

	meta := util.MapStr{
		"batch_number":    batchNumber,
		"file_id":         entry.Id,
		"path_display":    entry.PathDisplay,
		"path_lower":      entry.PathLower,
		"content_hash":    entry.ContentHash,
		"rev":             entry.Rev,
		"server_modified": entry.ServerModified,
		"client_modified": entry.ClientModified,
		"tag":             entry.Tag,
	}
	document.Metadata = meta

	processor.Collect(pipeCtx, connector, datasource, document)
}

func (processor *Processor) getIcon(filename string, isFolder bool) string {
	if isFolder {
		return "font_filetype-folder"
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".docx", ".doc":
		return "docx"
	case ".xlsx", ".xls":
		return "xlsx"
	case ".pptx", ".ppt":
		return "pptx"
	case ".pdf":
		return "pdf"
	case ".paper":
		return "paper"
	case ".gdoc":
		return "gdoc"
	case ".gsheet":
		return "gexcel"
	case ".gslides":
		return "gppt"
	default:
		return "default"
	}
}
