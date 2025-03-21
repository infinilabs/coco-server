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
		log.Error(err)
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	oldAppConfig := common.AppConfig()
	if appConfig.LLMConfig != nil {
		//merge settings
		llmCfg := common.LLMConfig{}
		err := mergeSettings(oldAppConfig.LLMConfig, appConfig.LLMConfig, &llmCfg)
		if err != nil {
			log.Error(err)
			h.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		oldAppConfig.LLMConfig = &llmCfg
	}
	if appConfig.ServerInfo != nil {
		//merge settings
		serverCfg := common.ServerInfo{}
		err := mergeSettings(oldAppConfig.ServerInfo, appConfig.ServerInfo, &serverCfg)
		if err != nil {
			log.Error(err)
			h.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		oldAppConfig.ServerInfo = &serverCfg
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
