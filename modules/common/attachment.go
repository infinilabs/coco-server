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

type Attachment struct {
	orm.ORMObjectBase // Embedding ORM base for persistence-related fields

	Name        string `json:"name,omitempty" elastic_mapping:"name:{type:keyword}"`
	Description string `json:"description,omitempty" elastic_mapping:"description:{type:keyword}"`
	Icon        string `json:"icon,omitempty" elastic_mapping:"icon:{enabled:false}"`
	MimeType    string `json:"mime_type,omitempty" elastic_mapping:"mime_type:{enabled:false}"`
	URL         string `json:"url,omitempty" elastic_mapping:"url:{enabled:false}"`
	Size        int    `json:"size,omitempty" elastic_mapping:"size:{type:long}"`

	Deleted       bool        `json:"deleted,omitempty" elastic_mapping:"deleted:{type:boolean}"`
	LastUpdatedBy *EditorInfo `json:"last_updated_by,omitempty" elastic_mapping:"last_updated_by:{type:object}"`

	Owner *UserInfo `json:"owner,omitempty" elastic_mapping:"owner:{type:object}"` // Document author or owner

	Metadata map[string]interface{} `json:"metadata,omitempty" elastic_mapping:"metadata:{type:object}"` // Additional accessible metadata (e.g., file version, permissions)
	Payload  map[string]interface{} `json:"payload,omitempty" elastic_mapping:"payload:{enabled:false}"` // Additional store-only metadata (e.g., file binary data)
}
