/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package profile

import (
	"fmt"
	"net/http"

	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
)

func init() {
	handler := APIHandler{}

	api.HandleUIMethod(api.GET, "/auth/sso_success", handler.ssoSuccess)
}

type APIHandler struct {
	api.Handler
}

func (h *APIHandler) ssoSuccess(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	payload := h.MustGetParameter(w, req, "payload")
	json := util.UrlDecode(payload)
	obj := util.MapStr{}
	util.MustFromJSONBytes([]byte(json), &obj)

	v, err := obj.GetValue("code")
	if err != nil {
		panic(err)
	}

	requestId, err := obj.GetValue("request_id")
	if err != nil {
		panic(err)
	}

	// Generate the HTML response with auto-redirect
	htmlContent := fmt.Sprintf(
		`<html>
        <head>
            <title>SSO Success</title>
            <meta http-equiv="refresh" content="5;url=coco://oauth_callback?code=%v&request_id=%v&provider=coco-cloud">
        </head>
        <body>
            <p>In order to continue, please click the link below if you are not redirected automatically within 5 seconds:</p>
            <a href="coco://oauth_callback?code=%v&request_id=%v&provider=coco-cloud">Launch Coco AI</a>
        </body>
    </html>`,
		v, requestId, v, requestId,
	)

	// Write the HTML content to the response with the appropriate Content-Type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(htmlContent))
}
