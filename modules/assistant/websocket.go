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

package assistant

import (
	"errors"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/kv"
	"net/http"
)

func (h APIHandler) GetUserWebsocketID(req *http.Request) (string, error) {
	//get websocket by user's id
	claims, err := core.ValidateLogin(req)
	if err != nil {
		return "", err
	}
	if claims != nil {

		websocketID := h.GetHeader(req, "WEBSOCKET-SESSION-ID", "")
		if websocketID != "" {
			log.Trace("get websocket session id from request header: ", websocketID)
			return websocketID, err
		}

		if claims.UserId != "" {
			v, err := kv.GetValue(common.WEBSOCKET_USER_SESSION, []byte(claims.UserId))
			if err != nil {
				return "", err
			}
			log.Trace("get websocket session id from kv: ", string(v))
			return string(v), nil
		}
	}
	return "", errors.New("not found")
}
