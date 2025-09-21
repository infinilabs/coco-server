/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package google_drive

import (
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"golang.org/x/oauth2"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
	"net/url"
	"time"
)

func (h *Plugin) connect(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	if h.oAuthConfig == nil {
		panic("invalid oauth config")
	}

	// Generate OAuth URL
	authURL := h.oAuthConfig.AuthCodeURL("", oauth2.AccessTypeOffline)

	// Parse the generated URL to append additional parameters
	parsedURL, err := url.Parse(authURL)
	if err != nil {
		panic("Failed to parse auth URL")
	}

	// Add the `approval_prompt=force` parameter to ensure the refresh token is returned
	query := parsedURL.Query()
	query.Set("approval_prompt", "force")
	parsedURL.RawQuery = query.Encode()

	// Return the updated URL with the necessary parameters
	authURL = parsedURL.String()

	http.Redirect(w, req, authURL, http.StatusFound)
}

// Endpoint to handle the OAuth redirect
func (h *Plugin) oAuthRedirect(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	// Extract the code from the query parameters
	code := req.URL.Query().Get("code")
	if code == "" {
		panic("invalid 'code' parameter")
	}

	// Exchange the authorization code for an access token
	token, err := h.oAuthConfig.Exchange(req.Context(), code)
	if err != nil {
		panic(err)
	}

	// Retrieve user info from Google
	client := h.oAuthConfig.Client(req.Context(), token)
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
	log.Infof("google drive authenticated user: ID=%s, Email=%s", userInfo.Sub, userInfo.Email)

	datasource := common.DataSource{
		SyncConfig: common.SyncConfig{Enabled: true, Interval: "30s"},
		Enabled:    true,
	}
	datasource.ID = util.MD5digest(fmt.Sprintf("%v,%v,%v", "google_drive", userInfo.Sub, userInfo.Email))
	datasource.Type = "connector"
	if userInfo.Name != "" {
		datasource.Name = userInfo.Name + "'s Google Drive"
	} else {
		datasource.Name = "My Google Drive"
	}
	datasource.Connector = common.ConnectorConfig{
		ConnectorID: "google_drive",
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
	}

	ctx := orm.NewContextWithParent(req.Context())
	err = orm.Save(ctx, &datasource)
	if err != nil {
		panic(err)
	}

	newRedirectUrl := fmt.Sprintf("/#/data-source/detail/%v", datasource.ID)

	h.Redirect(w, req, newRedirectUrl)
}

func (h *Plugin) reset(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	//from context
	datasourceID := h.GetParameter(req, "datasource")

	err := kv.DeleteKey("/connector/google_drive/lastModifiedTime", []byte(datasourceID))
	if err != nil {
		panic(err)
	}

	h.WriteAckOKJSON(w)
}

func (this *Plugin) saveLastModifiedTime(datasourceID string, lastModifiedTime string) error {
	err := kv.AddValue("/connector/google_drive/lastModifiedTime", []byte(datasourceID), []byte(lastModifiedTime))
	return err
}

func (this *Plugin) getLastModifiedTime(datasourceID string) (string, error) {
	data, err := kv.GetValue("/connector/google_drive/lastModifiedTime", []byte(datasourceID))
	if err != nil {
		return "", err
	}

	return string(data), nil
}
