#// Scenario 3:
#//
#// Users with [edit] permission can modify assistant settings.


#//----------------------------------------------------------------------------
#//
#// Login
#//
#//----------------------------------------------------------------------------


#// 1
#//
#// Log in to account admin
POST /account/login
{
  "email": "admin@mail.com",
  "password": "$[[env.ADMIN_PASSWORD]]"
}
# assert: (200, {status: "ok"}),
#
# register: [
#   { admin_token: "_ctx.response.body_json.access_token" },
# ]


#// 2
#//
#// Log in to account a
POST /account/login
{
  "email": "a@mail.com",
  "password": "$[[env.A_PASSWORD]]"
}
# assert: (200, {status: "ok"}),
#
# register: [
#   { a_token: "_ctx.response.body_json.access_token" },
# ]


#// 3
#//
#// Log in to account b
POST /account/login
{
  "email": "b@mail.com",
  "password": "$[[env.B_PASSWORD]]"
}
# assert: (200, {status: "ok"}),
#
# register: [
#   { b_token: "_ctx.response.body_json.access_token" },
# ]


#// 4
#//
#// Log in to account c
POST /account/login
{
  "email": "c@mail.com",
  "password": "$[[env.C_PASSWORD]]"
}
# assert: (200, {status: "ok"}),
#
# register: [
#   { c_token: "_ctx.response.body_json.access_token" },
# ]


#// 5
#//
#// Log in to account d
POST /account/login
{
  "email": "d@mail.com",
  "password": "$[[env.D_PASSWORD]]"
}
# assert: (200, {status: "ok"}),
#
# register: [
#   { d_token: "_ctx.response.body_json.access_token" },
# ]


#//----------------------------------------------------------------------------
#//
#// Case 1:
#//  1. User [admin] grants user [a] [edit] permission to assistant [aichat_a]
#//  2. User [admin] grants user [b] [view] permission to assistant [aichat_a]
#//  3. User [admin] grants user [c] [share] permission to assistant [aichat_a]
#//  4. Users [a, b, c] have the corresponding permission
#//  5. [aichat_a] is invisible to user [d]
#//
#//----------------------------------------------------------------------------


