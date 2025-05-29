POST $[[SETUP_INDEX_PREFIX]]assistant/$[[SETUP_DOC_TYPE]]/default
{
  "id" : "default",
  "created" : "2025-04-14T14:24:06.066519+08:00",
  "updated" : "2025-04-15T11:07:07.261101+08:00",
  "name" : "Coco AI",
  "description" : "Default Coco AI chat assistant",
  "icon" : "font_Robot-outlined",
  "type" : "simple",
  "answering_model" : $[[SETUP_ASSISTANT_ANSWERING_MODEL]],
  "datasource" : {
    "enabled" : true,
    "ids" : [
      "*"
    ],
    "visible" : true
  },
  "mcp_servers" : {
    "enabled" : true,
    "ids" : [
      "*"
    ],
    "visible" : true
  },
  "keepalive" : "30m",
  "enabled" : true,
  "chat_settings" : {
    "greeting_message" : "Hi! I’m Coco, nice to meet you. I can help answer your questions by tapping into the internet and your data sources. How can I assist you today?",
    "suggested" : {
      "enabled" : false,
      "questions" : [ ]
    },
    "input_preprocess_tpl" : "",
    "history_message" : {
      "number" : 5,
      "compression_threshold" : 1000,
      "summary" : true
    }
  },
  "builtin" : true,
  "role_prompt" : ""
}

POST $[[SETUP_INDEX_PREFIX]]assistant/$[[SETUP_DOC_TYPE]]/ai_overview
{
    "id": "ai_overview",
    "created": "2025-05-28T09:29:42.689775563+08:00",
    "updated": "2025-05-28T09:32:39.310853954+08:00",
    "name": "AI Overview",
    "description": "AI Overview for search results helps you quickly grasp key information and core insights.",
    "icon": "font_Brain02",
    "type": "simple",
    "answering_model": {
      "provider_id": "qianwen",
      "name": "qwen-max",
      "settings": {
        "reasoning": false,
        "temperature": 0,
        "top_p": 0,
        "presence_penalty": 0,
        "frequency_penalty": 0,
        "max_tokens": 0,
        "max_length": 0
      },
      "prompt": {
        "template": "{{.query}}",
        "input_vars": null
      }
    },
    "datasource": {
      "enabled": false,
      "ids": [
        "*"
      ],
      "visible": false,
      "enabled_by_default": false
    },
    "tools": {
      "enabled": false,
      "builtin": {
        "calculator": false,
        "wikipedia": false,
        "duckduckgo": false,
        "scraper": false
      }
    },
    "mcp_servers": {
      "enabled": false,
      "ids": [
        "*"
      ],
      "visible": false,
      "model": null,
      "max_iterations": 5,
      "enabled_by_default": false
    },
    "keepalive": "30m",
    "enabled": true,
    "chat_settings": {
      "greeting_message": "",
      "suggested": {
        "enabled": false,
        "questions": []
      },
      "input_preprocess_tpl": "",
      "history_message": {
        "number": 5,
        "compression_threshold": 1000,
        "summary": true
      }
    },
    "builtin": false,
    "role_prompt": "You are an information summarization assistant, specializing in summarizing, consolidating, and synthesizing content retrieved by Coco AI search. Your task is to extract the most relevant information for the user from the search results and provide a clear, concise, and well-organized overview.\n\nPlease follow these rules:\nSummarize only the content returned from the current search. Do not speculate or introduce external information.\nPresent the summary in a structured format (e.g., bullet points, paragraphs, or heading-content layout) to help users quickly grasp the core information.\nWhen there is a large amount of content, focus on common themes, key insights, and clear conclusions rather than listing each item individually.\nIf the results contain multiple sources or differing viewpoints, highlight similarities and differences.\nIf the search results are too messy or irrelevant, briefly explain that the content cannot be summarized effectively and suggest refining the search query.\nThe output language should match the user’s original query language."
  }