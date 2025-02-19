/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package oauth

import (
	"infini.sh/coco/plugins/security/config"
	"infini.sh/framework/core/api"
)

func Init(oathConfig map[string]config.OAuthConfig) {

	if len(oathConfig) > 0 {

		apiHandler.Init(oathConfig)

		api.HandleUIMethod(api.GET, "/sso/login/", apiHandler.ssoLoginIndex)
		api.HandleUIMethod(api.GET, "/sso/login/:provider", apiHandler.AuthHandler)
		api.HandleUIMethod(api.GET, "/sso/callback/:provider", apiHandler.CallbackHandler)

		api.HandleUIMethod(api.GET, "/auth/request_access_token", apiHandler.requestToken)

	}
}
