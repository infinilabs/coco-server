/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"fmt"
	"infini.sh/framework/core/global"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

func (h APIHandler) Logout(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	api.DestroySession(w, r)
	h.WriteOKJSON(w, util.MapStr{
		"status": "ok",
	})
}

func (h APIHandler) Profile(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	if !api.IsAuthEnable() {
		panic("auth is not enabled")
	}

	reqUser, err := security.GetUserFromContext(r.Context())
	if err != nil || reqUser == nil {
		panic("invalid user")
	}

	var data []byte
	if global.Env().SystemConfig.WebAppConfig.Security.Managed {
		data, err = kv.GetValue(core.UserProfileKey, []byte(reqUser.GetKey()))
		if err != nil {
			panic(err)
		}
	} else {
		data, err = kv.GetValue(core.DefaultSettingBucketKey, []byte(core.DefaultUserProfileKey))
		if err != nil {
			panic(err)
		}
	}

	h.WriteBytes(w, data, 200)
}

func (h APIHandler) UpdatePassword(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if global.Env().SystemConfig.WebAppConfig.Security.Managed {
		panic("should not be invoked as in managed mode")
	}

	reqUser, err := security.GetUserFromContext(r.Context())
	if err != nil {
		panic(err)
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	err = h.DecodeJSON(r, &req)
	if err != nil {
		h.ErrorInternalServer(w, err.Error())
		return
	}

	err, success := h.checkPassword(req.OldPassword)
	if !success {
		h.WriteError(w, "failed to login", 403)
		return
	}

	err = SavePassword(req.NewPassword)
	if err != nil {
		h.ErrorInternalServer(w, err.Error())
		return
	}
	h.WriteOKJSON(w, api.UpdateResponse(reqUser.Login))
	return
}

func SavePassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = kv.AddValue(core.DefaultSettingBucketKey, []byte(core.DefaultUserPasswordKey), hash)
	return err
}

func (h APIHandler) Login(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	if global.Env().SystemConfig.WebAppConfig.Security.Managed {
		panic("should not be invoked as in managed mode")
	}

	var req struct {
		Password string `json:"password"`
	}

	var fromForm = false
	var requestID = h.GetParameter(r, "request_id")

	// Check content type and parse accordingly
	contentType := r.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(contentType, "application/json"):
		// Handle JSON input
		err := h.DecodeJSON(r, &req)
		if err != nil {
			h.ErrorInternalServer(w, "invalid JSON format")
			return
		}

	case strings.HasPrefix(contentType, "application/x-www-form-urlencoded"),
		strings.HasPrefix(contentType, "multipart/form-data"):
		// Handle form input
		if err := r.ParseForm(); err != nil {
			h.ErrorInternalServer(w, "failed to parse form data")
			return
		}
		fromForm = true
		req.Password = r.PostFormValue("password")

	default:
		h.WriteError(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}

	// Validate password exists
	if req.Password == "" {
		h.WriteError(w, "password is required", http.StatusBadRequest)
		return
	}

	// Rest of your existing logic
	err, success := h.checkPassword(req.Password)
	if !success {
		h.WriteError(w, "failed to login", http.StatusForbidden)
		return
	}

	var user = &security.UserProfile{
		Name: core.DefaultUserLogin,
	}
	user.ID = core.DefaultUserLogin

	sessionInfo := security.UserSessionInfo{}
	sessionInfo.Source = "simple"
	sessionInfo.Provider = "simple"
	sessionInfo.Login = core.DefaultUserLogin

	//sessionInfo.Profile = user
	sessionInfo.Roles = []string{security.RoleAdmin}

	err, token := AddUserAccessTokenToSession(w, r, &sessionInfo)
	if err != nil {
		h.ErrorInternalServer(w, "failed to authorize user")
		return
	}

	if fromForm {
		h.Redirect(w, r, fmt.Sprintf("/login/success?request_id=%v&code=%v", requestID, token["access_token"]))
	} else {
		h.WriteOKJSON(w, token)
	}
}

func AddUserAccessTokenToSession(w http.ResponseWriter, r *http.Request, user *security.UserSessionInfo) (error, map[string]interface{}) {

	if user == nil {
		panic("invalid user")
	}

	// Generate access token
	token, err := GenerateJWTAccessToken(user)
	if err != nil {
		return err, nil
	}

	api.SetSession(w, r, core.UserAccessTokenSessionName, token["access_token"])
	return nil, token
}

func (h APIHandler) checkPassword(password string) (error, bool) {
	savedPassword, err := kv.GetValue(core.DefaultSettingBucketKey, []byte(core.DefaultUserPasswordKey))
	if err != nil {
		return err, false
	}

	err = bcrypt.CompareHashAndPassword([]byte(savedPassword), []byte(password))
	if err != nil {
		return err, false
	}
	return nil, true
}
