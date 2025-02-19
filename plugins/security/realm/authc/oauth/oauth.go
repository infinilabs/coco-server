/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"infini.sh/coco/plugins/security/config"
	"infini.sh/coco/plugins/security/core"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

func New(cfg2 config.OAuthConfig) *OAuthRealm {

	var realm = &OAuthRealm{
		//config: cfg2,
		//ldapCfg: config.OAuthConfig{
		//	Port:           cfg2.Port,
		//	Host:           cfg2.Host,
		//	TLS:            nil,
		//	BindDN:         cfg2.BindDn,
		//	BindPassword:   cfg2.BindPassword,
		//	Attributes:     nil,
		//	BaseDN:         cfg2.BaseDn,
		//	UserFilter:     cfg2.UserFilter,
		//	GroupFilter:    "",
		//	UIDAttribute:   cfg2.UidAttribute,
		//	GroupAttribute: cfg2.GroupAttribute,
		//},
	}
	//realm.ldapFunc = ldap.GetAuthenticateFunc(&realm.ldapCfg)
	return realm
}

func (h *APIHandler) getDefaultRoles(provider string) []core.UserRole {

	oAuthConfig, _ := h.mustGetAuthConfig(provider)
	if len(oAuthConfig.DefaultRoles) == 0 {
		return nil
	}

	if len(h.defaultOAuthRoles) > 0 {
		return h.defaultOAuthRoles
	}

	roles := h.getRolesByRoleIDs(oAuthConfig.DefaultRoles)
	if len(roles) > 0 {
		h.defaultOAuthRoles = roles
	}
	return roles
}

func (h *APIHandler) getRolesByRoleIDs(roles []string) []core.UserRole {
	out := []core.UserRole{}
	//for _, v := range roles {
	//	role, err := h.Adapter.Role.Get(v)
	//	if err != nil {
	//		if !strings.Contains(err.Error(), "not found") {
	//			panic(err)
	//		}
	//
	//		//try name
	//		role, err = h.Adapter.Role.GetBy("name", v)
	//		if err != nil {
	//			continue
	//		}
	//	}
	//	out = append(out, UserRole{ID: role.ID, Name: role.Name})
	//}
	return out
}

const oauthSession string = "oauth-session"

func (h *APIHandler) mustGetAuthConfig(provider string) (config.OAuthConfig, oauth2.Config) {
	oAuthConfig, ok := h.oAuthConfig[provider]
	if !ok {
		panic(errors.Errorf("oauth provider %s not found", provider))
	}

	oAuth2Config, ok := h.oauthCfg[provider]
	if !ok {
		panic(errors.Errorf("oauth provider %s not found", provider))
	}
	return oAuthConfig, oAuth2Config
}

func (h *APIHandler) ssoLoginIndex(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	provider := h.MustGetParameter(w, req, "provider")
	product := h.MustGetParameter(w, req, "product")
	requestID := h.MustGetParameter(w, req, "request_id")

	// Define supported SSO providers
	providers := []string{"github", "google"}

	// Generate HTML links for each provider
	var builder strings.Builder
	builder.WriteString("<html><head><title>SSO Login</title></head><body>")
	builder.WriteString("<h1>Select an SSO Provider</h1><ul>")

	for _, p := range providers {
		link := fmt.Sprintf(
			`<a href="/sso/login/%s?provider=%s&product=%s&request_id=%s">%s Login</a>`,
			p, provider, product, requestID, strings.Title(p),
		)
		builder.WriteString(fmt.Sprintf("<li>%s</li>", link))
	}

	builder.WriteString("</ul></body></html>")

	// Write the HTML response
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(builder.String()))

}

func (h *APIHandler) AuthHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	b := make([]byte, 16)
	rand.Read(b)

	state := base64.URLEncoding.EncodeToString(b)

	oauthProviderName := p.MustGetParameter("provider")
	requestID := h.GetParameter(r, "request_id")

	log.Tracef("oauth_redirect, provider: %v, request_id:%v", oauthProviderName, requestID)

	session, err := api.GetSessionStore(r, oauthSession)
	session.Values["state"] = state
	session.Values["request_id"] = requestID
	session.Values["provider"] = oauthProviderName
	session.Values["redirect_url"] = h.Get(r, "redirect_url", "")
	session.Values["product"] = h.Get(r, "product", "")
	session.Values["domain"] = h.Get(r, "domain", "")

	if session == nil {
		panic(errors.New("session is nil"))
	}

	oAuthConfig, oauthCfg := h.mustGetAuthConfig(oauthProviderName)

	err = session.Save(r, w)
	if err != nil {
		log.Error(err)
		http.Redirect(w, r, joinError(oAuthConfig.FailedPage, err), 302)
		return
	}

	url := oauthCfg.AuthCodeURL(state)
	http.Redirect(w, r, url, 302)
}

