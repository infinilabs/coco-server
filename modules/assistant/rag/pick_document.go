// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
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

package rag

import "infini.sh/framework/core/util"

type PickedDocument struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Explain string `json:"explain"`
}

func PickedDocumentFromString(str string) ([]PickedDocument, error) {
	str = util.TrimLeftStr(str, "<JSON>")
	str = util.TrimRightStr(str, "</JSON>")
	str = util.TrimSpaces(str)
	obj := []PickedDocument{}
	err := util.FromJSONBytes([]byte(str), &obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
