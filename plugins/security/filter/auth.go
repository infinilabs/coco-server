/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package filter

import (
	"net/http"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	common "infini.sh/framework/core/api/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

func init() {
	api.RegisterUIFilter(&AuthFilter{})
}

type AuthFilter struct {
	api.Handler
}

func (f *AuthFilter) GetPriority() int {
	return 200
}
func (f *AuthFilter) ApplyFilter(
	method string,
	pattern string,
	options *api.HandlerOptions,
	next httprouter.Handle,
) httprouter.Handle {

	//option not enabled
	if options == nil || (!options.RequireLogin && !options.OptionLogin) || !common.IsAuthEnable() {
		log.Debug(method, ",", pattern, ",skip auth")
		return next
	}

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		claims, err1 := core.ValidateLogin(w, r)

		if global.Env().IsDebug {
			log.Debug(method, ",", pattern, ",", util.MustToJSON(claims), ",", err1)
		}

		if claims != nil && claims.IsValid() {
			//update the initial permission
			claims.UserAssignedPermission = security.GetUserPermissions(claims)
			r = r.WithContext(security.AddUserToContext(r.Context(), claims))
		}

		if !options.OptionLogin {
			if claims == nil {
				o := api.PrepareErrorJson("invalid login", 401)
				f.WriteJSON(w, o, 401)
				return
			}

			//error 403 for empty permission
			if claims.UserAssignedPermission == nil || claims.UserAssignedPermission.AllowedPermissions == nil || claims.UserAssignedPermission.AllowedPermissions.IsEmpty() {
				api.WriteAccessDeniedError(w, "no permission")
				return
			}

			if err1 != nil {
				f.WriteErrorObject(w, err1, 401)
				return
			}
		}

		next(w, r, ps)
	}
}
