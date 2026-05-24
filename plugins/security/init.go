/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"infini.sh/coco/core"
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
			api.HandleUIMethod(api.GET, "/account/profile", apiHandler.Profile, api.RequireLogin(), api.Feature(core.FeatureCORS))
			api.HandleUIMethod(api.OPTIONS, "/account/profile", apiHandler.Profile, api.RequireLogin(), api.Feature(core.FeatureCORS))
			api.HandleUIMethod(api.POST, "/account/login", apiHandler.Login)
			api.HandleUIMethod(api.PUT, "/account/password", apiHandler.UpdatePassword, api.RequireLogin())
		}
	})

}
