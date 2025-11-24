#// Scenario 3:
#//
#// Users with [edit] permission can modify LLM provider settings


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
#//  1. User [admin] grants user [a] [edit] permission to [llm_a]
#//  2. User [admin] grants user [b] [view] permission to [llm_a]
#//  3. User [admin] grants user [c] [share] permission to [llm_a]
#//  4. Users [a, b, c] have the corresponding permission
#//  5. [llm_a] is invisible to user [d]
#//
#//----------------------------------------------------------------------------


#// 6
#//
#// User [admin] grants user [a] [edit] permission to [llm_a]
POST /resources/llm-provider/$[[env.LLM_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "llm-provider",
      "resource_id": "$[[env.LLM_A_ID]]",
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
#       "resource_id": "$[[env.LLM_A_ID]]",
#       "principal_id": "$[[env.A_ID]]",
#       "permission": $[[env.EDIT_ACCESS]]
#     }
#   ]
# })


#// 7
#//
#// User [admin] grants user [b] [view] permission to [llm_a]
POST /resources/llm-provider/$[[env.LLM_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "llm-provider",
      "resource_id": "$[[env.LLM_A_ID]]",
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
#       "resource_id": "$[[env.LLM_A_ID]]",
#       "principal_id": "$[[env.B_ID]]",
#       "permission": $[[env.VIEW_ACCESS]]
#     }
#   ]
# })


#// 8
#//
#// User [admin] grants user [c] [share] permission to [llm_a]
POST /resources/llm-provider/$[[env.LLM_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "llm-provider",
      "resource_id": "$[[env.LLM_A_ID]]",
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
#       "resource_id": "$[[env.LLM_A_ID]]",
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
    "resource_id": "$[[env.LLM_A_ID]]",
    "resource_type": "llm-provider"
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
#    "resource_id": "$[[env.LLM_A_ID]]",
#    "principal_id": "$[[env.A_ID]]",
#    "permission": $[[env.EDIT_ACCESS]]
#   },
#   {
#    "resource_id": "$[[env.LLM_A_ID]]",
#    "principal_id": "$[[env.B_ID]]",
#    "permission": $[[env.VIEW_ACCESS]]
#   },
#   {
#    "resource_id": "$[[env.LLM_A_ID]]",
#    "principal_id": "$[[env.C_ID]]",
#    "permission": $[[env.SHARE_ACCESS]]
#   }
# ])


#// 10
#//
#// User [d] cannot see llm provider [llm_a]
GET /model_provider/_search?&from=0&size=100&query=llm_a&t=1763461094530
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
#//  1. User [a] renames [llm_a] [llms_1]
#//  2. User [b, c] can see [llms_1]
#//  3. User [d] cannot see [llms_1]
#//
#//----------------------------------------------------------------------------


#// 11
#//
#// User [a] renames [llm_a] [llms_1]
PUT /model_provider/$[[env.LLM_A_ID]]
{
  "name": "llms_1",
  "icon": "font_Qianfan",
  "api_type": "ollama",
  "api_key": "",
  "base_url": "http://s",
  "models": [
    {
      "name": "s",
      "settings": {
        "reasoning": false,
        "temperature": 0.7,
        "top_p": 0.9,
        "presence_penalty": 0,
        "frequency_penalty": 0,
        "max_tokens": 4000,
        "max_length": 0
      }
    }
  ],
  "description": "",
  "enabled": true
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"_id":"$[[env.LLM_A_ID]]","result":"updated"})


#// 12
#//
#// User [b] can see [llms_1]
GET /model_provider/_search?&from=0&size=100&query=llms_1&t=1763461094531
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
#     { "_source.name": "llms_1" }
#   ]
# })


#// 13
#//
#// User [c] can see [llms_1]
GET /model_provider/_search?&from=0&size=100&query=llms_1&t=1763461094532
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
#     { "_source.name": "llms_1" }
#   ]
# })


#// 14
#//
#// User [d] cannot see [llms_1]
GET /model_provider/_search?&from=0&size=100&query=llms_1&t=1763461094533
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
#//  1. User [admin] deletes [llms_1]
#//  2. [llms_1] is invisible to users [a, b, c, d]
#//
#//----------------------------------------------------------------------------


#// 15
#//
#// User [admin] deletes [llms_1]
DELETE /model_provider/$[[env.LLM_A_ID]]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"_id":"$[[env.LLM_A_ID]]","result":"deleted"})


#// 16
#//
#// User [a] cannot see [llms_1]
GET /model_provider/_search?&from=0&size=100&query=llms_1&t=1763461094534
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


#// 17
#//
#// User [b] cannot see [llms_1]
GET /model_provider/_search?&from=0&size=100&query=llms_1&t=1763461094535
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


#// 18
#//
#// User [c] cannot see [llms_1]
GET /model_provider/_search?&from=0&size=100&query=llms_1&t=1763461094536
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


#// 19
#//
#// User [d] cannot see [llms_1]
GET /model_provider/_search?&from=0&size=100&query=llms_1&t=1763461094537
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
