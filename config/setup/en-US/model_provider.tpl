# model provider
POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/deepseek
{
  "id" : "deepseek",
  "created" : "2025-03-28T10:25:39.7741+08:00",
  "updated" : "2025-03-28T11:14:47.103278+08:00",
  "name" : "Deepseek",
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
  "description": "Provide efficient and flexible large model API services, supporting complex tasks with a high cost-performance advantage."
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
  "description": "Provide advanced GPT-series large model API services (such as GPT-4/ChatGPT), supporting multimodal interactions and enterprise-level AI solutions, with a mature API ecosystem and top-tier general intelligence performance."
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
  "description": "Deploy mainstream open-source models with a single click to enable private AI inference and fine-tuning, ensuring data privacy and local control."
}

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/gitee_ai
 {
   "id" : "gitee_ai",
   "created" : "2025-03-28T10:25:39.7741+08:00",
   "updated" : "2025-03-28T11:14:47.103278+08:00",
   "name" : "Gitee AI",
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
   "description" : "Gitee AI brings together the latest and most popular AI models, offering a one-stop service for model experience, inference, fine-tuning, deployment, and application."
 }

POST $[[SETUP_INDEX_PREFIX]]model-provider/$[[SETUP_DOC_TYPE]]/qianwen
{
  "id" : "qianwen",
  "created" : "2025-03-28T10:25:39.7741+08:00",
  "updated" : "2025-03-28T11:14:47.103278+08:00",
  "name" : "Tongyi Qianwen",
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
  "description" : "Aliyun's self-developed Tongyi large model supports full-modal model service calls, offering powerful inference capabilities with high efficiency and low cost to meet a wide range of business scenarios."
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
    "description" : "A fully compatible alternative to OpenAI's API, offering lower-cost and higher-concurrency model calls, supporting private deployment and multi-model hosting."
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
  "description": "Coco AI Custom Model Provider for Configuring Default AI Assistant."
}