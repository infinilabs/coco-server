/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"net/http"

	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/security"
)

// AuthFilter resolves the request's user via the registered auth providers
// (bearer token / session cookie / api token / app integration) and attaches
// the resulting UserSessionInfo to the request context so downstream filters
// like PermissionFilter can read it.
type AuthFilter struct {
	api.Handler
}

func (f *AuthFilter) GetPriority() int {
	// Lower priority than PermissionFilter (500) so this wraps the chain on
	// the outside and runs *before* permission checks.
	return 100
}

func (f *AuthFilter) ApplyFilter(
	method string,
	pattern string,
	options *api.HandlerOptions,
	next httprouter.Handle,
) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		claims, err := security.ValidateLogin(w, r)
		if err == nil && claims != nil {
			ctx := security.AddUserToContext(r.Context(), claims)
			r = r.WithContext(ctx)
		}
		next(w, r, ps)
	}
}

func init() {
	api.RegisterUIFilter(&AuthFilter{})
}
