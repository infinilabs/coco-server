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
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/queue"
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

func buildFullPath(folderID string, folderMap map[string]*FolderNode) string {
	var parts []string
	current := folderMap[folderID]

	for current != nil {
		parts = append([]string{current.Name}, parts...)
		if current.ParentID == "" || current.ParentID == current.ID {
			break
		}
		current = folderMap[current.ParentID]
	}

	return "/" + strings.Join(parts, "/")
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

func (this *Processor) startIndexingFiles(connector *common.Connector, datasource *common.DataSource, tok *oauth2.Token) {
	var filesProcessed = 0
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

		if filesProcessed > 0 {
			log.Infof("[connector][google_drive] successfully indexed [%v]  files", filesProcessed) //TODO unify logging format
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
	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		panic(err)
	}

	// Init root folder
	// /My Drive
	rootFolderID, rootFolderName := getRootFolderID(srv)
	rootDoc := common.CreateHierarchyPathFolderDoc(datasource, rootFolderID, rootFolderName, []string{})
	rootDoc.URL = fmt.Sprintf("https://drive.google.com/file/d/%s/view", rootFolderID)
	this.saveDocToQueue(rootDoc, filesProcessed)

	// /Shared with me
	shareWithMe := common.CreateHierarchyPathFolderDoc(datasource, "share_with_me", "Shared with me", []string{})
	rootDoc.URL = fmt.Sprintf("https://drive.google.com/file/d/%s/view", "share_with_me")

	this.saveDocToQueue(shareWithMe, filesProcessed)

	ft := &FolderTreeBuilder{FolderMap: map[string]*FolderNode{}}
	ft.AddFolder(rootFolderID, rootFolderName, "", false, false, nil, "")

	batchNumber := util.GetUUID()

	// Fetch all directories
	var nextPageToken string
	for {
		if global.ShuttingDown() {
			break
		}

		call := srv.Files.List().
			PageSize(int64(datasource.SyncConfig.PageSize)).
			OrderBy("name").
			Q("mimeType='application/vnd.google-apps.folder' and trashed=false").
			Fields("nextPageToken, files(id, name, parents)")

		r, err := call.PageToken(nextPageToken).Do()
		if err != nil {
			panic(errors.Errorf("Failed to fetch directories: %v", err))
		}

		// Save directories in the map
		for _, i := range r.Files {
			if global.ShuttingDown() {
				return
			}
			//TODO, should save to store in case there are so many crazy directories, OOM risk
			log.Debugf("google drive directory: ID=%s, Name=%s, Parents=%v", i.Id, i.Name, i.Parents)

			meta, err := srv.Files.Get(i.Id).
				Fields("id, name, permissions(id,emailAddress,role,type,permissionDetails), shared").
				Context(ctx).
				Do()
			if err != nil {
				log.Errorf("failed to get permissions for folder %s: %v", i.Id, err)
				continue
			}

			parent := ""
			if len(i.Parents) > 0 {
				if i.Id != parent {
					parent = i.Parents[0]
				}
			}
			shared := meta.Shared
			hasACL := checkExplicit(meta.Permissions)

			ft.AddFolder(i.Id, i.Name, parent, shared, hasACL, meta.Permissions, i.ModifiedTime)
		}

		nextPageToken = r.NextPageToken
		if nextPageToken == "" {
			break
		}
	}

	ft.BuildSorted()
	ormCtx := orm.NewContext().DirectAccess()
	for _, node := range ft.Sorted {

		if global.ShuttingDown() {
			return
		}

		// Build full path array (including self)
		var fullParts []string
		curr := node
		for curr != nil {
			fullParts = append([]string{curr.Name}, fullParts...)
			curr = ft.FolderMap[curr.ParentID]
		}
		node.FullPathArray = fullParts

		// Build parent path array (excluding self)
		var parentParts []string
		curr = ft.FolderMap[node.ParentID] // Start from parent, not self
		for curr != nil {
			parentParts = append([]string{curr.Name}, parentParts...)
			curr = ft.FolderMap[curr.ParentID]
		}
		node.ParentPathArray = parentParts

		folderDoc := common.CreateHierarchyPathFolderDoc(datasource, node.ID, node.Name, parentParts)
		rootDoc.URL = fmt.Sprintf("https://drive.google.com/file/d/%s/view", node.ID)

		node.FullPath = "/" + strings.Join(fullParts, "/")

		this.saveDocToQueue(folderDoc, filesProcessed)

		parent, ok := ft.FolderMap[node.ParentID]
		if ok && isSamePermission(node.Permissions, parent.Permissions) {
		} else {
			log.Tracef("Folder: %s, permissions: %v", node.FullPath, util.MustToJSON(node.Permissions))
			if len(node.Permissions) > 0 {
				ep := security.ExternalPermission{
					BatchNumber:  batchNumber,
					Source:       datasource.ID,
					ExternalID:   node.ID,
					ResourceID:   node.ID, //TODO, need hash to an internal ID
					ResourceType: security.FolderResource,
					ResourcePath: node.FullPath,
					Explicit:     true,
					ParentID:     node.ParentID,

					Permissions: []security.ExternalPermissionEntry{},
				}

				for _, perm := range node.Permissions {
					entry := security.ExternalPermissionEntry{}
					entry.PrincipalType = perm.Type
					entry.PrincipalID = perm.Id
					entry.PrimaryIdentity = perm.EmailAddress
					entry.DisplayName = perm.DisplayName
					entry.Role = perm.Role
					entry.Inherited = false
					ep.Permissions = append(ep.Permissions, entry)
				}

				if node.ModifiedTime != "" {
					parsedTime, err := time.Parse(time.RFC3339Nano, node.ModifiedTime)
					if err == nil {
						ep.ExternalUpdatedAt = &parsedTime
					}
				}

				ep.ID = util.MD5digest(fmt.Sprintf("%v-external-permission-%v", datasource.ID, node.ID))
				ep.System = datasource.System

				//save external permissions
				err := orm.Save(ormCtx, &ep)
				if err != nil {
					panic(err)
				}
			}

		}

	}

	// Fetch all files
	var query string

	//get last access time from kv
	lastModifiedTimeStr, _ := this.GetLastModifiedTime(datasource.ID)

	log.Tracef("get last modified time: %v", lastModifiedTimeStr)

	if lastModifiedTimeStr != "" { //TODO, if the files are newly shared and with old timestamp and we may missed
		// Parse last indexed time
		parsedTime, err := time.Parse(time.RFC3339Nano, lastModifiedTimeStr)
		if err != nil {
			panic(errors.Errorf("Invalid time format: %v", err))
		}
		lastModifiedTimeStr = parsedTime.Format(time.RFC3339Nano)
		query = fmt.Sprintf("modifiedTime > '%s'", lastModifiedTimeStr)
	}

	var lastModifyTime *time.Time

	// Start pagination loop
	nextPageToken = ""
	for {
		if global.ShuttingDown() {
			break
		}

		call := srv.Files.List().PageSize(int64(datasource.SyncConfig.PageSize)).OrderBy("modifiedTime asc")

		if query != "" {
			call = call.Q(query)
		}

		r, err := call.
			PageToken(nextPageToken).
			Fields("nextPageToken, files(id, name, parents, mimeType, size, owners(emailAddress, displayName), createdTime, " +
				"modifiedTime, lastModifyingUser(emailAddress, displayName), iconLink, fileExtension, description, hasThumbnail," +
				"kind, labelInfo, parents, properties, shared, sharingUser(emailAddress, displayName), spaces, " +
				"starred, driveId, thumbnailLink, videoMediaMetadata, webViewLink, imageMediaMetadata)").Do()
		if err != nil {
			panic(err)
		}

		log.Tracef("fetched %v files", len(r.Files))

		// Process files in the current page
		for _, i := range r.Files {

			//TODO configurable
			if i.Name == ".DS_Store" {
				continue
			}

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

				// Track the most recent "ModifiedTime"
				if updatedAt != nil && (lastModifyTime == nil || updatedAt.After(*lastModifyTime)) {
					lastModifyTime = updatedAt
				}
			}

			log.Tracef("Google Drive File: %s (ID: %s) | CreatedAt: %s | UpdatedAt: %s | Parents: %s", i.Name, i.Id, createdAt, updatedAt, i.Parents)

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

			var parentCategoryArray []string
			var parentPermissions []*drive.Permission
			if i.Parents != nil && len(i.Parents) == 1 {
				v, ok := ft.FolderMap[i.Parents[0]]
				if ok && v != nil {
					parentPermissions = v.Permissions
					if ok {
						parentCategoryArray = v.FullPathArray
						log.Tracef("file: %v, full path: %v", i.Name, v.FullPath)
					}
				}
			} else {
				if i.Shared {
					parentCategoryArray = []string{"Shared with me"}
				} else {
					parentCategoryArray = []string{"/"}
				}
			}

			if len(parentCategoryArray) > 0 {
				path := common.GetFullPathForCategories(parentCategoryArray)
				document.Category = path
				document.Categories = parentCategoryArray
				if document.System == nil {
					document.System = util.MapStr{}
				}
				document.System[common.SystemHierarchyPathKey] = path
			} else {
				log.Warnf("empty category, file: %v,  parents: %v", i.Name, i.Parents)
			}

			log.Trace(document.Category, " // ", document.Categories, " // ", document.Title)

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

			if i.Permissions != nil {
				log.Debugf("permission for file: %v(%v) %v", i.Id, i.Name, util.ToJson(i.Permissions, true))

				//TODO check dedicated file permission, save to external permission
				if document.Type != string(security.FolderResource) && !isSamePermission(i.Permissions, parentPermissions) {
					log.Info("different file permission: ", util.MustToJSON(i), ",vs: ", util.MustToJSON(parentPermissions))
				}
			}

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

			this.saveDocToQueue(document, filesProcessed)
		}

		// After processing all files, save the most recent modified time for next indexing
		if lastModifyTime != nil {
			// Save the lastModifyTime (for example, in a KV store or file)
			lastModifiedTimeStr = lastModifyTime.Format(time.RFC3339Nano)
			err := this.SaveLastModifiedTime(lastModifiedTimeStr, datasource.ID)
			if err != nil {
				panic(err)
			}
			log.Debugf("Last modified time to be saved: %s", lastModifiedTimeStr)
		}

		// Break the loop if no next page token
		if r.NextPageToken == "" {
			break
		}
		nextPageToken = r.NextPageToken
	}
}

func (this *Processor) saveDocToQueue(document common.Document, filesProcessed int) {

	log.Debugf("save file: %v, %v, %v, %v, %v", document.ID, document.Category, document.Categories, document.Type, document.Title)

	// Convert to JSON and push to queue
	data := util.MustToJSONBytes(document)
	if global.Env().IsDebug {
		log.Tracef(string(data))
	}
	err := queue.Push(queue.SmartGetOrInitConfig(this.Queue), data)
	if err != nil {
		panic(err)
	}
	filesProcessed++
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
