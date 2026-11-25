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

	// 搜索设置 · 热搜与推荐: the feature switch turns recommendations off
	// entirely (stored topics and the coco.yml fallback both) without
	// wiping the configuration — re-enabling restores it.
	if searchSettingsSnapshot().RecommendTopicsDisabled {
		h.WriteJSON(w, core.RecommendResponse{}, 200)
		return
	}

	// 搜索设置 · 热搜与推荐: operator-managed topics take precedence over the
	// static `recommend:` block in coco.yml — empty keeps the file fallback
	// (historic behavior).
	if topics := searchSettingsSnapshot().RecommendTopics; len(topics) > 0 {
		response := core.RecommendResponse{Total: len(topics)}
		for i, topic := range topics {
			response.Recommendations = append(response.Recommendations, core.RecommendEntityCard{
				Title: topic,
				Score: float64(len(topics) - i),
			})
		}
		h.WriteJSON(w, response, 200)
		return
	}

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
