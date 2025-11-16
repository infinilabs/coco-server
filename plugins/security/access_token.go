/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"fmt"
	"github.com/emirpasic/gods/sets/hashset"
	"infini.sh/framework/core/api"
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

func GetPermissionKeys(u *security.UserSessionInfo) []security.PermissionKey {
	//TODO cache, catch permission updates
	keys := security.MustGetPermissionKeysByRole(u.Roles)
	if len(u.Permissions) > 0 {
		keys = append(keys, u.Permissions...)
	}
	return keys
}

func GetPermissionHashSet(u *security.UserSessionInfo) *hashset.Set {
	//TODO cache, catch permission updates
	keys := GetPermissionKeys(u)
	set := security.ConvertPermissionKeysToHashSet(keys)
	return set
}

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
		"expire_in":    time.Now().Unix() + 86400, //24h
	}

	data["status"] = "ok"

	return data, err

}

func (h *APIHandler) RequestAccessToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	//user already login
	reqUser, err := security.GetUserFromContext(req.Context())
	if reqUser == nil || err != nil {
		panic(err)
	}

	reqBody := struct {
		Name        string                   `json:"name"`                  //custom access token name
		Permissions []security.PermissionKey `json:"permissions,omitempty"` //custom access token name
	}{}
	err = api.DecodeJSON(req, &reqBody)
	if err != nil {
		panic(err)
	}
	if reqBody.Name == "" {
		reqBody.Name = GenerateApiTokenName("")
	}

	permission := security.MustGetPermissionKeysByUserID(reqUser.MustGetUserID())
	if len(reqBody.Permissions) > 0 {
		//the permissions should be within' user's own permission scope
		if !util.IsSuperset(security.ConvertPermissionKeysToHashSet(permission), security.ConvertPermissionKeysToHashSet(reqBody.Permissions)) {
			panic("invalid permissions")
		}
	}

	res, err := CreateAPIToken(reqUser.MustGetUserID(), reqBody.Name, "general", permission)
	if err != nil {
		panic(err)
	}

	api.WriteJSON(w, res, 200)
}

func CreateAPIToken(userID string, tokenName, typeName string, permissions []security.PermissionKey) (util.MapStr, error) {

	if tokenName == "" {
		tokenName = GenerateApiTokenName("")
	}

	res := util.MapStr{}
	accessTokenStr := util.GetUUID() + util.GenerateRandomString(64)
	res["access_token"] = accessTokenStr
	expiredAT := time.Now().Add(365 * 24 * time.Hour).Unix()
	res["expire_in"] = expiredAT

	accessToken := security.AccessToken{}
	tokenID := util.GetUUID()
	accessToken.ID = tokenID
	accessToken.AccessToken = accessTokenStr
	accessToken.SetOwnerID(userID)

	accessToken.Type = typeName
	accessToken.Permissions = permissions
	accessToken.ExpireIn = expiredAT
	accessToken.Name = tokenName

	ctx := orm.NewContext()
	ctx.DirectAccess()
	ctx.Refresh = orm.WaitForRefresh
	err := orm.Create(ctx, &accessToken)
	if err != nil {
		panic(err)
	}

	// save access token to store
	err = kv.AddValue(core.KVAccessTokenBucket, []byte(accessTokenStr), util.MustToJSONBytes(accessToken))
	if err != nil {
		panic(err)
	}
	return res, err
}

func (h *APIHandler) SearchAccessToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//handle url query args, convert to query builder
	builder, err := orm.NewQueryBuilderFromRequest(req, "name")
	if err != nil {
		panic(err)
	}

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &security.AccessToken{})

	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		panic(err)
	}

	_, err = api.Write(w, res.Payload.([]byte))
	if err != nil {
		api.Error(w, err)
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

	api.WriteDeletedOKJSON(w, tokenID)
}

func GetToken(token string) (*security.AccessToken, error) {
	tokenBytes, err := kv.GetValue(core.KVAccessTokenBucket, []byte(token))
	if err != nil {
		panic(err)
	}
	var accessToken = security.AccessToken{}
	err = util.FromJSONBytes(tokenBytes, &accessToken)
	if err != nil {
		panic(err)
	}
	return &accessToken, nil
}

func (h *APIHandler) UpdateAccessToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	reqUser, err := security.GetUserFromContext(req.Context())
	if reqUser == nil || err != nil {
		panic(err)
	}
	reqBody := struct {
		Name        string                   `json:"name,omitempty"`        //custom access token name
		Permissions []security.PermissionKey `json:"permissions,omitempty"` //custom access token name
	}{}
	err = api.DecodeJSON(req, &reqBody)
	if err != nil {
		panic(err)
	}
	if reqBody.Name == "" {
		api.WriteError(w, "name is required", 400)
		return
	}
	tokenID := ps.ByName("token_id")

	ctx := orm.NewContextWithParent(req.Context())
	token := security.AccessToken{}
	token.ID = tokenID

	exists, err := orm.GetV2(ctx, &token)
	if err != nil {
		panic(err)
	}
	if !exists {
		api.WriteError(w, "access token not found", 404)
		return
	}

	if token.Name != "" {
		token.Name = reqBody.Name
	}
	if len(reqBody.Permissions) > 0 {
		//the permissions should be within' user's own permission scope
		newPermission := security.ConvertPermissionKeysToHashSet(token.Permissions)
		if !util.IsSuperset(GetPermissionHashSet(reqUser), newPermission) {
			panic("invalid permissions")
		}
		token.Permissions = reqBody.Permissions
	}

	ctx.Refresh = orm.WaitForRefresh
	err = orm.Save(ctx, &token)
	if err != nil {
		panic(err)
	}

	// save access token to store
	err = kv.AddValue(core.KVAccessTokenBucket, []byte(token.AccessToken), util.MustToJSONBytes(token))
	if err != nil {
		panic(err)
	}

	api.WriteUpdatedOKJSON(w, tokenID)
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
