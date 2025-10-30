/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */
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
	HeaderAPIToken             = "X-API-TOKEN"
	HeaderIntegrationID        = "APP-INTEGRATION-ID"
)

func ValidateLoginByAPITokenHeader(w http.ResponseWriter, r *http.Request) (claims *security.UserClaims, err error) {
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

	accessToken := security.AccessToken{}
	util.MustFromJSONBytes(bytes, &accessToken)

	if global.Env().IsDebug {
		log.Debug("get AccessToken from store:", util.MustToJSON(accessToken))
	}

	expireAtTime := time.Unix(accessToken.ExpireIn, 0) // Convert to time.Time
	if time.Now().After(expireAtTime) {
		return nil, errors.Error("token expired")
	}

	// Safely extract fields with type assertions
	claims = security.NewUserClaims()
	claims.System = accessToken.System
	claims.Provider = accessToken.Provider
	claims.Login = accessToken.Login
	claims.Roles = accessToken.Roles
	claims.Permissions = accessToken.Permissions

	claims.Source = "token"
	return claims, nil
}

func ValidateLoginByAuthorizationHeader(w http.ResponseWriter, r *http.Request) (claims *security.UserClaims, err error) {
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

	token, err := jwt.ParseWithClaims(tokenString, security.NewUserClaims(), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		secret, err := GetSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to get secret key: %v", err)
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	//validate bind tenant
	claims, ok = token.Claims.(*security.UserClaims)

	if ok && token.Valid {
		if claims.Login == "" {
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
	claims.Source = "bearer"
	return claims, nil
}

func ValidateLoginByAccessTokenSession(w http.ResponseWriter, r *http.Request) (claims *security.UserClaims, err error) {
	exists, sessToken := api.GetSession(w, r, UserAccessTokenSessionName)
	if !exists || sessToken == nil {
		return nil, errors.Error("invalid session")
	}

	tokenStr, ok := sessToken.(string)
	if !ok {
		return nil, errors.New("authorization token is empty")
	}

	// Preallocate to avoid nil pointer during JSON unmarshal
	claims = security.NewUserClaims()

	token, err1 := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		secret, err := GetSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to get secret key: %v", err)
		}
		return []byte(secret), nil
	})
	if err1 != nil {
		return nil, err1
	}

	if token.Valid {
		if claims.Login == "" {
			return nil, errors.New("user id is empty")
		}
		if !claims.VerifyExpiresAt(time.Now(), true) {
			return nil, errors.New("token is expired")
		}
	}

	claims.Source = "session"
	return claims, nil
}

func ValidateLoginByAccessTokenSession1(w http.ResponseWriter, r *http.Request) (claims *security.UserClaims, err error) {
	exists, sessToken := api.GetSession(w, r, UserAccessTokenSessionName)

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

	token, err1 := jwt.ParseWithClaims(tokenStr, security.NewUserClaims(), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		secret, err := GetSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to get secret key: %v", err)
		}
		return []byte(secret), nil
	})
	if err1 != nil {
		return
	}

	//validate bind tenant
	claims, ok = token.Claims.(*security.UserClaims)

	if ok && token.Valid {
		if claims.Login == "" {
			err = errors.New("user id is empty")
			return
		}
		if !claims.VerifyExpiresAt(time.Now(), true) {
			err = errors.New("token is expire in")
			return
		}
	}
	claims.Source = "session"
	return claims, nil
}

func ValidateLogin(w http.ResponseWriter, r *http.Request) (session *security.UserSessionInfo, err error) {

	claims, err := ValidateLoginByAccessTokenSession(w, r)

	if claims == nil || !claims.UserSessionInfo.ValidInfo() {
		claims, err = ValidateLoginByAuthorizationHeader(w, r)
	}

	if claims == nil || !claims.UserSessionInfo.ValidInfo() {
		claims, err = ValidateLoginByAPITokenHeader(w, r)
	}

	if claims == nil || !claims.UserSessionInfo.ValidInfo() || err != nil {
		err = errors.Errorf("invalid user info: %v", err)
		return
	}

	return claims.UserSessionInfo, nil
}
