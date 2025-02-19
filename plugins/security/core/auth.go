// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Console is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"fmt"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"net/http"
)

// Handler is the object of http handler
type Handler struct {
	api.Handler
}

func (handler Handler) WriteError(w http.ResponseWriter, err interface{}, status int) {
	if v, ok := err.(error); ok {
		handler.Handler.WriteError(w, v.Error(), status)
		return
	}
	handler.Handler.WriteError(w, fmt.Sprintf("%v", err), status)
}

func RequireLogin(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		if api.IsAuthEnable() {
			claims, err := ValidateLogin(r)
			if err != nil {

				w.WriteHeader(http.StatusUnauthorized)
				w.Write(util.MustToJSONBytes(api.PrepareErrorJson("Unauthorized access.", http.StatusUnauthorized)))
				return
			}
			r = r.WithContext(NewUserContext(r.Context(), claims))
		}

		h(w, r, ps)
	}
}

func (handler Handler) RequirePermission(h httprouter.Handle, permissions ...string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		if global.Env().SetupRequired() {
			return
		}

		if api.IsAuthEnable() {
			claims, err := ValidateLogin(r)
			if err != nil {
				handler.WriteError(w, err.Error(), http.StatusUnauthorized)
				return
			}
			err = ValidatePermission(claims, permissions)
			if err != nil {
				handler.WriteError(w, err.Error(), http.StatusForbidden)
				return
			}
			r = r.WithContext(NewUserContext(r.Context(), claims))
		}

		h(w, r, ps)
	}
}

//func (handler Handler) RequireClusterPermission(h httprouter.Handle, permissions ...string) httprouter.Handle {
//	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
//
//		if api.IsAuthEnable() {
//			id := ps.ByName("id")
//			claims, err := security.ValidateLogin(r.Header.Get("Authorization"))
//			if err != nil {
//				handler.WriteError(w, err.Error(), http.StatusUnauthorized)
//				return
//			}
//			r = r.WithContext(security.NewUserContext(r.Context(), claims))
//			hasAllPrivilege, clusterIDs := security.GetCurrentUserCluster(r)
//			if !hasAllPrivilege && (len(clusterIDs) == 0 || !util.StringInArray(clusterIDs, id)) {
//				w.WriteHeader(http.StatusForbidden)
//				w.Write([]byte(http.StatusText(http.StatusForbidden)))
//				return
//			}
//		}
//
//		h(w, r, ps)
//	}
//}

func (handler Handler) GetCurrentUser(req *http.Request) string {
	//if api.IsAuthEnable() {
	claims, ok := req.Context().Value("user").(*UserClaims)
	if ok {
		return claims.Login
	}
	//}
	return ""
}
