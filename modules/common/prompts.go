/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

// Define the prompt template
const QueryIntentPromptTemplate = `You are an AI assistant trained to understand and analyze user queries.
You will be given a conversation below and a follow-up question. You need to rephrase the follow-up question if needed so it is a standalone question that can be used by the LLM to search the knowledge base for information.

Conversation:
{{.history}}

Tool List:
{{.tool_list}}

Network search sources List:
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

const PickingDocPromptTemplate = `
You are an AI assistant trained to select the most relevant documents for further processing and to answer user queries.
We have already queried the backend database and retrieved a list of documents that may help answer the user's query. And also invoke some external tools provided by MCP servers. 
Your task is to choose the best documents for further processing.
The user has provided the following query:
{{.query}}

The primary intent behind this query is:
{{.intent}}

The following documents are fetched from database:
{{.docs}}

Please review these documents and identify which ones best related to user's query. 
Choose no more than 5 relevant documents. These documents may be entirely unrelated, so prioritize those that provide direct answers or valuable context.
If the document is unrelated not certain, don't include it.
For each document, provide a brief explanation of why it was selected.
Your decision should based solely on the information provided below. \nIf the information is insufficient, please indicate that you need more details to assist effectively.
Don't make anything up, which means if you can't identify which document best match the user's query, you should output nothing.
Make sure the output is concise and easy to process.
Wrap the JSON result in <JSON></JSON> tags.
"\nThe expected output format is:
<JSON>
[
{ "id": "<id of Doc 1>", "title": "<title of Doc 1>", "explain": "<Explain for Doc 1>"  },
{ "id": "<id of Doc 2>", "title": "<title of Doc 2>", "explain": "<Explain for Doc 2>"  },
]
</JSON>
`
