/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package google_drive

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

func getIcon(fileType string) string {
	i := getType(fileType)
	if fileType == "folder" || i == "folder" {
		return "font_filetype-folder"
	}
	return i
}

func getType(fileType string) string {
	switch fileType {
	case "application/vnd.google-apps.document":
		return "document"
	case "application/vnd.google-apps.form":
		return "form"
	case "application/pdf":
		return "pdf"
	case "application/vnd.google-apps.presentation":
		return "presentation"
	case "application/vnd.google-apps.spreadsheet":
		return "spreadsheet"
	case "application/vnd.google-apps.drawing":
		return "drawing"
	case "application/vnd.google-apps.folder":
		return string(security.FolderResource)
	case "application/vnd.google-apps.fusiontable":
		return "fusiontable"
	case "application/vnd.google-apps.jam":
		return "jam"
	case "application/vnd.google-apps.map":
		return "map"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": // MS Excel
		return "ms_excel"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation": // MS PowerPoint
		return "ms_powerpoint"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document": // MS Word
		return "ms_word"
	case "application/vnd.google-apps.script":
		return "script"
	case "application/vnd.google-apps.site":
		return "site"
	case "application/vnd.google-apps.video":
		return "video"
	case "application/zip":
		return "zip"
	case "image/jpeg", "image/png", "image/gif", "image/tiff", "image/bmp": // Image formats
		return "photo"
	case "audio/mpeg", "audio/wav", "audio/mp3", "audio/ogg": // Audio formats
		return "audio"
	default:
		return "default" // Default fallback
	}
}

// Function to get the root folder ID
func getRootFolderID(srv *drive.Service) (string, string) {
	rootFolder, err := srv.Files.Get("root").Fields("id, name").Do()
	if err != nil {
		panic(errors.Errorf("Unable to get root folder ID: %v", err))
	}
	return rootFolder.Id, rootFolder.Name
}

type FolderNode struct {
	ID              string
	Name            string
	ParentID        string
	Shared          bool
	HasACL          bool
	Processed       bool
	Permissions     []*drive.Permission
	FullPath        string
	FullPathArray   []string
	ParentPathArray []string // Path without self (parent folders only)
	ModifiedTime    string
}

type FolderTreeBuilder struct {
	FolderMap map[string]*FolderNode
	Sorted    []*FolderNode
}

func (ft *FolderTreeBuilder) AddFolder(id, name, parentID string, shared bool, hasACL bool, perms []*drive.Permission, modifiedTime string) {
	ft.FolderMap[id] = &FolderNode{
		ID:           id,
		Name:         name,
		ParentID:     parentID,
		Shared:       shared,
		HasACL:       hasACL,
		Permissions:  perms,
		ModifiedTime: modifiedTime,
	}
}

func (ft *FolderTreeBuilder) BuildSorted() {
	visited := map[string]bool{}
	var dfs func(id string)
	dfs = func(id string) {
		if visited[id] {
			return
		}
		node, ok := ft.FolderMap[id]
		if !ok {
			return
		}
		if node.ParentID != "" {
			dfs(node.ParentID)
		}
		visited[id] = true
		ft.Sorted = append(ft.Sorted, node)
	}
	for id := range ft.FolderMap {
		dfs(id)
	}
}

func (ft *FolderTreeBuilder) IsExplicitACL(folderID string) bool {
	node, ok := ft.FolderMap[folderID]
	if !ok {
		return false
	}
	if node.HasACL {
		return true
	}
	if node.ParentID == "" || node.ParentID == folderID {
		return false
	}
	return ft.IsExplicitACL(node.ParentID)
}

func hasExplicitSharedPermission(file *drive.File) bool {
	if !file.Shared || len(file.Permissions) == 0 {
		return false
	}

	for _, perm := range file.Permissions {
		// Skip owner/self
		if perm.Role == "owner" || perm.Role == "organizer" {
			continue
		}
		// Any other visible user/group is treated as explicit
		if perm.Type == "user" || perm.Type == "group" {
			if perm.Role == "reader" || perm.Role == "writer" {
				return true
			}
		}
	}
	return false
}

func checkExplicit(perms []*drive.Permission) bool {
	for _, perm := range perms {
		if (perm.Type == "user" || perm.Type == "group") &&
			(perm.Role == "reader" || perm.Role == "writer") {
			return true
		}
	}
	return false
}

