/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */
package core

import (
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"net/http"
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

	cfg, _ := InternalGetIntegration(integrationID)
	if cfg != nil {
		if cfg.Guest.Enabled && cfg.Guest.RunAs != "" {

			claims = security.NewUserClaims()
			claims.SetUserID(cfg.Guest.RunAs)

			claims.Provider = ProviderIntegration
			claims.Login = cfg.Guest.RunAs
			claims.UserID = cfg.Guest.RunAs
			claims.UserAssignedPermission = security.GetUserPermissions(claims.UserSessionInfo)
			return claims, nil
		}
	}

	return nil, errors.Error("invalid claims")
}
