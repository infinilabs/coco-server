package feishu

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

// saveLastModifiedTime saves the last modified time for incremental sync per datasource
func (this *Plugin) saveLastModifiedTime(datasourceID string, lastModifiedTime string) error {
	bucket := fmt.Sprintf("/connector/%s/lastModifiedTime", this.PluginType)
	return kv.AddValue(bucket, []byte(datasourceID), []byte(lastModifiedTime))
}

// getLastModifiedTime retrieves the last modified time for incremental sync per datasource
func (this *Plugin) getLastModifiedTime(datasourceID string) (string, error) {
	bucket := fmt.Sprintf("/connector/%s/lastModifiedTime", this.PluginType)
	data, err := kv.GetValue(bucket, []byte(datasourceID))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// connect handles the OAuth authorization request
func connect(w http.ResponseWriter, req *http.Request, _ httprouter.Params, pluginType PluginType, oauthConfig *OAuthConfig) {
	// Check if OAuth is properly configured in connector
	if oauthConfig == nil || oauthConfig.ClientID == "" || oauthConfig.ClientSecret == "" {
		http.Error(w, "OAuth not configured in connector. Please configure client_id and "+
			"client_secret in the connector settings.", http.StatusServiceUnavailable)
		return
	}

	// Generate OAuth authorization URL for Feishu/Lark
	// Feishu OAuth uses client_id instead of app_id

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
			host = "localhost:8080" // fallback
		}

		redirectURL = fmt.Sprintf("%s://%s%s", scheme, host, redirectURL)
		oauthConfig.RedirectURL = redirectURL
	}

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&scope=%s",
		oauthConfig.AuthURL,
		oauthConfig.ClientID,
		url.QueryEscape(redirectURL),
		"drive:drive space:document:retrieve offline_access",
	)

	log.Debugf("[%s connector] Redirecting to OAuth URL: %s", pluginType, authURL)

	// Redirect user to Feishu/Lark OAuth page
	http.Redirect(w, req, authURL, http.StatusTemporaryRedirect)
}

// oAuthRedirect handles the OAuth callback
func oAuthRedirect(w http.ResponseWriter, req *http.Request, ps httprouter.Params, pluginType PluginType, oauthConfig *OAuthConfig) {
	// Check if OAuth is properly configured in connector
	if oauthConfig == nil || oauthConfig.ClientID == "" || oauthConfig.ClientSecret == "" {
		http.Error(w, "OAuth not configured in connector. Please configure client_id and "+
			"client_secret in the connector settings.", http.StatusServiceUnavailable)
		return
	}

	// Extract authorization code from query parameters
	code := ps.ByName("code")
	if code == "" {
		// Try query parameter
		code = req.URL.Query().Get("code")
	}
	if code == "" {
		http.Error(w, "Missing authorization code.", http.StatusBadRequest)
		return
	}

	log.Debugf("[%s connector] Received authorization code", pluginType)

	// Create a temporary plugin instance to use helper methods
	p := &Plugin{}
	p.SetPluginType(pluginType)
	p.OAuthConfig = oauthConfig

	// Exchange authorization code for access token
	token, err := p.exchangeCodeForToken(code)
	if err != nil {
		log.Errorf("[%s connector] Failed to exchange code for token: %v", pluginType, err)
		http.Error(w, "Failed to exchange authorization code for token.", http.StatusInternalServerError)
		return
	}

	// Get user profile information
	profile, err := p.getUserProfile(token.AccessToken)
	if err != nil {
		log.Errorf("[%s connector] Failed to get user profile: %v", pluginType, err)
		http.Error(w, "Failed to get user profile information.", http.StatusInternalServerError)
		return
	}

	log.Infof("[%s connector] Successfully authenticated user: %v", pluginType, profile)

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

	datasource.ID = util.MD5digest(fmt.Sprintf("%v,%v", pluginType, userID))
	datasource.Type = "connector"

	// Set datasource name
	if name, ok := profile["name"].(string); ok && name != "" {
		datasource.Name = fmt.Sprintf("%s's %s", name, pluginType)
	} else {
		datasource.Name = fmt.Sprintf("My %s", pluginType)
	}

	// Create datasource config with OAuth tokens
	datasource.Connector = common.ConnectorConfig{
		ConnectorID: string(pluginType),
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
		log.Errorf("[%s connector] Failed to save datasource: %v", pluginType, err)
		http.Error(w, "Failed to save datasource.", http.StatusInternalServerError)
		return
	}

	log.Infof("[%s connector] Successfully created datasource: %s", pluginType, datasource.ID)

	// Redirect to datasource detail page
	newRedirectURL := fmt.Sprintf("/#/data-source/detail/%v", datasource.ID)
	http.Redirect(w, req, newRedirectURL, http.StatusTemporaryRedirect)
}

// feishuConnect handles OAuth authorization for Feishu
func feishuConnect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	// Get OAuth config from connector
	oauthConfig, err := getOAuthConfigFromConnector(ps.ByName("id"), PluginTypeFeishu)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get OAuth config: %v", err), http.StatusInternalServerError)
		return
	}
	connect(w, req, ps, PluginTypeFeishu, oauthConfig)
}

// feishuOAuthRedirect handles OAuth callback for Feishu
func feishuOAuthRedirect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	oauthConfig, err := getOAuthConfigFromConnector(ps.ByName("id"), PluginTypeFeishu)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get OAuth config: %v", err), http.StatusInternalServerError)
		return
	}
	oAuthRedirect(w, req, ps, PluginTypeFeishu, oauthConfig)
}

// larkConnect handles OAuth authorization for Lark
func larkConnect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	oauthConfig, err := getOAuthConfigFromConnector(ps.ByName("id"), PluginTypeLark)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get OAuth config: %v", err), http.StatusInternalServerError)
		return
	}
	connect(w, req, ps, PluginTypeLark, oauthConfig)
}

// larkOAuthRedirect handles OAuth callback for Lark
func larkOAuthRedirect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	oauthConfig, err := getOAuthConfigFromConnector(ps.ByName("id"), PluginTypeLark)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get OAuth config: %v", err), http.StatusInternalServerError)
		return
	}
	oAuthRedirect(w, req, ps, PluginTypeLark, oauthConfig)
}

// getOAuthConfigFromConnector retrieves OAuth configuration from connector
func getOAuthConfigFromConnector(connectorID string, pluginType PluginType) (*OAuthConfig, error) {
	apiConfig := getAPIConfig(pluginType)

	oauthConfig := &OAuthConfig{
		AuthURL:     apiConfig.AuthURL,
		TokenURL:    apiConfig.TokenURL,
		RedirectURL: fmt.Sprintf("/connector/%s/%s/oauth_redirect", connectorID, pluginType),
	}

	// Try to load connector to get OAuth credentials
	connector := common.Connector{}
	connector.ID = connectorID
	exists, err := orm.Get(&connector)
	if err == nil && exists && connector.Config != nil {
		if clientID, ok := connector.Config["client_id"].(string); ok {
			oauthConfig.ClientID = clientID
		}
		if clientSecret, ok := connector.Config["client_secret"].(string); ok {
			oauthConfig.ClientSecret = clientSecret
		}
		if authURL, ok := connector.Config["auth_url"].(string); ok {
			oauthConfig.AuthURL = authURL
		}
		if tokenURL, ok := connector.Config["token_url"].(string); ok {
			oauthConfig.TokenURL = tokenURL
		}
	}

	return oauthConfig, nil
}
