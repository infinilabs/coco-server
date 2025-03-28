// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
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

package system

import (
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"net/http"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}
	api.HandleUIMethod(api.GET, "/provider/_info", handler.providerInfo, api.AllowPublicAccess())
	api.HandleUIMethod(api.POST, "/setup/_initialize", handler.setupServer, api.AllowPublicAccess())
	api.HandleUIMethod(api.GET, "/settings", handler.getServerSettings, api.RequireLogin())
	api.HandleUIMethod(api.PUT, "/settings", handler.updateServerSettings, api.RequireLogin())
}

func (h *APIHandler) providerInfo(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	info := common.AppConfig()

	json := util.MustToJSONBytes(&info.ServerInfo)
	output := util.MapStr{}
	err := util.FromJSONBytes(json, &output)
	if err != nil {
		panic(err)
	}

	overallHealthType := global.Env().GetOverallHealth()
	obj := util.MapStr{
		"status": overallHealthType.ToString(),
	}

	services := global.Env().GetServicesHealth()
	if len(services) > 0 {
		obj["services"] = services
	}

	isSetup := checkSetupStatus()
	output["setup_required"] = !isSetup
	output["health"] = obj

	if info.ServerInfo != nil && info.ServerInfo.Managed {
		if info.ServerInfo.Provider.AuthProvider.SSO.URL == "" {
			panic("sso url can't be nil")
		}
	}

	h.WriteJSON(w, output, 200)
}
