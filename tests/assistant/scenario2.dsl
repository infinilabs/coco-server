#// Scenario 2:
#//
#// Users with [share] permission can grant other users permission.


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
#//  1. User [admin] grants user [a] [share] permission to assistant [aichat_a]
#//  2. User [a] has that [share] permission to it and can create conversation 
#//     with it
#//
#//----------------------------------------------------------------------------


#// 6
#//
#// User [admin] grants user [a] [share] permission to assistant [aichat_a]
POST /resources/assistant/$[[env.AICHAT_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "assistant",
      "resource_id": "$[[env.AICHAT_A_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.A_ID]]",
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
#       "principal_id": "$[[env.A_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]] 
#     } 
#   ]
# })


#// 7
#//
#// Verify that user [a] has [share] permission to [aichat_a]
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.AICHAT_A_ID]]",
    "resource_type": "assistant"
  }
]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, [{
#   "resource_id": "$[[env.AICHAT_A_ID]]",
#   "principal_id": "$[[env.A_ID]]",
#   "permission": $[[env.SHARE_ACCESS]]
#   }],
# )


#// 8
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


#//----------------------------------------------------------------------------
#//
#// Case 2:
#//  1. User [a] grants user [b] [edit] permission to assistant [aichat_a]
#//  2. User [a] grants user [c] [share] permission to assistant [aichat_a]
#//  3. Verify that user [b,c] have the corresponding permission
#//  4. Verify that [aichat_a] is invisible to user [d]
#//
#//----------------------------------------------------------------------------


#// 9
#//
#// User [a] grants user [b] [edit] permission to assistant [aichat_a]
POST /resources/assistant/$[[env.AICHAT_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "assistant",
      "resource_id": "$[[env.AICHAT_A_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.B_ID]]",
      "permission": $[[env.EDIT_ACCESS]]
    }
  ],
  "revokes": []
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# register: [
#   { grant_user_a_assistant_a_user_b: "_ctx.response.body_json.created.0.id" },
# ],
#
# assert: (200, { 
#   "created": [
#     { 
#       "resource_id": "$[[env.AICHAT_A_ID]]", 
#       "principal_id": "$[[env.B_ID]]", 
#       "permission": $[[env.EDIT_ACCESS]] 
#     } 
#   ]
# })


#// 10
#//
#// User [a] grants user [c] [share] permission to assistant [aichat_a]
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
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# register: [
#   { grant_user_a_assistant_a_user_c: "_ctx.response.body_json.created.0.id" },
# ],
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


#// 11
#//
#// Verify that user [b, c] have the corresponding permission to [aichat_a]
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
#     "permission": $[[env.SHARE_ACCESS]]
#   },
#   {
#     "resource_id": "$[[env.AICHAT_A_ID]]",
#     "principal_id": "$[[env.B_ID]]",
#     "permission": $[[env.EDIT_ACCESS]]
#   },
#   {
#     "resource_id": "$[[env.AICHAT_A_ID]]",
#     "principal_id": "$[[env.C_ID]]",
#     "permission": $[[env.SHARE_ACCESS]]
#   }
# ]
# )


#// 12
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


#// 13
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


#// 14
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
#//  1. User [c] grants user [a] [edit] permission to assistant [aichat_a]
#//  3. Verify the permissions granted on [aichat_a]:
#//     a: edit
#//     b: edit
#//     c: share
#//  4. Verify that [aichat_a] is invisible to user [d]
#//
#//----------------------------------------------------------------------------


#// 15
#//
#// User [c] grants user [a] [edit] permission to assistant [aichat_a]
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
#     {Authorization: "Bearer $[[c_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# register: [
#   { grant_user_c_assistant_a_user_a: "_ctx.response.body_json.updated.0.id" },
# ],
#
# sleep: {
#   "sleep_in_milli_seconds": 1000
# },
#
# assert: (200, { 
#   "updated": [
#     { 
#       "resource_id": "$[[env.AICHAT_A_ID]]", 
#       "principal_id": "$[[env.A_ID]]", 
#       "permission": $[[env.EDIT_ACCESS]] 
#     } 
#   ]
# })


#// 16
#//
#// Verify the permissions granted on [aichat_a]
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
#     "principal_id": "$[[env.B_ID]]",
#     "permission": $[[env.EDIT_ACCESS]]
#   },
#   {
#     "resource_id": "$[[env.AICHAT_A_ID]]",
#     "principal_id": "$[[env.C_ID]]",
#     "permission": $[[env.SHARE_ACCESS]]
#   },
#   {
#     "resource_id": "$[[env.AICHAT_A_ID]]",
#     "principal_id": "$[[env.A_ID]]",
#     "permission": $[[env.EDIT_ACCESS]]
#   }
# ]
# )


#// 17
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


#// 18
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


#// 19
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
#// Case 4:
#//  1. User [admin] revokes the permissions to [aichat_a] from user [a, b, c]
#//  2. Verify that users [a, b, c, d] cannot see the [aichat_a]
#//
#//----------------------------------------------------------------------------

#// 20
#//
#// User [admin] revokes the permissions to [aichat_a] from user [a, b, c]
POST /resources/assistant/$[[env.AICHAT_A_ID]]/share
{
  "revokes": [
    {
      "id": "$[[grant_user_a_assistant_a_user_b]]",
      "principal_type": "user",
      "principal_id": "$[[env.B_ID]]",
      "permission": $[[env.EDIT_ACCESS]]
    },
    {
      "id": "$[[grant_user_a_assistant_a_user_c]]",
      "principal_type": "user",
      "principal_id": "$[[env.C_ID]]",
      "permission": $[[env.SHARE_ACCESS]]
    },
    {
      "id": "$[[grant_user_c_assistant_a_user_a]]",
      "principal_type": "user",
      "principal_id": "$[[env.A_ID]]",
      "permission": $[[env.EDIT_ACCESS]]
    }
  ]
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, { 
#   "deleted": [
#     { 
#       "resource_id": "$[[env.AICHAT_A_ID]]",
#       "principal_id": "$[[env.B_ID]]",
#       "permission": $[[env.EDIT_ACCESS]]
#     }, 
#     { 
#       "resource_id": "$[[env.AICHAT_A_ID]]",
#       "principal_id": "$[[env.C_ID]]",
#       "permission": $[[env.SHARE_ACCESS]]
#     }, 
#     { 
#       "resource_id": "$[[env.AICHAT_A_ID]]",
#       "principal_id": "$[[env.A_ID]]",
#       "permission": $[[env.EDIT_ACCESS]]
#     }, 
#   ]
# })


#// 21
#//
#// User [a] cannot see assistant [aichat_a]
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


#// 22
#//
#// User [b] cannot see assistant [aichat_a]
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


#// 23
#//
#// User [c] cannot see assistant [aichat_a]
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


#// 24
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