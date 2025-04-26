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

const GenerateAnswerPromptTemplate = `
You are a helpful assistant designed to help users access own data and understand their tasks.
Your responses should be clear, concise, and based solely on the information provided below.
You will be given a conversation below and a follow-up question. 
You need to rephrase the follow-up question if needed so it is a standalone question that can be 
used by the LLM to search the knowledge base for information.

{{.context}}

The user has provided the following query:
{{.query}}


Please generate your response using the information above, prioritizing LLM tool outputs when available. 
Ensure your response is thoughtful, accurate, and well-structured.
If the provided information is insufficient, let the user know more details are needed — or offer a friendly, conversational response instead.
For complex answers, format your response using clear and well-organized **Markdown** to improve readability.
`
