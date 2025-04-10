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

type MCPServer struct {
	CombinedFullText
	Name        string   `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"`
	Description string   `json:"description" elastic_mapping:"description:{type:keyword,copy_to:combined_fulltext}"`
	Endpoint    string   `json:"endpoint" elastic_mapping:"endpoint:{type:keyword}"`
	Tags        []string `json:"tags" elastic_mapping:"tags:{type:keyword,copy_to:combined_fulltext}"`
	Token       string   `json:"token" elastic_mapping:"token:{type:keyword}"`
}
