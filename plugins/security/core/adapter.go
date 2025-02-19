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
	"infini.sh/framework/core/orm"
)

type IRole interface {
	Get(id string) (Role, error)
	GetBy(field string, value interface{}) (Role, error)
	Update(role *Role) error
	Create(role *Role) (string, error)
	Delete(id string) error
	Search(keyword string, from, size int) (orm.Result, error)
}

type IUser interface {
	Get(id string) (User, error)
	GetBy(field string, value interface{}) (*User, error)
	Update(user *User) error
	Create(user *User) (string, error)
	Delete(id string) error
	Search(keyword string, from, size int) (orm.Result, error)
}

type SecurityRealm interface {
	GetType() string
	Authenticate(username, password string) (bool, *User, error) // Return true if authentication is successful, otherwise false
	Authorize(user *User) (bool, error)                          // Return true if authorization is granted, otherwise false
}

type Adapter struct {
	Role IRole
	User IUser
}

var adapterHandlers = map[string]Adapter{}

func RegisterAdapter(typ string, handler Adapter) {
	adapterHandlers[typ] = handler
}

func GetAdapter(typ string) Adapter {
	handler, ok := adapterHandlers[typ]
	if !ok {
		panic(fmt.Errorf("dal handler %s not found", typ))
	}
	return handler
}
