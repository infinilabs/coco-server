/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package integration

import "infini.sh/framework/core/api"

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}

	api.HandleUIMethod(api.POST, "/integration/", handler.create, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/integration/:id", handler.get, api.RequireLogin())
	api.HandleUIMethod(api.PUT, "/integration/:id", handler.update, api.RequireLogin())
	api.HandleUIMethod(api.DELETE, "/integration/:id", handler.delete, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/integration/_search", handler.search, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/integration/_search", handler.search, api.RequireLogin())
}
