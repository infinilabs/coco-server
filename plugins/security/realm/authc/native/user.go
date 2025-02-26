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
	"fmt"
	"infini.sh/coco/plugins/security/core"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"strings"
)

type User struct {
}

func (dal *User) Get(id string) (core.User, error) {
	user := core.User{}
	user.ID = id
	_, err := orm.Get(&user)
	return user, err
}

func (dal *User) GetBy(field string, value interface{}) (*core.User, error) {
	user := &core.User{}
	err, result := orm.GetBy(field, value, core.User{})
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

func (dal *User) Update(user *core.User) error {

	return orm.Update(nil, user)
}

func (dal *User) Create(user *core.User) (string, error) {
	user.ID = util.GetUUID()
	return user.ID, orm.Save(nil, user)
}

func (dal *User) Delete(id string) error {
	user := core.User{}
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

	err, result := orm.Search(core.User{}, &query)
	return result, err
}
