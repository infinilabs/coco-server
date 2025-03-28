/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package search

import (
	"infini.sh/coco/plugins/security/filter"
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}

	api.HandleUIMethod(api.GET, "/query/_suggest", handler.suggest, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/query/_recommend", handler.recommend, api.RequireLogin())
	api.HandleUIMethod(api.OPTIONS, "/query/_search", handler.search, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.GET, "/query/_search", handler.search, api.RequireLogin(), api.Feature(filter.FeatureCORS))

}