#// 6
#//
#// User [admin] grants user [a] [edit] permission to assistant [aichat_a]
POST /resources/assistant/$[[env.AICHAT_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "assistant",
      "resource_id": "$[[env.AICHAT_A_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.A_ID]]",
      "permission": $[[env.EDIT_ACCESS]]
    }
  ],
  "revokes": []
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "created": [
#     { 
#       "resource_id": "$[[env.AICHAT_A_ID]]", 
#       "principal_id": "$[[env.A_ID]]", 
#       "permission": $[[env.EDIT_ACCESS]] 
#     } 
#   ]
# })



#// 7
#//
#// User [admin] grants user [b] [view] permission to assistant [aichat_a]
POST /resources/assistant/$[[env.AICHAT_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "assistant",
      "resource_id": "$[[env.AICHAT_A_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.B_ID]]",
      "permission": $[[env.VIEW_ACCESS]]
    }
  ],
  "revokes": []
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "created": [
#     { 
#       "resource_id": "$[[env.AICHAT_A_ID]]", 
#       "principal_id": "$[[env.B_ID]]", 
#       "permission": $[[env.VIEW_ACCESS]] 
#     } 
#   ]
# })


#// 8
#//
#// User [admin] grants user [c] [share] permission to assistant [aichat_a]
POST /resources/assistant/$[[env.AICHAT_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "assistant",
      "resource_id": "$[[env.AICHAT_A_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.C_ID]]",
      "permission": $[[env.SHARE_ACCESS]]
    }
  ],
  "revokes": []
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "created": [
#     { 
#       "resource_id": "$[[env.AICHAT_A_ID]]", 
#       "principal_id": "$[[env.C_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]] 
#     } 
#   ]
# })



#// 9
#//
#// Verify that users [a, b, c] have the corresponding permission
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.AICHAT_A_ID]]",
    "resource_type": "assistant"
  }
]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, [
#   {
#     "resource_id": "$[[env.AICHAT_A_ID]]",
#     "principal_id": "$[[env.A_ID]]",
#     "permission": $[[env.EDIT_ACCESS]]
#   },
#   {
#     "resource_id": "$[[env.AICHAT_A_ID]]",
#     "principal_id": "$[[env.B_ID]]",
#     "permission": $[[env.VIEW_ACCESS]]
#   },
#   {
#     "resource_id": "$[[env.AICHAT_A_ID]]",
#     "principal_id": "$[[env.C_ID]]",
#     "permission": $[[env.SHARE_ACCESS]]
#   }
# ],
# )


#// 10
#//
#// User [a] can create a new conversation with [aichat_a]
POST /chat/_create?search=false&deep_thinking=false&mcp=false&datasource=hacker_news&assistant_id=$[[env.AICHAT_A_ID]]
{"message":"hello"}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: { "_ctx.response.status": 200 }


#// 11
#//
#// User [b] can create a new conversation with [aichat_a]
POST /chat/_create?search=false&deep_thinking=false&mcp=false&datasource=hacker_news&assistant_id=$[[env.AICHAT_A_ID]]
{"message":"hello"}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[b_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: { "_ctx.response.status": 200 }


#// 12
#//
#// User [c] can create a new conversation with [aichat_a]
POST /chat/_create?search=false&deep_thinking=false&mcp=false&datasource=hacker_news&assistant_id=$[[env.AICHAT_A_ID]]
{"message":"hello"}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[c_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: { "_ctx.response.status": 200 }


#// 13
#//
#// User [d] cannot see assistant [aichat_a]
GET /assistant/_search?&from=0&size=100&query=&t=1763461094530
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "hits.total.value": 0
# })


#//----------------------------------------------------------------------------
#//
#// Case 2:
#//  1. User [a] renames [aichat_a] [aichat_1]
#//  2. User [b, c] can see assistant [aichat_1]
#//  3. User [d] cannot see assistant [aichat_1]
#//
#//----------------------------------------------------------------------------

#// 14
#//
#// User [a] renames [aichat_a] [aichat_1]
PUT /assistant/$[[env.AICHAT_A_ID]]
{
  "id": "$[[env.AICHAT_A_ID]]",
  "name": "aichat_1",
  "description": "",
  "icon": "font_coco",
  "category": "",
  "type": "simple",
  "answering_model": {
    "provider_id": "deepseek",
    "name": "deepseek-chat",
    "settings": {
      "reasoning": false,
      "temperature": 0.7,
      "top_p": 0.9,
      "presence_penalty": 0,
      "frequency_penalty": 0,
      "max_tokens": 4000,
      "max_length": 0
    },
    "prompt": {
      "template": "You are a helpful AI assistant.\n  You will be given a conversation below and a follow-up question.\n\n  {{.context}}\n\n  The user has provided the following query:\n  {{.query}}\n\n  Ensure your response is thoughtful, accurate, and well-structured.\n  For complex answers, format your response using clear and well-organized **Markdown** to improve readability.",
      "input_vars": null
    },
    "id": "deepseek_deepseek-chat"
  },
  "role_prompt": "You are a personal AI assistant designed by Coco AI(https://coco.rs), the backend team is behind INFINI Labs(https://infinilabs.com).",
  "chat_settings": {
    "greeting_message": "Hi! I’m Coco, nice to meet you. I can help answer your questions by tapping into the internet and your data sources. How can I assist you today?",
    "suggested": {
      "enabled": false,
      "questions": []
    },
    "placeholder": "",
    "history_message": {
      "number": 5,
      "compression_threshold": 1000,
      "summary": true
    }
  },
  "enabled": true,
  "datasource": {
    "enabled": true,
    "ids": [
      "*"
    ],
    "visible": true,
    "enabled_by_default": false,
    "filter": null
  },
  "mcp_servers": {
    "enabled": true,
    "ids": [
      "*"
    ],
    "visible": true,
    "model": null,
    "max_iterations": 5,
    "enabled_by_default": false
  },
  "upload": {
    "enabled": false,
    "allowed_file_extensions": [
      "*"
    ],
    "max_file_size_in_bytes": 1048576,
    "max_file_count": 6
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
  "keepalive": "30m"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"_id":"$[[env.AICHAT_A_ID]]","result":"updated"} )


#// 15
#//
#// User [b] can see assistant [aichat_1]
GET /assistant/_search?&from=0&size=100&query=&t=1763461094530
# request: {
#   headers: [
#     {Authorization: "Bearer $[[b_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "hits.total.value": 1,
#   "hits.hits": [
#     { "_source.name": "aichat_1" }
#   ]
# })


#// 16
#//
#// User [c] can see assistant [aichat_1]
GET /assistant/_search?&from=0&size=100&query=&t=1763461094530
# request: {
#   headers: [
#     {Authorization: "Bearer $[[c_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "hits.total.value": 1,
#   "hits.hits": [
#     { "_source.name": "aichat_1" }
#   ]
# })

#// 17
#//
#// User [d] cannot see assistant [aichat_a]
GET /assistant/_search?&from=0&size=100&query=&t=1763461094530
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "hits.total.value": 0
# })


#//----------------------------------------------------------------------------
#//
#// Case 3:
#//  1. User [admin] deletes [aichat_1]
#//  2. [aichat_1] is invisible to users [b, c, d]
#//
#//----------------------------------------------------------------------------

#// 18
#//
#// User [admin] deletes [aichat_1]
DELETE /assistant/$[[env.AICHAT_A_ID]]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"_id":"$[[env.AICHAT_A_ID]]","result":"deleted"})


#// 17
#//
#// User [a] cannot see assistant [aichat_1]
GET /assistant/_search?&from=0&size=100&query=&t=1763461094530
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "hits.total.value": 0
# })


#// 18
#//
#// User [b] cannot see assistant [aichat_1]
GET /assistant/_search?&from=0&size=100&query=&t=1763461094530
# request: {
#   headers: [
#     {Authorization: "Bearer $[[b_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "hits.total.value": 0
# })


#// 19
#//
#// User [c] cannot see assistant [aichat_1]
GET /assistant/_search?&from=0&size=100&query=&t=1763461094530
# request: {
#   headers: [
#     {Authorization: "Bearer $[[c_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "hits.total.value": 0
# })


#// 20
#//
#// User [d] cannot see assistant [aichat_1]
GET /assistant/_search?&from=0&size=100&query=&t=1763461094530
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "hits.total.value": 0
# })