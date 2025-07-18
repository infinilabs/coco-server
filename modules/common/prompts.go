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

// Define the prompt template
const QueryIntentPromptTemplate = `You are an AI assistant trained to understand and analyze user queries.
You will be given a conversation below and a follow-up question. You need to rephrase the follow-up question if needed so it is a standalone question that can be used by the LLM to search the knowledge base for information.

Conversation:
{{.history}}

Tool List:
{{.tool_list}}

Network sources List:
{{.network_sources}}

The user has provided the following query:
{{.query}}

You need help to figure out the following tasks:
- Please analyze the query and identify the user's primary intent. Determine if they are looking for information, making a request, or seeking clarification. brief as field: 'intent',
- Categorize the intent in </Category>,  and rephrase the query in several different forms to improve clarity.
- Provide possible variations of the query in <Query/> and identify relevant keywords in </Keyword> in JSON array format.
- Provide possible related queries in <Suggestion/> and expand the related query for query suggestion.
- Based on the tool list provided, analyze the user's query whether need to call external tools, output as field: 'need_call_tools'
- Based on the network source list provided, analyze the user's query whether need to perform a network search, in order to get more information, output as field: 'need_network_search'
- Analyze the user's query whether need to plan some complex sub-tasks in order to achieve the goal, output as field: 'need_plan_tasks'


Please make sure the output is concise, well-organized, and easy to process.
Please present these possible query and keyword items in both English and Chinese.

If the possible query is in English, keep the original English one, and translate it to Chinese and keep it as a new query, to be clear, you should output: [Apple, 苹果], neither just 'Apple' nor just '苹果'.
Wrap the valid JSON result in <JSON></JSON> tags.

Your output should look like this format:
<JSON>
{
  "category": "<Intent's Category>",
  "intent": "<User's Intent>",
  "query": [
    "<Rephrased Query 1>",
    "<Rephrased Query 2>",
    "<Rephrased Query 3>"
  ],
  "keyword": [
    "<Keyword 1>",
    "<Keyword 2>",
    "<Keyword 3>"
  ],
  "suggestion": [
    "<Suggest Query 1>",
    "<Suggest Query 2>",
    "<Suggest Query 3>"
  ],
  "need_plan_tasks":<true or false>,
  "need_call_tools":<true or false>,
  "need_network_search":<true or false>
}
</JSON>`

const GenerateAnswerPromptTemplate = `
You are a helpful AI assistant.
You will be given a conversation below and a follow-up question.

{{.context}}

The user has provided the following query:
{{.query}}

Ensure your response is thoughtful, accurate, and well-structured.
For complex answers, format your response using clear and well-organized **Markdown** to improve readability.
`
