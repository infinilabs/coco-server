/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package search

import (
	"infini.sh/coco/plugins/security/core"
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}

	api.HandleUIMethod(api.GET, "/query/_suggest", core.RequireLogin(handler.suggest))
	api.HandleUIMethod(api.GET, "/query/_recommend", core.RequireLogin(handler.recommend))
	api.HandleUIMethod(api.GET, "/query/_search", core.RequireLogin(handler.search))

}
