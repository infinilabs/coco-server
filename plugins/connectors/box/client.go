/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package box

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
)

const (
	BaseURL           = "https://api.box.com"
	TokenEndpoint     = "/oauth2/token"
	PingEndpoint      = "/2.0/users/me"
	FolderEndpoint    = "/2.0/folders/%s/items"
	UsersEndpoint     = "/2.0/users"
	DefaultTimeout    = 30 * time.Second
	TokenExpiryBuffer = 5 * time.Minute
	DefaultPageSize   = 100
)

const (
	AccountTypeFree       = "box_free"
	AccountTypeEnterprise = "box_enterprise"
)

const (
	FileTypeFile   = "file"
	FileTypeFolder = "folder"
)

// BoxFile represents a file or folder item from Box API
type BoxFile struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Size         int64                  `json:"size"`
	Description  string                 `json:"description"`
	ItemStatus   string                 `json:"item_status"`
	SequenceID   string                 `json:"sequence_id"`
	ETag         string                 `json:"etag"`
	CreatedAt    time.Time              `json:"created_at"`
	ModifiedAt   time.Time              `json:"modified_at"`
	URL          string                 `json:"url"`
	DownloadURL  string                 `json:"download_url"`
	ThumbnailURL string                 `json:"thumbnail_url"`
	SharedLink   map[string]interface{} `json:"shared_link"`
	CreatedBy    map[string]interface{} `json:"created_by"`
	ModifiedBy   map[string]interface{} `json:"modified_by"`
	OwnedBy      map[string]interface{} `json:"owned_by"`
	Parent       map[string]interface{} `json:"parent"`
	Extension    string                 `json:"extension"`
}

// FolderItemsResponse represents the response from Box folder items API
type FolderItemsResponse struct {
	TotalCount int        `json:"total_count"`
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	Entries    []*BoxFile `json:"entries"`
}

// TokenCache manages access token caching with expiration
type TokenCache struct {
	mu           sync.RWMutex
	accessToken  string
	refreshToken string
	expiry       time.Time
}

func (tc *TokenCache) Get() (string, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	if tc.accessToken == "" || time.Now().After(tc.expiry.Add(-TokenExpiryBuffer)) {
		return "", false
	}
	return tc.accessToken, true
}

func (tc *TokenCache) Set(accessToken, refreshToken string, expiresIn int64) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.accessToken = accessToken
	if refreshToken != "" {
		tc.refreshToken = refreshToken
	}
	tc.expiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
}

func (tc *TokenCache) GetRefreshToken() string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.refreshToken
}

func (tc *TokenCache) SetRefreshToken(refreshToken string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.refreshToken = refreshToken
}

// BoxClient handles communication with the Box API
type BoxClient struct {
	config     *Config
	httpClient *http.Client
	tokenCache *TokenCache
	baseURL    string
}

// NewBoxClient creates a new Box client
func NewBoxClient(config *Config) *BoxClient {
	// Initialize token cache with refresh token for Free accounts
	tokenCache := &TokenCache{}
	if config.IsEnterprise == AccountTypeFree && config.RefreshToken != "" {
		tokenCache.SetRefreshToken(config.RefreshToken)
	}

	return &BoxClient{
		config: config,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		tokenCache: tokenCache,
		baseURL:    BaseURL,
	}
}

// NewBoxClientWithTokens creates a new Box client with pre-obtained OAuth tokens
// This is useful when creating a client right after OAuth authentication
func NewBoxClientWithTokens(config *Config, accessToken, refreshToken string, expiresIn int64) *BoxClient {
	tokenCache := &TokenCache{}

	// Set both access token and refresh token
	tokenCache.Set(accessToken, refreshToken, expiresIn)

	return &BoxClient{
		config: config,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		tokenCache: tokenCache,
		baseURL:    BaseURL,
	}
}

// TokenResponse represents the Box OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Authenticate authenticates with Box and retrieves an access token
func (c *BoxClient) Authenticate() error {
	log.Debugf("[box client] Generating an access token for account type: %s", c.config.IsEnterprise)

	var data url.Values

	if c.config.IsEnterprise == AccountTypeFree {
		// Box Free Account: use refresh_token grant
		refreshToken := c.tokenCache.GetRefreshToken()
		if refreshToken == "" {
			return fmt.Errorf("refresh_token is required for Box Free Account")
		}

		data = url.Values{}
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", refreshToken)
		data.Set("client_id", c.config.ClientID)
		data.Set("client_secret", c.config.ClientSecret)
	} else {
		// Box Enterprise Account: use client_credentials grant
		if c.config.EnterpriseID == "" {
			return fmt.Errorf("enterprise_id is required for Box Enterprise Account")
		}

		data = url.Values{}
		data.Set("grant_type", "client_credentials")
		data.Set("client_id", c.config.ClientID)
		data.Set("client_secret", c.config.ClientSecret)
		data.Set("box_subject_type", "enterprise")
		data.Set("box_subject_id", c.config.EnterpriseID)
	}

	tokenURL := c.baseURL + TokenEndpoint
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	// Cache the token
	c.tokenCache.Set(tokenResp.AccessToken, tokenResp.RefreshToken, tokenResp.ExpiresIn)

	// Log token info
	if c.config.IsEnterprise == AccountTypeFree {
		if tokenResp.RefreshToken != "" {
			log.Debugf("[box client] Successfully authenticated (Free Account), token expires in %d seconds, refresh_token updated", tokenResp.ExpiresIn)
		} else {
			log.Debugf("[box client] Successfully authenticated (Free Account), token expires in %d seconds", tokenResp.ExpiresIn)
		}
	} else {
		log.Debugf("[box client] Successfully authenticated (Enterprise Account), token expires in %d seconds", tokenResp.ExpiresIn)
	}

	return nil
}

