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

package native

import (
	"infini.sh/coco/plugins/security/core"
)

var handler core.Adapter

func init() {
	handler = core.Adapter{
		User: &User{},
		Role: &Role{},
	}
	core.RegisterAdapter(providerName, handler)
}

const providerName = "native"

type NativeRealm struct {
	// Implement any required fields
}

func (r *NativeRealm) GetType() string {
	return providerName
}

func (r *NativeRealm) Authenticate(username, password string) (bool, *core.User, error) {
	// Implement the authentication logic
	// Retrieve the user profile upon successful authentication
	// Return the authentication result, user profile, and any potential error

	var err error

	//user, err := handler.User.GetBy("name", username)
	//if err != nil {
	//	return false, user, err
	//}
	//if user == nil {
	//	return false, nil, fmt.Errorf("user account [%s] not found", username)
	//}
	//
	//err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	//if err == bcrypt.ErrMismatchedHashAndPassword {
	//	err = errors.New("incorrect password")
	//}
	//
	//if err == nil {
	//	user.AuthProvider = providerName
	//	return true, user, nil
	//}

	return false, nil, err
}

func (r *NativeRealm) Authorize(user *core.User) (bool, error) {

	//var _, privilege = user.GetPermissions()
	//
	//if len(privilege) == 0 {
	//	log.Error("no privilege assigned to user:", user)
	//	return false, errors.New("no privilege assigned to this user:" + user.Username)
	//}

	return true, nil
}
