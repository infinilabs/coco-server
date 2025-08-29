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
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/orm"

	log "github.com/cihub/seelog"
)

// connect handles the OAuth authorization request
func (h *Plugin) connect(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	// Get datasource ID from query parameters
	datasourceID := h.MustGetParameter(w, req, "datasource_id")
	if datasourceID == "" {
		http.Error(w, "Missing datasource ID.", http.StatusBadRequest)
		return
	}

	// Get datasource configuration (existing flow)
	datasource := common.DataSource{}
	datasource.ID = datasourceID
	exists, err := orm.Get(&datasource)
	if !exists || err != nil {
		http.Error(w, "Datasource not found. Please create a datasource first with client_id and "+
			"client_secret.", http.StatusNotFound)
		return
	}

	// Parse datasource config
	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		http.Error(w, "Invalid datasource configuration.", http.StatusInternalServerError)
		return
	}

	obj := Config{}
	err = cfg.Unpack(&obj)
	if err != nil {
		http.Error(w, "Failed to parse datasource configuration.", http.StatusInternalServerError)
		return
	}

	// Check if OAuth is properly configured in datasource
	if obj.ClientID == "" || obj.ClientSecret == "" {
		http.Error(w, "OAuth not configured in datasource. Please configure client_id and "+
			"client_secret in the datasource settings.", http.StatusServiceUnavailable)
		return
	}

	// Generate OAuth authorization URL for Feishu
	// Feishu OAuth uses client_id instead of app_id
	// We'll use state parameter to pass datasource ID

	// Build full redirect_uri from current request
	redirectURI := h.OAuthConfig.RedirectURI
	if !strings.HasPrefix(redirectURI, "http://") && !strings.HasPrefix(redirectURI, "https://") {
		// Extract scheme and host from current request
		scheme := "http"
		if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}

		host := req.Host
		if host == "" {
			host = "localhost:8080" // fallback
		}

		redirectURI = fmt.Sprintf("%s://%s%s", scheme, host, redirectURI)
	}

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		h.OAuthConfig.AuthURL,
		obj.ClientID,
		url.QueryEscape(redirectURI),
		"drive:drive space:document:retrieve offline_access",
		url.QueryEscape(datasourceID),
	)

	log.Debugf("[feishu connector] Redirecting to OAuth URL: %s", authURL)

	// Redirect user to Feishu OAuth page
	http.Redirect(w, req, authURL, http.StatusTemporaryRedirect)
}

// oAuthRedirect handles the OAuth callback
func (h *Plugin) oAuthRedirect(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	// Get datasource ID from state parameter
	state := h.MustGetParameter(w, req, "state")
	if state == "" {
		http.Error(w, "Missing state parameter.", http.StatusBadRequest)
		return
	}

	// Decode state parameter
	stateData, err := url.QueryUnescape(state)
	if err != nil {
		http.Error(w, "Invalid state parameter.", http.StatusBadRequest)
		return
	}

	// Handle existing datasource flow (original logic)
	datasourceID := stateData

	// Get datasource configuration
	datasource := common.DataSource{}
	datasource.ID = datasourceID
	exists, err := orm.Get(&datasource)
	if !exists || err != nil {
		http.Error(w, "Datasource not found.", http.StatusNotFound)
		return
	}

	// Parse datasource config
	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		http.Error(w, "Invalid datasource configuration.", http.StatusInternalServerError)
		return
	}

	obj := Config{}
	err = cfg.Unpack(&obj)
	if err != nil {
		http.Error(w, "Failed to parse datasource configuration.", http.StatusInternalServerError)
		return
	}

	// Check if OAuth is properly configured in datasource
	if obj.ClientID == "" || obj.ClientSecret == "" {
		http.Error(w, "OAuth not configured in datasource. Please configure client_id and "+
			"client_secret in the datasource settings.", http.StatusServiceUnavailable)
		return
	}

	// Extract authorization code from query parameters
	code := h.MustGetParameter(w, req, "code")
	if code == "" {
		http.Error(w, "Missing authorization code.", http.StatusBadRequest)
		return
	}

	log.Debugf("[feishu connector] Received authorization code for datasource: %s", datasourceID)

	// Exchange authorization code for access token
	token, err := h.exchangeCodeForToken(code, obj)
	if err != nil {
		log.Errorf("[feishu connector] Failed to exchange code for token: %v", err)
		http.Error(w, "Failed to exchange authorization code for token.", http.StatusInternalServerError)
		return
	}

	// Get user profile information
	profile, err := h.getUserProfile(token.AccessToken)
	if err != nil {
		log.Errorf("[feishu connector] Failed to get user profile: %v", err)
		http.Error(w, "Failed to get user profile information.", http.StatusInternalServerError)
		return
	}

	log.Infof("[feishu connector] Successfully authenticated user: %v", profile)

	// Update existing datasource with OAuth tokens
	obj.AccessToken = token.AccessToken
	obj.RefreshToken = token.RefreshToken
	obj.TokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).Format(time.RFC3339)

	// Save refresh token expiry if provided
	if token.RefreshTokenExpiresIn > 0 {
		obj.RefreshTokenExpiry = time.Now().
			Add(time.Duration(token.RefreshTokenExpiresIn) * time.Second).Format(time.RFC3339)
	}

	obj.Profile = profile

	// Update datasource config
	datasource.Connector.Config = obj

	// Save updated datasource
	ctx := orm.NewContextWithParent(req.Context())
	err = orm.Update(ctx, &datasource)
	if err != nil {
		log.Errorf("[feishu connector] Failed to update datasource: %v", err)
		http.Error(w, "Failed to save OAuth tokens.", http.StatusInternalServerError)
		return
	}

	log.Infof("[feishu connector] Successfully updated datasource with OAuth tokens: %s", datasourceID)

	// Redirect to datasource detail page
	newRedirectURL := fmt.Sprintf("/#/data-source/detail/%v", datasource.ID)
	http.Redirect(w, req, newRedirectURL, http.StatusTemporaryRedirect)
}
