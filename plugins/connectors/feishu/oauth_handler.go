package feishu

import (
	"encoding/json"
	"fmt"

	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"

	log "github.com/cihub/seelog"
)

// OAuthHandler handles OAuth operations for Feishu/Lark connectors
type OAuthHandler struct {
	PluginType  PluginType
	APIConfig   *APIConfig
	OAuthConfig *OAuthConfig
}

// NewOAuthHandler creates a new OAuth handler instance
func NewOAuthHandler(pluginType PluginType, oauthConfig *OAuthConfig) *OAuthHandler {
	return &OAuthHandler{
		PluginType:  pluginType,
		APIConfig:   getAPIConfig(pluginType),
		OAuthConfig: oauthConfig,
	}
}

// Token represents an OAuth token response
type Token struct {
	Code                  int    `json:"code"`
	AccessToken           string `json:"access_token"`
	ExpiresIn             int    `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	TokenType             string `json:"token_type"`
	Scope                 string `json:"scope"`
	Error                 string `json:"error"`
	ErrorDescription      string `json:"error_description"`
}

// exchangeCodeForToken exchanges authorization code for access token
func (h *OAuthHandler) exchangeCodeForToken(code string) (*Token, error) {
	if h.OAuthConfig == nil {
		return nil, errors.Errorf("OAuth config not initialized")
	}

	payload := map[string]interface{}{
		"client_id":     h.OAuthConfig.ClientID,
		"client_secret": h.OAuthConfig.ClientSecret,
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  h.OAuthConfig.RedirectURL,
	}

	log.Debugf("[%s connector] Exchanging code for token at: %s", h.PluginType, h.OAuthConfig.TokenURL)

	req := util.NewPostRequest(h.OAuthConfig.TokenURL, util.MustToJSONBytes(payload))
	req.AddHeader("Content-Type", "application/json")

	res, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.Errorf("%s API error, no response", h.PluginType)
	}

	if res.StatusCode >= 300 {
		return nil, errors.Errorf("%s API error: status %d, body: %s", h.PluginType, res.StatusCode, string(res.Body))
	}

	var tokenResponse Token
	if err := json.Unmarshal(res.Body, &tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}

// getUserProfile retrieves user profile information using access token
func (h *OAuthHandler) getUserProfile(accessToken string) (util.MapStr, error) {
	req := util.NewGetRequest(h.APIConfig.UserInfoURL, nil)
	req.AddHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	res, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.Errorf("%s API error, no response", h.PluginType)
	}

	if res.StatusCode >= 300 {
		return nil, errors.Errorf("%s API error: status %d, body: %s", h.PluginType, res.StatusCode, string(res.Body))
	}

	var response struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data util.MapStr `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &response); err != nil {
		return nil, err
	}

	if response.Code != 0 {
		return nil, errors.Errorf("%s API error: %s", h.PluginType, response.Msg)
	}

	return response.Data, nil
}

// refreshAccessTokenWithConnectorConfig refreshes the access token using refresh token
func (h *OAuthHandler) refreshAccessTokenWithConnectorConfig(refreshToken string, connectorConfig util.MapStr) (*Token, error) {
	// Extract OAuth credentials from connector config
	clientID, _ := connectorConfig["client_id"].(string)
	clientSecret, _ := connectorConfig["client_secret"].(string)

	if clientID == "" || clientSecret == "" {
		return nil, errors.Errorf("OAuth client_id and client_secret not found in connector config")
	}

	tokenURL := h.APIConfig.RefreshTokenURL

	// Allow override from connector config
	if url, ok := connectorConfig["token_url"].(string); ok && url != "" {
		tokenURL = url
	}

	payload := map[string]interface{}{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}

	log.Debugf("[%s connector] Refreshing access token at: %s", h.PluginType, tokenURL)

	req := util.NewPostRequest(tokenURL, util.MustToJSONBytes(payload))
	req.AddHeader("Content-Type", "application/json")

	res, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.Errorf("%s API error, no response", h.PluginType)
	}

	if res.StatusCode >= 300 {
		return nil, errors.Errorf("%s API error: status %d, body: %s", h.PluginType, res.StatusCode, string(res.Body))
	}

	var tokenResponse Token
	if err := json.Unmarshal(res.Body, &tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}
