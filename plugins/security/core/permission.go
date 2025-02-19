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
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/api/routetree"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
	"sync"
)

var permissionsMap = map[string]interface{}{}
var permissionsLocker = sync.Mutex{}

func RegisterPermission(typ string, permissions interface{}) {
	permissionsLocker.Lock()
	defer permissionsLocker.Unlock()
	permissionsMap[typ] = permissions
}

func GetPermissions(typ string) interface{} {
	permissionsLocker.Lock()
	defer permissionsLocker.Unlock()
	return permissionsMap[typ]
}

var RoleMap = make(map[string]Role)

type Token struct {
	JwtStr   string `json:"jwt_str"`
	Value    string `json:"value"`
	ExpireIn int64  `json:"expire_in"`
}

var userTokenLocker = sync.RWMutex{}
var tokenMap = make(map[string]Token)

const KVUserToken = "user_token"

func SetUserToken(key string, token Token) {
	userTokenLocker.Lock()
	tokenMap[key] = token
	userTokenLocker.Unlock()
	_ = kv.AddValue(KVUserToken, []byte(key), util.MustToJSONBytes(token))
}
func GetUserToken(key string) *Token {
	userTokenLocker.RLock()
	defer userTokenLocker.RUnlock()
	if token, ok := tokenMap[key]; ok {
		return &token
	}
	tokenBytes, err := kv.GetValue(KVUserToken, []byte(key))
	if err != nil {
		log.Errorf("get user token from kv error: %v", err)
		return nil
	}
	if tokenBytes == nil {
		return nil
	}
	token := Token{}
	util.MustFromJSONBytes(tokenBytes, &token)
	return &token
}

func DeleteUserToken(key string) {
	userTokenLocker.Lock()
	delete(tokenMap, key)
	userTokenLocker.Unlock()
	_ = kv.DeleteKey(KVUserToken, []byte(key))
}

var apiPermissionRouter = map[string]*routetree.Router{}
var apiPermissionLocker = sync.Mutex{}

func RegisterAPIPermissionRouter(typ string, router *routetree.Router) {
	apiPermissionLocker.Lock()
	defer apiPermissionLocker.Unlock()
	apiPermissionRouter[typ] = router
}

func GetAPIPermissionRouter(typ string) *routetree.Router {
	apiPermissionLocker.Lock()
	defer apiPermissionLocker.Unlock()
	return apiPermissionRouter[typ]
}
