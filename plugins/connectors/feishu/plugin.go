/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package feishu

import (
	"encoding/json"
	"fmt"
	"infini.sh/coco/core"
	"net/url"
	"strings"
	"time"

	"infini.sh/coco/modules/common"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

// PluginType represents the type of plugin (feishu or lark)
type PluginType string

const (
	PluginTypeFeishu PluginType = "feishu"
	PluginTypeLark   PluginType = "lark"
	ConnectorFeishu             = "feishu"
	ConnectorLark               = "lark"
)

// APIConfig holds API endpoints for different plugin types
type APIConfig struct {
	BaseURL         string
	AuthURL         string
	TokenURL        string
	RefreshTokenURL string
	UserInfoURL     string
	DriveURL        string
}

// getAPIConfig returns the appropriate API configuration based on plugin type
func getAPIConfig(pluginType PluginType) *APIConfig {
	switch pluginType {
	case PluginTypeFeishu:
		return &APIConfig{
			BaseURL:         "https://open.feishu.cn",
			AuthURL:         "https://accounts.feishu.cn/open-apis/authen/v1/authorize",
			TokenURL:        "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
			RefreshTokenURL: "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
			UserInfoURL:     "https://open.feishu.cn/open-apis/authen/v1/user_info",
			DriveURL:        "https://open.feishu.cn/open-apis/drive/v1/files",
		}
	case PluginTypeLark:
		return &APIConfig{
			BaseURL:         "https://open.larksuite.com",
			AuthURL:         "https://accounts.larksuite.com/open-apis/authen/v1/authorize",
			TokenURL:        "https://open.larksuite.com/open-apis/authen/v2/oauth/token",
			RefreshTokenURL: "https://open.larksuite.com/open-apis/authen/v2/oauth/token",
			UserInfoURL:     "https://open.larksuite.com/open-apis/authen/v1/user_info",
			DriveURL:        "https://open.larksuite.com/open-apis/drive/v1/files",
		}
	default:
		// Default to feishu
		return getAPIConfig(PluginTypeFeishu)
	}
}

type Plugin struct {
	cmn.ConnectorProcessorBase
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
	ClientID        string `config:"client_id" json:"client_id"`
	ClientSecret    string `config:"client_secret" json:"client_secret"`
	UserAccessToken string `config:"user_access_token" json:"user_access_token"`
}

type Config struct {
	// OAuth token fields (for datasource config)
	AccessToken        string      `config:"access_token" json:"access_token"`
	RefreshToken       string      `config:"refresh_token" json:"refresh_token"`
	TokenExpiry        string      `config:"token_expiry" json:"token_expiry"`
	RefreshTokenExpiry string      `config:"refresh_token_expiry" json:"refresh_token_expiry"`
	Profile            util.MapStr `config:"profile" json:"profile"`
	// Connector-level config
	DocumentTypes []string `config:"document_types" json:"document_types"`
	PageSize      int      `config:"page_size" json:"page_size"`
}

// SearchContext holds parameters for recursive file search
type SearchContext struct {
	Ctx             *pipeline.Context
	Connector       *core.Connector
	Token           string
	FolderToken     string
	DocTypes        []string
	PageSize        int
	DataSource      *core.DataSource
	ParentPath      string
	ParentPathArray []string
	LastKnown       time.Time
	LatestSeen      *time.Time
}

func (this *Plugin) Name() string {
	return string(this.PluginType)
}

// NewFeishu creates a new Feishu pipeline processor
func NewFeishu(c *config.Config) (pipeline.Processor, error) {
	runner := &Plugin{}
	runner.SetPluginType(PluginTypeFeishu)
	runner.Init(c, runner)
	return runner, nil
}

// NewLark creates a new Lark pipeline processor
func NewLark(c *config.Config) (pipeline.Processor, error) {
	runner := &Plugin{}
	runner.SetPluginType(PluginTypeLark)
	runner.Init(c, runner)
	return runner, nil
}

func init() {
	// Register pipeline processors
	pipeline.RegisterProcessorPlugin(ConnectorFeishu, NewFeishu)
	pipeline.RegisterProcessorPlugin(ConnectorLark, NewLark)

	// Register OAuth routes for Feishu
	api.HandleUIMethod(api.GET, "/connector/:id/feishu/connect", handleOAuthConnect(PluginTypeFeishu), api.RequireLogin())
	api.HandleUIMethod(api.GET, "/connector/:id/feishu/oauth_redirect", handleOAuthRedirect(PluginTypeFeishu), api.RequireLogin())

	// Register OAuth routes for Lark
	api.HandleUIMethod(api.GET, "/connector/:id/lark/connect", handleOAuthConnect(PluginTypeLark), api.RequireLogin())
	api.HandleUIMethod(api.GET, "/connector/:id/lark/oauth_redirect", handleOAuthRedirect(PluginTypeLark), api.RequireLogin())
}

func (this *Plugin) Fetch(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	if connector == nil || datasource == nil {
		return errors.Errorf("invalid connector config")
	}

	cfg := Config{}
	this.MustParseConfig(datasource, &cfg)

	// Get access token - support both user_access_token and OAuth tokens
	var token string

	// First try OAuth tokens from datasource config
	if cfg.AccessToken != "" {
		// Check if access token is expired
		if cfg.TokenExpiry != "" {
			if expiry, err := time.Parse(time.RFC3339, cfg.TokenExpiry); err == nil {
				if time.Now().After(expiry) && cfg.RefreshToken != "" {
					// Check if refresh token is also expired
					if cfg.RefreshTokenExpiry != "" {
						if refreshExpiry, err := time.Parse(time.RFC3339, cfg.RefreshTokenExpiry); err == nil {
							if time.Now().After(refreshExpiry) {
								return errors.Errorf("[%s connector] both access token and "+
									"refresh token expired for datasource [%s]", this.PluginType, datasource.Name)
							}
						}
					}

					// Access token expired but refresh token still valid, try to refresh
					// Create OAuth handler to refresh token
					handler := NewOAuthHandler(this.PluginType, nil)
					newToken, err := handler.refreshAccessTokenWithConnectorConfig(cfg.RefreshToken, connector.Config)
					if err != nil {
						return errors.Errorf("[%s connector] failed to refresh token: %v", this.PluginType, err)
					}

					// Update datasource with new token
					cfg.AccessToken = newToken.AccessToken
					cfg.RefreshToken = newToken.RefreshToken
					cfg.TokenExpiry = time.Now().
						Add(time.Duration(newToken.ExpiresIn) * time.Second).Format(time.RFC3339)

					// Update refresh token expiry if provided
					if newToken.RefreshTokenExpiresIn > 0 {
						cfg.RefreshTokenExpiry = time.Now().
							Add(time.Duration(newToken.RefreshTokenExpiresIn) * time.Second).Format(time.RFC3339)
					}

					// Save updated token to datasource
					datasource.Connector.Config = cfg
					ormCtx := orm.NewContext().DirectAccess()
					if err := orm.Update(ormCtx, datasource); err != nil {
						return errors.Errorf("[%s connector] failed to save refreshed token: %v", this.PluginType, err)
					}
				}
			}
		}
		token = cfg.AccessToken
	} else {
		// Fallback to user_access_token from connector config
		if userAccessToken, ok := connector.Config["user_access_token"].(string); ok && userAccessToken != "" {
			token = strings.TrimSpace(userAccessToken)
		}
	}

	if token == "" {
		return errors.Errorf("[%s connector] missing access token for datasource [%s]", this.PluginType, datasource.Name)
	}

	// Set default document types if not specified
	var docTypes []string
	if len(cfg.DocumentTypes) > 0 {
		docTypes = cfg.DocumentTypes
	} else {
		// Try to get from connector config
		if dtypes, ok := connector.Config["document_types"].([]interface{}); ok {
			for _, dt := range dtypes {
				if str, ok := dt.(string); ok {
					docTypes = append(docTypes, str)
				}
			}
		}
		if len(docTypes) == 0 {
			docTypes = []string{"doc", "sheet", "slides", "mindnote", "bitable", "file", "docx", "folder", "shortcut"}
		}
	}

	pageSize := cfg.PageSize
	if pageSize <= 0 {
		// Try to get from connector config
		if ps, ok := connector.Config["page_size"].(int); ok && ps > 0 {
			pageSize = ps
		} else if ps, ok := connector.Config["page_size"].(float64); ok && ps > 0 {
			pageSize = int(ps)
		} else {
			pageSize = 100
		}
	}

	// Incremental sync: get last modified time saved for this datasource
	var lastKnown time.Time
	if lastStr, _ := this.GetLastModifiedTime(datasource.ID); lastStr != "" {
		lastKnown = getTime(lastStr)
		// Add a small buffer to ensure we don't miss documents due to timing issues
		lastKnown = lastKnown.Add(-1 * time.Minute)
	}
	var latestSeen time.Time

	// Start recursive search from root
	// Initialize path as "/" and categories as [] to align with Google Drive behavior
	searchCtx := &SearchContext{
		Ctx:             ctx,
		Connector:       connector,
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
		if err := this.SaveLastModifiedTime(datasource.ID, latestSeen.UTC().Format(time.RFC3339)); err != nil {
			log.Warnf("[%s connector] failed to save last modified time: %v", this.PluginType, err)
		}
	}

	log.Infof("[%s connector] sync completed for datasource: ID: %s, Name: %s", this.PluginType, datasource.ID, datasource.Name)
	return nil
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
			log.Errorf("[%s connector] Error parsing API response: %v, body: %s", this.PluginType, err, string(resBody))
			break // Skip this page and continue with next iteration
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

					// Collect folder directory document
					this.Collect(ctx.Ctx, ctx.Connector, ctx.DataSource, folderDoc)

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
						Ctx:             ctx.Ctx,
						Connector:       ctx.Connector,
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
			doc := core.Document{
				Source: core.DataSourceReference{
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
			if ctx.LatestSeen != nil {
				if ctx.LatestSeen.IsZero() || updatedAt.After(*ctx.LatestSeen) {
					*ctx.LatestSeen = updatedAt
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
			doc.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", ctx.DataSource.Connector.ConnectorID, ctx.DataSource.ID, key))

			if global.Env().IsDebug {
				log.Tracef("collecting document: %v", util.MustToJSON(doc))
			}
			this.Collect(ctx.Ctx, ctx.Connector, ctx.DataSource, doc)
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

// Note: Utility functions (getString, getTime, getBool, getIcon) moved to util.go for better code organization
