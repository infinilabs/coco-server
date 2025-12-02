package feishu

import (
	"fmt"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/connector"
	"infini.sh/coco/plugins/connectors"
	"net/http"
	"net/url"
	"strings"
	"time"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

// Note: saveLastModifiedTime and getLastModifiedTime methods are inherited from ConnectorProcessorBase
// No need to override them here unless custom behavior is needed

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

	redirectURL := resolveRedirectURL(oauthConfig, req)

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

	// Create OAuth handler to process the callback
	handler := NewOAuthHandler(pluginType, oauthConfig)

	// Exchange authorization code for access token
	token, err := handler.exchangeCodeForToken(code)
	if err != nil {
		_ = log.Errorf("[%s connector] Failed to exchange code for token: %v", pluginType, err)
		http.Error(w, "Failed to exchange authorization code for token.", http.StatusInternalServerError)
		return
	}

	// Get user profile information
	profile, err := handler.getUserProfile(token.AccessToken)
	if err != nil {
		_ = log.Errorf("[%s connector] Failed to get user profile: %v", pluginType, err)
		http.Error(w, "Failed to get user profile information.", http.StatusInternalServerError)
		return
	}

	log.Infof("[%s connector] Successfully authenticated user: %v", pluginType, profile)

	// Create datasource with OAuth tokens
	datasource := core.DataSource{
		SyncConfig: core.SyncConfig{Enabled: true, Interval: "30s"},
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
	datasource.Connector = core.ConnectorConfig{
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

// handleOAuthConnect is a generic handler factory for OAuth authorization
func handleOAuthConnect(pluginType PluginType) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		oauthConfig, err := getOAuthConfigFromConnector(ps.ByName("id"), pluginType)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get OAuth config: %v", err), http.StatusInternalServerError)
			return
		}
		connect(w, req, ps, pluginType, oauthConfig)
	}
}

// handleOAuthRedirect is a generic handler factory for OAuth callback
func handleOAuthRedirect(pluginType PluginType) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		oauthConfig, err := getOAuthConfigFromConnector(ps.ByName("id"), pluginType)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get OAuth config: %v", err), http.StatusInternalServerError)
			return
		}
		oauthConfig.RedirectURL = resolveRedirectURL(oauthConfig, req)
		oAuthRedirect(w, req, ps, pluginType, oauthConfig)
	}
}

// getOAuthConfigFromConnector retrieves OAuth configuration from connector
func getOAuthConfigFromConnector(connectorID string, pluginType PluginType) (*OAuthConfig, error) {
	apiConfig := getAPIConfig(pluginType)

	oauthConfig := &OAuthConfig{
		AuthURL:     apiConfig.AuthURL,
		TokenURL:    apiConfig.TokenURL,
		RedirectURL: fmt.Sprintf("/connector/%s/%s/oauth_redirect", connectorID, pluginType),
	}

	if connectorID == "" {
		return nil, fmt.Errorf("connector id is empty")
	}
	cfg, err := connector.GetConnectorByID(connectorID)
	if err != nil || cfg == nil {
		return nil, fmt.Errorf("invalid connector config")
	}
	err = connectors.ParseConnectorBaseConfigure(cfg, &oauthConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid oauth config parse from connector")
	}
	if oauthConfig.ClientID == "" || oauthConfig.ClientSecret == "" || len(oauthConfig.RedirectURL) == 0 {
		return nil, fmt.Errorf("missing %s OAuth credentials", pluginType)
	}
	log.Infof("oauthConfig: %v", oauthConfig)

	return oauthConfig, nil
}
