POST $[[SETUP_INDEX_PREFIX]]assistant/$[[SETUP_DOC_TYPE]]/default
{
  "id" : "default",
  "created" : "2025-04-14T14:24:06.066519+08:00",
  "updated" : "2025-04-15T11:07:07.261101+08:00",
  "name" : "Coco AI",
  "description" : "默认 Coco AI 聊天助手",
  "icon" : "font_Robot-outlined",
  "type" : "simple",
  "answering_model": {
      "provider_id": "$[[SETUP_LLM_PROVIDER_ID]]",
      "name": "$[[SETUP_LLM_DEFAULT_MODEL_ID]]",
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
        "template": "You will be given a conversation and a follow-up question.\n\nIf necessary, rephrase the follow-up question into a standalone question so that the LLM can retrieve information from its knowledge base.\n\n⸻\n\nContext:\n\n{{.context}}\n\nUser Query:\n\n{{.query}}\n\n⸻\n\nPlease generate your response based on the information above, prioritizing the output from LLM tools (if available).\n\nEnsure your response is thoughtful, accurate, and well-structured.\n\nIf the information provided is insufficient, let the user know that more details are needed, or respond in a friendly and conversational tone.\n\nFor complex answers, use clear and well-organized Markdown format to improve readability.",
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
    "greeting_message" : "你好！我是 Coco，很高兴认识你。我可以通过访问互联网和你的数据源来帮助回答你的问题。今天我能为你做些什么？",
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
    "description": "用于搜索结果的 AI Overview，帮助你快速洞察关键信息、核心观点",
    "icon": "font_Brain02",
    "type": "simple",
    "answering_model": {
      "provider_id": "$[[SETUP_LLM_PROVIDER_ID]]",
      "name": "$[[SETUP_LLM_DEFAULT_MODEL_ID]]",
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
    "builtin": true,
    "role_prompt": "你是一个信息总结助手，专门负责对由 Coco AI 搜索得到的结果内容进行总结、归纳与概括。你的任务是从搜索结果中提取出用户最关心的信息，提供清晰、简洁、有条理的概览。\n\n请遵循以下规则：\n你只总结用户本次搜索返回的内容，不推测或引入外部信息。\n当搜索结果内容较多时，请优先提取共同主题、主要观点和明显的结论，避免逐条复述。\n如果搜索结果中包含多个来源或多种观点，请指出异同。\n如搜索结果过于杂乱或无效，请简要说明无法总结的原因，并建议用户尝试优化搜索关键词。\n不使用 Markdown 格式, 使用纯文本输出摘要. 摘要总体的字符总数不超过 250 个字符.\n输出语言与用户问题一致。\n"
  }