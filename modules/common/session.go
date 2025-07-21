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

package common

import "infini.sh/framework/core/orm"

type Session struct {
	orm.ORMObjectBase
	Status               string `config:"status" json:"status,omitempty" elastic_mapping:"status:{type:keyword}"`
	Title                string `json:"title,omitempty" elastic_mapping:"title:{type:text,copy_to:combined_fulltext,fields:{text: {type: text}, pinyin: {type: text, analyzer: pinyin_analyzer}}}"` // Document title
	Summary              string `config:"summary" json:"summary,omitempty" elastic_mapping:"summary:{type:text}"`
	ManuallyRenamedTitle bool   `config:"manually_renamed_title" json:"manually_renamed_title,omitempty" elastic_mapping:"manually_renamed_title:{type:boolean}"`

	Visible bool `json:"visible" elastic_mapping:"visible:{type:boolean}"` // Whether the connector is enabled or not

	Context *SessionContext `config:"context" json:"context,omitempty" elastic_mapping:"context:{type:object}"`
}

type SessionContext struct {
	Attachments []string `config:"attachments" json:"attachments,omitempty" elastic_mapping:"attachments:{type:keyword}"`
}
