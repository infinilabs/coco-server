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

/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package core

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/golang-jwt/jwt"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	"net/http"
	"strings"
	"time"
)

const (
	UserAccessTokenSessionName = "user_session_access_token"
	KVAccessTokenBucket        = "access_token"
	HeaderAPIToken       = "X-API-TOKEN"
	HeaderIntegrationID  = "APP-INTEGRATION-ID"
)

func ValidateLoginByAPITokenHeader(r *http.Request) (claims *security.UserClaims, err error) {
	apiToken := r.Header.Get(HeaderAPIToken)

	if apiToken == "" {
		return nil, errors.Error("api token not found")
	}

	bytes, err := kv.GetValue(KVAccessTokenBucket, []byte(apiToken))
	if err != nil {
		return nil, err
	}

	if bytes == nil || len(bytes) == 0 {
		return nil, errors.Errorf("invalid %s", HeaderAPIToken)
	}

	data := security.AccessToken{}
	util.MustFromJSONBytes(bytes, &data)

	if global.Env().IsDebug {
		log.Debug("get AccessToken from store:", util.MustToJSON(data))
	}

	expireAtTime := time.Unix(data.ExpireIn, 0) // Convert to time.Time
	if time.Now().After(expireAtTime) {
		panic("Token expired")
	}

	// Safely extract fields with type assertions
	claims = &security.UserClaims{}
	claims.UserSession = &security.UserSession{}
	claims.Provider = data.Provider
	claims.Login = data.Login
	claims.UserID = data.UserID
	claims.Roles = data.Roles

	//claims. //
	//permissions

	claims.Provider = "token"
	return claims, nil
}

func ValidateLoginByAuthorizationHeader(r *http.Request) (claims *security.UserClaims, err error) {
	var (
		authorization = r.Header.Get("Authorization")
		ok            bool
	)

	if authorization == "" {
		return nil, errors.Error("Authorization not found")
	}

	fields := strings.Fields(authorization)
	if fields[0] != "Bearer" || len(fields) != 2 {
		err = errors.New("authorization header is invalid")
		return nil, err
	}
	tokenString := fields[1]

	token, err := jwt.ParseWithClaims(tokenString, &security.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(GetSecret()), nil
	})
	if err != nil {
		return nil, err
	}
	//validate bind tenant
	claims, ok = token.Claims.(*security.UserClaims)
	if ok && token.Valid {
		if claims.UserID == "" {
			err = errors.New("user id is empty")
			return nil, err
		}
		if !claims.VerifyExpiresAt(time.Now(), true) {
			err = errors.New("token is expire in")
			return nil, err
		}
	}
	if claims == nil {
		return nil, errors.Error("invalid claims")
	}
	claims.Provider = "bearer"
	return claims, nil
}

func ValidateLoginByAccessTokenSession(r *http.Request) (claims *security.UserClaims, err error) {
	exists, sessToken := api.GetSession(r, UserAccessTokenSessionName)

	if !exists || sessToken == nil {
		return nil, errors.Error("invalid session")
	}

	var (
		tokenStr string
		ok       bool
	)
	if tokenStr, ok = sessToken.(string); !ok {
		err = errors.New("authorization token is empty")
		return
	}

	token, err1 := jwt.ParseWithClaims(tokenStr, &security.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(GetSecret()), nil
	})
	if err1 != nil {
		return
	}

	//validate bind tenant
	claims, ok = token.Claims.(*security.UserClaims)
	if ok && token.Valid {
		if claims.UserID == "" {
			err = errors.New("user id is empty")
			return
		}
		if !claims.VerifyExpiresAt(time.Now(), true) {
			err = errors.New("token is expire in")
			return
		}
	}
	claims.Provider = "session"
	return claims, nil
}

func ValidateLogin(r *http.Request) (claims *security.UserClaims, err error) {

	claims, err = ValidateLoginByAccessTokenSession(r)

	if claims == nil {
		claims, err = ValidateLoginByAuthorizationHeader(r)
	}

	if claims == nil {
		claims, err = ValidateLoginByAPITokenHeader(r)
	}

	if claims == nil || err != nil {
		err = errors.Errorf("invalid user info: %v", err)
		return
	}

	return claims, nil
}
