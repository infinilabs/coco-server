/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package datasource

import (
	"context"
	"net/http"
	"time"

	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
)

const testConnectionTimeout = 15 * time.Second

type testConnectionRequest struct {
	ConnectorID string                 `json:"connector_id"`
	Config      map[string]interface{} `json:"config"`
}

func (h *APIHandler) testConnection(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var body testConnectionRequest
	h.MustDecodeJSON(req, &body)

	if body.ConnectorID == "" {
		h.WriteJSON(w, util.MapStr{
			"success": false,
			"error":   "connector_id is required",
		}, http.StatusBadRequest)
		return
	}

	tester, ok := core.GetConnectionTester(body.ConnectorID)
	if !ok {
		h.WriteJSON(w, util.MapStr{
			"success": false,
			"error":   "test connection is not supported for this connector",
		}, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(req.Context(), testConnectionTimeout)
	defer cancel()

	err := tester.TestConnection(ctx, body.Config)
	if err != nil {
		h.WriteJSON(w, util.MapStr{
			"success": false,
			"error":   err.Error(),
		}, http.StatusOK)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"success": true,
	}, http.StatusOK)
}
