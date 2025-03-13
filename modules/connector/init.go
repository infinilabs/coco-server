/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connector

import (
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}

	api.HandleUIMethod(api.POST, "/connector/", handler.create, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/connector/:id", handler.get, api.RequireLogin())
	api.HandleUIMethod(api.PUT, "/connector/:id", handler.update, api.RequireLogin())
	api.HandleUIMethod(api.DELETE, "/connector/:id", handler.delete, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/connector/_search", handler.search, api.RequireLogin())

	api.HandleUIMethod(api.POST, "/datasource/", handler.createDatasource, api.RequireLogin())
	api.HandleUIMethod(api.DELETE, "/datasource/:id", handler.deleteDatasource, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/datasource/:id", handler.getDatasource, api.RequireLogin())
	api.HandleUIMethod(api.PUT, "/datasource/:id", handler.updateDatasource, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/datasource/_search", handler.searchDatasource, api.RequireLogin())

	//shortcut to indexing docs into this datasource
	api.HandleUIMethod(api.POST, "/datasource/:id/_doc", handler.createDocInDatasource, api.RequireLogin())

}
