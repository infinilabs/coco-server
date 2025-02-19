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

/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package api

import (
	"infini.sh/coco/plugins/security/core"
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	core.Handler
	core.Adapter
}

const adapterType = "native"

var apiHandler = APIHandler{Adapter: core.GetAdapter(adapterType)} //TODO handle hard coded

func Init() {

	//api.HandleAPIMethod(api.GET, "/permission/:type", CocoApiHandler.RequireLogin(CocoApiHandler.ListPermission))
	//
	//api.HandleAPIMethod(api.POST, "/role/:type", CocoApiHandler.RequirePermission(CocoApiHandler.CreateRole, enum.RoleAllPermission...))
	//api.HandleAPIMethod(api.GET, "/role/:id", CocoApiHandler.RequirePermission(CocoApiHandler.GetRole, enum.RoleReadPermission...))
	//api.HandleAPIMethod(api.DELETE, "/role/:id", CocoApiHandler.RequirePermission(CocoApiHandler.DeleteRole, enum.RoleAllPermission...))
	//api.HandleAPIMethod(api.PUT, "/role/:id", CocoApiHandler.RequirePermission(CocoApiHandler.UpdateRole, enum.RoleAllPermission...))
	//api.HandleAPIMethod(api.GET, "/role/_search", CocoApiHandler.RequirePermission(CocoApiHandler.SearchRole, enum.RoleReadPermission...))

	//api.HandleAPIMethod(api.POST, "/user", CocoApiHandler.RequirePermission(CocoApiHandler.CreateUser, enum.UserAllPermission...))
	//api.HandleAPIMethod(api.GET, "/user/:id", CocoApiHandler.RequirePermission(CocoApiHandler.GetUser, enum.UserReadPermission...))
	//api.HandleAPIMethod(api.DELETE, "/user/:id", CocoApiHandler.RequirePermission(CocoApiHandler.DeleteUser, enum.UserAllPermission...))
	//api.HandleAPIMethod(api.PUT, "/user/:id", CocoApiHandler.RequirePermission(CocoApiHandler.UpdateUser, enum.UserAllPermission...))
	//api.HandleAPIMethod(api.GET, "/user/_search", CocoApiHandler.RequirePermission(CocoApiHandler.SearchUser, enum.UserReadPermission...))
	//api.HandleAPIMethod(api.PUT, "/user/:id/password", CocoApiHandler.RequirePermission(CocoApiHandler.UpdateUserPassword, enum.UserAllPermission...))

	//api.HandleAPIMethod(api.POST, "/account/login", CocoApiHandler.Login)
	api.HandleUIMethod(api.POST, "/account/logout", apiHandler.Logout)
	api.HandleUIMethod(api.DELETE, "/account/logout", apiHandler.Logout)

	api.HandleUIMethod(api.GET, "/account/profile", core.RequireLogin(apiHandler.Profile))
	//api.HandleAPIMethod(api.PUT, "/account/password", CocoApiHandler.RequireLogin(CocoApiHandler.UpdatePassword))

}
