/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package feishu

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

// PluginType represents the type of plugin (feishu or lark)
type PluginType string

const (
	PluginTypeFeishu PluginType = "feishu"
	PluginTypeLark   PluginType = "lark"
)

// APIConfig holds API endpoints for different plugin types
type APIConfig struct {
	BaseURL     string
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	DriveURL    string
}

// getAPIConfig returns the appropriate API configuration based on plugin type
func getAPIConfig(pluginType PluginType) *APIConfig {
	switch pluginType {
	case PluginTypeFeishu:
		return &APIConfig{
			BaseURL:     "https://open.feishu.cn",
			AuthURL:     "https://accounts.feishu.cn/open-apis/authen/v1/authorize",
			TokenURL:    "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
			UserInfoURL: "https://open.feishu.cn/open-apis/authen/v1/user_info",
			DriveURL:    "https://open.feishu.cn/open-apis/drive/v1/files",
		}
	case PluginTypeLark:
		return &APIConfig{
			BaseURL:     "https://open.larksuite.com",
			AuthURL:     "https://accounts.larksuite.com/open-apis/authen/v1/authorize",
			TokenURL:    "https://open.larksuite.com/open-apis/authen/v2/oauth/token",
			UserInfoURL: "https://open.larksuite.com/open-apis/authen/v1/user_info",
			DriveURL:    "https://open.larksuite.com/open-apis/drive/v1/files",
		}
	default:
		// Default to feishu
		return getAPIConfig(PluginTypeFeishu)
	}
}

type Plugin struct {
	api.Handler
	Enabled     bool               `config:"enabled"`
	Queue       *queue.QueueConfig `config:"queue"`
	Interval    string             `config:"interval"`
	PageSize    int                `config:"page_size"`
	OAuthConfig *OAuthConfig       `config:"o_auth_config"`
	// Plugin type to determine API endpoints
	PluginType PluginType
	// API configuration based on plugin type
	apiConfig *APIConfig
}

// SetPluginType sets the plugin type and initializes API configuration
func (this *Plugin) SetPluginType(pluginType PluginType) {
	this.PluginType = pluginType
	this.apiConfig = getAPIConfig(pluginType)
}

// GetPluginType returns the current plugin type
func (this *Plugin) GetPluginType() PluginType {
	return this.PluginType
}

// GetAPIConfig returns the current API configuration
func (this *Plugin) GetAPIConfig() *APIConfig {
	if this.apiConfig == nil {
		this.apiConfig = getAPIConfig(this.PluginType)
	}
	return this.apiConfig
}

type OAuthConfig struct {
	// OAuth endpoints
	AuthURL     string `config:"auth_url" json:"auth_url"`
	TokenURL    string `config:"token_url" json:"token_url"`
	RedirectURL string `config:"redirect_url" json:"redirect_url"`
	// OAuth credentials
	ClientID        string   `config:"client_id" json:"client_id"`
	ClientSecret    string   `config:"client_secret" json:"client_secret"`
	DocumentTypes   []string `config:"document_types" json:"document_types"`
	UserAccessToken string   `config:"user_access_token" json:"user_access_token"`
}

type Config struct {
	// OAuth token fields (for datasource config)
	AccessToken        string      `config:"access_token" json:"access_token"`
	RefreshToken       string      `config:"refresh_token" json:"refresh_token"`
	TokenExpiry        string      `config:"token_expiry" json:"token_expiry"`
	RefreshTokenExpiry string      `config:"refresh_token_expiry" json:"refresh_token_expiry"`
	Profile            util.MapStr `config:"profile" json:"profile"`
}

