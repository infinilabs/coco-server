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
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
)

const Secret = "coco"

var secretKey string

func GetSecret() string {

	if secretKey != "" {
		return secretKey
	}

	exists, err := kv.ExistsKey("Coco", []byte(Secret))
	if err != nil {
		panic(err)
	}
	if !exists {
		key := util.GetUUID()
		err = kv.AddValue("Coco", []byte(Secret), []byte(key))
		if err != nil {
			panic(err)
		}
		secretKey = key
	} else {
		v, err := kv.GetValue("Coco", []byte(Secret))
		if err != nil {
			panic(err)
		}
		if len(v) > 0 {
			secretKey = string(v)
		}
	}

	if secretKey == "" {
		panic("invalid secret")
	}

	return secretKey
}
