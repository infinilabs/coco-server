POST $[[SETUP_INDEX_PREFIX]]assistant/$[[SETUP_DOC_TYPE]]/default
{
  "id" : "default",
  "created" : "2025-04-14T14:24:06.066519+08:00",
  "updated" : "2025-04-15T11:07:07.261101+08:00",
  "name" : "Coco AI",
  "description" : "default Coco AI chat assistant",
  "icon" : "font_Robot-outlined",
  "type" : "deep_think",
  "config" : {
    "intent_analysis_model" : {
      "name" : "tongyi-intent-detect-v3",
      "provider_id" : "coco"
    },
    "picking_doc_model" : {
      "name" : "deepseek-r1-distill-qwen-32b",
      "provider_id" : "coco"
    }
  },
  "answering_model" : {
    "provider_id" : "coco",
    "name" : "deepseek-r1"
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