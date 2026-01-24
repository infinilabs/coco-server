/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package dropbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
	"golang.org/x/oauth2"
)

// resolveRedirectURL builds full redirect URL from current request
func resolveRedirectURL(oauthConfig *oauth2.Config, req *http.Request) string {
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

func getOAuthConfig(connectorID string) (*oauth2.Config, error) {
	if connectorID == "" {
		return nil, fmt.Errorf("connector id is empty")
	}

	// Default OAuth config
	oAuthConfig := &oauth2.Config{
		RedirectURL: fmt.Sprintf("/connector/%s/dropbox/oauth_redirect", connectorID),
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://www.dropbox.com/oauth2/authorize",
			TokenURL: "https://api.dropboxapi.com/oauth2/token",
		},
		Scopes: []string{
			"files.content.read",
			"files.metadata.read",
			"sharing.read",
			"account_info.read",
			"team_data.member",
		},
	}

	// Try to load connector to get OAuth credentials
	connector := core.Connector{}
	connector.ID = connectorID
	exists, err := orm.Get(&connector)
	if err == nil && exists && connector.Config != nil {
		if clientID, ok := connector.Config["client_id"].(string); ok {
			oAuthConfig.ClientID = clientID
		}
		if clientSecret, ok := connector.Config["client_secret"].(string); ok {
			oAuthConfig.ClientSecret = clientSecret
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to load connector: %v", err)
	}

	return oAuthConfig, nil
}

func connect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	connectorID := ps.MustGetParameter("id")
	oAuthConfig, err := getOAuthConfig(connectorID)
	if err != nil {
		http.Error(w, fmt.Sprintf("OAuth config error: %v", err), http.StatusBadRequest)
		return
	}

	// Check if OAuth is properly configured in connector
	if oAuthConfig.ClientID == "" || oAuthConfig.ClientSecret == "" {
		http.Error(w, "OAuth not configured in connector. "+
			"Please configure client_id and client_secret in the connector settings.", http.StatusServiceUnavailable)
		return
	}

	// Resolve full redirect URL from current request
	resolveRedirectURL(oAuthConfig, req)

	// Generate OAuth URL
	// Dropbox requires 'token_access_type=offline' to get a refresh token
	authURL := oAuthConfig.AuthCodeURL("", oauth2.SetAuthURLParam("token_access_type", "offline"))

	log.Infof("[dropbox connector] Redirecting to Dropbox OAuth: %s", authURL)
	http.Redirect(w, req, authURL, http.StatusFound)
}

// Endpoint to handle the OAuth redirect
func oAuthRedirect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	connectorID := ps.MustGetParameter("id")
	oAuthConfig, err := getOAuthConfig(connectorID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get OAuth config: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if OAuth is properly configured in connector
	if oAuthConfig == nil || oAuthConfig.ClientID == "" || oAuthConfig.ClientSecret == "" {
		http.Error(w, "OAuth not configured in connector. Please configure client_id and "+
			"client_secret in the connector settings.", http.StatusServiceUnavailable)
		return
	}

	// Resolve full redirect URL
	resolveRedirectURL(oAuthConfig, req)

	// Extract the code from the query parameters
	code := req.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code.", http.StatusBadRequest)
		return
	}

	log.Debugf("[dropbox connector] Received authorization code")

	// Exchange the authorization code for an access token
	// Use a detached context with timeout to ensure token exchange completes even if the request context is canceled
	oauthCtx, cancel := context.WithTimeout(req.Context(), 60*time.Second)
	defer cancel()

	// Inject a custom HTTP client into the context to ensure proxy settings are respected
	// oauth2.Exchange uses the client from the context if available
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	ctxWithClient := context.WithValue(oauthCtx, oauth2.HTTPClient, client)

	token, err := oAuthConfig.Exchange(ctxWithClient, code)
	if err != nil {
		_ = log.Errorf("[dropbox connector] Failed to exchange code for token: %v", err)
		http.Error(w, "Failed to exchange authorization code for token.", http.StatusInternalServerError)
		return
	}

	// Retrieve user info from Dropbox
	// Use a client that inherits the context and transport settings
	client = oAuthConfig.Client(ctxWithClient, token)

	// Dropbox API to get current account info
	resp, err := client.Post(fmt.Sprintf("%s/users/get_current_account", BaseURL), "application/json", strings.NewReader("null"))
	if err != nil {
		_ = log.Errorf("[dropbox connector] Failed to fetch user info: %v", err)
		http.Error(w, "Failed to fetch user info.", http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = log.Errorf("[dropbox connector] Unexpected status code fetching user info: %d, body: %s", resp.StatusCode, string(bodyBytes))
		http.Error(w, "Failed to fetch user info.", http.StatusInternalServerError)
		return
	}

	// Parse the user info
	var userInfo struct {
		AccountId string `json:"account_id"`
		Name      struct {
			DisplayName string `json:"display_name"`
		} `json:"name"`
		Email           string `json:"email"`
		ProfilePhotoUrl string `json:"profile_photo_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		_ = log.Errorf("[dropbox connector] Failed to parse user info: %v", err)
		http.Error(w, "Failed to parse user info.", http.StatusInternalServerError)
		return
	}

	log.Debugf("dropbox authenticated user: ID=%s, Email=%s", userInfo.AccountId, userInfo.Email)

	datasource := core.DataSource{
		SyncConfig: core.SyncConfig{Enabled: true, Interval: "30s"},
		Enabled:    true,
	}
	datasource.ID = util.MD5digest(fmt.Sprintf("%v,%v,%v,%v", "dropbox", connectorID, userInfo.AccountId, userInfo.Email))
	datasource.Type = "connector"
	datasource.Icon = "default"
	if userInfo.Name.DisplayName != "" {
		datasource.Name = userInfo.Name.DisplayName + "'s Dropbox"
	} else {
		datasource.Name = "My Dropbox"
	}

	// Store profile info
	profileMap := util.MapStr{
		"account_id": userInfo.AccountId,
		"email":      userInfo.Email,
		"name":       userInfo.Name.DisplayName,
		"picture":    userInfo.ProfilePhotoUrl,
	}

	datasource.Connector = core.ConnectorConfig{
		ConnectorID: connectorID,
		Config: util.MapStr{
			"client_id":     oAuthConfig.ClientID,
			"client_secret": oAuthConfig.ClientSecret,
			"refresh_token": token.RefreshToken,
			"profile":       profileMap,
		},
	}

	if token.RefreshToken == "" {
		log.Warnf("refresh token was not granted for: %v", datasource.Name)
	}

	ctx := orm.NewContextWithParent(req.Context())
	common.MarkDatasourceNotDeleted(datasource.ID)
	err = orm.Save(ctx, &datasource)
	if err != nil {
		_ = log.Errorf("[dropbox connector] Failed to save datasource: %v", err)
		http.Error(w, "Failed to save datasource.", http.StatusInternalServerError)
		return
	}

	log.Infof("[dropbox connector] Successfully created datasource: %s", datasource.ID)

	newRedirectUrl := fmt.Sprintf("/#/data-source/detail/%v", datasource.ID)

	http.Redirect(w, req, newRedirectUrl, http.StatusSeeOther)
}
