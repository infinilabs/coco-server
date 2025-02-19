/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package oauth

import (
	"fmt"
	"golang.org/x/oauth2"
	"infini.sh/coco/plugins/security/config"
	"infini.sh/coco/plugins/security/core"
	"infini.sh/framework/core/orm"
	ccache "infini.sh/framework/lib/cache"
)

type APIHandler struct {
	core.Handler
	cCache *ccache.LayeredCache

	oAuthConfig       map[string]config.OAuthConfig
	defaultOAuthRoles []core.UserRole

	oauthCfg map[string]oauth2.Config
}

func (h *APIHandler) Init(oathConfig map[string]config.OAuthConfig) {

	h.oAuthConfig = make(map[string]config.OAuthConfig)
	h.oauthCfg = make(map[string]oauth2.Config)

	for k, cfg := range oathConfig {
		h.oAuthConfig[k] = cfg
		oauthCfg := oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  cfg.AuthorizeUrl,
				TokenURL: cfg.TokenUrl,
			},
			RedirectURL: cfg.RedirectUrl,
			Scopes:      cfg.Scopes,
		}
		h.oauthCfg[k] = oauthCfg
	}

}

func GetExternalUserProfileID(provider string, login string) string {
	return fmt.Sprintf("%v-%v", provider, login)
}

func (h *APIHandler) saveExternalUser(provider string, login string, payload interface{}, user *core.User) (*core.ExternalUserProfile, error) {
	obj := core.ExternalUserProfile{}
	obj.ID = GetExternalUserProfileID(provider, login)
	obj.UserID = user.ID
	obj.AuthProvider = provider
	obj.Login = login
	obj.Payload = payload

	return &obj, orm.Save(nil, &obj)
}

func (h *APIHandler) getExternalUserBy(provider string, login string) *core.ExternalUserProfile {

	obj := core.ExternalUserProfile{}
	obj.ID = GetExternalUserProfileID(provider, login)

	exists, err := orm.Get(&obj)
	if exists && err == nil && obj.Login == login {
		return &obj
	}
	return nil
}

var apiHandler = APIHandler{
	cCache: ccache.Layered(ccache.Configure().MaxSize(10000).ItemsToPrune(100)),
}
