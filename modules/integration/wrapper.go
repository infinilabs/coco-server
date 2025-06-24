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

package integration

import (
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"io"
	"net/http"
	"strings"
)

var ver = util.GetUUID()

func (h *APIHandler) widgetWrapper(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	if h.wrapperTemplate == nil {
		panic("invalid wrapper template")
	}

	integrationID := ps.MustGetParameter("id")
	obj := common.Integration{}
	obj.ID = integrationID

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    integrationID,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	if !obj.Enabled {
		h.WriteJavascriptHeader(w)
		h.WriteHeader(w, 200)
		return
	}

	info := common.AppConfig()
	token := obj.Token

	str := h.wrapperTemplate.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		switch tag {
		case "ID":
			return w.Write([]byte(integrationID))
		case "VER":
			return w.Write([]byte(ver))
		case "ENDPOINT":
			endpoint := strings.TrimRight(info.ServerInfo.Endpoint, "/")
			return w.Write([]byte(endpoint))
		case "TOKEN":
			endpoint := token
			return w.Write([]byte(endpoint))
		}
		return -1, nil
	})
	h.WriteJavascriptHeader(w)
	_, _ = h.Write(w, []byte(str))
	h.WriteHeader(w, 200)
}
