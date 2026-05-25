/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	"net/http"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/service"
	"infini.sh/coco/modules/common"
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
	api.HandleUIMethod(api.POST, "/setup/_initialize", handler.setupInitialize, api.AllowPublicAccess())
	api.HandleUIMethod(api.POST, "/setup/_initialize/default_model", handler.setupInitializeDefaultModel, api.RequireLogin(), api.RequirePermission(updatePermission))

	api.HandleUIMethod(api.OPTIONS, "/settings", handler.getServerSettings, api.RequirePermission(readPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.GET, "/settings", handler.getServerSettings, api.RequirePermission(readPermission), api.Feature(core.FeatureCORS), api.Feature(core.FeatureMaskSensitiveField),
		api.Feature(core.FeatureRemoveSensitiveField))
	api.HandleUIMethod(api.PUT, "/settings", handler.updateServerSettings, api.RequirePermission(updatePermission))

	//list all icons for connectors
	api.HandleUIMethod(api.GET, "/icons/list", handler.getIcons, api.AllowPublicAccess())

	api.RegisterAppSetting("setup_required", func() interface{} {
		return !isSetupDone()
	})

	api.RegisterAppSetting("search_settings", func() interface{} {
		info := common.AppConfig()
		return info.SearchSettings
	})

}

func (h *APIHandler) providerInfo(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	output := util.MapStr{}

	info := common.AppConfig()
	json := util.MustToJSONBytes(&info.ServerInfo)
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

	output["health"] = obj

	stats := util.MapStr{}
	claims, _ := security.ValidateLogin(w, req)
	if claims != nil {
		count, err := service.CountAssistants()
		if err != nil {
			panic(err)
		}
		stats["assistant_count"] = count
	}
	output["stats"] = stats

	h.WriteJSON(w, output, 200)
}
