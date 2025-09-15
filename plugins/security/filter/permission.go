/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package filter

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/api"
	common "infini.sh/framework/core/api/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	ccache "infini.sh/framework/lib/cache"
	"net/http"
	"time"
)

func init() {
	api.RegisterUIFilter(&PermissionFilter{})
}

type PermissionFilter struct {
	api.Handler
}

func (f *PermissionFilter) GetPriority() int {
	return 500
}

func (f *PermissionFilter) ApplyFilter(
	method string,
	pattern string,
	options *api.HandlerOptions,
	next httprouter.Handle,
) httprouter.Handle {

	if options == nil || options.RequirePermission == nil || len(options.RequirePermission) == 0 || !common.IsAuthEnable() {
		log.Debug(method, ",", pattern, ",skip permission check")
		return next
	}

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		reqUser, err := security.GetUserFromContext(r.Context())
		if reqUser == nil || err != nil {
			o := api.PrepareErrorJson("invalid login", 401)
			f.WriteJSON(w, o, 401)
			return
		}

		//bypass admin
		if reqUser.Roles != nil && util.AnyInArrayEquals(reqUser.Roles, security.RoleAdmin) {
			next(w, r, ps)
			return
		}

		if reqUser.UserAssignedPermission == nil || reqUser.UserAssignedPermission.NeedRefresh() {
			reqUser.UserAssignedPermission = GetUserPermissions(reqUser)
		}

		if reqUser.UserAssignedPermission == nil || options.RequirePermission == nil || len(options.RequirePermission) == 0 {
			panic("invalid permission state")
		}

		if global.Env().IsDebug {
			log.Tracef("perm key: %v", options.RequirePermission)
		}

		if reqUser.UserAssignedPermission.Validate(security.MustRegisterPermissionByKeys(options.RequirePermission)) {
			next(w, r, ps)
		} else {
			f.WriteErrorObject(w, errors.Errorf("permission [%v] not allowed", options.RequirePermission), 403)
		}
	}
}

var permissionCache = ccache.Layered(ccache.Configure().MaxSize(10000).ItemsToPrune(100))

func GetUserPermissions(shortUser *security.UserSessionInfo) *security.UserAssignedPermission {

	var skipCache = false
	if shortUser.UserAssignedPermission != nil && shortUser.UserAssignedPermission.NeedRefresh() {
		skipCache = true
	}

	if !skipCache {
		v := permissionCache.Get(PermissionCache, shortUser.GetKey())
		if v != nil {
			if !v.Expired() {
				x, ok := v.Value().(*security.UserAssignedPermission)
				if ok {
					if global.Env().IsDebug {
						log.Debug("hit permission cache")
						x.Dump()
					}
					return x
				}
			}
		}
	}

	var allowedPermissions = []string{}
	if len(shortUser.Roles) > 0 {
		for _, v := range shortUser.Roles {
			perms, ok := security.GetPermissionsForRole(v)
			if !ok {
				panic(errors.Errorf("invalid role: %v", v))
			}
			allowedPermissions = append(allowedPermissions, perms...)
		}
	}

	//user, err := security.GetUser(shortUser.UserID)
	//if err != nil {
	//	panic(err)
	//}

	//privilege := api2.GetUserAllowedPrivileges(shortUser, user)
	//log.Debugf("get user's privileges: %v, %v", shortUser.UserID, privilege)

	//for _, v := range privilege {
	//	p := security.DefaultRBAC.GetPrivilege(v)
	//	if p != nil {
	//		for resource, n := range p.Grants {
	//			for x, _ := range n {
	//				id := security.GetSimplePermission(permission.CategoryPlatform, resource, string(x))
	//				allowedPermissions = append(allowedPermissions, id)
	//				log.Debugf("register permission: %v, category: %v, resource: %v, action: %v", id, permission.CategoryPlatform, resource, string(x))
	//			}
	//		}
	//	}
	//}

	//log.Error("user's permissioins:", allowedPermissions)
	perms := security.NewUserAssignedPermission(allowedPermissions, nil)
	if perms != nil {
		permissionCache.Set(PermissionCache, shortUser.GetKey(), perms, util.GetDurationOrDefault("30m", time.Duration(30)*time.Minute))
		return perms
	}
	return nil
}

const PermissionCache = "UserPermissionCache"
