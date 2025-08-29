/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/api"
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

const (
	ConnectorFeishu = "feishu"
)

type Plugin struct {
	api.Handler
	Enabled  bool               `config:"enabled"`
	Queue    *queue.QueueConfig `config:"queue"`
	Interval string             `config:"interval"`
	PageSize int                `config:"page_size"`

	OAuthConfig *OAuthConfig
}

type OAuthConfig struct {
	AuthURL     string `config:"auth_uri" json:"auth_url"`
	TokenURL    string `config:"token_uri" json:"token_url"`
	RedirectURI string `config:"redirect_uris" json:"redirect_uri"`
}

type Config struct {
	ClientID     string `config:"client_id" json:"client_id"`
	ClientSecret string `config:"client_secret" json:"client_secret"`
	// Document types to search for (doc, sheet, slides, etc.)
	DocumentTypes []string `config:"document_types"`
	// Legacy support for user_access_token
	UserAccessToken string `config:"user_access_token"`
	// OAuth token fields (for datasource config)
	AccessToken  string      `config:"access_token" json:"access_token"`
	RefreshToken string      `config:"refresh_token" json:"refresh_token"`
	TokenExpiry  string      `config:"token_expiry" json:"token_expiry"`
	Profile      util.MapStr `config:"profile" json:"profile"`
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}

// Setup initializes the plugin
func (this *Plugin) Setup() {
	ok, err := env.ParseConfig("connector.feishu", &this)
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	if !this.Enabled {
		return
	}
	if this.PageSize <= 0 {
		this.PageSize = 100
	}
	if this.Queue == nil {
		this.Queue = &queue.QueueConfig{Name: "indexing_documents"}
	}
	this.Queue = queue.SmartGetOrInitConfig(this.Queue)

	// Set default OAuth configuration if not provided
	if this.OAuthConfig == nil {
		// OAuth configuration should be loaded from connector.tpl
		// These are fallback defaults if config is not loaded properly
		this.OAuthConfig = &OAuthConfig{
			AuthURL:     "https://accounts.feishu.cn/open-apis/authen/v1/authorize",
			TokenURL:    "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
			RedirectURI: "/connector/feishu/oauth_redirect", // Will be dynamically built from request
		}
	}

	// Register OAuth routes
	log.Debugf("[feishu connector] Attempting to register OAuth routes...")
	api.HandleUIMethod(api.GET, "/connector/feishu/connect", this.connect, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/connector/feishu/oauth_redirect", this.oAuthRedirect, api.RequireLogin())
	log.Infof("[feishu connector] OAuth routes registered successfully")
}

func (this *Plugin) Start() error {
	if this.Enabled {
		task.RegisterScheduleTask(task.ScheduleTask{
			ID:          util.GetUUID(),
			Group:       "connectors",
			Singleton:   true,
			Interval:    util.GetDurationOrDefault(this.Interval, time.Second*30).String(),
			Description: "indexing feishu cloud documents",
			Task: func(ctx context.Context) {
				connector := common.Connector{}
				connector.ID = ConnectorFeishu
				exists, err := orm.Get(&connector)
				if !exists {
					log.Debugf("Connector %s not found", connector.ID)
					return
				}
				if err != nil {
					panic(errors.Errorf("invalid %s connector:%v", connector.ID, err))
				}

				q := orm.Query{}
				q.Size = this.PageSize
				q.Conds = orm.And(orm.Eq("connector.id", connector.ID), orm.Eq("sync_enabled", true))
				var results []common.DataSource
				err, _ = orm.SearchWithJSONMapper(&results, &q)
				if err != nil {
					panic(err)
				}

				for _, item := range results {
					toSync, err := connectors.CanDoSync(item)
					if err != nil {
						_ = log.Errorf("error checking syncable with datasource [%s]: %v", item.Name, err)
						continue
					}
					if !toSync {
						continue
					}
					log.Debugf("fetch feishu cloud docs: ID: %s, Name: %s", item.ID, item.Name)
					this.fetchFeishuCloudDocs(&connector, &item)
				}
			},
		})
	}
	return nil
}

func (this *Plugin) Stop() error {
	return nil
}

func (this *Plugin) Name() string {
	return ConnectorFeishu
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}

func (this *Plugin) fetchFeishuCloudDocs(connector *common.Connector, datasource *common.DataSource) {
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
		// Check if token is expired
		if obj.TokenExpiry != "" {
			if expiry, err := time.Parse(time.RFC3339, obj.TokenExpiry); err == nil {
				if time.Now().After(expiry) && obj.RefreshToken != "" {
					// Token expired, try to refresh
					newToken, err := this.refreshAccessToken(obj.RefreshToken, obj)
					if err != nil {
						_ = log.Errorf("[feishu connector] failed to refresh token: %v", err)
						// Continue with expired token, API will return error
					} else {
						// Update datasource with new token
						obj.AccessToken = newToken.AccessToken
						obj.TokenExpiry = newToken.Expiry.Format(time.RFC3339)

						// Save updated token to datasource
						datasource.Connector.Config = obj
						ctx := orm.NewContext().DirectAccess()
						if err := orm.Update(ctx, datasource); err != nil {
							_ = log.Errorf("[feishu connector] failed to save refreshed token: %v", err)
						}
					}
				}
			}
		}
		token = obj.AccessToken
	} else if obj.UserAccessToken != "" {
		// Fallback to user_access_token
		token = strings.TrimSpace(obj.UserAccessToken)
	}

	if token == "" {
		_ = log.Errorf("[feishu connector] missing access token for datasource [%s]", datasource.Name)
		return
	}

	// Validate token format (basic check)
	if !strings.HasPrefix(token, "t-") && !strings.HasPrefix(token, "u-") {
		_ = log.Warnf("[feishu connector] access token format may be invalid for datasource [%s]", datasource.Name)
	}

	// 1) Search cloud documents
	// Set default document types if not specified
	docTypes := obj.DocumentTypes
	if len(docTypes) == 0 {
		docTypes = []string{"doc", "sheet", "slides", "mindnote", "bitable", "file", "docx", "folder", "shortcut"}
	}

	pageSize := this.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	// Start recursive search from root
	this.searchFilesRecursively(token, "", docTypes, pageSize, datasource)
}

