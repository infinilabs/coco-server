// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Console is offered under the GNU Affero General Public License v3.0
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

/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	log "github.com/cihub/seelog"
	"github.com/golang-jwt/jwt"
	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
	"net/http"
	"time"
)

func GenerateJWTAccessToken(provider string, login string, user *core.User) (map[string]interface{}, error) {

	var data map[string]interface{}

	token1 := jwt.NewWithClaims(jwt.SigningMethodHS256, core.UserClaims{
		ShortUser: &core.ShortUser{
			Provider: provider,
			Login:    login,
			UserId:   user.ID,
			Roles:    []string{},
		},
		RegisteredClaims: &jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	})

	tokenString, err := token1.SignedString([]byte(core.Secret))
	if tokenString == "" || err != nil {
		return nil, errors.Errorf("failed to generate access_token for user: %v", user)
	}

	token := Token{ExpireIn: time.Now().Unix() + 86400}
	SetUserToken(user.ID, token)

	data = util.MapStr{
		"access_token": tokenString,
		"username":     login,
		"id":           user.ID,
		"expire_in":    86400,
	}

	data["status"] = "ok"

	return data, err

}

func (h *APIHandler) RequestAccessToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	//user already login
	reqUser, err := core.UserFromContext(req.Context())
	if reqUser == nil || err != nil {
		panic(err)
	}

	username := reqUser.Login
	userid := reqUser.UserId
	provider := "access_token"

	res := util.MapStr{}
	accessToken := util.GetUUID() + util.GenerateRandomString(64)
	res["access_token"] = accessToken
	expiredAT := time.Now().Add(365 * 24 * time.Hour).Unix()
	res["expire_in"] = expiredAT

	newPayload := util.MapStr{}
	newPayload["access_token"] = accessToken
	newPayload["provider"] = provider
	newPayload["login"] = username
	newPayload["userid"] = userid
	newPayload["expire_in"] = expiredAT

	log.Trace("generate and save access_token:", util.MustToJSON(newPayload))

	// save access token to store
	err = kv.AddValue("access_token", []byte(accessToken), util.MustToJSONBytes(newPayload))
	if err != nil {
		panic(err)
	}
	h.WriteJSON(w, res, 200)
}
