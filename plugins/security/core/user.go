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
	rbac "infini.sh/console/core/security"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"strings"
)

type User struct {
	orm.ORMObjectBase

	Name        string      `json:"name"  elastic_mapping:"name: { type: keyword }"`
	Email       string      `json:"email" elastic_mapping:"email: { type: keyword }"`
	Phone       string      `json:"phone" elastic_mapping:"phone: { type: keyword }"`
	Tags        []string    `json:"tags" elastic_mapping:"tags: { type: keyword }"`
	AvatarUrl   string      `json:"avatar" elastic_mapping:"avatar: { type: keyword }"`
	Roles       []UserRole  `json:"roles" elastic_mapping:"roles: { type: object }"`
	Preferences Preferences `json:"preferences"`
}

type NativeUser struct {
	orm.ORMObjectBase
	Username string `json:"username"  elastic_mapping:"username: { type: keyword }"`
	Password string `json:"password"  elastic_mapping:"password: { type: keyword }"`
}

func (dal *User) Get(id string) (rbac.User, error) {
	user := rbac.User{}
	user.ID = id
	_, err := orm.Get(&user)
	return user, err
}

func (dal *User) GetBy(field string, value interface{}) (*rbac.User, error) {
	user := &rbac.User{}
	err, result := orm.GetBy(field, value, rbac.User{})
	if err != nil {
		return nil, err
	}
	if len(result.Result) == 0 {
		return nil, nil
	}
	userBytes, err := util.ToJSONBytes(result.Result[0])
	if err != nil {
		return nil, err
	}
	util.FromJSONBytes(userBytes, &user)
	return user, err
}

func (dal *User) Update(user *rbac.User) error {

	return orm.Update(nil, user)
}

func (dal *User) Create(user *rbac.User) (string, error) {
	user.ID = util.GetUUID()
	return user.ID, orm.Save(nil, user)
}

func (dal *User) Delete(id string) error {
	user := rbac.User{}
	user.ID = id
	return orm.Delete(nil, user)
}

func (dal *User) Search(keyword string, from, size int) (orm.Result, error) {
	query := orm.Query{}

	queryDSL := `{"query":{"bool":{"must":[%s]}}, "from": %d,"size": %d}`
	mustBuilder := &strings.Builder{}

	if keyword != "" {
		mustBuilder.WriteString(fmt.Sprintf(`{"query_string":{"default_field":"*","query": "%s"}}`, keyword))
	}
	queryDSL = fmt.Sprintf(queryDSL, mustBuilder.String(), from, size)
	query.RawQuery = []byte(queryDSL)

	err, result := orm.Search(rbac.User{}, &query)
	return result, err
}

type ExternalUserProfile struct {
	orm.ORMObjectBase
	UserID       string      `json:"user_id"  elastic_mapping:"user_id: { type: keyword }"`
	AuthProvider string      `json:"provider"  elastic_mapping:"provider: { type: keyword }"`
	Login        string      `json:"login"  elastic_mapping:"login: { type: keyword }"`
	Payload      interface{} `json:"payload" elastic_mapping:"payload: { type: object }"`
}

// Preferences represents the user's preferences for theme and language.
type Preferences struct {
	Theme    string `json:"theme"`
	Language string `json:"language"`
}

func (user *User) GetPermissions() (roles []string, privileges []string) {
	for _, v := range user.Roles {
		role, ok := RoleMap[v.Name]
		if ok {
			roles = append(roles, v.Name)
			privileges = append(privileges, role.Privilege.Platform...)
		}
	}
	return roles, privileges
}

type UserRole struct {
	ID   string `json:"id" elastic_mapping:"id: { type: keyword }"`
	Name string `json:"name" elastic_mapping:"name: { type: keyword }"`
}
