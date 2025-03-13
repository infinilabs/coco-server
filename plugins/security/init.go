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

package security

import (
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	api.Handler
}

var apiHandler = APIHandler{}

func init() {

	//login page
	api.HandleUIMethod(api.GET, "/login/", apiHandler.LoginPage)
	api.HandleUIMethod(api.GET, "/login/success", apiHandler.LoginSuccess)

	api.HandleUIMethod(api.POST, "/account/login", apiHandler.Login)
	api.HandleUIMethod(api.POST, "/account/logout", apiHandler.Logout, api.OptionLogin())

	api.HandleUIMethod(api.GET, "/account/profile", apiHandler.Profile, api.RequireLogin())
	api.HandleUIMethod(api.PUT, "/account/password", apiHandler.UpdatePassword, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/auth/request_access_token", apiHandler.RequestAccessToken, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/auth/access_token/_cat", apiHandler.CatAccessToken, api.RequireLogin())
	api.HandleUIMethod(api.DELETE, "/auth/access_token/:token_id", apiHandler.DeleteAccessToken, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/auth/access_token/:token_id/_rename", apiHandler.RenameAccessToken, api.RequireLogin())

}
