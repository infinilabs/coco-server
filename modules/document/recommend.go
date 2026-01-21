/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"net/http"

	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
)

func (h *APIHandler) recommend(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	// Parse the request body
	var recommendReq core.RecommendRequest
	if err := h.DecodeJSON(req, &recommendReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tag := ps.ByName("tag") //eg: hot

	if h.recommendConfigs != nil {
		v, ok := h.recommendConfigs[tag]
		if ok {
			h.WriteJSON(w, v, 200)
			return
		}

		v, ok = h.recommendConfigs["default"]
		if ok {
			h.WriteJSON(w, v, 200)
			return
		}
	}

	response := core.RecommendResponse{}
	h.WriteJSON(w, response, 200)

}
