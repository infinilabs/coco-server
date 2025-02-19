/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package indexing

import (
	"infini.sh/coco/plugins/security/core"
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}

	//for internal document management, security should be enabled
	api.HandleUIMethod(api.POST, "/document/", core.RequireLogin(handler.createDoc))
	api.HandleUIMethod(api.GET, "/document/:doc_id", core.RequireLogin(handler.getDoc))
	api.HandleUIMethod(api.PUT, "/document/:doc_id", core.RequireLogin(handler.updateDoc))
	api.HandleUIMethod(api.DELETE, "/document/:doc_id", core.RequireLogin(handler.deleteDoc))
	api.HandleUIMethod(api.GET, "/document/_search", core.RequireLogin(handler.searchDocs))
}
