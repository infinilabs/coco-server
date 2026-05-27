/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */
package core

import (
	"net/http"

	"infini.sh/framework/core/api"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
)

const (
	HeaderIntegrationID = "APP-INTEGRATION-ID"
)

func init() {
	security.RegisterHTTPAuthFilterProvider("app_integration_id", ValidateLoginByIntegrationHeader)
}

func InternalGetIntegration(id string) (*Integration, error) {
	obj := Integration{}
	obj.ID = id
	ctx := orm.NewContext()
	ctx.DirectReadAccess()

	ctx.PermissionScope(security.PermissionScopePlatform)

	exists, err := orm.GetV2(ctx, &obj)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("integration not found")
	}
	return &obj, nil
}

func ValidateLoginByIntegrationHeader(w http.ResponseWriter, r *http.Request) (claims *security.UserClaims, err error) {
	integrationID := r.Header.Get(HeaderIntegrationID)

	if integrationID == "" {
		return nil, errors.Error("api integration not found")
	}

	// Workaround: security.ValidateLogin iterates auth providers via
	// sync.Map.Range whose order is non-deterministic. If this provider runs
	// before session_token / bearer_token and the request already carries a
	// valid session or Bearer token, returning nil here causes Range to
	// continue and lets the higher-priority provider identify the real user,
	// preventing an already-logged-in user from being treated as a guest.
	//
	// TODO: the framework's HTTP auth backend should support a priority/order
	// mechanism so that session_token and bearer_token always take precedence
	// over integration auth without requiring this workaround.
	if _, sessToken := api.GetSession(w, r, security.UserAccessTokenSessionName); sessToken != nil {
		return nil, errors.Error("session present, skipping integration auth")
	}


	if r.Header.Get("Authorization") != "" {
		return nil, errors.Error("bearer token present, skipping integration auth")
	}

	cfg, _ := InternalGetIntegration(integrationID)
	if cfg != nil {
		if cfg.Guest.Enabled && cfg.Guest.RunAs != "" {

			claims = security.NewUserClaims()
			claims.SetUserID(cfg.Guest.RunAs)

			claims.Provider = ProviderIntegration
			claims.Login = cfg.Guest.RunAs
			claims.UserID = cfg.Guest.RunAs
			claims.Permissions = security.GetAllPermissionsForUser(claims.UserSessionInfo)
			//claims.Permissions = security.MustGetPermissionKeysByUser(r.Context(), cfg.Guest.RunAs)
			//log.Info("integration:", integrationID, ", run as:", cfg.Guest.RunAs, ",permissions:", claims.Permissions)
			return claims, nil
		}
	}

	return nil, errors.Error("invalid claims")
}
