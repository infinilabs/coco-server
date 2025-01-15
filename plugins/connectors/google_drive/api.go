/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package google_drive

import (
	"encoding/base64"
	"fmt"
	"golang.org/x/oauth2"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
	"net/http"
	"net/url"
	"strings"
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
	tenantID := customArgs["tenant"]
	userID := customArgs["user"]
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

	err = h.saveToken(tenantID, userID, token)
	if err != nil {
		panic(err)
	}

	newRedirectUrl := util.JoinPath(redirectPath, "?source=google_drive")
	h.Redirect(w, req, newRedirectUrl)
}

func (h *Plugin) getTenantKey(tenantID, userID string) string {
	return strings.Join([]string{tenantID, userID}, ",")
}

func (h *Plugin) reset(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	//from context
	tenantID := "test"
	userID := "test"

	tenantKey := h.getTenantKey(tenantID, userID)
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

func (this *Plugin) saveLastModifiedTime(tenantID, userID string, lastModifiedTime string) error {
	tenantKey := this.getTenantKey(tenantID, userID)
	err := kv.AddValue("/connector/google_drive/lastModifiedTime", []byte(tenantKey), []byte(lastModifiedTime))
	return err
}

func (this *Plugin) getLastModifiedTime(tenantID, userID string) (string, error) {
	tenantKey := this.getTenantKey(tenantID, userID)
	data, err := kv.GetValue("/connector/google_drive/lastModifiedTime", []byte(tenantKey))
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (this *Plugin) saveToken(tenantID, userID string, token *oauth2.Token) error {
	tenantKey := this.getTenantKey(tenantID, userID)
	err := kv.AddValue("/connector/google_drive/token", []byte(tenantKey), util.MustToJSONBytes(token))
	return err
}

func (this *Plugin) getToken(tenantID, userID string) (bool, *oauth2.Token, error) {
	tenantKey := this.getTenantKey(tenantID, userID)
	data, err := kv.GetValue("/connector/google_drive/token", []byte(tenantKey))
	if err != nil {
		return false, nil, err
	}

	if data != nil {
		obj := oauth2.Token{}
		err = util.FromJSONBytes(data, &obj)
		if err != nil {
			return false, nil, err
		}

		return true, &obj, nil
	}

	return false, nil, nil
}
