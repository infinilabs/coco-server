/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	"fmt"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
	"net/http"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}
	api.HandleUIMethod(api.GET, "/auth/sso_success", handler.ssoSuccess)
}

func (h *APIHandler) ssoSuccess(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	payload := h.MustGetParameter(w, req, "payload")
	json := util.UrlDecode(payload)
	//log.Error(json)
	//id := ps.MustGetParameter("id")
	obj := util.MapStr{}
	util.MustFromJSONBytes([]byte(json), &obj)

	v, err := obj.GetValue("code")
	if err != nil {
		panic(err)
	}

	h.WriteBytes(w, []byte(fmt.Sprintf("<a href=coco://oauth_callback?code=%v&provider=coco-cloud>In order to continue, please click here to launch Coco AI</a>", v)), 200)

	//obj := common.Connector{}
	//obj.ID = id
	//
	//exists, err := orm.Get(&obj)
	//if !exists || err != nil {
	//	h.WriteJSON(w, util.MapStr{
	//		"_id":   id,
	//		"found": false,
	//	}, http.StatusNotFound)
	//	return
	//}
	//
	//h.WriteJSON(w, util.MapStr{
	//	"found":   true,
	//	"_id":     id,
	//	"_source": obj,
	//}, 200)
}
