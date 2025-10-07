/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"infini.sh/coco/plugins/security/filter"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/global"
)

type APIHandler struct {
	api.Handler
}

var apiHandler = APIHandler{}

func init() {

	global.RegisterFuncBeforeSetup(func() {
		//for not managed only
		if !global.Env().SystemConfig.WebAppConfig.Security.Managed {
			api.HandleUIMethod(api.GET, "/account/profile", apiHandler.Profile, api.RequireLogin())
			api.HandleUIMethod(api.POST, "/account/login", apiHandler.Login)
			api.HandleUIMethod(api.PUT, "/account/password", apiHandler.UpdatePassword, api.RequireLogin())
		}
	})

	api.HandleUIMethod(api.POST, "/account/logout", apiHandler.Logout, api.OptionLogin())

	api.HandleUIMethod(api.POST, "/auth/access_token", apiHandler.RequestAccessToken, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/auth/access_token/_search", apiHandler.SearchAccessToken, api.RequireLogin(), api.Feature(filter.FeatureMaskSensitiveField))
	api.HandleUIMethod(api.DELETE, "/auth/access_token/:token_id", apiHandler.DeleteAccessToken, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/auth/access_token/:token_id/_rename", apiHandler.RenameAccessToken, api.RequireLogin())

}
