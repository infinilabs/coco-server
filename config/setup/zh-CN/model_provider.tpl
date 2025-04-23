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
     {"name":"gpt-4o-mini"},
     {"name":"deepseek-r1"}
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