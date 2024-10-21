/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rag

import (
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}
	api.HandleAPIMethod(api.POST, "/search/_suggest", handler.suggest)
	api.HandleAPIMethod(api.POST, "/search/_recommend", handler.recommend)
	api.HandleAPIMethod(api.POST, "/search/_search", handler.search)

	api.HandleAPIMethod(api.POST, "/chat/_history", handler.search)
	api.HandleAPIMethod(api.POST, "/chat/:id/_open", handler.search)
	api.HandleAPIMethod(api.POST, "/chat/:id/_clear", handler.search)
	api.HandleAPIMethod(api.POST, "/chat/:id/_close", handler.search)
	api.HandleAPIMethod(api.POST, "/chat/:id/_history", handler.search)
}