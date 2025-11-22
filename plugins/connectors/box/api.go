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

	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

// Global client cache to store authenticated Box clients
var (
	clientCache sync.Map // key: datasource ID, value: *BoxClient
)

// GetCachedClient retrieves a cached Box client for a datasource
func GetCachedClient(datasourceID string) (*BoxClient, bool) {
	if client, ok := clientCache.Load(datasourceID); ok {
		if boxClient, ok := client.(*BoxClient); ok {
			return boxClient, true
		}
	}
	return nil, false
}

// CacheClient stores a Box client for a datasource
func CacheClient(datasourceID string, client *BoxClient) {
	clientCache.Store(datasourceID, client)
	log.Debugf("[box connector] Cached client for datasource: %s", datasourceID)
}

// RemoveCachedClient removes a cached Box client for a datasource
func RemoveCachedClient(datasourceID string) {
	clientCache.Delete(datasourceID)
	log.Debugf("[box connector] Removed cached client for datasource: %s", datasourceID)
}

// OAuthConfig represents Box OAuth configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// getOAuthConfigFromConnector retrieves OAuth configuration from connector
func getOAuthConfigFromConnector(connectorID string) (*OAuthConfig, error) {
	if connectorID == "" {
		return nil, fmt.Errorf("connector id is empty")
	}

	oauthConfig := &OAuthConfig{
		RedirectURL: fmt.Sprintf("/connector/%s/box/oauth_redirect", connectorID),
	}

	// Try to load connector to get OAuth credentials
	connector := core.Connector{}
	connector.ID = connectorID
	exists, err := orm.Get(&connector)
	if err == nil && exists && connector.Config != nil {
		if clientID, ok := connector.Config["client_id"].(string); ok {
			oauthConfig.ClientID = clientID
		}
		if clientSecret, ok := connector.Config["client_secret"].(string); ok {
			oauthConfig.ClientSecret = clientSecret
		}
	}

	return oauthConfig, nil
}

// resolveRedirectURL builds full redirect URL from current request
func resolveRedirectURL(oauthConfig *OAuthConfig, req *http.Request) string {
	// Build full redirect_url from current request
	redirectURL := oauthConfig.RedirectURL
	if !strings.HasPrefix(redirectURL, "http://") && !strings.HasPrefix(redirectURL, "https://") {
		// Extract scheme and host from current request
		scheme := "http"
		if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}

		host := req.Host
		if host == "" {
			host = "localhost:9000" // fallback
		}

		redirectURL = fmt.Sprintf("%s://%s%s", scheme, host, redirectURL)
		oauthConfig.RedirectURL = redirectURL
	}
	return redirectURL
}

// connect initiates the OAuth flow by redirecting to Box authorization page
func connect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	connectorID := ps.MustGetParameter("id")

	oauthConfig, err := getOAuthConfigFromConnector(connectorID)
	if err != nil {
		http.Error(w, fmt.Sprintf("OAuth config error: %v", err), http.StatusBadRequest)
		return
	}

	// Check if OAuth is properly configured in connector
	if oauthConfig.ClientID == "" || oauthConfig.ClientSecret == "" {
		http.Error(w, "OAuth not configured in connector. Please configure client_id and client_secret in the connector settings.", http.StatusServiceUnavailable)
		return
	}

	// Resolve full redirect URL from current request
	redirectURL := resolveRedirectURL(oauthConfig, req)

	// Build Box OAuth authorization URL
	authURL := "https://account.box.com/api/oauth2/authorize"
	params := url.Values{}
	params.Set("client_id", oauthConfig.ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", redirectURL)

	fullAuthURL := fmt.Sprintf("%s?%s", authURL, params.Encode())

	log.Infof("[box connector] Redirecting to Box OAuth: %s", fullAuthURL)
	http.Redirect(w, req, fullAuthURL, http.StatusFound)
}

