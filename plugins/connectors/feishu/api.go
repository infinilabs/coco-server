/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package feishu

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

// connect handles the OAuth authorization request
func (h *Plugin) connect(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	// Check if OAuth is properly configured in connector
	if h.OAuthConfig == nil || h.OAuthConfig.ClientID == "" || h.OAuthConfig.ClientSecret == "" {
		http.Error(w, "OAuth not configured in connector. Please configure client_id and "+
			"client_secret in the connector settings.", http.StatusServiceUnavailable)
		return
	}

	// Generate OAuth authorization URL for Feishu
	// Feishu OAuth uses client_id instead of app_id

	// Build full redirect_url from current request
	redirectURL := h.OAuthConfig.RedirectURL
	if !strings.HasPrefix(redirectURL, "http://") && !strings.HasPrefix(redirectURL, "https://") {
		// Extract scheme and host from current request
		scheme := "http"
		if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}

		host := req.Host
		if host == "" {
			host = "localhost:8080" // fallback
		}

		redirectURL = fmt.Sprintf("%s://%s%s", scheme, host, redirectURL)
		h.OAuthConfig.RedirectURL = redirectURL
	}

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&scope=%s",
		h.OAuthConfig.AuthURL,
		h.OAuthConfig.ClientID,
		url.QueryEscape(redirectURL),
		"drive:drive space:document:retrieve offline_access",
	)

	log.Debugf("[%s connector] Redirecting to OAuth URL: %s", h.PluginType, authURL)

	// Redirect user to Feishu OAuth page
	http.Redirect(w, req, authURL, http.StatusTemporaryRedirect)
}

// oAuthRedirect handles the OAuth callback
func (h *Plugin) oAuthRedirect(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	// Check if OAuth is properly configured in connector
	if h.OAuthConfig == nil || h.OAuthConfig.ClientID == "" || h.OAuthConfig.ClientSecret == "" {
		http.Error(w, "OAuth not configured in connector. Please configure client_id and "+
			"client_secret in the connector settings.", http.StatusServiceUnavailable)
		return
	}

	// Extract authorization code from query parameters
	code := h.MustGetParameter(w, req, "code")
	if code == "" {
		http.Error(w, "Missing authorization code.", http.StatusBadRequest)
		return
	}

	log.Debugf("[%s connector] Received authorization code", h.PluginType)

	// Exchange authorization code for access token
	token, err := h.exchangeCodeForToken(code)
	if err != nil {
		log.Errorf("[%s connector] Failed to exchange code for token: %v", h.PluginType, err)
		http.Error(w, "Failed to exchange authorization code for token.", http.StatusInternalServerError)
		return
	}

	// Get user profile information
	profile, err := h.getUserProfile(token.AccessToken)
	if err != nil {
		log.Errorf("[%s connector] Failed to get user profile: %v", h.PluginType, err)
		http.Error(w, "Failed to get user profile information.", http.StatusInternalServerError)
		return
	}

	log.Infof("[%s connector] Successfully authenticated user: %v", h.PluginType, profile)

	// Create datasource with OAuth tokens
	datasource := common.DataSource{
		SyncConfig: common.SyncConfig{Enabled: true, Interval: "30s"},
		Enabled:    true,
	}

	// Generate unique datasource ID based on connector type and user info
	userID := ""
	if userIDStr, ok := profile["user_id"].(string); ok {
		userID = userIDStr
	} else if email, ok := profile["email"].(string); ok {
		userID = email
	} else {
		userID = "unknown"
	}

	datasource.ID = util.MD5digest(fmt.Sprintf("%v,%v", h.PluginType, userID))
	datasource.Type = "connector"

	// Set datasource name
	if name, ok := profile["name"].(string); ok && name != "" {
		datasource.Name = fmt.Sprintf("%s's %s", name, h.PluginType)
	} else {
		datasource.Name = fmt.Sprintf("My %s", h.PluginType)
	}

	// Create datasource config with OAuth tokens
	datasource.Connector = common.ConnectorConfig{
		ConnectorID: string(h.PluginType),
		Config: util.MapStr{
			"access_token":  token.AccessToken,
			"refresh_token": token.RefreshToken,
			"token_expiry":  time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).Format(time.RFC3339),
			"profile":       profile,
		},
	}

	// Add refresh token expiry if provided
	if token.RefreshTokenExpiresIn > 0 {
		if configMap, ok := datasource.Connector.Config.(util.MapStr); ok {
			configMap["refresh_token_expiry"] = time.Now().
				Add(time.Duration(token.RefreshTokenExpiresIn) * time.Second).Format(time.RFC3339)
		}
	}

	// Check if refresh token is missing or empty
	if token.RefreshToken == "" {
		log.Warnf("refresh token was not granted for: %v", datasource.Name)
	}

	// Save datasource
	ctx := orm.NewContextWithParent(req.Context())
	err = orm.Save(ctx, &datasource)
	if err != nil {
		log.Errorf("[%s connector] Failed to save datasource: %v", h.PluginType, err)
		http.Error(w, "Failed to save datasource.", http.StatusInternalServerError)
		return
	}

	log.Infof("[%s connector] Successfully created datasource: %s", h.PluginType, datasource.ID)

	// Redirect to datasource detail page
	newRedirectURL := fmt.Sprintf("/#/data-source/detail/%v", datasource.ID)
	http.Redirect(w, req, newRedirectURL, http.StatusTemporaryRedirect)
}
