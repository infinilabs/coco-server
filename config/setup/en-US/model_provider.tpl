# model provider
POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/deepseek
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "deepseek",
  "created" : "2025-03-28T10:25:39.7741+08:00",
  "updated" : "2025-03-28T11:14:47.103278+08:00",
  "name" : "Deepseek",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://api.deepseek.com/v1",
  "icon" : "font_deepseek",
  "models" : [
    {"name":"deepseek-chat", "settings":{"reasoning":false}},
    {"name":"deepseek-reasoner", "settings":{"reasoning":true}}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "Provide efficient and flexible large model API services, supporting complex tasks with a high cost-performance advantage."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/openai
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "openai",
  "created" : "2025-03-28T10:24:37.843478+08:00",
  "updated" : "2025-03-31T20:16:18.517692+08:00",
  "name" : "OpenAI",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://api.openai.com",
  "icon" : "/assets/icons/llm/openai.svg",
  "models" : [
     {"name":"gpt-4o-mini", "settings":{"reasoning":false}},
     {"name":"gpt-4o", "settings":{"reasoning":false}}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "Provide advanced GPT-series large model API services (such as GPT-4/ChatGPT), supporting multimodal interactions and enterprise-level AI solutions, with a mature API ecosystem and top-tier general intelligence performance."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/ollama
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "ollama",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "Ollama",
  "api_key" : "",
  "api_type" : "ollama",
  "base_url" : "http://127.0.0.1:11434",
  "icon" : "/assets/icons/llm/ollama.svg",
  "models" : [
     {"name":"qwen2.5:32b", "settings":{"reasoning":false}},
     {"name":"deepseek-r1:32b", "settings":{"reasoning":true}},
     {"name":"deepseek-r1:14b", "settings":{"reasoning":true}},
     {"name":"deepseek-r1:8b", "settings":{"reasoning":true}}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "Deploy mainstream open-source models with a single click to enable private AI inference and fine-tuning, ensuring data privacy and local control."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/gitee_ai
 {
  "_system": {
             "owner_id": "$[[SETUP_OWNER_ID]]"
           },
   "id" : "gitee_ai",
   "created" : "2025-03-28T10:25:39.7741+08:00",
   "updated" : "2025-03-28T11:14:47.103278+08:00",
   "name" : "Gitee AI",
   "api_key" : "",
   "api_type" : "openai",
   "icon" : "font_gitee",
   "models" : [
     {
       "name" : "deepseek-ai/DeepSeek-R1",
        "settings" : {
          "reasoning" : true
        }
     },
     {
       "name" : "deepseek-ai/DeepSeek-V3",
       "settings" : {
          "reasoning" : false
       }
     },
     {
       "name" : "deepseek-ai/DeepSeek-R1-Distill-Qwen-7B",
        "settings" : {
          "reasoning" : true
        }
     }
   ],
   "base_url" : "https://ai.gitee.com",
   "enabled" : false,
   "builtin" : true,
   "description" : "Gitee AI brings together the latest and most popular AI models, offering a one-stop service for model experience, inference, fine-tuning, deployment, and application."
 }

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/qianwen
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "qianwen",
  "created" : "2025-03-28T10:25:39.7741+08:00",
  "updated" : "2025-03-28T11:14:47.103278+08:00",
  "name" : "Tongyi Qianwen",
  "api_key" : "",
  "api_type" : "openai",
  "icon" : "font_tongyiqianwenTongyi-Qianwen",
  "models" : [
    {
      "name" : "tongyi-intent-detect-v3",
      "settings" : {
        "reasoning" : false
      }
    },
    {
      "name" : "deepseek-r1-distill-qwen-32b",
       "settings" : {
        "reasoning" : true
      }
    },
    {
      "name" : "deepseek-r1",
       "settings" : {
        "reasoning" : true
      }
    },
     {
       "name" : "qwen-max",
        "settings" : {
          "reasoning" : false
        }
     },
     {
       "name" : "qwq-plus",
        "settings" : {
          "reasoning" : true
        }
     },
     {
       "name" : "qwen2.5-32b-instruct",
        "settings" : {
          "reasoning" : false
        }
     }
  ],
  "base_url" : "https://dashscope.aliyuncs.com/compatible-mode/v1",
  "enabled" : false,
  "builtin" : true,
  "description" : "Aliyun's self-developed Tongyi large model supports full-modal model service calls, offering powerful inference capabilities with high efficiency and low cost to meet a wide range of business scenarios."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/openai_compatible
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
    "id" : "openai_compatible",
    "created" : "2025-03-28T10:25:39.7741+08:00",
    "updated" : "2025-03-28T11:14:47.103278+08:00",
    "name" : "OpenAI-API-compatible",
    "api_key" : "",
    "api_type" : "openai",
    "icon" : "font_openai",
    "models" : [
      {
        "name" : "deepseek-r1",
        "settings" : {
          "reasoning" : true
        }
      }
    ],
    "base_url" : "",
    "enabled" : false,
    "builtin" : true,
    "description" : "A fully compatible alternative to OpenAI's API, offering lower-cost and higher-concurrency model calls, supporting private deployment and multi-model hosting."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/coco
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "coco",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "Coco AI",
  "api_key" : "$[[SETUP_LLM_API_KEY]]",
  "api_type" : "$[[SETUP_LLM_API_TYPE]]",
  "base_url" : "$[[SETUP_LLM_BASE_URL]]",
  "icon" : "font_coco",
  "models" : [
     $[[SETUP_LLM_DEFAULT_MODEL]]
  ],
  "enabled" : $[[SETUP_LLM_ENABLED]],
  "builtin" : true,
  "description": "Coco AI Custom Model Provider for Configuring Default AI Assistant."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/silicon_flow
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "silicon_flow",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "SiliconFlow",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://api.siliconflow.cn",
  "icon" : "font_siliconflow",
  "models" : [
    {"name": "BAAI/bge-m3", "settings":{"reasoning":false}},
    {"name": "deepseek-ai/DeepSeek-R1", "settings":{"reasoning":true}},
    {"name": "deepseek-ai/DeepSeek-V3", "settings":{"reasoning":false}}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "Siliconflow provides access to various models (LLM, text embedding, reordering, STT, TTS), which can be configured through model name, API key, and other parameters."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/tencent_hunyuan
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "tencent_hunyuan",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "Tencent Hunyuan",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://api.hunyuan.cloud.tencent.com",
  "icon" : "font_hunyuan",
  "models" : [
    {"name": "hunyuan-pro", "settings":{"reasoning":false}},
    {"name": "hunyuan-standard", "settings":{"reasoning":false}},
    {"name": "hunyuan-lite", "settings":{"reasoning":false}},
    {"name": "hunyuan-standard-256k", "settings":{"reasoning":false}},
    {"name": "hunyuan-vision", "settings":{"reasoning":false}},
    {"name": "hunyuan-code", "settings":{"reasoning":false}},
    {"name": "hunyuan-role", "settings":{"reasoning":false}},
    {"name": "hunyuan-turbo", "settings":{"reasoning":false}}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "Tencent Hunyuan provides models such as hunyuan-standard, hunyuan-standard-256k, hunyuan-pro, hunyuan-role..."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/gemini
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "gemini",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "Gemini",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://generativelanguage.googleapis.com",
  "icon" : "font_gemini-ai",
  "models" : [
    {"name": "gemini-2.0-flash", "settings":{"reasoning":false}},
    {"name": "gemini-1.5-flash", "settings":{"reasoning":false}},
    {"name": "gemini-1.5-pro", "settings":{"reasoning":true}}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "Google's Gemini model"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/moonshot
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "moonshot",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "Moonshot",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://api.moonshot.cn",
  "icon" : "font_Moonshot",
  "models" : [
    {"name": "moonshot-v1-auto", "settings":{"reasoning":true}}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "Moonshot provides models such as moonshot-v1-8k, moonshot-v1-32k, and moonshot-v1-128k."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/minimax
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "minimax",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "Minimax",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://api.minimax.chat/v1/",
  "icon" : "font_MiniMax",
  "models" : [
    {"name": "abab5.5s", "settings":{"reasoning":false}},
    {"name": "abab6.5s", "settings":{"reasoning":false}},
    {"name": "abab6.5g", "settings":{"reasoning":false}},
    {"name": "abab6.5t", "settings":{"reasoning":true}},
    {"name": "minimax-01", "settings":{"reasoning":true}}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "MiniMax is an advanced AI platform that provides a suite of powerful models designed for various applications, including LLMs."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/volcanoArk
{ "_system": {
             "owner_id": "$[[SETUP_OWNER_ID]]"
           },
  "id" : "volcanoArk",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "VolcanoArk",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://ark.cn-beijing.volces.com/api/v3/",
  "icon" : "font_VolcanoArk",
  "models" : [
    {"name": "doubao-1.5-vision-pro", "settings":{"reasoning":false}},
    {"name": "doubao-1.5-pro-32k", "settings":{"reasoning":false}},
    {"name": "doubao-1.5-pro-32k-character", "settings":{"reasoning":false}},
    {"name": "Doubao-1.5-pro-256k", "settings":{"reasoning":false}}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "The models provided by VolcanoArk, such as Doubao-pro-4k, Doubao-pro-32k, and Doubao-pro-128k."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/qianfan
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id" : "qianfan",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "Qianfan",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://qianfan.baidubce.com/v2/",
  "icon" : "font_Qianfan",
  "models" : [
    {"name": "ERNIE-4.0", "settings":{"reasoning":false}},
    {"name": "ERNIE 4.0 Trubo", "settings":{"reasoning":false}},
    {"name": "ERNlE Speed", "settings":{"reasoning":false}},
    {"name": "ERNIE Lite", "settings":{"reasoning":false}},
    {"name": "BGE Large ZH", "settings":{"reasoning":false}},
    {"name": "BGE Large EN", "settings":{"reasoning":false}}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "Pre-set the full series of Wenxin large models and over a hundred selected third-party models."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider$[[SETUP_SCHEMA_VER]]/$[[SETUP_DOC_TYPE]]/cohere
{
 "_system": {
            "owner_id": "$[[SETUP_OWNER_ID]]"
          },
  "id": "cohere",
  "created": "2025-03-28T11:30:00.000000+08:00",
  "updated": "2025-03-28T11:30:00.000000+08:00",
  "name": "Cohere",
  "api_key": "",
  "api_type": "cohere",
  "base_url": "https://api.cohere.ai/v1",
  "icon": "/assets/icons/llm/cohere.svg",
  "models": [
    {"name": "command-r-plus", "settings":{"reasoning":true}},
    {"name": "command-r", "settings":{"reasoning":true}},
    {"name": "embed-english-v3", "settings":{"reasoning":false}},
    {"name": "embed-multilingual-v3", "settings":{"reasoning":false}}
  ],
  "enabled": false,
  "builtin": true,
  "description": "Cohere provides advanced APIs for natural language understanding, generation, and embeddings (including the Command-R and Embed v3 series). These models are optimized for reasoning, retrieval-augmented generation (RAG), multilingual semantic search, and enterprise-scale AI applications, offering strong performance and cost-efficiency."
}