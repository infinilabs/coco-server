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

package realm

import (
	"infini.sh/coco/plugins/security/api"
	"infini.sh/coco/plugins/security/config"
	"infini.sh/coco/plugins/security/core"
	"infini.sh/coco/plugins/security/realm/authc/oauth"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	log "src/github.com/cihub/seelog"
)

var realms = []core.SecurityRealm{}

func Init(config *config.Config) {

	if !config.Enabled {
		return
	}

	if len(config.Authentication.Realms.OAuth) > 0 {
		for _, v := range config.Authentication.Realms.OAuth {
			{
				realm := oauth.New(v)
				realms = append(realms, realm)
			}
		}
	}

	if global.Env().IsDebug {
		log.Tracef("config: %v", util.MustToJSON(config))
	}

	oauth.Init(config.Authentication.Realms.OAuth)
	api.Init()

}

func Authenticate(username, password string) (bool, *core.User, error) {

	for i, realm := range realms {
		ok, user, err := realm.Authenticate(username, password)
		log.Debugf("authenticate result: %v, user: %v, err: %v, realm: %v", ok, user, err, i)
		if ok && user != nil && err == nil {
			return true, user, nil
		}
	}
	if global.Env().IsDebug {
		log.Errorf("failed to authenticate user: %v", username)
	}
	return false, nil, errors.Errorf("failed to authenticate user: %v", username)
}

//func Authorize(user *core.User) (bool, error) {
//
//	for i, realm := range realms {
//		//skip if not the same auth provider, TODO: support cross-provider authorization
//		if user.AuthProvider != realm.GetType() {
//			continue
//		}
//
//		ok, err := realm.Authorize(user)
//		log.Debugf("authorize result: %v, user: %v, err: %v, realm: %v", ok, user, err, i)
//		if ok && err == nil {
//			//return on any success, TODO, maybe merge all roles and privileges from all realms
//			return true, nil
//		}
//	}
//
//	roles, privilege := user.GetPermissions()
//	if len(roles) == 0 && len(privilege) == 0 {
//		if global.Env().IsDebug {
//			log.Errorf("failed to authorize user: %v", user.Username)
//		}
//		return false, errors.New("no roles or privileges")
//	}
//
//	return false, errors.Errorf("failed to authorize user: %v", user.Username)
//
//}
