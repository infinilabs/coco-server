/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"fmt"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
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

	if global.Env().SystemConfig.WebAppConfig.Security.Managed {
		panic("should not be invoked as in managed mode")
	}

	reqUser, err := security.GetUserFromContext(r.Context())
	if err != nil || reqUser == nil {
		panic("invalid user")
	}

	//TODO get from user's profile, or fallback to account info

	_, user, err := security.GetUserByID(reqUser.MustGetUserID())
	if err != nil {
		panic(err)
	}
	if user == nil {
		panic("user not found")
	}

	profile := security.UserProfile{Name: user.Name}
	profile.Email = user.Email
	profile.ID = user.ID
	profile.Name = user.Name
	profile.Permissions = security.MustGetPermissionKeysByRole(user.Roles)

	h.WriteJSON(w, profile, 200)
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

	id := reqUser.MustGetUserID()
	err, account, success := h.checkPasswordForUserID(id, req.OldPassword)
	if !success {
		h.WriteError(w, "failed to login", 403)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	account.Password = string(hash)
	ctx := orm.NewContextWithParent(r.Context())
	ctx.Refresh = orm.WaitForRefresh
	err = orm.Save(ctx, account)
	if err != nil {
		panic(err)
	}

	h.WriteUpdatedOKJSON(w, id)
	return
}

func (h APIHandler) Login(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	if global.Env().SystemConfig.WebAppConfig.Security.Managed {
		panic("should not be invoked as in managed mode")
	}

	var req struct {
		Email    string `json:"email,omitempty"`
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

	sessionInfo := security.UserSessionInfo{}

	if req.Email == "" {
		err, success := h.checkPassword(req.Password)
		if err != nil {
			panic(err)
		}
		if !success {
			h.WriteError(w, "failed to login", http.StatusForbidden)
			return
		}

		sessionInfo.Provider = core.DefaultSimpleAuthBackend
		sessionInfo.Login = core.DefaultSimpleAuthUserLogin
		sessionInfo.Roles = []string{security.RoleAdmin}
		sessionInfo.SetGetUserID(core.DefaultSimpleAuthUserLogin)
	} else {
		err, account, success := h.checkPasswordForEmail(req.Email, req.Password)
		if err != nil {
			panic(err)
		}
		if !success {
			h.WriteError(w, "failed to login", http.StatusForbidden)
			return
		}

		sessionInfo.Provider = security.DefaultNativeAuthBackend
		sessionInfo.Login = account.Email
		sessionInfo.Roles = account.Roles
		sessionInfo.SetGetUserID(account.ID)
	}

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

func (h APIHandler) checkPasswordForEmail(email, password string) (error, *security.UserAccount, bool) {

	exists, account, err := security.MustGetAuthenticationProvider(security.DefaultNativeAuthBackend).GetUserByLogin(email)
	if err != nil {
		return err, nil, false
	}
	if !exists || account == nil || account.Password == "" {
		//user not exists
		return nil, nil, false
	}
	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password))
	if err == nil {
		return nil, account, true
	}
	return nil, nil, false
}

func (h APIHandler) checkPasswordForUserID(id, password string) (error, *security.UserAccount, bool) {

	_, account, err := security.GetUserByID(id)
	if err != nil {
		return err, nil, false
	}
	if account == nil || account.Password == "" {
		//user not exists
		return nil, nil, false
	}
	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password))
	if err == nil {
		return nil, account, true
	}
	return nil, nil, false
}

func (h APIHandler) checkPassword(password string) (error, bool) {
	savedPassword, err := kv.GetValue(core.DefaultSettingBucketKey, []byte(core.DefaultUserPasswordKey))
	if err != nil {
		return err, false
	}

	if savedPassword == nil || len(savedPassword) == 0 {
		panic("previous password was not set")
	}

	err = bcrypt.CompareHashAndPassword([]byte(savedPassword), []byte(password))
	if err != nil {
		return err, false
	}
	return nil, true
}
