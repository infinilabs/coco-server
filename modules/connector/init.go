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

	api.HandleAPIMethod(api.POST, "/connector/", handler.create)
	api.HandleAPIMethod(api.GET, "/connector/:id", handler.get)
	api.HandleAPIMethod(api.PUT, "/connector/:id", handler.update)
	api.HandleAPIMethod(api.DELETE, "/connector/:id", handler.delete)
	api.HandleAPIMethod(api.GET, "/connector/_search", handler.search)

	api.HandleAPIMethod(api.POST, "/datasource/", handler.createDatasource)
	api.HandleAPIMethod(api.DELETE, "/datasource/:id", handler.deleteDatasource)
	api.HandleAPIMethod(api.GET, "/datasource/:id", handler.getDatasource)
	api.HandleAPIMethod(api.PUT, "/datasource/:id", handler.updateDatasource)
	api.HandleAPIMethod(api.GET, "/datasource/_search", handler.searchDatasource)

}