// oAuthRedirect handles the OAuth callback from Box
func oAuthRedirect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	connectorID := ps.MustGetParameter("id")

	oauthConfig, err := getOAuthConfigFromConnector(connectorID)
	if err != nil {
		http.Error(w, fmt.Sprintf("OAuth config error: %v", err), http.StatusBadRequest)
		return
	}

	// Check if OAuth is properly configured in connector
	if oauthConfig.ClientID == "" || oauthConfig.ClientSecret == "" {
		http.Error(w, "OAuth not configured in connector. Please configure client_id and client_secret in the connector settings.", http.StatusServiceUnavailable)
		return
	}

	// Resolve full redirect URL
	redirectURL := resolveRedirectURL(oauthConfig, req)

	// Extract the authorization code from query parameters
	code := req.URL.Query().Get("code")
	if code == "" {
		errorMsg := req.URL.Query().Get("error")
		errorDesc := req.URL.Query().Get("error_description")
		log.Errorf("[box connector] OAuth error: %s - %s", errorMsg, errorDesc)
		http.Error(w, fmt.Sprintf("Authorization failed: %s", errorDesc), http.StatusBadRequest)
		return
	}

	log.Infof("[box connector] Received authorization code, exchanging for tokens...")
	log.Debugf("[box connector] Using redirect_uri: %s", redirectURL)

	// Exchange authorization code for tokens
	// Box API requires application/x-www-form-urlencoded format
	tokenURL := "https://api.box.com/oauth2/token"
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", oauthConfig.ClientID)
	data.Set("client_secret", oauthConfig.ClientSecret)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		log.Errorf("[box connector] Failed to exchange code for token: %v", err)
		http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Errorf("[box connector] Box token API error: status %d, body: %s", resp.StatusCode, string(body))
		http.Error(w, fmt.Sprintf("Box token API error: status %d, body: %s", resp.StatusCode, string(body)), http.StatusInternalServerError)
		return
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		log.Errorf("[box connector] Failed to decode token response: %v", err)
		http.Error(w, fmt.Sprintf("Failed to decode token response: %v", err), http.StatusInternalServerError)
		return
	}
	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" {
		log.Error("[box connector] Received empty tokens from Box")
		http.Error(w, "Received invalid tokens from Box", http.StatusInternalServerError)
		return
	}

	log.Infof("[box connector] Successfully obtained tokens, refresh_token: %s...", tokenResp.RefreshToken[:10])

	// Get user profile to set datasource name
	userProfile, err := getUserProfile(tokenResp.AccessToken)
	if err != nil {
		log.Warnf("[box connector] Failed to get user profile: %v, will use default name", err)
	}

	// Create datasource with obtained tokens
	datasourceName := "Box - Unknown User"
	if userProfile != nil && userProfile.Name != "" {
		datasourceName = fmt.Sprintf("Box - %s", userProfile.Name)
	}

	datasource := &core.DataSource{
		Name:    datasourceName,
		Type:    "connector",
		Enabled: true,
		SyncConfig: core.SyncConfig{
			Enabled:  true,
			Interval: "30s",
		},
		Connector: core.ConnectorConfig{
			ConnectorID: connectorID,
			Config: util.MapStr{
				"is_enterprise": AccountTypeFree,
				"client_id":     oauthConfig.ClientID,
				"client_secret": oauthConfig.ClientSecret,
				"refresh_token": tokenResp.RefreshToken,
				"_access_token": tokenResp.AccessToken, // Store for reference (optional)
				"_token_expiry": time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339),
				"_user_profile": userProfile,
			},
		},
	}
	datasource.ID = util.GetUUID()

	// Verify the connection by creating and testing the client
	log.Infof("[box connector] Verifying connection with obtained tokens...")
	clientConfig := &Config{
		IsEnterprise: AccountTypeFree,
		ClientID:     oauthConfig.ClientID,
		ClientSecret: oauthConfig.ClientSecret,
		RefreshToken: tokenResp.RefreshToken,
	}

	// Create client with the OAuth tokens we just obtained
	client := NewBoxClientWithTokens(clientConfig, tokenResp.AccessToken, tokenResp.RefreshToken, tokenResp.ExpiresIn)

	// Test the connection
	if err := client.Ping(); err != nil {
		log.Errorf("[box connector] Failed to verify connection: %v", err)
		http.Error(w, fmt.Sprintf("Failed to verify connection: %v", err), http.StatusInternalServerError)
		return
	}
	log.Infof("[box connector] Connection verified successfully")

	// Save datasource
	ctx := orm.NewContextWithParent(req.Context())
	err = orm.Save(ctx, &datasource)
	if err != nil {
		log.Errorf("[box connector] Failed to create datasource: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create datasource: %v", err), http.StatusInternalServerError)
		return
	}

	log.Infof("[box connector] Successfully created datasource: %s (ID: %s)", datasource.Name, datasource.ID)

	// Cache the authenticated client for future use
	CacheClient(datasource.ID, client)

	// Redirect to datasource detail page
	newRedirectURL := fmt.Sprintf("/#/data-source/detail/%v", datasource.ID)
	http.Redirect(w, req, newRedirectURL, http.StatusTemporaryRedirect)
}

// BoxUserProfile represents Box user profile information
type BoxUserProfile struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Name  string `json:"name"`
	Login string `json:"login"`
}

// getUserProfile retrieves the current user's profile from Box
func getUserProfile(accessToken string) (*BoxUserProfile, error) {
	req, err := http.NewRequest("GET", "https://api.box.com/2.0/users/me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get user profile failed with status %d: %s", resp.StatusCode, string(body))
	}

	var profile BoxUserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode profile response: %w", err)
	}

	return &profile, nil
}
