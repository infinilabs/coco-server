/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	"net/http"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/security/filter"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

type APIHandler struct {
	api.Handler
}

const Category = "coco"
const Resource = "system"

func init() {

	readPermission := security.GetSimplePermission(Category, Resource, string(security.Read))
	updatePermission := security.GetSimplePermission(Category, Resource, string(security.Update))

	security.GetOrInitPermissionKey(readPermission)
	security.GetOrInitPermissionKey(updatePermission)

	handler := APIHandler{}
	api.HandleUIMethod(api.GET, "/provider/_info", handler.providerInfo, api.AllowPublicAccess())
	api.HandleUIMethod(api.POST, "/setup/_initialize", handler.setupServer, api.AllowPublicAccess())

	api.HandleUIMethod(api.OPTIONS, "/settings", handler.getServerSettings, api.RequirePermission(readPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.GET, "/settings", handler.getServerSettings, api.RequirePermission(readPermission), api.Feature(filter.FeatureCORS), api.Feature(filter.FeatureMaskSensitiveField),
		api.Feature(filter.FeatureRemoveSensitiveField))
	api.HandleUIMethod(api.PUT, "/settings", handler.updateServerSettings, api.RequirePermission(updatePermission))

	//list all icons for connectors
	api.HandleUIMethod(api.GET, "/icons/list", handler.getIcons, api.AllowPublicAccess())
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

	isSetup := isAlreadyDoneSetup()
	if !isSetup {
		output["setup_required"] = true
	}

	output["health"] = obj

	if global.Env().SystemConfig.WebAppConfig.Security.Managed {
		output["managed"] = true
		if info.ServerInfo.Provider.AuthProvider.SSO.URL == "" {
			panic("sso url can't be nil")
		}
	}
	stats := util.MapStr{}
	claims, _ := core.ValidateLogin(req)
	if claims != nil {
		count, err := common.CountAssistants()
		if err != nil {
			panic(err)
		}
		stats["assistant_count"] = count
	}
	output["stats"] = stats

	h.WriteJSON(w, output, 200)
}
