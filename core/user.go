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
	"infini.sh/framework/core/orm"
)

type User struct {
	orm.ORMObjectBase

	Name        string      `json:"name"  elastic_mapping:"name: { type: keyword }"`
	Email       string      `json:"email" elastic_mapping:"email: { type: keyword }"`
	Phone       string      `json:"phone" elastic_mapping:"phone: { type: keyword }"`
	Tags        []string    `json:"tags" elastic_mapping:"tags: { type: keyword }"`
	AvatarUrl   string      `json:"avatar" elastic_mapping:"avatar: { type: keyword }"`
	Preferences Preferences `json:"preferences"`
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