func isSamePermission(a, b []*drive.Permission) bool {
	if len(a) != len(b) {
		return false
	}
	mapify := func(perms []*drive.Permission) map[string]string {
		m := map[string]string{}
		for _, p := range perms {
			key := fmt.Sprintf("%s:%s", p.EmailAddress, p.Role)
			m[key] = p.Type
		}
		return m
	}

	mapA := mapify(a)
	mapB := mapify(b)

	if len(mapA) != len(mapB) {
		return false
	}

	for k, v := range mapA {
		if mapB[k] != v {
			return false
		}
	}
	return true
}

func (this *Processor) IndexingFolder(pipeCtx *pipeline.Context, connector *common.Connector, datasource *common.DataSource, doc common.Document, srv *drive.Service, q string, batchNumber string) {

	//save current folder to index
	log.Tracef("saving folder: %v, %v, %v", doc.Category, doc.Title, doc.ID)
	this.Collect(pipeCtx, connector, datasource, doc)

	var nextPageToken string
	for {
		if global.ShuttingDown() {
			break
		}

		call := srv.Files.List().
			PageSize(int64(datasource.SyncConfig.PageSize)).
			//OrderBy("name asc").
			Q(q).
			IncludeItemsFromAllDrives(true).
			SupportsAllDrives(true). //
			Fields("nextPageToken, files(id, name, parents, mimeType, size, owners(emailAddress, displayName), createdTime, modifiedTime, lastModifyingUser(emailAddress, displayName), iconLink, fileExtension, description, hasThumbnail, kind, labelInfo, parents, properties, shared, sharingUser(emailAddress, displayName), spaces, starred, driveId, thumbnailLink, videoMediaMetadata, webViewLink, imageMediaMetadata, permissions(id,emailAddress,displayName,role,type,permissionDetails))")

		r, err := call.PageToken(nextPageToken).Do()
		if err != nil {
			if r != nil {
				s, err := r.MarshalJSON()
				log.Error(string(s), err)
			}
			panic(errors.Errorf("Failed to fetch directories: %v", err))
		}

		for _, i := range r.Files {

			if global.ShuttingDown() {
				return
			}

			//	//TODO, should save to store in case there are so many crazy directories, OOM risk
			//log.Infof("google drive directory: ID=%s, Name=%s, Parents=%v", i.Id, i.Name, i.Parents)
			//	meta, err := srv.Files.Get(i.Id).
			//		Fields("id, name, permissions(id,emailAddress,displayName,role,type,permissionDetails), shared").
			//		Context(ctx).
			//		Do()
			//	if err != nil {
			//		log.Errorf("failed to get permissions for folder %s (%s): %v", i.Name, i.Id, err)
			//		// Check if it's a rate limit error
			//		if strings.Contains(err.Error(), "rateLimitExceeded") || strings.Contains(err.Error(), "userRateLimitExceeded") {
			//			log.Warnf("rate limit exceeded when fetching folder permissions, consider implementing backoff")
			//		}
			//		continue
			//	}
			//

			//	shared := meta.Shared
			//	hasACL := checkExplicit(meta.Permissions)
			//
			//	if parent == "" && shared {
			//		parent = shareWithMeID
			//	}
			//	ft.AddFolder(i.Id, i.Name, parent, shared, hasACL, meta.Permissions, i.ModifiedTime)
			//	if ok && isSamePermission(node.Permissions, parent.Permissions) {
			//	} else {
			//		log.Tracef("Folder: %s, permissions: %v", node.FullPath, util.MustToJSON(node.Permissions))
			//		if len(node.Permissions) > 0 {
			//			ep := security.ExternalPermission{
			//				BatchNumber:  batchNumber,
			//				Source:       datasource.ID,
			//				ExternalID:   node.ID,
			//				ResourceID:   node.ID, //TODO, need hash to an internal ID
			//				ResourceType: security.FolderResource,
			//				ResourcePath: node.FullPath,
			//				Explicit:     true,
			//				ParentID:     node.ParentID,
			//
			//				Permissions: []security.ExternalPermissionEntry{},
			//			}
			//
			//			for _, perm := range node.Permissions {
			//				entry := security.ExternalPermissionEntry{}
			//				entry.PrincipalType = perm.Type
			//				entry.PrincipalID = perm.Id
			//				entry.PrimaryIdentity = perm.EmailAddress
			//
			//				// Ensure DisplayName is never empty
			//				entry.DisplayName = perm.DisplayName
			//				if entry.DisplayName == "" && perm.EmailAddress != "" {
			//					// Fallback to email username if display name is missing
			//					parts := strings.Split(perm.EmailAddress, "@")
			//					if len(parts) > 0 {
			//						entry.DisplayName = parts[0]
			//					}
			//				}
			//
			//				entry.Role = perm.Role
			//				entry.Inherited = false
			//				ep.Permissions = append(ep.Permissions, entry)
			//			}
			//
			//			if node.ModifiedTime != "" {
			//				parsedTime, err := time.Parse(time.RFC3339Nano, node.ModifiedTime)
			//				if err == nil {
			//					ep.ExternalUpdatedAt = &parsedTime
			//				}
			//			}
			//
			//			ep.ID = util.MD5digest(fmt.Sprintf("%v-external-permission-%v", datasource.ID, node.ID))
			//			ep.System = datasource.System
			//
			//			//save external permissions
			//			err := orm.Save(ormCtx, &ep)
			//			if err != nil {
			//				panic(err)
			//			}
			//		}
			//
			//	}

			parent := []string{}
			parent = append(parent, doc.Categories...)
			parent = append(parent, doc.Title)

			if i.MimeType == "application/vnd.google-apps.folder" {
				folder := this.createFolderDoc(i.Id, i.Name, parent, datasource)
				this.IndexingFolder(pipeCtx, connector, datasource, folder, srv, fmt.Sprintf("'%v' in parents and trashed = false", i.Id), batchNumber)
				continue
			} else {
				this.IndexingFile(pipeCtx, connector, datasource, parent, doc, srv, i, batchNumber)
			}
		}

		nextPageToken = r.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
	return
}

func (this *Processor) IndexingFile(pipeCtx *pipeline.Context, connector *common.Connector, datasource *common.DataSource, parent []string, folder common.Document, srv *drive.Service, i *drive.File, batchNumber string) {

	//if processedFileIDs[i.Id] {
	//check bitmap, if files are processed already, no duplicated progressing
	//	log.Tracef("Skipping already processed file: %s", i.Id)
	//	continue
	//}

	//TODO configurable
	if i.Name == ".DS_Store" {
		return
	}

	log.Tracef("saving file: %v, %v", i.Name, i.MimeType)

	var createdAt, updatedAt *time.Time
	if i.CreatedTime != "" {
		parsedTime, err := time.Parse(time.RFC3339Nano, i.CreatedTime)
		if err == nil {
			createdAt = &parsedTime
		}
	}
	if i.ModifiedTime != "" {
		parsedTime, err := time.Parse(time.RFC3339Nano, i.ModifiedTime)
		if err == nil {
			updatedAt = &parsedTime
		}

		//// Track the most recent "ModifiedTime"
		//if updatedAt != nil && (*lastModifyTime == nil || updatedAt.After(**lastModifyTime)) {
		//	*lastModifyTime = updatedAt
		//}
	}

	// Map Google Drive file to Document struct
	document := common.Document{
		Source: common.DataSourceReference{
			ID:   datasource.ID,
			Name: datasource.Name,
			Type: "connector",
		},
		Title:   i.Name,
		Summary: i.Description,
		Type:    getType(i.MimeType),
		Size:    int(i.Size),
		URL:     fmt.Sprintf("https://drive.google.com/file/d/%s/view", i.Id),
		Owner: &common.UserInfo{
			UserAvatar: i.Owners[0].PhotoLink,
			UserName:   i.Owners[0].DisplayName,
			UserID:     i.Owners[0].EmailAddress,
		},
		Icon:      getIcon(i.MimeType),
		Thumbnail: i.ThumbnailLink,
	}

	document.System = datasource.System
	document.ID = common.GetDocID(datasource.ID, i.Id)
	document.Created = createdAt
	document.Updated = updatedAt

	if len(parent) > 0 {
		path := common.GetFullPathForCategories(parent)
		document.Category = path
		document.Categories = parent
		if document.System == nil {
			document.System = util.MapStr{}
		}
		document.System[common.SystemHierarchyPathKey] = path
	} else {
		log.Warnf("empty category, file: %v,  parents: %v", i.Name, i.Parents)
	}

	meta := util.MapStr{
		"batch_number":   batchNumber,
		"drive_id":       i.DriveId,
		"file_id":        i.Id,
		"email":          i.Owners[0].EmailAddress,
		"file_extension": i.FileExtension,
		"kind":           i.Kind,
		"mimetype":       i.MimeType,
		"shared_with_me": i.SharedWithMeTime,
		"sharing_user":   i.SharingUser,
		"shared":         i.Shared,
		"spaces":         i.Spaces,
		"starred":        i.Starred,
		"web_view_link":  i.WebViewLink,
		"labels":         i.LabelInfo,
		"parents":        i.Parents,
		"permissions":    i.Permissions,
		"permission_ids": i.PermissionIds,
		"properties":     i.Properties,
	}

	document.Metadata = meta.RemoveNilItems()

	//if i.Permissions != nil {
	//	// Ensure DisplayName is populated for file permissions
	//	for j := range i.Permissions {
	//		perm := i.Permissions[j]
	//		if perm.DisplayName == "" && perm.EmailAddress != "" {
	//			// Fallback to email username if display name is missing
	//			parts := strings.Split(perm.EmailAddress, "@")
	//			if len(parts) > 0 {
	//				perm.DisplayName = parts[0]
	//			}
	//		}
	//	}
	//
	//	log.Debugf("permission for file: %v(%v) %v", i.Id, i.Name, util.ToJson(i.Permissions, true))
	//
	//	//TODO check dedicated file permission, save to external permission
	//	if document.Type != string(security.FolderResource) && !isSamePermission(i.Permissions, parentPermissions) {
	//		log.Debug("different file permission: ", util.MustToJSON(i), ",vs: ", util.MustToJSON(parentPermissions))
	//	}
	//}

	if i.LastModifyingUser != nil {
		document.LastUpdatedBy = &common.EditorInfo{
			UserInfo: &common.UserInfo{
				UserAvatar: i.LastModifyingUser.PhotoLink,
				UserName:   i.LastModifyingUser.DisplayName,
				UserID:     i.LastModifyingUser.EmailAddress,
			},
			UpdatedAt: updatedAt,
		}
	}

	document.Payload = util.MapStr{}

	// Handle optional fields
	if i.SharingUser != nil {
		document.Payload["sharingUser"] = common.UserInfo{
			UserAvatar: i.SharingUser.PhotoLink,
			UserName:   i.SharingUser.DisplayName,
			UserID:     i.SharingUser.EmailAddress,
		}
	}

	if i.VideoMediaMetadata != nil {
		document.Payload["video_metadata"] = i.VideoMediaMetadata
	}

	if i.ImageMediaMetadata != nil {
		document.Payload["image_metadata"] = i.ImageMediaMetadata
	}

	this.Collect(pipeCtx, connector, datasource, document)
}

func (this *Processor) createFolderDoc(id, name string, parent []string, datasource *common.DataSource) common.Document {
	shareWithMe := common.CreateHierarchyPathFolderDoc(datasource, id, name, parent)
	shareWithMe.URL = fmt.Sprintf("https://drive.google.com/file/d/%s/view", id)
	return shareWithMe
}

func (this *Processor) startIndexingFiles(pipeCtx *pipeline.Context, connector *common.Connector, datasource *common.DataSource, tok *oauth2.Token) {
	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}
				log.Error("error on indexing google drive files,", v)
			}
		}
	}()

	if datasource.SyncConfig.PageSize <= 0 {
		datasource.SyncConfig.PageSize = 100
	}

	oAuthConfig := getOAuthConfig(connector.ID)
	if oAuthConfig == nil {
		panic("invalid oauth config")
	}

	client := oAuthConfig.Client(context.Background(), tok)
	client.Timeout = this.timeout
	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		panic(err)
	}

	batchNumber := util.GetUUID()

	//My Drive
	rootFolderID, rootFolderName := getRootFolderID(srv)
	rootDoc := this.createFolderDoc(rootFolderID, rootFolderName, []string{}, datasource)

	q := fmt.Sprintf("'%v' in parents and trashed = false", rootFolderID)
	this.IndexingFolder(pipeCtx, connector, datasource, rootDoc, srv, q, batchNumber)

	// /Shared with me
	shareWithMe := this.createFolderDoc("share_with_me", "Shared with me", []string{}, datasource)

	this.IndexingFolder(pipeCtx, connector, datasource, shareWithMe, srv, "sharedWithMe", batchNumber)

}

