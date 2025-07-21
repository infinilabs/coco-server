POST $[[SETUP_INDEX_PREFIX]]assistant$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/default
{
  "id" : "default",
  "created" : "2025-04-14T14:24:06.066519+08:00",
  "updated" : "2025-04-15T11:07:07.261101+08:00",
  "name" : "Coco AI",
  "description" : "Default Coco AI chat assistant",
  "icon" : "font_Robot-outlined",
  "type" : "simple",
  "answering_model": {
    "provider_id": "$[[SETUP_LLM_PROVIDER_ID]]",
    "name": "$[[SETUP_LLM_DEFAULT_MODEL_ID]]",
    "settings": {
      "reasoning": $[[SETUP_LLM_REASONING]],
      "temperature": 0,
      "top_p": 0,
      "presence_penalty": 0,
      "frequency_penalty": 0,
      "max_tokens": 0,
      "max_length": 0
    },
    "prompt": {
      "template": "You are a helpful AI assistant. \n You will be given a conversation below and a follow-up question.\n \n {{.context}}\n \n The user has provided the following query:\n {{.query}}\n \n Ensure your response is thoughtful, accurate, and well-structured.\n For complex answers, format your response using clear and well-organized **Markdown** to improve readability.",
      "input_vars": null
    }
  },
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

POST $[[SETUP_INDEX_PREFIX]]assistant$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/ai_overview
{
    "id": "ai_overview",
    "created": "2025-05-28T09:29:42.689775563+08:00",
    "updated": "2025-05-28T09:32:39.310853954+08:00",
    "name": "AI Overview",
    "description": "AI Overview for search results helps you quickly grasp key information and core insights.",
    "icon": "font_Brain02",
    "type": "simple",
    "answering_model": {
      "provider_id": "$[[SETUP_LLM_PROVIDER_ID]]",
      "name": "$[[SETUP_LLM_DEFAULT_MODEL_ID]]",
      "settings": {
        "reasoning": $[[SETUP_LLM_REASONING]],
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
    "builtin": true,
"role_prompt": "You are an information summarization assistant, specialized in summarizing, condensing, and organizing the results retrieved by Coco AI Search. Your task is to extract the most relevant information that the user cares about and provide a clear, concise, and well-structured overview.\n\nPlease follow these rules:\nOnly summarize the content returned by the current search; do not infer or introduce external information.\nWhen the search results are lengthy, prioritize extracting common themes, main points, and clear conclusions, and avoid listing each result individually.\nIf the results include multiple sources or perspectives, highlight the similarities and differences.\nIf the results are too chaotic or irrelevant, briefly explain why a summary cannot be provided and suggest the user refine their search keywords.\nDo not use Markdown formatting; output the summary as plain text. The total character count of the summary must not exceed 250 characters.\nThe output language should match the language of the user's query.\n"
}