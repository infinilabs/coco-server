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
	httprouter "infini.sh/framework/core/api/router"
	"net/http"
	"strings"
)

func (h APIHandler) LoginPage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	provider := h.MustGetParameter(w, req, "provider")
	product := h.MustGetParameter(w, req, "product")
	requestID := h.MustGetParameter(w, req, "request_id")

	// Generate HTML form with password input
	var builder strings.Builder
	builder.WriteString(`<html>
		<head><title>Login</title></head>
		<body>
			<h1>Login Coco Server</h1>
			<form method="post" action="/account/login?provider=`)
	builder.WriteString(provider)
	builder.WriteString("&product=")
	builder.WriteString(product)
	builder.WriteString("&request_id=")
	builder.WriteString(requestID)
	builder.WriteString(`">
				<div style="margin: 10px 0">
					<label for="password">Password:</label>
					<input type="password" id="password" name="password" required>
				</div>
				<button type="submit">Login</button>
			</form>
		</body>
	</html>`)

	// Write the HTML response
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(builder.String()))
}
