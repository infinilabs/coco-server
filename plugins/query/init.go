/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package query

import (
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}
	api.HandleAPIMethod(api.POST, "/query/_suggest", handler.suggest)
	api.HandleAPIMethod(api.POST, "/query/_recommend", handler.recommend)
	api.HandleAPIMethod(api.POST, "/query/_search", handler.search)

}