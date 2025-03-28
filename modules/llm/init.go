/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package llm

import "infini.sh/framework/core/api"

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}

	api.HandleUIMethod(api.POST, "/model_provider/", handler.create, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/model_provider/:id", handler.get, api.RequireLogin())
	api.HandleUIMethod(api.PUT, "/model_provider/:id", handler.update, api.RequireLogin())
	api.HandleUIMethod(api.DELETE, "/model_provider/:id", handler.delete, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/model_provider/_search", handler.search, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/model_provider/_search", handler.search, api.RequireLogin())
}