// getRootFolderOwner determines who owns the root folder of a file's path hierarchy
func (this *Processor) getRootFolderOwner(parentInfos []struct {
	pathArray   []string
	permissions []*drive.Permission
	folderNode  *FolderNode
}, ft *FolderTreeBuilder, srv *drive.Service, currentUserEmail string) string {
	if len(parentInfos) == 0 {
		return ""
	}

	// Use the first parent as starting point to trace up to root
	startNode := parentInfos[0].folderNode

	// Trace up the hierarchy to find the root folder
	currentNode := startNode
	for {
		if currentNode.ParentID == "" || currentNode.ParentID == currentNode.ID {
			// Found root folder
			break
		}

		// Move up to parent
		if parentNode, ok := ft.FolderMap[currentNode.ParentID]; ok {
			currentNode = parentNode
		} else {
			break
		}
	}

	// Check if we can determine the root folder owner
	if currentNode != nil {
		// Try to get root folder metadata to check ownership
		rootMeta, err := srv.Files.Get(currentNode.ID).
			Fields("id, name, owners(emailAddress), shared").
			Context(context.Background()).Do()

		if err == nil && rootMeta != nil && len(rootMeta.Owners) > 0 {
			rootOwnerEmail := rootMeta.Owners[0].EmailAddress
			log.Debugf("root folder %s (%s) owner: %s, current user: %s",
				rootMeta.Name, rootMeta.Id, rootOwnerEmail, currentUserEmail)
			return rootOwnerEmail
		}
	}

	// Fallback: if we can't determine root owner, assume it's the current user
	// This is a conservative approach - better to assume ownership than incorrectly categorize as shared
	log.Debugf("cannot determine root folder owner, assuming current user owns the folder hierarchy")
	return currentUserEmail
}

