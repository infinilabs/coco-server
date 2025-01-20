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
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"io"
	"os"
	"runtime"
	"time"
)

func getIcon(fileType string) string {
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
		return "folder"
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

func (this *Plugin) startIndexingFiles(connector *common.Connector, datasource *common.DataSource, tenantID, userID string, tok *oauth2.Token) {
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

	client := this.oAuthConfig.Client(context.Background(), tok)
	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		panic(err)
	}

	// All directories
	var directoryMap = map[string]common.RichLabel{}

	// Root Folder
	rootFolderID, rootFolderName := getRootFolderID(srv)
	directoryMap[rootFolderID] = common.RichLabel{Key: rootFolderID, Label: rootFolderName, Icon: "folder"}

	// Fetch all directories
	var nextPageToken string
	for {
		if global.ShuttingDown() {
			break
		}

		call := srv.Files.List().
			PageSize(int64(this.PageSize)).
			OrderBy("name").
			Q("mimeType='application/vnd.google-apps.folder' and trashed=false").
			Fields("nextPageToken, files(id, name, parents)")

		r, err := call.PageToken(nextPageToken).Do()
		if err != nil {
			panic(errors.Errorf("Failed to fetch directories: %v", err))
		}

		// Save directories in the map
		for _, i := range r.Files {
			//TODO, should save to store in case there are so many crazy directories, OOM risk
			directoryMap[i.Id] = common.RichLabel{Key: i.Id, Label: i.Name, Icon: "folder"}
			log.Debugf("google drive directory: ID=%s, Name=%s, Parents=%v", i.Id, i.Name, i.Parents)
		}

		nextPageToken = r.NextPageToken
		if nextPageToken == "" {
			break
		}
	}

	log.Infof("fetched %d google drive directories", len(directoryMap))

	// Fetch all files
	var query string

	//get last access time from kv
	lastModifiedTimeStr, _ := this.getLastModifiedTime(tenantID, userID, datasource.ID)

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

		call := srv.Files.List().PageSize(int64(this.PageSize)).OrderBy("modifiedTime asc")

		if query != "" {
			call = call.Q(query)
		}

		r, err := call.
			PageToken(nextPageToken).
			Fields("nextPageToken, files(id, name, mimeType, size, owners(emailAddress, displayName), createdTime, " +
				"modifiedTime, lastModifyingUser(emailAddress, displayName), iconLink, fileExtension, description, hasThumbnail," +
				"kind, labelInfo, parents, properties, shared, sharingUser(emailAddress, displayName), spaces, " +
				"starred, driveId, thumbnailLink, videoMediaMetadata, webViewLink, imageMediaMetadata)").Do()
		if err != nil {
			panic(err)
		}

		log.Debugf("fetched %v files", len(r.Files))

		// Process files in the current page
		for _, i := range r.Files {
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

			if i.MimeType == "application/vnd.google-apps.folder" {
				directoryMap[i.Id] = common.RichLabel{Key: i.Id, Label: i.Name, Icon: "folder"}
			}
			categories := []common.RichLabel{}
			if len(i.Parents) > 0 {
				for _, v := range i.Parents {
					folderName, ok := directoryMap[v]
					if ok {
						//log.Debugf("folder: %v, %v", folderName, v)
						categories = append(categories, folderName)
					} else {
						log.Errorf("missing folder info: %v", v)
						//TODO, if the parent_id is not found, delay to handle this file, maybe newly added file, and the folder meta is aware
					}
				}
			}

			log.Debugf("Google Drive File: %s (ID: %s) | CreatedAt: %s | UpdatedAt: %s", i.Name, i.Id, createdAt, updatedAt)

			// Map Google Drive file to Document struct
			document := common.Document{
				Source: common.DataSourceReference{
					ID:   datasource.ID,
					Name: datasource.Name,
					Type: "connector",
				},
				Title:   i.Name,
				Summary: i.Description,
				Type:    i.MimeType,
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

			if len(categories) > 0 {
				document.RichCategories = categories
			}

			document.ID = i.Id //add tenant namespace and then hash
			document.Created = createdAt
			document.Updated = updatedAt

			document.Metadata = util.MapStr{
				"drive_id":       i.DriveId,
				"file_id":        i.Id,
				"email":          i.Owners[0].EmailAddress,
				"file_extension": i.FileExtension,
				"kind":           i.Kind,
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

			if i.LastModifyingUser != nil {
				document.LastUpdatedBy = &common.EditorInfo{
					UserInfo: common.UserInfo{
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

		// After processing all files, save the most recent modified time for next indexing
		if lastModifyTime != nil {
			// Save the lastModifyTime (for example, in a KV store or file)
			lastModifiedTimeStr = lastModifyTime.Format(time.RFC3339Nano)
			err := this.saveLastModifiedTime(tenantID, userID, lastModifiedTimeStr, datasource.ID)
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
	res, err := srv.Files.Get(fileID).Download()
	if err != nil {
		return fmt.Errorf("Unable to download file: %v", err)
	}
	defer res.Body.Close()

	// Create the local file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("Unable to create local file: %v", err)
	}
	defer f.Close()

	// Write the contents to the local file
	if _, err := io.Copy(f, res.Body); err != nil {
		return fmt.Errorf("Unable to save file locally: %v", err)
	}
	return nil
}

// exportFile exports a Google Docs-type file (Docs, Sheets, Slides) to a specific format
func exportFile(srv *drive.Service, fileID, mimeType, outputPath string) error {
	res, err := srv.Files.Export(fileID, mimeType).Download()
	if err != nil {
		return fmt.Errorf("Unable to export file: %v", err)
	}
	defer res.Body.Close()

	// Create the local file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("Unable to create file: %v", err)
	}
	defer f.Close()

	// Write the contents to the local file
	if _, err := io.Copy(f, res.Body); err != nil {
		return fmt.Errorf("Unable to save file locally: %v", err)
	}
	return nil
}
