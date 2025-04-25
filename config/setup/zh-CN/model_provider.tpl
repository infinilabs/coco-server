# model provider
POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/deepseek
{
  "id" : "deepseek",
  "created" : "2025-03-28T10:25:39.7741+08:00",
  "updated" : "2025-03-28T11:14:47.103278+08:00",
  "name" : "深度求索",
  "api_key" : "",
  "api_type" : "deepseek",
  "base_url" : "https://api.deepseek.com/v1",
  "icon" : "font_deepseek",
  "models" : [
    {"name":"deepseek-chat"},
    {"name":"deepseek-reasoner"}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "提供高效灵活的大模型API服务，支持复杂场景任务，具备高性价比优势。"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/openai
{
  "id" : "openai",
  "created" : "2025-03-28T10:24:37.843478+08:00",
  "updated" : "2025-03-31T20:16:18.517692+08:00",
  "name" : "OpenAI",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://api.openai.com",
  "icon" : "/assets/icons/llm/openai.svg",
  "models" : [
     {"name":"gpt-4o-mini"},
     {"name":"gpt-4o"}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "提供先进的GPT系列大模型（如GPT-4/ChatGPT），支持多模态交互与企业级AI解决方案，具备成熟的API生态与顶尖的通用智能表现。"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/ollama
{
  "id" : "ollama",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "Ollama",
  "api_key" : "",
  "api_type" : "ollama",
  "base_url" : "http://127.0.0.1:11434",
  "icon" : "/assets/icons/llm/ollama.svg",
  "models" : [
       {"name":"qwen2.5:32b"},
       {"name":"deepseek-r1:32b"},
       {"name":"deepseek-r1:14b"},
       {"name":"deepseek-r1:8b"}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "一键部署主流开源模型，实现私有化 AI 推理与微调，保障数据隐私与本地化控制。"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/gitee_ai
 {
   "id" : "gitee_ai",
   "created" : "2025-03-28T10:25:39.7741+08:00",
   "updated" : "2025-03-28T11:14:47.103278+08:00",
   "name" : "模力方舟",
   "api_key" : "",
   "api_type" : "openai",
   "icon" : "font_gitee",
   "models" : [
     {
       "name" : "deepseek-ai/DeepSeek-R1"
     },
     {
       "name" : "deepseek-ai/DeepSeek-V3"
     },
     {
       "name" : "deepseek-ai/DeepSeek-R1-Distill-Qwen-7B"
     }
   ],
   "base_url" : "https://ai.gitee.com",
   "enabled" : false,
   "builtin" : true,
   "description" : "模力方舟（Gitee AI），汇聚了最新最热的 AI 模型，提供模型体验、推理、微调、部署和应用的一站式服务。"
 }

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/qianwen
{
  "id" : "qianwen",
  "created" : "2025-03-28T10:25:39.7741+08:00",
  "updated" : "2025-03-28T11:14:47.103278+08:00",
  "name" : "通义千问",
  "api_key" : "",
  "api_type" : "openai",
  "icon" : "font_tongyiqianwenTongyi-Qianwen",
  "models" : [
    {
      "name" : "tongyi-intent-detect-v3"
    },
    {
      "name" : "deepseek-r1-distill-qwen-32b"
    },
    {
      "name" : "deepseek-r1"
    },
    {
      "name" : "qwen-max"
    },
    {
      "name" : "qwq-plus"
    }
  ],
  "base_url" : "https://dashscope.aliyuncs.com/compatible-mode/v1",
  "enabled" : false,
  "builtin" : true,
  "description" : "阿里云自研的通义大模型，支持全模态模型服务调用，强推理高效率低成本，满足更多业务场景。"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/openai_compatible
{
    "id" : "openai_compatible",
    "created" : "2025-03-28T10:25:39.7741+08:00",
    "updated" : "2025-03-28T11:14:47.103278+08:00",
    "name" : "OpenAI-API-compatible",
    "api_key" : "",
    "api_type" : "openai",
    "icon" : "font_openai",
    "models" : [
      {
        "name" : "deepseek-r1"
      }
    ],
    "base_url" : "",
    "enabled" : false,
    "builtin" : true,
    "description" : "全兼容 OpenAI API 接口的替代方案，提供更低成本/更高并发的模型调用，支持私有化部署与多模型托管。"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/coco
{
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
  "description": "Coco AI 自定义模型提供商，用于配置默认 AI 助手"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/silicon_flow
{
  "id" : "silicon_flow",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "硅基流动",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://api.siliconflow.cn",
  "icon" : "font_siliconflow",
  "models" : [
    {"name": "BAAI/bge-m3"},
    {"name": "deepseek-ai/DeepSeek-R1"},
    {"name": "deepseek-ai/DeepSeek-V3"}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "硅基流动提供对各种模型（LLM、文本嵌入、重排序、STT、TTS）的访问，可通过模型名称、API密钥和其他参数进行配置"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/tencent_hunyuan
{
  "id" : "tencent_hunyuan",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "腾讯混元",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://api.hunyuan.cloud.tencent.com",
  "icon" : "font_hunyuan",
  "models" : [
    {"name": "hunyuan-pro"},
    {"name": "hunyuan-standard"},
    {"name": "hunyuan-lite"},
    {"name": "hunyuan-standard-256k"},
    {"name": "hunyuan-vision"},
    {"name": "hunyuan-code"},
    {"name": "hunyuan-role"},
    {"name": "hunyuan-turbo"}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "腾讯混元提供的模型，例如 hunyuan-standard、 hunyuan-standard-256k, hunyuan-pro, hunyuan-role…"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/gemini
{
  "id" : "gemini",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "Gemini",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://generativelanguage.googleapis.com",
  "icon" : "font_gemini-ai",
  "models" : [
    {"name": "gemini-1.5-flash"},
    {"name": "gemini-1.5-pro"}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "谷歌提供的 Gemini 模型"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/moonshot
{
  "id" : "moonshot",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "月之暗面",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://api.moonshot.cn",
  "icon" : "font_Moonshot",
  "models" : [
    {"name": "moonshot-v1-auto"}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "Moonshot 提供的模型，例如 moonshot-v1-8k、moonshot-v1-32k 和 moonshot-v1-128k。"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/minimax
{
  "id" : "minimax",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "Minimax",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://api.minimax.chat/v1/",
  "icon" : "font_MiniMax",
  "models" : [
    {"name": "abab5.5s"},
    {"name": "abab6.5s"},
    {"name": "abab6.5g"},
    {"name": "abab6.5t"},
    {"name": "minimax-01"}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "MiniMax 是一个先进的AI平台，提供一系列为各种应用设计的强大模型，包括LLMs。"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/volcanoArk
{
  "id" : "volcanoArk",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "火山方舟",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://ark.cn-beijing.volces.com/api/v3/",
  "icon" : "font_VolcanoArk",
  "models" : [
    {"name": "doubao-1.5-vision-pro"},
    {"name": "doubao-1.5-pro-32k"},
    {"name": "doubao-1.5-pro-32k-character"},
    {"name": "Doubao-1.5-pro-256k"}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "火山方舟提供的模型，例如 Doubao-pro-4k、Doubao-pro-32k 和 Doubao-pro-128k。"
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/qianfan
{
  "id" : "qianfan",
  "created" : "2025-03-28T10:24:22.378929+08:00",
  "updated" : "2025-03-28T11:22:57.605814+08:00",
  "name" : "百度云千帆",
  "api_key" : "",
  "api_type" : "openai",
  "base_url" : "https://qianfan.baidubce.com/v2/",
  "icon" : "font_Qianfan",
  "models" : [
    {"name": "ERNIE-4.0"},
    {"name": "ERNIE 4.0 Trubo"},
    {"name": "ERNlE Speed"},
    {"name": "ERNIE Lite"},
    {"name": "BGE Large ZH"},
    {"name": "BGE Large EN"}
  ],
  "enabled" : false,
  "builtin" : true,
  "description": "预置全系列文心大模型与上百个精选第三方模型"
}