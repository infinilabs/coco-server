package google_drive

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"io"
	"net/http"
	"os"
	"time"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, tokenPath string) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenPath, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		panic(err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		panic(err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	log.Debug("Saving credential file to: %s", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func startIndexingFiles(credentialsPath,tokenPath string, outputQueue *queue.QueueConfig) {
	ctx := context.Background()
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		panic(err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, slides.DriveScope) //drive.DriveMetadataReadonlyScope
	if err != nil {
		panic(err)
	}
	client := getClient(config,tokenPath)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		panic(err)
	}

	r, err := srv.Files.List().PageSize(10).
		Fields("nextPageToken, files(id, name, mimeType, size, owners(emailAddress, displayName), createdTime, " +
			"modifiedTime, lastModifyingUser(emailAddress, displayName), iconLink, fileExtension, description, hasThumbnail," +
			" kind, labelInfo, parents, properties, shared, sharingUser(emailAddress, displayName), spaces, " +
			"starred, driveId, thumbnailLink, videoMediaMetadata, webViewLink, imageMediaMetadata)").Do()
	if err != nil {
		panic(err)
	}
	if len(r.Files) == 0 {
		log.Debug("No files found.")
	} else {
		for _, i := range r.Files {
			var createdAt, updatedAt *time.Time
			if i.CreatedTime != "" {
				parsedTime, err := time.Parse(time.RFC3339, i.CreatedTime)
				if err == nil {
					createdAt = &parsedTime
				}
			}
			if i.ModifiedTime != "" {
				parsedTime, err := time.Parse(time.RFC3339, i.ModifiedTime)
				if err == nil {
					updatedAt = &parsedTime
				}
			}
			// Map Google Drive file to Document struct
			document := common.Document{
				Source:  "google_drive",
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
				Icon:      i.IconLink,
				Thumbnail: i.ThumbnailLink,
				Metadata: util.MapStr{
					"driveId":        i.DriveId,
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
				},
			}

			document.ID=i.Id
			document.Created = createdAt
			document.Updated = updatedAt

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

			// Handle optional fields like VideoMediaMetadata and ImageMediaMetadata
			if i.SharingUser != nil {
				document.Metadata["sharingUser"] = common.UserInfo{
					UserAvatar: i.SharingUser.PhotoLink,
					UserName:   i.SharingUser.DisplayName,
					UserID:     i.SharingUser.EmailAddress,
				}
			}

			if i.VideoMediaMetadata != nil {
				document.Metadata["video_metadata"] = i.VideoMediaMetadata
			}

			if i.ImageMediaMetadata != nil {
				document.Metadata["image_metadata"] = i.ImageMediaMetadata
			}

			data:=util.MustToJSONBytes(document)
			if global.Env().IsDebug{
				log.Tracef(string(data))
			}
			err:=queue.Push(queue.SmartGetOrInitConfig(outputQueue),data)
			if err!=nil{
				panic(err)
			}
		}
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
