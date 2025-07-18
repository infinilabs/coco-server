/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package google_drive

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"golang.org/x/oauth2"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func encodeState(args map[string]string) string {
	var stateParts []string
	for key, value := range args {
		stateParts = append(stateParts, fmt.Sprintf("%s=%s", key, url.QueryEscape(value)))
	}
	return base64.URLEncoding.EncodeToString([]byte(strings.Join(stateParts, "&")))
}

func decodeState(state string) (map[string]string, error) {
	decoded, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return nil, err
	}

	args := map[string]string{}
	for _, part := range strings.Split(string(decoded), "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			args[kv[0]], _ = url.QueryUnescape(kv[1])
		}
	}
	return args, nil
}

func (h *Plugin) connect(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	//TODO handle tenant info
	// Custom arguments to pass
	customArgs := map[string]string{
		"tenant":   "test",
		"user":     "test",
		"redirect": "/connector/connect_success",
	}

	// Encode customArgs into a single string
	state := encodeState(customArgs)

	if h.oAuthConfig == nil {
		panic("invalid oauth config")
	}

	// Generate OAuth URL
	authURL := h.oAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

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

	// Retrieve the state parameter from the query
	state := req.URL.Query().Get("state")
	if state == "" {
		panic(errors.New("Missing 'state' parameter"))
	}

	// Decode the state parameter
	customArgs, err := decodeState(state)
	if err != nil {
		panic(err)
	}

	//// Access custom arguments
	//tenantID := customArgs["tenant"]
	//userID := customArgs["user"]
	redirectPath := customArgs["redirect"]

	// Extract the code from the query parameters
	code := req.URL.Query().Get("code")
	if code == "" {
		panic(err)
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
		SyncEnabled: true,
		Enabled:     true,
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

	err = orm.Save(nil, &datasource)
	if err != nil {
		panic(err)
	}

	newRedirectUrl := util.JoinPath(redirectPath, "?source=google_drive")
	h.Redirect(w, req, newRedirectUrl)
}

func (h *Plugin) getTenantKey(tenantID, userID, datasourceID string) string {
	return strings.Join([]string{tenantID, userID, datasourceID}, ",")
}

func (h *Plugin) reset(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	//from context
	tenantID := "test"
	userID := "test"
	datasourceID := h.GetParameter(req, "datasource")

	tenantKey := h.getTenantKey(tenantID, userID, datasourceID)
	err := kv.DeleteKey("/connector/google_drive/lastModifiedTime", []byte(tenantKey))
	if err != nil {
		panic(err)
	}
	err = kv.DeleteKey("/connector/google_drive/token", []byte(tenantKey))
	if err != nil {
		panic(err)
	}

	h.WriteAckOKJSON(w)
}

func (this *Plugin) saveLastModifiedTime(tenantID, userID, datasourceID string, lastModifiedTime string) error {
	tenantKey := this.getTenantKey(tenantID, userID, datasourceID)
	err := kv.AddValue("/connector/google_drive/lastModifiedTime", []byte(tenantKey), []byte(lastModifiedTime))
	return err
}

func (this *Plugin) getLastModifiedTime(tenantID, userID, datasourceID string) (string, error) {
	tenantKey := this.getTenantKey(tenantID, userID, datasourceID)
	data, err := kv.GetValue("/connector/google_drive/lastModifiedTime", []byte(tenantKey))
	if err != nil {
		return "", err
	}

	return string(data), nil
}
