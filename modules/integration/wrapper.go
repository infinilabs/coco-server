/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package integration

import (
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	"io"
	"net/http"
	"strings"
)

var ver = util.GetUUID()

func (h *APIHandler) widgetWrapper(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	integrationID := ps.MustGetParameter("id")
	obj := common.Integration{}
	obj.ID = integrationID
	ctx := orm.NewContextWithParent(req.Context()).DirectReadAccess()
	ctx.Set(orm.ReadPermissionCheckingScope, security.PermissionScopePublic)

	exists, err := orm.GetV2(ctx, &obj)
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

	var str string

	switch obj.Type {
	//'embedded', 'floating', 'all', 'fullscreen'
	case "fullscreen":
		if h.fullscreenWrapperTemplate == nil {
			panic("invalid wrapper template")
		}

		info := common.AppConfig()
		token := obj.Token

		str = h.fullscreenWrapperTemplate.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
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
		break
	default:

		if h.searchBoxWrapperTemplate == nil {
			panic("invalid wrapper template")
		}

		info := common.AppConfig()
		token := obj.Token

		str = h.searchBoxWrapperTemplate.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
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
	}
	h.WriteJavascriptHeader(w)
	_, _ = h.Write(w, []byte(str))
	h.WriteHeader(w, 200)

}
