#// Scenario 1:
#//
#// Owner (admin) grants other users permission, then other users will have
#// the corresponding permission.


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
#// Test
#//
#//----------------------------------------------------------------------------


#// 6
#//
#// User [admin] grants user [a] [view] permission to assistant [aichat_a]
POST /resources/assistant/$[[env.AICHAT_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "assistant",
      "resource_id": "$[[env.AICHAT_A_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.A_ID]]",
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
#       "principal_id": "$[[env.A_ID]]", 
#       "permission": $[[env.VIEW_ACCESS]] 
#     } 
#   ]
# })


#// 7
#//
#// User [admin] grants user [b] [edit] permission to assistant [aichat_b]
POST /resources/assistant/$[[env.AICHAT_B_ID]]/share
{
  "shares": [
    {
      "resource_type": "assistant",
      "resource_id": "$[[env.AICHAT_B_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.B_ID]]",
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
#       "resource_id": "$[[env.AICHAT_B_ID]]", 
#       "principal_id": "$[[env.B_ID]]", 
#       "permission": $[[env.EDIT_ACCESS]] 
#     } 
#   ]
# })


#// 8
#//
#// User [admin] grants user [c] [share] permission to assistant [aichat_c]
POST /resources/assistant/$[[env.AICHAT_C_ID]]/share
{
  "shares": [
    {
      "resource_type": "assistant",
      "resource_id": "$[[env.AICHAT_C_ID]]",
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
#       "resource_id": "$[[env.AICHAT_C_ID]]", 
#       "principal_id": "$[[env.C_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]] 
#     } 
#   ]
# })

#// 9
#//
#// User [admin] grants user [d] [view] permission to assistant [aichat_d]
POST /resources/assistant/$[[env.AICHAT_D_ID]]/share
{
  "shares": [
    {
      "resource_type": "assistant",
      "resource_id": "$[[env.AICHAT_D_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.D_ID]]",
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
#       "resource_id": "$[[env.AICHAT_D_ID]]", 
#       "principal_id": "$[[env.D_ID]]", 
#       "permission": $[[env.VIEW_ACCESS]] 
#     } 
#   ]
# })

#// 10
#//
#// User [a] can see assistant [aichat_a]
GET /assistant/_search?&from=0&size=100&query=&t=1763461094530
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "hits.total.value": 1,
#   "hits.hits": [
#     { "_source.name": "$[[env.AICHAT_A_NAME]]" }  
#   ]
# })


#// 11
#//
#// User [b] can see assistant [aichat_b]
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
#     { "_source.name": "$[[env.AICHAT_B_NAME]]" }  
#   ]
# })


#// 12
#//
#// User [c] can see assistant [aichat_c]
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
#     { "_source.name": "$[[env.AICHAT_C_NAME]]" }  
#   ]
# })


#// 13
#//
#// User [d] can see assistant [aichat_d]
GET /assistant/_search?&from=0&size=100&query=&t=1763461094530
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "hits.total.value": 1,
#   "hits.hits": [
#     { "_source.name": "$[[env.AICHAT_D_NAME]]" }  
#   ]
# })

#// 14
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


#// 15
#//
#// User [b] can create a new conversation with [aichat_b]
POST /chat/_create?search=false&deep_thinking=false&mcp=false&datasource=hacker_news&assistant_id=$[[env.AICHAT_B_ID]]
{"message":"hello"}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[b_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: { "_ctx.response.status": 200 }


#// 16
#//
#// User [c] can create a new conversation with [aichat_c]
POST /chat/_create?search=false&deep_thinking=false&mcp=false&datasource=hacker_news&assistant_id=$[[env.AICHAT_C_ID]]
{"message":"hello"}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[c_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: { "_ctx.response.status": 200 }


#// 17
#//
#// User [d] can create a new conversation with [aichat_d]
POST /chat/_create?search=false&deep_thinking=false&mcp=false&datasource=hacker_news&assistant_id=$[[env.AICHAT_D_ID]]
{"message":"hello"}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: { "_ctx.response.status": 200 }