// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package security

import (
	"fmt"
	httprouter "infini.sh/framework/core/api/router"
	"net/http"
)

func (h *APIHandler) LoginSuccess(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	requestId := h.GetParameter(req, "request_id")
	code := h.GetParameter(req, "code")

	callbackUrl := fmt.Sprintf("coco://oauth_callback?code=%v&request_id=%v&provider=coco-cloud", code, requestId)

	// Generate the HTML response with auto-redirect
	htmlContent := fmt.Sprintf(
		`<html>
        <head>
            <title>Login Success</title>
            <meta http-equiv="refresh" content="5;url=%v">
        </head>
        <body>
            <p>In order to continue, please click the link below if you are not redirected automatically within 5 seconds:</p>
            <a href="%v">Launch Coco AI</a>


<p>
If the redirect doesn’t work, you can copy the following URL and paste it into the Connect settings window in Coco AI.
<pre>
				%v
			</pre>
</p>
        </body>
    </html>`,
		callbackUrl, callbackUrl, callbackUrl,
	)

	// Write the HTML content to the response with the appropriate Content-Type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(htmlContent))
}