// downloadOrExportFile downloads or exports a Google Drive file based on its MIME type
func downloadOrExportFile(srv *drive.Service, fileID, outputPath string) error {
	// Get file metadata to check MIME type
	file, err := srv.Files.Get(fileID).Fields("mimeType", "name").Do()
	if err != nil {
		return fmt.Errorf("Unable to retrieve file metadata: %v", err)
	}

	// Determine if the file can be downloaded directly or needs export
	switch file.MimeType {
	case "application/vnd.google-apps.document":
		return exportFile(srv, fileID, "text/plain", outputPath)
	case "application/vnd.google-apps.spreadsheet":
		return exportFile(srv, fileID, "text/csv", outputPath)
	case "application/vnd.google-apps.presentation":
		return exportFile(srv, fileID, "text/plain", outputPath)
	case " application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return exportFile(srv, fileID, "text/plain", outputPath)
	default:
		return downloadFile(srv, fileID, outputPath)
	}
}

// downloadFile directly downloads a binary file from Google Drive
func downloadFile(srv *drive.Service, fileID, outputPath string) error {
	// Download the file from Google Drive
	resp, err := srv.Files.Get(fileID).Download()
	if err != nil {
		return fmt.Errorf("failed to download file with ID %q: %w", fileID, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Errorf("warning: failed to close response body: %v", cerr)
		}
	}()

	// Create the output file on disk
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create local file at %q: %w", outputPath, err)
	}
	defer func() {
		if cerr := outFile.Close(); cerr != nil {
			log.Errorf("warning: failed to close output file: %v", cerr)
		}
	}()

	// Copy the content to the local file
	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write to file %q: %w", outputPath, err)
	}

	return nil
}

