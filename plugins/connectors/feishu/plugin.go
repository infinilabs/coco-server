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

// SearchContext holds parameters for recursive file search
type SearchContext struct {
	Token           string
	FolderToken     string
	DocTypes        []string
	PageSize        int
	DataSource      *common.DataSource
	ParentPath      string
	ParentPathArray []string
	LastKnown       time.Time
	LatestSeen      *time.Time
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
	log.Debug("starting feishu plugin")
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
		lastKnown = getTime(lastStr)
		// Add a small buffer to ensure we don't miss documents due to timing issues
		lastKnown = lastKnown.Add(-1 * time.Minute)
	}
	var latestSeen time.Time

	// Start recursive search from root
	// Initialize path as "/" and categories as ["/"] to align with Google Drive behavior
	searchCtx := &SearchContext{
		Token:           token,
		FolderToken:     "",
		DocTypes:        docTypes,
		PageSize:        pageSize,
		DataSource:      datasource,
		ParentPath:      "/",
		ParentPathArray: []string{},
		LastKnown:       lastKnown,
		LatestSeen:      &latestSeen,
	}
	this.searchFilesRecursively(searchCtx)

	// Save last modified time for next incremental run
	if !latestSeen.IsZero() {
		_ = this.saveLastModifiedTime(datasource.ID, latestSeen.UTC().Format(time.RFC3339))
	}

	log.Infof("[%s connector] sync completed for datasource: ID: %s, Name: %s", this.PluginType, datasource.ID, datasource.Name)
}

// searchFilesRecursively recursively searches for files in folders
func (this *Plugin) searchFilesRecursively(ctx *SearchContext) {

	var nextPageToken string
	for {
		resBody, err := this.listFilesInFolder(ctx.Token, ctx.FolderToken, nextPageToken, ctx.PageSize)
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
			if !this.isSupportedDocumentType(docType, ctx.DocTypes) {
				continue
			}

			// If it's a folder, create directory document and recursively search it
			if docType == "folder" {
				if folderToken := getString(m, "token"); folderToken != "" {
					// Build next path
					folderName := getString(m, "name")
					if folderName == "" {
						folderName = getString(m, "title")
					}
					if folderName == "" {
						continue
					}

					// Create folder directory document
					folderDoc := common.CreateHierarchyPathFolderDoc(
						ctx.DataSource,
						folderToken,
						folderName,
						ctx.ParentPathArray,
					)
					folderDoc.URL = getString(m, "url")
					folderDoc.Metadata = util.MapStr{
						"folder_type":  "folder",
						"folder_token": folderToken,
						"platform":     string(this.PluginType),
					}

					// Add folder metadata
					if createdTime := getString(m, "created_time"); createdTime != "" {
						folderDoc.Metadata["created_time"] = createdTime
					}
					if modifiedTime := getString(m, "modified_time"); modifiedTime != "" {
						folderDoc.Metadata["modified_time"] = modifiedTime
					}

					// Save folder directory to queue
					dataBytes := util.MustToJSONBytes(&folderDoc)
					if err := queue.Push(queue.SmartGetOrInitConfig(this.Queue), dataBytes); err != nil {
						_ = log.Errorf("[%s connector] failed to push folder directory to queue: %v", this.PluginType, err)
					}

					// Compute new path string
					var nextPath string
					if ctx.ParentPath == "" || ctx.ParentPath == "/" {
						nextPath = "/" + folderName
					} else {
						nextPath = ctx.ParentPath + "/" + folderName
					}
					// Compute new path array (copy then append)
					nextPathArray := append(append([]string(nil), ctx.ParentPathArray...), folderName)

					// Create new context for recursive call
					nextCtx := &SearchContext{
						Token:           ctx.Token,
						FolderToken:     folderToken,
						DocTypes:        ctx.DocTypes,
						PageSize:        ctx.PageSize,
						DataSource:      ctx.DataSource,
						ParentPath:      nextPath,
						ParentPathArray: nextPathArray,
						LastKnown:       ctx.LastKnown,
						LatestSeen:      ctx.LatestSeen,
					}
					this.searchFilesRecursively(nextCtx)
				}
				continue
			}

			// Process document
			doc := common.Document{
				Source: common.DataSourceReference{
					ID:   ctx.DataSource.ID,
					Type: "connector",
					Name: ctx.DataSource.Name,
				},
			}
			doc.System = ctx.DataSource.System

			// Extract document information
			title := getString(m, "name")
			if title == "" {
				title = getString(m, "title")
			}
			doc.Title = title
			doc.Type = docType
			doc.Icon = getIcon(docType)
			doc.URL = getString(m, "url")

			// Use GetFullPathForCategories to build hierarchy path
			doc.Category = common.GetFullPathForCategories(ctx.ParentPathArray)
			doc.Categories = ctx.ParentPathArray

			if doc.System == nil {
				doc.System = util.MapStr{}
			}
			doc.System[common.SystemHierarchyPathKey] = doc.Category

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

			// Only skip if we have a valid lastKnown time and the document is not newer
			// This ensures we don't skip documents that might have been missed in previous syncs
			if !ctx.LastKnown.IsZero() && !updatedAt.IsZero() && !updatedAt.After(ctx.LastKnown) {
				continue
			}

			// Track latest seen modified time
			if ctx.LatestSeen.IsZero() || updatedAt.After(*ctx.LatestSeen) {
				*ctx.LatestSeen = updatedAt
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
			doc.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", ctx.DataSource.Connector.ConnectorID, ctx.DataSource.ID, key))

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
	// Fixed ordering: EditedTime DESC
	apiURL += "&order_by=EditedTime&direction=DESC"

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

// getIcon returns the appropriate icon for a Salesforce object type
func getIcon(docType string) string {
	switch docType {
	case "doc", "sheet", "slides", "mindnote", "bitable", "file", "docx":
		return docType
	default:
		return "default"
	}
}