func joinError(url string, err error) string {
	if err != nil {
		return url + "?err=" + util.UrlEncode(err.Error())
	}
	return url
}

func safeDereference(strPtr *string) string {
	if strPtr != nil {
		return *strPtr
	}
	return ""
}

func (h *APIHandler) CallbackHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	provider := p.MustGetParameter("provider")
	oAuthConfig, oauthCfg := h.mustGetAuthConfig(provider)

	session, err := api.GetSessionStore(r, oauthSession)
	if err != nil || session == nil {
		log.Error("sso callback failed to get session_store, aborted")
		http.Redirect(w, r, joinError(oAuthConfig.FailedPage, err), 302)
		return
	}

	defer func() {
		session.Options.MaxAge = -1
		err = session.Save(r, w)
		if err != nil {
			log.Error(err)
		}
	}()

	if r.URL.Query().Get("state") != session.Values["state"] {
		log.Error("failed to sso, no state match; possible csrf OR cookies not enabled")
		http.Redirect(w, r, joinError(oAuthConfig.FailedPage, err), 302)
		return
	}

	oAuthProvider := session.Values["provider"].(string)
	product := session.Values["product"].(string)
	oAuthRequestID := session.Values["request_id"].(string)

	log.Debugf("oauth_callback, provider:%v vs %v, request_id:%v", oAuthProvider, provider, oAuthRequestID)

	if provider != oAuthProvider {
		panic("invalid provider")
	}

	tkn, err := oauthCfg.Exchange(oauth2.NoContext, r.URL.Query().Get("code"))
	if err != nil {
		log.Error("failed to sso, there was an issue getting your token: ", err)
		http.Redirect(w, r, joinError(oAuthConfig.FailedPage, err), 302)
		return
	}

	if !tkn.Valid() {
		log.Error("failed to sso, retrieved invalid token")
		http.Redirect(w, r, joinError(oAuthConfig.FailedPage, err), 302)
		return
	}

	state := session.Values["state"].(string)
	var username, nicename, email, avatar string
	var userPayload interface{}

	payload := util.MapStr{
		"product":       product,
		"request_id":    oAuthRequestID,
		"auth_provider": provider,
		"state":         state,
	}

	switch provider {
	case "google":
		client := oauthCfg.Client(context.Background(), tkn)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
		if err != nil {
			panic(fmt.Errorf("failed to fetch user info: %w", err))
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

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

		userPayload = userInfo
		username = userInfo.Sub
		nicename = userInfo.Name
		avatar = userInfo.Picture
		email = userInfo.Email

		//// Use the unique user ID (userInfo.Sub) or email (userInfo.Email) as the unique identifier
		//log.Infof("google drive authenticated user: ID=%s, Email=%v", userInfo.Sub, userInfo)

	case "github":
		//get user info
		client := github.NewClient(oauthCfg.Client(oauth2.NoContext, tkn))
		user, res, err := client.Users.Get(oauth2.NoContext, "")
		if err != nil || user == nil || *user.Login == "" {
			if res != nil {
				log.Error("failed to sso, error getting name:", err, res.String())
			}
			break
		}
		userPayload = *user

		username = safeDereference(user.Login)
		nicename = safeDereference(user.Name)
		avatar = safeDereference(user.AvatarURL)
		email = safeDereference(user.Email)

		//if user.Name != nil && *user.Name != "" {
		//	payload["nickname"] = *user.Name
		//}
		//if user.Email != nil && *user.Email != "" {
		//	payload["email"] = *user.Email
		//}
		//if user.AvatarURL != nil && *user.AvatarURL != "" {
		//	payload["avatar_url"] = *user.AvatarURL
		//}

		//update access_token for product
		//api.coco.rs -> db -> access_token valid

		//if dbUser != nil {
		//	//generate access token
		//	dbUser.AuthProvider = ProviderGithub
		//	data, err := GenerateAccessToken(dbUser, nil)
		//	if err != nil {
		//		log.Error(data, err)
		//		break
		//	}
		//	payload := util.MapStr{
		//		"sso_bound": true,
		//		"domain": session.Values["domain"],
		//	}
		//	api.SetSession(w, r, UserTokenSessionName, data["access_token"])
		//	url := oAuthConfig.SuccessPage + "?payload=" + util.UrlEncode(util.MustToJSON(payload))
		//	http.Redirect(w, r, url, 302)
		//	return
		//
		//}
		////jump to sso register page
		//payload := util.MapStr{
		//	"username": *user.Login,
		//	"auth_provider": ProviderGithub,
		//	"sso_bound": false,
		//	"domain": session.Values["domain"],
		//}
		//if user.Name != nil && *user.Name != "" {
		//	payload["nickname"] = *user.Name
		//}
		//if user.Email != nil && *user.Email != "" {
		//	payload["email"] = *user.Email
		//}
		//
		//if user.AvatarURL != nil && *user.AvatarURL != "" {
		//	payload["avatar_url"]= *user.AvatarURL
		//}

	}

	valid := false
	if username != "" {
		payload["username"] = username
		valid = true
	}

	if valid {
		//handle coco app
		tempToken := fmt.Sprintf("%v%v", util.GetUUID(), util.GenerateRandomString(64))

		//TODO
		//save userinfo with token to store
		userInfo := h.getExternalUserBy(provider, username)
		var userId string
		if userInfo == nil {

			//TODO, auto save this user to db if in our white list

			user := core.User{}
			userId = GetExternalUserProfileID(provider, username)
			user.ID = userId
			user.Name = nicename
			user.Email = email
			user.AvatarUrl = avatar
			err = orm.Save(nil, &user)
			if err != nil {
				panic(err)
			}

			userInfo, err = h.saveExternalUser(provider, username, userPayload, &user)
			if err != nil {
				panic(err)
			}

		} else {
			userId = userInfo.UserID
		}

		if userInfo != nil {
			//if the user is valid, assign the access token

			if userId == "" || username == "" {
				panic("invalid user info")
			}

			tempPayload := util.MapStr{}
			tempPayload["provider"] = provider
			tempPayload["login"] = username
			tempPayload["userid"] = userId
			tempPayload["code"] = tempToken
			tempPayload["request_id"] = oAuthRequestID
			//save a temp expired token within 15minutes
			h.cCache.GetOrCreateSecondaryCache("sso_temp_token").Set(oAuthRequestID, tempPayload, 15*time.Minute)

			log.Trace("set sso temp token: ", oAuthRequestID)

			payload["code"] = tempToken

			//TODO if the user was not found, refused to access
			//TODO save user's profile

			log.Trace("callback: %v", util.MustToJSON(payload))

			//use server's redirect_url first
			if url, ok := session.Values["redirect_url"].(string); ok && url != "" {
				url = url + "?payload=" + util.UrlEncode(util.MustToJSON(payload))
				http.Redirect(w, r, url, 302)
				return
			}

			url := oAuthConfig.SuccessPage + "?payload=" + util.UrlEncode(util.MustToJSON(payload))
			http.Redirect(w, r, url, 302)
			return
		}
	}

	http.Redirect(w, r, joinError(oAuthConfig.FailedPage, err), 302)
}