// GetAccessToken returns a valid access token, refreshing if necessary
func (c *BoxClient) GetAccessToken() (string, error) {
	// Try to get cached token
	if token, valid := c.tokenCache.Get(); valid {
		return token, nil
	}

	// Token expired or not found, authenticate
	log.Debug("[box client] No valid token cache found; fetching new token")
	if err := c.Authenticate(); err != nil {
		return "", err
	}

	// Get the newly cached token
	token, _ := c.tokenCache.Get()
	return token, nil
}

// Ping tests the connection to Box
func (c *BoxClient) Ping() error {
	resp, err := c.Get(PingEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to execute ping request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ping failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Debug("[box client] Successfully pinged Box API")
	return nil
}

// Get makes an authenticated GET request to the Box API
func (c *BoxClient) Get(endpoint string, params url.Values) (*http.Response, error) {
	return c.GetWithHeaders(endpoint, params, nil)
}

// GetWithHeaders makes an authenticated GET request to the Box API with custom headers
// Additional headers can be provided (e.g., "as-user" for enterprise accounts)
func (c *BoxClient) GetWithHeaders(endpoint string, params url.Values, headers map[string]string) (*http.Response, error) {
	accessToken, err := c.GetAccessToken()
	if err != nil {
		return nil, err
	}

	requestURL := c.baseURL + endpoint
	if params != nil && len(params) > 0 {
		requestURL = requestURL + "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	// Set additional headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	// Handle 401 Unauthorized - token might be expired
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		// Force re-authentication
		c.tokenCache.accessToken = ""
		log.Warn("[box client] Received 401, re-authenticating...")

		if err := c.Authenticate(); err != nil {
			return nil, fmt.Errorf("re-authentication failed: %w", err)
		}

		// Retry the request with new token
		accessToken, _ = c.tokenCache.Get()
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to retry request: %w", err)
		}
	}

	return resp, nil
}

// GetFolderItems retrieves items in a folder with pagination
// For enterprise accounts, userID should be provided to fetch items as that user
func (c *BoxClient) GetFolderItems(folderID string, offset, limit int, userID string) (*FolderItemsResponse, error) {
	endpoint := fmt.Sprintf(FolderEndpoint, folderID)

	params := url.Values{}
	params.Set("offset", fmt.Sprintf("%d", offset))
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("fields", "id,type,name,size,description,item_status,sequence_id,etag,created_at,modified_at,url,created_by,modified_by,owned_by,parent,extension")

	// Prepare headers for enterprise accounts
	var headers map[string]string
	if userID != "" {
		headers = map[string]string{"as-user": userID}
	}

	resp, err := c.GetWithHeaders(endpoint, params, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder items: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get folder items failed with status %d: %s", resp.StatusCode, string(body))
	}

	var itemsResp FolderItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&itemsResp); err != nil {
		return nil, fmt.Errorf("failed to decode folder items response: %w", err)
	}

	return &itemsResp, nil
}

// BoxUser represents a user from Box API
type BoxUser struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Name  string `json:"name"`
	Login string `json:"login"`
}

// UsersResponse represents the response from Box users API
type UsersResponse struct {
	TotalCount int       `json:"total_count"`
	Offset     int       `json:"offset"`
	Limit      int       `json:"limit"`
	Entries    []BoxUser `json:"entries"`
}

// GetUsers retrieves all users in the enterprise (Enterprise account only)
func (c *BoxClient) GetUsers() ([]BoxUser, error) {
	if c.config.IsEnterprise != AccountTypeEnterprise {
		return nil, fmt.Errorf("GetUsers is only available for Enterprise accounts")
	}

	var allUsers []BoxUser
	offset := 0
	limit := DefaultPageSize

	for {
		params := url.Values{}
		params.Set("offset", fmt.Sprintf("%d", offset))
		params.Set("limit", fmt.Sprintf("%d", limit))

		resp, err := c.Get(UsersEndpoint, params)
		if err != nil {
			return nil, fmt.Errorf("failed to get users: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("get users failed with status %d: %s", resp.StatusCode, string(body))
		}

		var usersResp UsersResponse
		if err := json.NewDecoder(resp.Body).Decode(&usersResp); err != nil {
			return nil, fmt.Errorf("failed to decode users response: %w", err)
		}

		allUsers = append(allUsers, usersResp.Entries...)

		// Check if we have more users
		if offset+limit >= usersResp.TotalCount {
			break
		}
		offset += limit
	}

	log.Debugf("[box client] Retrieved %d users from enterprise", len(allUsers))
	return allUsers, nil
}
