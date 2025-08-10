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
	"infini.sh/framework/core/orm"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

func GenerateJWTAccessToken(user *security.UserSessionInfo) (map[string]interface{}, error) {

	var data map[string]interface{}
	t := time.Now()
	if user.LastLogin.Timestamp == nil {
		user.LastLogin.Timestamp = &t
	}

	token1 := jwt.NewWithClaims(jwt.SigningMethodHS256, security.UserClaims{
		UserSessionInfo: user,
		RegisteredClaims: &jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	})

	secret, err := core.GetSecret()
	if err != nil {
		return nil, errors.Errorf("failed to get secret key: %v", err)
	}

	tokenString, err := token1.SignedString([]byte(secret))
	if tokenString == "" || err != nil {
		return nil, errors.Errorf("failed to generate access_token for user: %v", user)
	}

	data = util.MapStr{
		"access_token": tokenString,
		"expire_in": time.Now().Unix() + 86400, //24h
	}

	data["status"] = "ok"

	return data, err

}

func (h *APIHandler) RequestAccessToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	ctx := orm.NewContextWithParent(req.Context())

	//user already login
	reqUser, err := security.GetUserFromContext(req.Context())
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
		reqBody.Name = GenerateApiTokenName("")
	}
	res, err := CreateAPIToken(ctx, reqBody.Name, "general", []string{security.RoleAdmin})
	if err != nil {
		panic(err)
	}

	h.WriteJSON(w, res, 200)
}

func CreateAPIToken(ctx *orm.Context, tokenName, typeName string, Roles []string) (util.MapStr, error) {
	if tokenName == "" {
		tokenName = GenerateApiTokenName("")
	}

	reqUser, err := security.GetUserFromContext(ctx)
	if reqUser == nil || err != nil {
		panic(err)
	}

	provider := "access_token"

	res := util.MapStr{}
	accessTokenStr := util.GetUUID() + util.GenerateRandomString(64)
	res["access_token"] = accessTokenStr
	expiredAT := time.Now().Add(365 * 24 * time.Hour).Unix()
	res["expire_in"] = expiredAT

	accessToken := security.AccessToken{}
	tokenID := util.GetUUID()
	accessToken.ID = tokenID
	accessToken.AccessToken = accessTokenStr
	accessToken.Provider = provider
	accessToken.Login = reqUser.Login
	accessToken.Type = typeName
	accessToken.Roles = Roles
	accessToken.Permissions = []string{}
	accessToken.ExpireIn = expiredAT
	accessToken.Name = tokenName

	ctx.Refresh = orm.WaitForRefresh
	err = orm.Create(ctx, &accessToken)
	if err != nil {
		panic(err)
	}

	// save access token to store
	err = kv.AddValue(core.KVAccessTokenBucket, []byte(accessTokenStr), util.MustToJSONBytes(accessToken))
	if err != nil {
		return nil, err
	}
	return res, err
}

func (h *APIHandler) SearchAccessToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//handle url query args, convert to query builder
	builder, err := orm.NewQueryBuilderFromRequest(req, "name")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &security.AccessToken{})

	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Payload.([]byte))
	if err != nil {
		h.Error(w, err)
	}
}

func (h *APIHandler) DeleteAccessToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	reqUser, err := security.GetUserFromContext(req.Context())
	if reqUser == nil || err != nil {
		panic(err)
	}
	tokenID := ps.ByName("token_id")

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh

	token := security.AccessToken{}
	token.ID = tokenID
	err = orm.Delete(ctx, &token)
	if err != nil {
		panic(err)
	}

	if token.AccessToken != "" {
		err = kv.DeleteKey(core.KVAccessTokenBucket, []byte(token.AccessToken))
		if err != nil {
			panic(err)
		}
	}

	h.WriteDeletedOKJSON(w, tokenID)
}

func GetToken(token string) (*security.AccessToken, error) {
	tokenBytes, err := kv.GetValue(core.KVAccessTokenBucket, []byte(token))
	if err != nil {
		return nil, err
	}
	var accessToken = security.AccessToken{}
	err = util.FromJSONBytes(tokenBytes, &accessToken)
	if err != nil {
		return nil, err
	}
	return &accessToken, nil
}

func (h *APIHandler) RenameAccessToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	reqUser, err := security.GetUserFromContext(req.Context())
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

	ctx := orm.NewContextWithParent(req.Context())
	token := security.AccessToken{}
	token.ID = tokenID

	exists, err := orm.GetV2(ctx, &token)
	if err != nil {
		h.WriteError(w, err.Error(), 400)
		return
	}
	if !exists {
		h.WriteError(w, "access token not found", 404)
		return
	}

	token.Name = reqBody.Name
	ctx.Refresh = orm.WaitForRefresh
	err = orm.Save(ctx, &token)
	if err != nil {
		h.WriteError(w, err.Error(), 400)
		return
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
