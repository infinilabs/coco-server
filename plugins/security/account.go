/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"net/http"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"

	"golang.org/x/crypto/bcrypt"
	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/security"
)

func (h APIHandler) Profile(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	if !api.IsAuthEnable() {
		panic("auth is not enabled")
	}

	if global.Env().SystemConfig.WebAppConfig.Security.Managed {
		panic("should not be invoked as in managed mode")
	}

	reqUser, err := security.GetUserFromContext(r.Context())
	if err != nil || reqUser == nil {
		api.WriteAuthRequiredError(w, "invalid user")
		return
	}

	if reqUser.Has(core.UserSessionInfoKeyIntegration) {
		log.Trace("user login via INTEGRATION, guest user!")
		api.WriteAuthRequiredError(w, "no profile for guest user")
		return
	}

	//TODO get from user's profile, or fallback to account info

	_, user, err := security.GetUserByID(reqUser.MustGetUserID())
	if err != nil {
		panic(err)
	}
	if user == nil {
		api.WriteAuthRequiredError(w, "user not found")
		return
	}

	profile := security.UserProfile{Name: user.Name}
	profile.Email = user.Email
	profile.ID = user.ID
	profile.Name = user.Name
	profile.Permissions = reqUser.GetPermissionKeys()

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
