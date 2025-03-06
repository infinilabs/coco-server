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
	api.HandleUIMethod(api.POST, "/document/", handler.createDoc, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/document/:doc_id", handler.getDoc, api.RequireLogin())
	api.HandleUIMethod(api.PUT, "/document/:doc_id", handler.updateDoc, api.RequireLogin())
	api.HandleUIMethod(api.DELETE, "/document/:doc_id", handler.deleteDoc, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/document/_search", handler.searchDocs, api.RequireLogin())
	api.HandleUIMethod(api.DELETE, "/document/", handler.batchDeleteDoc, api.RequireLogin())
}
