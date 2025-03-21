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
	"fmt"
	"github.com/buger/jsonparser"
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

const (
	KVAccessTokenIDBucket = "access_token_id"
	KVUserTokenBucket     = "user_token_id"
)

func (h *APIHandler) RequestAccessToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	//user already login
	reqUser, err := core.UserFromContext(req.Context())
	if reqUser == nil || err != nil {
		panic(err)
	}
	tokenIDs, err := getTokenIDs(reqUser.UserId)
	if err != nil {
		panic(err)
	}

	//if len(tokenIDs) >= 5 {
	//	h.WriteError(w, "Access token limit exceeded. Maximum allowed: 5.", 400)
	//	return
	//}

	reqBody := struct {
		Name string `json:"name"` //custom access token name
	}{}
	err = h.DecodeJSON(req, &reqBody)
	if err != nil {
		panic(err)
	}
	if reqBody.Name == "" {
		reqBody.Name = GenerateApiTokenName("")
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
	tokenID := util.GetUUID()
	newPayload["id"] = tokenID
	newPayload["access_token"] = accessToken
	newPayload["provider"] = provider
	newPayload["login"] = username
	newPayload["userid"] = userid
	newPayload["expire_in"] = expiredAT
	newPayload["name"] = reqBody.Name

	log.Trace("generate and save access_token:", util.MustToJSON(newPayload))

	// save access token to store
	err = kv.AddValue(core.KVAccessTokenBucket, []byte(accessToken), util.MustToJSONBytes(newPayload))
	if err != nil {
		panic(err)
	}
	// save relationship between token and token id
	err = kv.AddValue(KVAccessTokenIDBucket, []byte(tokenID), []byte(accessToken))
	if err != nil {
		log.Error("failed to save access_token_id:", err)
	}
	// save relationship between user and token id
	tokenIDs[tokenID] = struct{}{}
	err = kv.AddValue(KVUserTokenBucket, []byte(userid), util.MustToJSONBytes(tokenIDs))
	if err != nil {
		panic(err)
	}

	h.WriteJSON(w, res, 200)
}

func getTokenIDs(userID string) (map[string]struct{}, error) {
	tokenIDsBytes, err := kv.GetValue(KVUserTokenBucket, []byte(userID))
	if err != nil {
		return nil, err
	}
	var tokenIDs = make(map[string]struct{})
	err = util.FromJSONBytes(tokenIDsBytes, &tokenIDs)
	if err != nil {
		return nil, err
	}
	return tokenIDs, nil
}

func (h *APIHandler) CatAccessToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//check login
	reqUser, err := core.UserFromContext(req.Context())
	if reqUser == nil || err != nil {
		panic(err)
	}

	tokenIDs, err := getTokenIDs(reqUser.UserId)
	if err != nil {
		panic(err)
	}
	accessTokens := make([]map[string]interface{}, 0)
	for tokenID := range tokenIDs {
		accessTokenBytes, err := kv.GetValue(KVAccessTokenIDBucket, []byte(tokenID))
		if err != nil {
			panic(err)
		}
		tokenV, err := kv.GetValue(core.KVAccessTokenBucket, accessTokenBytes)
		if err != nil {
			panic(err)
		}
		var accessToken map[string]interface{}
		util.MustFromJSONBytes(tokenV, &accessToken)
		if strToken, ok := accessToken["access_token"].(string); ok {
			if len(strToken) > 8 {
				accessToken["access_token"] = strToken[0:4] + "***************" + strToken[len(strToken)-4:]
			}
		}
		accessTokens = append(accessTokens, accessToken)
	}

	h.WriteJSON(w, accessTokens, 200)
}

func (h *APIHandler) DeleteAccessToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	reqUser, err := core.UserFromContext(req.Context())
	if reqUser == nil || err != nil {
		panic(err)
	}
	tokenID := ps.ByName("token_id")
	tokenBytes, err := kv.GetValue(KVAccessTokenIDBucket, []byte(tokenID))
	if err != nil {
		panic(err)
	}
	if tokenBytes == nil {
		h.WriteError(w, "token not found", 404)
		return
	}
	tokenV, err := kv.GetValue(core.KVAccessTokenBucket, tokenBytes)
	if err != nil {
		panic(err)
	}
	userID, err := jsonparser.GetString(tokenV, "userid")
	if err != nil {
		panic(err)
	}
	if userID != reqUser.UserId {
		h.WriteError(w, "permission denied", 403)
		return
	}
	err = kv.DeleteKey(core.KVAccessTokenBucket, tokenBytes)
	if err != nil {
		panic(err)
	}
	// delete relationship between token and token id
	err = kv.DeleteKey(KVAccessTokenIDBucket, []byte(tokenID))
	if err != nil {
		panic(err)
	}
	// update relationship between user and token id
	tokenIDs, err := getTokenIDs(userID)
	if err != nil {
		panic(err)
	}
	delete(tokenIDs, tokenID)
	err = kv.AddValue(KVUserTokenBucket, []byte(userID), util.MustToJSONBytes(tokenIDs))
	if err != nil {
		panic(err)
	}

	h.WriteDeletedOKJSON(w, tokenID)
}

func (h *APIHandler) RenameAccessToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	reqUser, err := core.UserFromContext(req.Context())
	if reqUser == nil || err != nil {
		panic(err)
	}
	reqBody := struct {
		Name string `json:"name"` //custom access token name
	}{}
	err = h.DecodeJSON(req, &reqBody)
	if err != nil {
		panic(err)
	}
	if reqBody.Name == "" {
		h.WriteError(w, "name is required", 400)
		return
	}
	tokenID := ps.ByName("token_id")
	tokenBytes, err := kv.GetValue(KVAccessTokenIDBucket, []byte(tokenID))
	if err != nil {
		panic(err)
	}
	if tokenBytes == nil {
		h.WriteError(w, "token not found", 404)
		return
	}
	tokenV, err := kv.GetValue(core.KVAccessTokenBucket, tokenBytes)
	if err != nil {
		panic(err)
	}
	userID, err := jsonparser.GetString(tokenV, "userid")
	if err != nil {
		panic(err)
	}
	if userID != reqUser.UserId {
		h.WriteError(w, "permission denied", 403)
		return
	}
	tokenV, err = jsonparser.Set(tokenV, []byte(fmt.Sprintf(`"%s"`, reqBody.Name)), "name")
	if err != nil {
		panic(err)
	}
	err = kv.AddValue(core.KVAccessTokenBucket, tokenBytes, tokenV)
	if err != nil {
		panic(err)
	}
	h.WriteUpdatedOKJSON(w, tokenID)
}

// GenerateApiTokenName generates a unique API token name
func GenerateApiTokenName(prefix string) string {
	if prefix == "" {
		prefix = "token"
	}
	timestamp := time.Now().UnixMilli()
	randomStr := util.GenerateRandomString(8)
	return fmt.Sprintf("%s_%d_%s", prefix, timestamp, randomStr)
}
