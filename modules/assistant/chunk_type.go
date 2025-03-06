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

package assistant

// Define constants for the various stages
const (
	QueryIntent  = "query_intent"
	QueryRewrite = "query_rewrite"
	FetchSource  = "fetch_source"
	PickSource   = "pick_source"
	DeepRead     = "deep_read"
	Think        = "think"    //reasoning message by LLM
	Response     = "response" //formal response by assistant
	References   = "references"
	ReplyEnd     = "reply_end"
)