func (h *APIHandler) getRoleMapping(provider, username string) []core.UserRole {
	roles := []core.UserRole{}

	if username != "" {
		oAuthConfig, _ := h.mustGetAuthConfig(provider)
		if len(oAuthConfig.RoleMapping) > 0 {
			r, ok := oAuthConfig.RoleMapping[username]
			if ok {
				roles = h.getRolesByRoleIDs(r)
			}
		}
	}

	if len(roles) == 0 {
		return h.getDefaultRoles(provider)
	}
	return roles
}

type OAuthRealm struct {
	// Implement any required fields
}

func (r *OAuthRealm) GetType() string {
	return "oauth"
}

// Authenticate for oauth user, username equals sso id (oauth provider + "_" + oauth username)
//
// password equals oauth random state value
func (r *OAuthRealm) Authenticate(username, password string) (bool, *core.User, error) {

	panic("invalid access")

	//parts := strings.Split(username, "_")
	//if len(parts) < 2 {
	//	return false, nil, fmt.Errorf("invalid oauth username")
	//}
	//authProvider, oauthUsername := parts[0], parts[1]
	//user, ssoUser, err := apiHandler.User.GetUserBySSOID(username)
	//if err != nil && !strings.Contains(err.Error(), "not found") {
	//	return false, nil, err
	//}
	//if ssoUser == nil || user == nil {
	//	return false, nil, fmt.Errorf("oauth account is not registered")
	//}
	//// validate oauth state
	//ssoLog := &UserSSOLog{}
	//ssoLog.ID = fmt.Sprintf("%s_%s", oauthUsername, password)
	//exists, err := orm.Get(ssoLog)
	//if err != nil {
	//	return exists, nil, err
	//}
	//if ssoLog.AuthProvider != authProvider || ssoLog.IsUsed {
	//	return false, nil, fmt.Errorf("sso oauth info not found")
	//}
	//ssoLog.IsUsed = true
	//err = orm.Update(nil, ssoLog)
	//if err != nil {
	//	log.Error(err)
	//}
	//user.AuthProvider = providerName
	//return true, user, nil
}

func (r *OAuthRealm) Authorize(user *core.User) (bool, error) {
	var _, privilege = user.GetPermissions()

	if len(privilege) == 0 {
		log.Error("no privilege assigned to user:", user)
		return false, fmt.Errorf("no privilege assigned to this user:" + user.ID)
	}

	return true, nil
}