// Token represents the OAuth token response
type Token struct {
	Code                  int    `json:"code"`
	AccessToken           string `json:"access_token"`
	ExpiresIn             int    `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	TokenType             string `json:"token_type"`
	Scope                 string `json:"scope"`
	Error                 string `json:"error"`
	ErrorDescription      string `json:"error_description"`
}

// Setup initializes the plugin
func (this *Plugin) Setup() {
	return
}

func (this *Plugin) Start() error {
	return nil
}

func (this *Plugin) Stop() error {
	return nil
}

func (this *Plugin) Name() string {
	return ""
}

func (this *Plugin) fetchCloudDocs(connector *common.Connector, datasource *common.DataSource) {
	if connector == nil || datasource == nil {
		panic("invalid connector config")
	}

	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		panic(err)
	}
	obj := Config{}
	err = cfg.Unpack(&obj)
	if err != nil {
		panic(err)
	}

	// Get access token - support both user_access_token and OAuth tokens
	var token string

	// First try OAuth tokens
	if obj.AccessToken != "" {
		// Check if access token is expired
		if obj.TokenExpiry != "" {
			if expiry, err := time.Parse(time.RFC3339, obj.TokenExpiry); err == nil {
				if time.Now().After(expiry) && obj.RefreshToken != "" {
					// Check if refresh token is also expired
					if obj.RefreshTokenExpiry != "" {
						if refreshExpiry, err := time.Parse(time.RFC3339, obj.RefreshTokenExpiry); err == nil {
							if time.Now().After(refreshExpiry) {
								_ = log.Errorf("[%s connector] both access token and "+
									"refresh token expired for datasource [%s]", this.PluginType, datasource.Name)
								// Both tokens expired, need to re-authenticate
								return
							}
						}
					}

					// Access token expired but refresh token still valid, try to refresh
					newToken, err := this.refreshAccessToken(obj.RefreshToken)
					if err != nil {
						_ = log.Errorf("[%s connector] failed to refresh token: %v", this.PluginType, err)
						// Continue with expired token, API will return error
					} else {
						// Update datasource with new token
						obj.AccessToken = newToken.AccessToken
						obj.RefreshToken = newToken.RefreshToken
						obj.TokenExpiry = time.Now().
							Add(time.Duration(newToken.ExpiresIn) * time.Second).Format(time.RFC3339)

						// Update refresh token expiry if provided
						if newToken.RefreshTokenExpiresIn > 0 {
							obj.RefreshTokenExpiry = time.Now().
								Add(time.Duration(newToken.RefreshTokenExpiresIn) * time.Second).Format(time.RFC3339)
						}

						// Save updated token to datasource
						datasource.Connector.Config = obj
						ctx := orm.NewContext().DirectAccess()
						if err := orm.Update(ctx, datasource); err != nil {
							_ = log.Errorf("[%s connector] failed to save refreshed token: %v", this.PluginType, err)
						}
					}
				}
			}
		}
		token = obj.AccessToken
	} else if this.OAuthConfig != nil && this.OAuthConfig.UserAccessToken != "" {
		// Fallback to user_access_token from connector config
		token = strings.TrimSpace(this.OAuthConfig.UserAccessToken)
	}

	if token == "" {
		_ = log.Errorf("[%s connector] missing access token for datasource [%s]", this.PluginType, datasource.Name)
		return
	}

	// 1) Search cloud documents
	// Set default document types if not specified
	var docTypes []string
	if this.OAuthConfig != nil && len(this.OAuthConfig.DocumentTypes) > 0 {
		docTypes = this.OAuthConfig.DocumentTypes
	} else {
		docTypes = []string{"doc", "sheet", "slides", "mindnote", "bitable", "file", "docx", "folder", "shortcut"}
	}

	pageSize := this.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	// Incremental sync: get last modified time saved for this datasource
	var lastKnown time.Time
	if lastStr, _ := this.getLastModifiedTime(datasource.ID); lastStr != "" {
		if t, err := time.Parse(time.RFC3339Nano, lastStr); err == nil {
			lastKnown = t
		} else if t2, err2 := time.Parse(time.RFC3339, lastStr); err2 == nil {
			lastKnown = t2
		}
	}
	lastKnown = time.Now().Add(-30000 * time.Hour)
	var latestSeen time.Time

	// Start recursive search from root
	// Initialize path as "/" and categories as ["/"] to align with Google Drive behavior
	this.searchFilesRecursively(token, "", docTypes, pageSize, datasource, "/", []string{"/"}, lastKnown, &latestSeen)

	// Save last modified time for next incremental run
	if !latestSeen.IsZero() {
		_ = this.saveLastModifiedTime(datasource.ID, latestSeen.UTC().Format(time.RFC3339Nano))
	}

	// Log sync completion for this datasource
	log.Infof("[%s connector] sync completed for datasource: ID: %s, Name: %s", this.PluginType, datasource.ID, datasource.Name)
}

// searchFilesRecursively recursively searches for files in folders
func (this *Plugin) searchFilesRecursively(
	token, folderToken string, docTypes []string, pageSize int, datasource *common.DataSource,
	parentPath string, parentPathArray []string, lastKnown time.Time, latestSeen *time.Time,
) {

	var nextPageToken string
	for {
		resBody, err := this.listFilesInFolder(token, folderToken, nextPageToken, pageSize)
		if err != nil {
			_ = log.Errorf("[%s connector] list files in folder failed: %v", this.PluginType, err)
			break
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(resBody, &parsed); err != nil {
			panic(errors.Errorf("Error parsing response: %v", err))
		}

		data, _ := parsed["data"].(map[string]interface{})
		if data == nil {
			break
		}

		items, _ := data["files"].([]interface{})
		for i, it := range items {
			m, _ := it.(map[string]interface{})
			docType := getString(m, "type")

			// Skip if not in supported document types
			if !this.isSupportedDocumentType(docType, docTypes) {
				continue
			}

			// If it's a folder, recursively search it and propagate the folder path
			if docType == "folder" {
				if subFolderToken := getString(m, "token"); subFolderToken != "" {
					// Build next path
					folderName := getString(m, "name")
					if folderName == "" {
						folderName = getString(m, "title")
					}
					// Compute new path string
					var nextPath string
					if parentPath == "" || parentPath == "/" {
						nextPath = "/" + folderName
					} else {
						nextPath = parentPath + "/" + folderName
					}
					// Compute new path array (copy then append)
					nextPathArray := append(append([]string(nil), parentPathArray...), folderName)

					this.searchFilesRecursively(token, subFolderToken, docTypes, pageSize, datasource, nextPath, nextPathArray, lastKnown, latestSeen)
				}
				continue
			}

			// Process document
			doc := common.Document{
				Source: common.DataSourceReference{
					ID:   datasource.ID,
					Type: "connector",
					Name: datasource.Name,
				},
			}
			doc.System = datasource.System

			// Extract document information
			title := getString(m, "name")
			if title == "" {
				title = getString(m, "title")
			}
			doc.Title = title
			doc.Type = docType
			doc.Icon = "default"
			doc.URL = getString(m, "url")

			// Set path related fields
			currPath := parentPath
			if currPath == "" {
				currPath = "/"
			}
			doc.Category = currPath
			if doc.System == nil {
				doc.System = util.MapStr{}
			}
			doc.System["parent_path"] = currPath
			if len(parentPathArray) > 0 {
				doc.Categories = parentPathArray
			}

			if ct := getTime(getString(m, "created_time")); !ct.IsZero() {
				doc.Created = &ct
			} else {
				now := time.Now()
				doc.Created = &now
			}
			if ut := getTime(getString(m, "modified_time")); !ut.IsZero() {
				doc.Updated = &ut
			} else {
				doc.Updated = doc.Created
			}

			// Incremental filter: skip if not newer than last known
			var updatedAt time.Time
			if doc.Updated != nil {
				updatedAt = *doc.Updated
			} else if doc.Created != nil {
				updatedAt = *doc.Created
			}
			if !lastKnown.IsZero() && (updatedAt.IsZero() || !updatedAt.After(lastKnown)) {
				continue
			}

			// Track latest seen modified time
			if latestSeen != nil {
				if latestSeen.IsZero() || updatedAt.After(*latestSeen) {
					*latestSeen = updatedAt
				}
			}
			// Content is not returned in search; keep metadata/payload
			doc.Payload = m

			// stable id
			key := doc.URL
			if key == "" {
				key = getString(m, "token")
			}
			if key == "" {
				key = fmt.Sprintf("%d", i)
			}
			doc.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", datasource.Connector.ConnectorID, datasource.ID, key))

			dataBytes := util.MustToJSONBytes(doc)
			if global.Env().IsDebug {
				log.Tracef(string(dataBytes))
			}
			if err := queue.Push(queue.SmartGetOrInitConfig(this.Queue), dataBytes); err != nil {
				panic(err)
			}
		}

		// Check if there are more pages
		if hasMore := getBool(data, "has_more"); !hasMore {
			break
		}
		if nextPageToken = getString(data, "page_token"); nextPageToken == "" {
			break
		}
	}
}

// exchangeCodeForToken exchanges authorization code for access token
func (this *Plugin) exchangeCodeForToken(code string) (*Token, error) {
	if this.OAuthConfig == nil {
		return nil, errors.Errorf("OAuth config not initialized")
	}
	payload := map[string]interface{}{
		"client_id":     this.OAuthConfig.ClientID,
		"client_secret": this.OAuthConfig.ClientSecret,
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  this.OAuthConfig.RedirectURL,
	}

	req := util.NewPostRequest(this.OAuthConfig.TokenURL, util.MustToJSONBytes(payload))
	req.AddHeader("Content-Type", "application/json")

	res, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.Errorf("%s API error, no response", this.PluginType)
	}

	if res.StatusCode >= 300 {
		return nil, errors.Errorf("%s API error: status %d, body: %s", this.PluginType, res.StatusCode, string(res.Body))
	}

	var tokenResponse Token
	if err := json.Unmarshal(res.Body, &tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}

// refreshAccessToken refreshes the access token using refresh token
func (this *Plugin) refreshAccessToken(refreshToken string) (*Token, error) {
	if this.OAuthConfig == nil {
		return nil, errors.Errorf("OAuth config not initialized")
	}
	payload := map[string]interface{}{
		"client_id":     this.OAuthConfig.ClientID,
		"client_secret": this.OAuthConfig.ClientSecret,
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}

	req := util.NewPostRequest(this.OAuthConfig.TokenURL, util.MustToJSONBytes(payload))
	req.AddHeader("Content-Type", "application/json")
	res, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.Errorf("%s API error, no response", this.PluginType)
	}
	if res.StatusCode >= 300 {
		return nil, errors.Errorf("%s API error: status %d, body: %s", this.PluginType, res.StatusCode, string(res.Body))
	}

	var tokenResponse Token
	if err := json.Unmarshal(res.Body, &tokenResponse); err != nil {
		return nil, err
	}
	return &tokenResponse, nil
}

// getUserProfile retrieves user profile information
func (this *Plugin) getUserProfile(accessToken string) (util.MapStr, error) {
	apiConfig := this.GetAPIConfig()
	req := util.NewGetRequest(apiConfig.UserInfoURL, nil)
	req.AddHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	res, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.Errorf("%s API error, no response", this.PluginType)
	}
	if res.StatusCode >= 300 {
		return nil, errors.Errorf("%s API error: status %d, body: %s", this.PluginType, res.StatusCode, string(res.Body))
	}

	var response struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data util.MapStr `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &response); err != nil {
		return nil, err
	}
	if response.Code != 0 {
		return nil, errors.Errorf("%s API error: %s", this.PluginType, response.Msg)
	}
	return response.Data, nil
}

