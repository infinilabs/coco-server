/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connector

import (
	"infini.sh/coco/plugins/security/core"
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}

	api.HandleUIMethod(api.POST, "/connector/", core.RequireLogin(handler.create))
	api.HandleUIMethod(api.GET, "/connector/:id", core.RequireLogin(handler.get))
	api.HandleUIMethod(api.PUT, "/connector/:id", core.RequireLogin(handler.update))
	api.HandleUIMethod(api.DELETE, "/connector/:id", core.RequireLogin(handler.delete))
	api.HandleUIMethod(api.GET, "/connector/_search", core.RequireLogin(handler.search))

	api.HandleUIMethod(api.POST, "/datasource/", core.RequireLogin(handler.createDatasource))
	api.HandleUIMethod(api.DELETE, "/datasource/:id", core.RequireLogin(handler.deleteDatasource))
	api.HandleUIMethod(api.GET, "/datasource/:id", core.RequireLogin(handler.getDatasource))
	api.HandleUIMethod(api.PUT, "/datasource/:id", core.RequireLogin(handler.updateDatasource))
	api.HandleUIMethod(api.GET, "/datasource/_search", core.RequireLogin(handler.searchDatasource))

}
