/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package indexing

import (
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}

	//for internal document management, security should be enabled
	api.HandleAPIMethod(api.POST, "/document/", handler.createDoc)
	api.HandleAPIMethod(api.GET, "/document/:doc_id", handler.getDoc)
	api.HandleAPIMethod(api.PUT, "/document/:doc_id", handler.updateDoc)
	api.HandleAPIMethod(api.DELETE, "/document/:doc_id", handler.deleteDoc)
	api.HandleAPIMethod(api.GET, "/document/_search", handler.searchDocs)
}