// exportFile exports a Google Docs-type file (Docs, Sheets, Slides) to a specific format
func exportFile(srv *drive.Service, fileID, mimeType, outputPath string) error {
	res, err := srv.Files.Export(fileID, mimeType).Download()
	if err != nil {
		return fmt.Errorf("Unable to export file: %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	// Create the local file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("Unable to create file: %v", err)
	}
	defer func() {
		_ = f.Close()
	}()

	// Write the contents to the local file
	if _, err := io.Copy(f, res.Body); err != nil {
		return fmt.Errorf("Unable to save file locally: %v", err)
	}
	return nil
}

// createDocumentFromFile creates a Document from a Google Drive file
func (this *Processor) createDocumentFromFile(file *drive.File, datasource *common.DataSource, folderPath []string, ft *FolderTreeBuilder, currentUserEmail string, isMyDrive bool) *common.Document {
	if file == nil {
		return nil
	}

	// Skip .DS_Store files
	if file.Name == ".DS_Store" {
		return nil
	}

	var createdAt, updatedAt *time.Time
	if file.CreatedTime != "" {
		parsedTime, err := time.Parse(time.RFC3339Nano, file.CreatedTime)
		if err == nil {
			createdAt = &parsedTime
		}
	}
	if file.ModifiedTime != "" {
		parsedTime, err := time.Parse(time.RFC3339Nano, file.ModifiedTime)
		if err == nil {
			updatedAt = &parsedTime
		}
	}

	log.Tracef("Google Drive File: %s (ID: %s) | CreatedAt: %s | UpdatedAt: %s | Parents: %s", file.Name, file.Id, createdAt, updatedAt, file.Parents)

	// Map Google Drive file to Document struct
	document := common.Document{
		Source: common.DataSourceReference{
			ID:   datasource.ID,
			Name: datasource.Name,
			Type: "connector",
		},
		Title:   file.Name,
		Summary: file.Description,
		Type:    getType(file.MimeType),
		Size:    int(file.Size),
		URL:     fmt.Sprintf("https://drive.google.com/file/d/%s/view", file.Id),
		Owner: &common.UserInfo{
			UserAvatar: file.Owners[0].PhotoLink,
			UserName:   file.Owners[0].DisplayName,
			UserID:     file.Owners[0].EmailAddress,
		},
		Icon:      getIcon(file.MimeType),
		Thumbnail: file.ThumbnailLink,
	}

	document.System = datasource.System
	document.ID = common.GetDocID(datasource.ID, file.Id)
	document.Created = createdAt
	document.Updated = updatedAt

	// Set category and path
	if len(folderPath) > 0 {
		document.Category = common.GetFullPathForCategories(folderPath)
		document.Categories = folderPath
		if document.System == nil {
			document.System = util.MapStr{}
		}
		document.System[common.SystemHierarchyPathKey] = document.Category
	}

	// Build metadata
	meta := util.MapStr{
		"batch_number":   util.GetUUID(),
		"drive_id":       file.DriveId,
		"file_id":        file.Id,
		"email":          file.Owners[0].EmailAddress,
		"file_extension": file.FileExtension,
		"kind":           file.Kind,
		"mimetype":       file.MimeType,
		"shared_with_me": file.SharedWithMeTime,
		"sharing_user":   file.SharingUser,
		"shared":         file.Shared,
		"spaces":         file.Spaces,
		"starred":        file.Starred,
		"web_view_link":  file.WebViewLink,
		"labels":         file.LabelInfo,
		"parents":        file.Parents,
		"permissions":    file.Permissions,
		"permission_ids": file.PermissionIds,
		"properties":     file.Properties,
	}

	document.Metadata = meta.RemoveNilItems()

	// Handle permissions
	if file.Permissions != nil {
		// Ensure DisplayName is populated for file permissions
		for j := range file.Permissions {
			perm := file.Permissions[j]
			if perm.DisplayName == "" && perm.EmailAddress != "" {
				// Fallback to email username if display name is missing
				parts := strings.Split(perm.EmailAddress, "@")
				if len(parts) > 0 {
					perm.DisplayName = parts[0]
				}
			}
		}
	}

	if file.LastModifyingUser != nil {
		document.LastUpdatedBy = &common.EditorInfo{
			UserInfo: &common.UserInfo{
				UserAvatar: file.LastModifyingUser.PhotoLink,
				UserName:   file.LastModifyingUser.DisplayName,
				UserID:     file.LastModifyingUser.EmailAddress,
			},
			UpdatedAt: updatedAt,
		}
	}

	document.Payload = util.MapStr{}

	// Handle optional fields
	if file.SharingUser != nil {
		document.Payload["sharingUser"] = common.UserInfo{
			UserAvatar: file.SharingUser.PhotoLink,
			UserName:   file.SharingUser.DisplayName,
			UserID:     file.SharingUser.EmailAddress,
		}
	}

	if file.VideoMediaMetadata != nil {
		document.Payload["video_metadata"] = file.VideoMediaMetadata
	}

	if file.ImageMediaMetadata != nil {
		document.Payload["image_metadata"] = file.ImageMediaMetadata
	}

	return &document
}
