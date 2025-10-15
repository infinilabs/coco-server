/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package google_drive

import (
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/connector"
	"infini.sh/coco/plugins/connectors"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
	"net/url"
	"time"
)

func getOAuthConfig(connectorID string) *oauth2.Config {
	if connectorID == "" {
		panic("connector id is empty")
	}

	cfg, err := connector.GetConnectorByID(connectorID)
	if err != nil || cfg == nil {
		panic("invalid connector config")
	}

	oauthCfg := Credential{
		AuthUri:  "https://accounts.google.com/o/oauth2/auth",
		TokenUri: "https://oauth2.googleapis.com/token",
	}
	err = connectors.ParseConnectorBaseConfigure(cfg, &oauthCfg)
	if err != nil {
		panic("invalid oauth config parse from connector")
	}

	if oauthCfg.ClientId == "" || oauthCfg.ClientSecret == "" || len(oauthCfg.RedirectUri) == 0 {
		panic("Missing Google OAuth credentials")
	}

	oAuthConfig := &oauth2.Config{
		ClientID:     oauthCfg.ClientId,
		ClientSecret: oauthCfg.ClientSecret,
		RedirectURL:  oauthCfg.RedirectUri,
		Endpoint:     google.Endpoint,
	}

	oAuthConfig.Scopes = []string{
		"https://www.googleapis.com/auth/drive.readonly",
		"https://www.googleapis.com/auth/drive.metadata.readonly", // Access Drive metadata
		"https://www.googleapis.com/auth/userinfo.email",          // Access the user's profile information
		"https://www.googleapis.com/auth/userinfo.profile",        // Access the user's profile information
	}
	return oAuthConfig
}

func connect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	connectorID := ps.MustGetParameter("id")
	oAuthConfig := getOAuthConfig(connectorID)
	if oAuthConfig == nil {
		panic("invalid oauth config")
	}

	// Generate OAuth URL with proper offline access
	authURL := oAuthConfig.AuthCodeURL("", oauth2.AccessTypeOffline)

	// Parse the generated URL to append additional parameters
	parsedURL, err := url.Parse(authURL)
	if err != nil {
		panic("Failed to parse auth URL")
	}

	// Add parameters to ensure refresh token is always granted
	// Use prompt=consent instead of deprecated approval_prompt=force
	query := parsedURL.Query()
	query.Set("prompt", "consent") // Force consent to ensure refresh token
	parsedURL.RawQuery = query.Encode()

	// Return the updated URL with the necessary parameters
	authURL = parsedURL.String()

	http.Redirect(w, req, authURL, http.StatusFound)
}

// Endpoint to handle the OAuth redirect
func oAuthRedirect(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	connectorID := ps.MustGetParameter("id")
	oAuthConfig := getOAuthConfig(connectorID)
	if oAuthConfig == nil {
		panic("invalid oauth config")
	}

	// Extract the code from the query parameters
	code := req.URL.Query().Get("code")
	if code == "" {
		panic("invalid 'code' parameter")
	}

	// Exchange the authorization code for an access token
	token, err := oAuthConfig.Exchange(req.Context(), code)
	if err != nil {
		panic(err)
	}

	// Retrieve user info from Google
	client := oAuthConfig.Client(req.Context(), token)
	client.Timeout = time.Duration(30) * time.Second
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		panic(fmt.Errorf("failed to fetch user info: %w", err))
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		panic(fmt.Errorf("unexpected status code fetching user info: %d", resp.StatusCode))
	}

	// Parse the user info
	var userInfo struct {
		Sub           string `json:"sub"`            // Unique Google user ID
		Email         string `json:"email"`          // User's email address
		VerifiedEmail bool   `json:"email_verified"` // Whether email is verified
		Name          string `json:"name"`           // User's full name
		Picture       string `json:"picture"`        // User's profile picture URL
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		panic(fmt.Errorf("failed to parse user info: %w", err))
	}
	// Use the unique user ID (userInfo.Sub) or email (userInfo.Email) as the unique identifier
	log.Debugf("google drive authenticated user: ID=%s, Email=%s", userInfo.Sub, userInfo.Email)

	datasource := common.DataSource{
		SyncConfig: common.SyncConfig{Enabled: true, Interval: "30s"},
		Enabled:    true,
	}
	datasource.ID = util.MD5digest(fmt.Sprintf("%v,%v,%v,%v", "google_drive", connectorID, userInfo.Sub, userInfo.Email))
	datasource.Type = "connector"
	datasource.Icon = "default"
	if userInfo.Name != "" {
		datasource.Name = userInfo.Name + "'s Google Drive"
	} else {
		datasource.Name = "My Google Drive"
	}

	datasource.Connector = common.ConnectorConfig{
		ConnectorID: connectorID,
		Config: util.MapStr{
			"access_token":  token.AccessToken,                 // Store access token
			"refresh_token": token.RefreshToken,                // Store refresh token
			"token_expiry":  token.Expiry.Format(time.RFC3339), // Format using RFC3339
			"profile":       userInfo,
		},
	}

	// Check if refresh token is missing or empty
	if token.RefreshToken == "" {
		log.Warnf("refresh token was not granted for: %v", datasource.Name)
		log.Warnf("This may cause issues with automatic token refresh. Consider re-authorizing with prompt=consent parameter.")
	}

	ctx := orm.NewContextWithParent(req.Context())
	common.MarkDatasourceNotDeleted(datasource.ID)
	err = orm.Save(ctx, &datasource)
	if err != nil {
		panic(err)
	}

	newRedirectUrl := fmt.Sprintf("/#/data-source/detail/%v", datasource.ID)

	http.Redirect(w, req, newRedirectUrl, http.StatusSeeOther)
}
