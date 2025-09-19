/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
	"net/http"
)

type ServerSettings struct {
}

func (h *APIHandler) getServerSettings(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	appConfig := common.AppConfig()
	h.WriteJSON(w, appConfig, http.StatusOK)
}

func (h *APIHandler) updateServerSettings(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	appConfig := common.Config{}
	if err := h.DecodeJSON(req, &appConfig); err != nil {
		_ = log.Error(err)
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	oldAppConfig := common.AppConfig()
	if appConfig.ServerInfo != nil {
		//merge settings
		serverCfg := common.ServerInfo{}
		err := mergeSettings(oldAppConfig.ServerInfo, appConfig.ServerInfo, &serverCfg)
		if err != nil {
			_ = log.Error(err)
			h.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		oldAppConfig.ServerInfo = &serverCfg
	}
	if appConfig.AppSettings != nil {
		//merge settings
		appSettings := common.AppSettings{}
		err := mergeSettings(oldAppConfig.AppSettings, appConfig.AppSettings, &appSettings)
		if err != nil {
			_ = log.Error(err)
			h.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		oldAppConfig.AppSettings = &appSettings
	}
	if appConfig.SearchSettings != nil {
		//merge settings
		searchSettings := common.SearchSettings{}
		err := mergeSettings(oldAppConfig.SearchSettings, appConfig.SearchSettings, &searchSettings)
		if err != nil {
			_ = log.Error(err)
			h.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		oldAppConfig.SearchSettings = &searchSettings
	}
	common.SetAppConfig(&oldAppConfig)
	h.WriteAckOKJSON(w)
}

func mergeSettings(old, new, merged interface{}) error {
	newSettings := util.MapStr{}
	buf := util.MustToJSONBytes(new)
	util.MustFromJSONBytes(buf, &newSettings)
	buf = util.MustToJSONBytes(old)
	oldSettings := util.MapStr{}
	util.MustFromJSONBytes(buf, &oldSettings)
	err := util.MergeFields(oldSettings, newSettings, true)
	if err != nil {
		return err
	}
	buf = util.MustToJSONBytes(oldSettings)
	util.MustFromJSONBytes(buf, merged)
	return nil
}
