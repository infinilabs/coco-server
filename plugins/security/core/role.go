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
	"infini.sh/coco/plugins/security/core/enum"
	"infini.sh/framework/core/orm"
	"time"
)

type Role struct {
	orm.ORMObjectBase

	Name        string        `json:"name"  elastic_mapping:"name: { type: keyword }"`
	Type        string        `json:"type" elastic_mapping:"type: { type: keyword }"`
	Description string        `json:"description"  elastic_mapping:"description: { type: text }"`
	Builtin     bool          `json:"builtin" elastic_mapping:"builtin: { type: boolean }"`
	Privilege   RolePrivilege `json:"privilege" elastic_mapping:"privilege: { type: object }"`
}

type RolePrivilege struct {
	Platform      []string               `json:"platform,omitempty" elastic_mapping:"platform: { type: keyword }"`
	Elasticsearch ElasticsearchPrivilege `json:"elasticsearch,omitempty" elastic_mapping:"elasticsearch: { type: object }"`
}

type ElasticsearchPrivilege struct {
	Cluster ClusterPrivilege `json:"cluster,omitempty" elastic_mapping:"cluster: { type: object }"`
	Index   []IndexPrivilege `json:"index,omitempty" elastic_mapping:"index: { type: object }"`
}

type InnerCluster struct {
	ID   string `json:"id" elastic_mapping:"id: { type: keyword }"`
	Name string `json:"name" elastic_mapping:"name: { type: keyword }"`
}

type ClusterPrivilege struct {
	Resources   []InnerCluster `json:"resources,omitempty" elastic_mapping:"resources: { type: object }"`
	Permissions []string       `json:"permissions,omitempty" elastic_mapping:"permissions: { type: keyword }"`
}

type IndexPrivilege struct {
	Name        []string `json:"name,omitempty" elastic_mapping:"name: { type: keyword }"`
	Permissions []string `json:"permissions,omitempty" elastic_mapping:"permissions: { type: keyword }"`
}

type RoleType = string

const (
	Platform      RoleType = "platform"
	Elasticsearch RoleType = "elasticsearch"
)

func IsAllowRoleType(roleType string) (err error) {
	if roleType != Platform && roleType != Elasticsearch {
		err = fmt.Errorf("invalid role type %s ", roleType)
		return
	}
	return
}

var BuiltinRoles = make(map[string]Role, 0)

const RoleAdminName = "Administrator"

func init() {
	now := time.Now()
	BuiltinRoles[RoleAdminName] = Role{
		ORMObjectBase: orm.ORMObjectBase{
			ID:      RoleAdminName,
			Created: &now,
		},
		Name: RoleAdminName,
		Type: "platform",
		Privilege: RolePrivilege{
			Platform: enum.AdminPrivilege,
			Elasticsearch: ElasticsearchPrivilege{
				Cluster: ClusterPrivilege{
					Resources:   []InnerCluster{{"*", "*"}},
					Permissions: []string{"*"},
				},
				Index: []IndexPrivilege{
					{Name: []string{"*"},
						Permissions: []string{"*"},
					},
				},
			},
		},
		Builtin:     true,
		Description: "Administrator is a super role.",
	}
}
