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

type Role struct {
}

func (dal *Role) Get(id string) (core.Role, error) {

	r, ok := core.BuiltinRoles[id]
	if ok {
		return r, nil
	}

	role := core.Role{}
	role.ID = id
	_, err := orm.Get(&role)
	return role, err
}

func (dal *Role) GetBy(field string, value interface{}) (core.Role, error) {
	role := core.Role{}
	err, result := orm.GetBy(field, value, &role)
	if result.Total > 0 {
		if len(result.Result) > 0 {
			bytes := util.MustToJSONBytes(result.Result[0])
			err := util.FromJSONBytes(bytes, &role)
			if err != nil {
				panic(err)
			}
			return role, nil
		}
	}
	return role, err
}

func (dal *Role) Update(role *core.Role) error {
	return orm.Save(nil, role)
}

func (dal *Role) Create(role *core.Role) (string, error) {
	role.ID = util.GetUUID()
	return role.ID, orm.Save(nil, role)
}

func (dal *Role) Delete(id string) error {
	role := core.Role{}
	role.ID = id
	return orm.Delete(nil, role)
}

func (dal *Role) Search(keyword string, from, size int) (orm.Result, error) {
	query := orm.Query{}

	queryDSL := `{"query":{"bool":{"must":[%s]}}, "from": %d,"size": %d}`
	mustBuilder := &strings.Builder{}

	if keyword != "" {
		mustBuilder.WriteString(fmt.Sprintf(`{"query_string":{"default_field":"*","query": "%s"}}`, keyword))
	}
	queryDSL = fmt.Sprintf(queryDSL, mustBuilder.String(), from, size)
	query.RawQuery = []byte(queryDSL)

	err, result := orm.Search(core.Role{}, &query)
	return result, err
}
