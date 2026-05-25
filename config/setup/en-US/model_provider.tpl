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
    {"name":"deepseek-v4-pro", "type":"language", "support_reasoning":true},
    {"name":"deepseek-v4-flash", "type":"language", "support_reasoning":true}
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
     {"name":"gpt-5.5", "type":"language", "support_reasoning":true},
     {"name":"gpt-5.5-pro", "type":"language", "support_reasoning":true},
     {"name":"gpt-5.4", "type":"language", "support_reasoning":true},
     {"name":"gpt-5.4-pro", "type":"language", "support_reasoning":true},
     {"name":"gpt-5.4-mini", "type":"language", "support_reasoning":true},
     {"name":"gpt-5.4-nano", "type":"language", "support_reasoning":true},
     {"name":"gpt-4.1", "type":"language"},
     {"name":"gpt-4o", "type":"language"},
     {"name":"gpt-4o-mini", "type":"language"},
     {"name":"text-embedding-3-large", "type":"embedding"},
     {"name":"text-embedding-3-small", "type":"embedding"},
     {"name":"text-embedding-ada-002", "type":"embedding"}
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
     {"name":"qwen2.5:32b", "type":"language"},
     {"name":"deepseek-r1:32b", "type":"language", "support_reasoning":true},
     {"name":"deepseek-r1:14b", "type":"language", "support_reasoning":true},
     {"name":"deepseek-r1:8b", "type":"language", "support_reasoning":true}
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
       "name" : "deepseek-ai/DeepSeek-R1", "type":"language",
       "support_reasoning" : true
     },
     {
       "name" : "deepseek-ai/DeepSeek-V3", "type":"language"
     },
     {
       "name" : "deepseek-ai/DeepSeek-R1-Distill-Qwen-7B", "type":"language",
       "support_reasoning" : true
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
    {"name":"qwen3.7-max", "type":"language", "support_reasoning":true},
    {"name":"qwen3.6-plus", "type":"language", "support_reasoning":true},
    {"name":"qwen3.6-flash", "type":"language", "support_reasoning":true},
    {"name":"qwen3.5-plus", "type":"language", "support_reasoning":true},
    {"name":"qwen3.5-flash", "type":"language", "support_reasoning":true},
    {"name":"deepseek-v4-pro", "type":"language", "support_reasoning":true},
    {"name":"deepseek-v4-flash", "type":"language", "support_reasoning":true},
    {"name":"kimi-k2.6", "type":"language", "support_reasoning":true},
    {"name":"glm-5.1", "type":"language", "support_reasoning":true},
    {"name":"MiniMax/MiniMax-M2.7", "type":"language", "support_reasoning":true},
    {"name":"mimo-v2.5-pro", "type":"language", "support_reasoning":true},
    {"name":"text-embedding-v4", "type":"embedding"}
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
        "name" : "deepseek-r1", "type":"language",
        "support_reasoning" : true
      }
    ],
    "base_url" : "",
    "enabled" : false,
    "builtin" : true,
    "description" : "A fully compatible alternative to OpenAI's API, offering lower-cost and higher-concurrency model calls, supporting private deployment and multi-model hosting."
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
    {"name": "BAAI/bge-m3", "type":"embedding"},
    {"name": "deepseek-ai/DeepSeek-R1", "type":"language", "support_reasoning":true},
    {"name": "deepseek-ai/DeepSeek-V3", "type":"language"}
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
    {"name": "hy3-preview", "type":"language", "support_reasoning":true},
    {"name": "hunyuan-t1-latest", "type":"language", "support_reasoning":true},
    {"name": "hunyuan-a13b", "type":"language", "support_reasoning":true},
    {"name": "hunyuan-turbos-latest", "type":"language"},
    {"name": "hunyuan-lite", "type":"language"},
    {"name": "hunyuan-role-latest", "type":"language"},
    {"name": "hy-mt2-pro", "type":"language"},
    {"name": "hunyuan-vision-1.5-instruct", "type":"vision"},
    {"name": "hunyuan-t1-vision-20250916", "type":"vision", "support_reasoning":true}
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
    {"name": "gemini-3.5-flash", "type":"language", "support_reasoning":true},
    {"name": "gemini-3.1-pro-preview", "type":"language", "support_reasoning":true},
    {"name": "gemini-3.1-flash-lite", "type":"language"},
    {"name": "gemini-2.5-pro", "type":"language", "support_reasoning":true},
    {"name": "gemini-2.5-flash", "type":"language", "support_reasoning":true},
    {"name": "gemini-2.5-flash-lite", "type":"language"},
    {"name": "gemini-embedding-2", "type":"embedding"},
    {"name": "gemini-embedding-001", "type":"embedding"}
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
    {"name": "kimi-k2.6", "type":"language", "support_reasoning":true},
    {"name": "kimi-k2.5", "type":"language", "support_reasoning":true},
    {"name": "moonshot-v1-128k", "type":"language"},
    {"name": "moonshot-v1-32k", "type":"language"},
    {"name": "moonshot-v1-8k", "type":"language"},
    {"name": "moonshot-v1-128k-vision-preview", "type":"vision"},
    {"name": "moonshot-v1-32k-vision-preview", "type":"vision"},
    {"name": "moonshot-v1-8k-vision-preview", "type":"vision"}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "Moonshot (Kimi) provides models such as kimi-k2.6, moonshot-v1-8k, moonshot-v1-32k, and moonshot-v1-128k."
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
    {"name": "MiniMax-M2.7", "type":"language", "support_reasoning":true},
    {"name": "MiniMax-M2.7-highspeed", "type":"language", "support_reasoning":true},
    {"name": "MiniMax-M2.5", "type":"language", "support_reasoning":true},
    {"name": "MiniMax-M2.5-highspeed", "type":"language", "support_reasoning":true},
    {"name": "M2-her", "type":"language"}
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
    {"name": "doubao-1.5-vision-pro", "type":"vision"},
    {"name": "doubao-1.5-pro-32k", "type":"language"},
    {"name": "doubao-1.5-pro-32k-character", "type":"language"},
    {"name": "Doubao-1.5-pro-256k", "type":"language"}
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
    {"name": "ernie-5.1", "type":"language", "support_reasoning":true},
    {"name": "ernie-5.0", "type":"language", "support_reasoning":true},
    {"name": "ernie-x1.1", "type":"language", "support_reasoning":true},
    {"name": "ernie-x1-turbo", "type":"language", "support_reasoning":true},
    {"name": "ernie-4.5-turbo", "type":"language"},
    {"name": "ernie-4.5-turbo-vl", "type":"vision"},
    {"name": "ernie-4.5", "type":"language"},
    {"name": "bge-large-zh", "type":"embedding"},
    {"name": "bge-large-en", "type":"embedding"},
    {"name": "embedding", "type":"embedding"}
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
  "api_type": "openai",
  "base_url": "https://api.cohere.ai/compatibility/v1",
  "icon": "/assets/icons/llm/cohere.svg",
  "models": [
    {"name": "command-r-plus", "type":"language", "support_reasoning":true},
    {"name": "command-r", "type":"language", "support_reasoning":true},
    {"name": "embed-english-v3", "type":"embedding"},
    {"name": "embed-multilingual-v3", "type":"embedding"}
  ],
  "enabled": false,
  "builtin": true,
  "description": "Cohere provides advanced APIs for natural language understanding, generation, and embeddings (including the Command-R and Embed v3 series). These models are optimized for reasoning, retrieval-augmented generation (RAG), multilingual semantic search, and enterprise-scale AI applications, offering strong performance and cost-efficiency."
}