// listFilesInFolder lists files in a specific folder
func (this *Plugin) listFilesInFolder(tenantToken, folderToken, pageToken string, pageSize int) ([]byte, error) {
	if pageSize <= 0 {
		pageSize = 100
	}

	apiConfig := this.GetAPIConfig()
	apiURL := apiConfig.DriveURL
	apiURL += "?page_size=" + fmt.Sprintf("%d", pageSize)
	if folderToken != "" {
		apiURL += "&folder_token=" + url.QueryEscape(folderToken)
	}
	if pageToken != "" {
		apiURL += "&page_token=" + url.QueryEscape(pageToken)
	}
	// Fixed ordering: EditedTime ASC
	apiURL += "&order_by=EditedTime&direction=ASC"

	req := util.NewGetRequest(apiURL, nil)
	req.AddHeader("Authorization", fmt.Sprintf("Bearer %s", tenantToken))
	req.AddHeader("Content-Type", "application/json")

	res, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.Errorf("%s API error, no response", this.PluginType)
	}
	if res.StatusCode >= 300 {
		return nil, errors.Errorf("%s API error: status %d, body: %s", this.PluginType, res.StatusCode, string(res.Body))
	}
	return res.Body, nil
}

// isSupportedDocumentType checks if a document type is in the supported list
func (this *Plugin) isSupportedDocumentType(docType string, supportedTypes []string) bool {
	for _, st := range supportedTypes {
		if st == docType {
			return true
		}
	}
	return false
}

func getString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z07:00"}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	// Fallback: numeric Unix timestamp (seconds/milliseconds/microseconds/nanoseconds)
	isDigits := true
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			isDigits = false
			break
		}
	}
	if isDigits {
		if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
			var sec int64
			var nsec int64
			switch {
			case len(s) <= 10: // seconds
				sec = ts
				nsec = 0
			case len(s) <= 13: // milliseconds
				sec = ts / 1_000
				nsec = (ts % 1_000) * int64(time.Millisecond)
			case len(s) <= 16: // microseconds
				sec = ts / 1_000_000
				nsec = (ts % 1_000_000) * int64(time.Microsecond)
			default: // nanoseconds
				sec = ts / 1_000_000_000
				nsec = ts % 1_000_000_000
			}
			return time.Unix(sec, nsec)
		}
	}
	return time.Time{}
}

func getBool(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(bool); ok {
			return s
		}
	}
	return false
}
