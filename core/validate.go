/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */
package core

import (
	"net/http"

	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
)

const (
	HeaderIntegrationID = "APP-INTEGRATION-ID"
)

func init() {
	// priority 50: runs after session_token (10) and bearer_token (20),
	// so integration guest auth only activates when no real login is present.
	security.RegisterHTTPAuthFilterProviderWithPriority("app_integration_id", ValidateLoginByIntegrationHeader, 50)
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

			claims.Provider = security.DefaultNativeAuthBackend
			claims.Login = cfg.Guest.RunAs
			claims.UserID = cfg.Guest.RunAs
			// Mark this session as integration-authenticated so downstream handlers
			// (e.g. /account/profile, enterprise tenant filter) can distinguish a
			// real native-backend login from an integration guest run-as session.
			claims.Set(UserSessionInfoKeyIntegration, integrationID)
			claims.Permissions = security.GetAllPermissionsForUser(claims.UserSessionInfo)
			//claims.Permissions = security.MustGetPermissionKeysByUser(r.Context(), cfg.Guest.RunAs)
			//log.Info("integration:", integrationID, ", run as:", cfg.Guest.RunAs, ",permissions:", claims.Permissions)
			return claims, nil
		}
	}

	return nil, errors.Error("invalid claims")
}
