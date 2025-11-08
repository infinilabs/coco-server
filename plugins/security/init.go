/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
)

type APIHandler struct {
	api.Handler
}

var apiHandler = APIHandler{}

func init() {

	global.RegisterFuncBeforeSetup(func() {
		//for not managed only
		if !global.Env().SystemConfig.WebAppConfig.Security.Managed {
			api.HandleUIMethod(api.GET, "/account/profile", apiHandler.Profile, api.RequireLogin(), api.Feature(core.FeatureCORS))
			api.HandleUIMethod(api.OPTIONS, "/account/profile", apiHandler.Profile, api.RequireLogin(), api.Feature(core.FeatureCORS))
			api.HandleUIMethod(api.POST, "/account/login", apiHandler.Login)
			api.HandleUIMethod(api.PUT, "/account/password", apiHandler.UpdatePassword, api.RequireLogin())
		}
	})

	api.HandleUIMethod(api.POST, "/account/logout", apiHandler.Logout, api.OptionLogin(), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/account/logout", apiHandler.Logout, api.OptionLogin(), api.Feature(core.FeatureCORS))

	createTokenPermission := security.GetSimplePermission("generic", "security:auth:api-token", security.Create)
	updateTokenPermission := security.GetSimplePermission("generic", "security:auth:api-token", security.Update)
	deleteTokenPermission := security.GetSimplePermission("generic", "security:auth:api-token", security.Delete)
	searchTokenPermission := security.GetSimplePermission("generic", "security:auth:api-token", security.Search)

	security.GetOrInitPermissionKeys(createTokenPermission, updateTokenPermission, deleteTokenPermission, searchTokenPermission)

	api.HandleUIMethod(api.POST, "/auth/access_token", apiHandler.RequestAccessToken, api.RequirePermission(createTokenPermission))
	api.HandleUIMethod(api.GET, "/auth/access_token/_search", apiHandler.SearchAccessToken, api.RequirePermission(searchTokenPermission), api.Feature(core.FeatureMaskSensitiveField))
	api.HandleUIMethod(api.DELETE, "/auth/access_token/:token_id", apiHandler.DeleteAccessToken, api.RequirePermission(deleteTokenPermission))
	api.HandleUIMethod(api.PUT, "/auth/access_token/:token_id", apiHandler.UpdateAccessToken, api.RequirePermission(updateTokenPermission))
}