// searchFilesRecursively recursively searches for files in folders
func (this *Plugin) searchFilesRecursively(token, folderToken string, docTypes []string, pageSize int, datasource *common.DataSource) {
	var nextPageToken string

	for {
		resBody, err := this.listFilesInFolder(token, folderToken, nextPageToken, pageSize)
		if err != nil {
			_ = log.Errorf("[feishu connector] list files in folder failed: %v", err)
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

			// If it's a folder, recursively search it
			if docType == "folder" {
				if subFolderToken := getString(m, "token"); subFolderToken != "" {
					this.searchFilesRecursively(token, subFolderToken, docTypes, pageSize, datasource)
				}
				continue
			}

			// Process document
			doc := common.Document{Source: common.DataSourceReference{ID: datasource.ID, Type: "connector", Name: datasource.Name}}
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

			if ct := getTime(getString(m, "created_time")); !ct.IsZero() {
				doc.Created = &ct
			}
			if ut := getTime(getString(m, "modified_time")); !ut.IsZero() {
				doc.Updated = &ut
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
func (this *Plugin) exchangeCodeForToken(code string, config Config) (*FeishuToken, error) {
	payload := map[string]interface{}{
		"client_id":     config.ClientID,
		"client_secret": config.ClientSecret,
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  this.OAuthConfig.RedirectURI,
	}

	req := util.NewPostRequest(this.OAuthConfig.TokenURL, util.MustToJSONBytes(payload))
	req.AddHeader("Content-Type", "application/json")

	res, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.Errorf("Feishu API error, no response")
	}

	if res.StatusCode >= 300 {
		return nil, errors.Errorf("Feishu API error: status %d, body: %s", res.StatusCode, string(res.Body))
	}

	var tokenResponse FeishuToken
	if err := json.Unmarshal(res.Body, &tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}

// refreshAccessToken refreshes the access token using refresh token
func (this *Plugin) refreshAccessToken(refreshToken string, config Config) (*FeishuToken, error) {
	payload := map[string]interface{}{
		"client_id":     config.ClientID,
		"client_secret": config.ClientSecret,
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"redirect_uri":  this.OAuthConfig.RedirectURI,
	}

	req := util.NewPostRequest(this.OAuthConfig.TokenURL, util.MustToJSONBytes(payload))
	req.AddHeader("Content-Type", "application/json")
	res, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.Errorf("Feishu API error, no response")
	}
	if res.StatusCode >= 300 {
		return nil, errors.Errorf("Feishu API error: status %d, body: %s", res.StatusCode, string(res.Body))
	}

	var tokenResponse FeishuToken
	if err := json.Unmarshal(res.Body, &tokenResponse); err != nil {
		return nil, err
	}
	return &tokenResponse, nil
}

// getUserProfile retrieves user profile information
func (this *Plugin) getUserProfile(accessToken string) (util.MapStr, error) {
	req := util.NewGetRequest("https://open.feishu.cn/open-apis/authen/v1/user_info", nil)
	req.AddHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	res, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.Errorf("Feishu API error, no response")
	}
	if res.StatusCode >= 300 {
		return nil, errors.Errorf("Feishu API error: status %d, body: %s", res.StatusCode, string(res.Body))
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
		return nil, errors.Errorf("Feishu API error: %s", response.Msg)
	}
	return response.Data, nil
}

// FeishuToken represents the OAuth token response
type FeishuToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
}

// listFilesInFolder lists files in a specific folder
func (this *Plugin) listFilesInFolder(tenantToken, folderToken, pageToken string, pageSize int) ([]byte, error) {
	if pageSize <= 0 {
		pageSize = 100
	}

	apiURL := "https://open.feishu.cn/open-apis/drive/v1/files"
	apiURL += "?page_size=" + fmt.Sprintf("%d", pageSize)
	if folderToken != "" {
		apiURL += "&folder_token=" + url.QueryEscape(folderToken)
	}
	if pageToken != "" {
		apiURL += "&page_token=" + url.QueryEscape(pageToken)
	}

	req := util.NewGetRequest(apiURL, nil)
	req.AddHeader("Authorization", fmt.Sprintf("Bearer %s", tenantToken))
	req.AddHeader("Content-Type", "application/json")

	res, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.Errorf("Feishu API error, no response")
	}
	if res.StatusCode >= 300 {
		return nil, errors.Errorf("Feishu API error: status %d, body: %s", res.StatusCode, string(res.Body))
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
