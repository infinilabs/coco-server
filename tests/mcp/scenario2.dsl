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
#//   1. User [admin] grants user [a] [share] permission to [mcp_server_a]
#//   2. User [a] have the corresponding permission
#//
#//----------------------------------------------------------------------------


#// 6
#//
#// User [admin] grants user [a] [share] permission to [mcp_server_a]
POST /resources/mcp-server/$[[env.MCP_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "mcp-server",
      "resource_id": "$[[env.MCP_A_ID]]",
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
#       "resource_id": "$[[env.MCP_A_ID]]", 
#       "principal_id": "$[[env.A_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]] 
#     } 
#   ]
# })


#// 7
#//
#// User [a] has the corresponding permission
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.MCP_A_ID]]",
    "resource_type": "mcp-server"
  }
]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# 
# assert: (200,  
# [ 
#   {
#     "permission": $[[env.SHARE_ACCESS]],
#     "principal_id": "$[[env.A_ID]]"
#   }
# ])


#//----------------------------------------------------------------------------
#//
#// Case 2:
#//   1. User [a] grants:
#//      1. [edit] permission of [mcp_server_a] to user [b]
#//      2. [share] permission of [mcp_server_a] to user [c]
#//   2. Users [b, c] have the corresponding permission
#//   3. User [d] don't have access to [mcp_server_a]
#//
#//----------------------------------------------------------------------------


#// 8
#//
#// User [a] grants user [b] [edit] permission to [mcp_server_a]
POST /resources/mcp-server/$[[env.MCP_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "mcp-server",
      "resource_id": "$[[env.MCP_A_ID]]",
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
#   { grant_user_a_mcp_a_user_b: "_ctx.response.body_json.created.0.id" },
# ],
#
# assert: (200, { 
#   "created": [
#     { 
#       "resource_id": "$[[env.MCP_A_ID]]", 
#       "principal_id": "$[[env.B_ID]]", 
#       "permission": $[[env.EDIT_ACCESS]] 
#     } 
#   ]
# })


#// 9
#//
#// User [a] grants user [c] [share] permission to [mcp_server_a]
POST /resources/mcp-server/$[[env.MCP_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "mcp-server",
      "resource_id": "$[[env.MCP_A_ID]]",
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
#   { grant_user_a_mcp_a_user_c: "_ctx.response.body_json.created.0.id" },
# ],
#
# assert: (200, { 
#   "created": [
#     { 
#       "resource_id": "$[[env.MCP_A_ID]]", 
#       "principal_id": "$[[env.C_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]] 
#     } 
#   ]
# })


#// 10
#//
#// Users [b, c] have the corresponding permissions
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.MCP_A_ID]]",
    "resource_type": "mcp-server"
  }
]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# 
# assert: (200,  
# [ 
#   {
#     "permission": $[[env.SHARE_ACCESS]],
#     "principal_id": "$[[env.A_ID]]"
#   },
#   {
#     "permission": $[[env.EDIT_ACCESS]],
#     "principal_id": "$[[env.B_ID]]"
#   },
#   {
#     "permission": $[[env.SHARE_ACCESS]],
#     "principal_id": "$[[env.C_ID]]"
#   }
# ])


#// 11
#//
#// User [d] cannot see mcp server [mcp_server_a]
GET /mcp_server/_search?&from=0&size=100&query=mcp_server
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
#//  1. User [c] grants user [a] [edit] permission to assistant [mcp_server_a]
#//  3. Verify the permissions granted on [mcp_server_a]:
#//     a: edit
#//     b: edit
#//     c: share
#//  4. Verify that [mcp_server_a] is invisible to user [d]
#//
#//----------------------------------------------------------------------------


#// 12
#//
#// User [c] grants user [a] [edit] permission to mcp server [mcp_server_a]
POST /resources/mcp-server/$[[env.MCP_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "mcp-server",
      "resource_id": "$[[env.MCP_A_ID]]",
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
# sleep: {
#   "sleep_in_milli_seconds": 1000
# },
#
# register: [
#   { grant_user_c_mcp_a_user_a: "_ctx.response.body_json.updated.0.id" },
# ],
#
# assert: (200, { 
#   "updated": [
#     { 
#       "resource_id": "$[[env.MCP_A_ID]]", 
#       "principal_id": "$[[env.A_ID]]", 
#       "permission": $[[env.EDIT_ACCESS]] 
#     } 
#   ]
# })


#// 13
#//
#// Verify the permissions granted on [mcp_server_a]
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.MCP_A_ID]]",
    "resource_type": "mcp-server"
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
#     "resource_id": "$[[env.MCP_A_ID]]",
#     "principal_id": "$[[env.B_ID]]",
#     "permission": $[[env.EDIT_ACCESS]]
#   },
#   {
#     "resource_id": "$[[env.MCP_A_ID]]",
#     "principal_id": "$[[env.C_ID]]",
#     "permission": $[[env.SHARE_ACCESS]]
#   },
#   {
#     "resource_id": "$[[env.MCP_A_ID]]",
#     "principal_id": "$[[env.A_ID]]",
#     "permission": $[[env.EDIT_ACCESS]]
#   }
# ]
# )


#// 14
#//
#// User [d] cannot see mcp server [mcp_server_a]
GET /mcp_server/_search?&from=0&size=100&query=mcp_server
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
#//  1. User [admin] revokes the [edit] permission to [mcp_server_a] from user [a]
#//  2. User [admin] revokes the [view] permission to [mcp_server_a] from user [b]
#//  3. User [admin] revokes the [share] permission to [mcp_server_a] from user [c]
#//  4. Verify that [mcp_server_a] is invisible to users [a, b, c]
#//
#//----------------------------------------------------------------------------


#// 15
#//
#// User [admin] revokes the [edit] permission of [mcp_server_a] from user [a]
POST /resources/mcp-server/$[[env.MCP_A_ID]]/share
{
  "revokes": [
    {
      "id": "$[[grant_user_c_mcp_a_user_a]]",
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
#       "resource_id": "$[[env.MCP_A_ID]]",
#       "permission": $[[env.EDIT_ACCESS]],
#       "principal_id": "$[[env.A_ID]]"
#     } 
#   ]
# })


#// 16
#//
#// User [admin] revokes the [edit] permission of [mcp_server_a] from user [b]
POST /resources/mcp-server/$[[env.MCP_A_ID]]/share
{
  "revokes": [
    {
      "id": "$[[grant_user_a_mcp_a_user_b]]",
      "principal_type": "user",
      "principal_id": "$[[env.B_ID]]",
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
#       "resource_id": "$[[env.MCP_A_ID]]",
#       "permission": $[[env.EDIT_ACCESS]],
#       "principal_id": "$[[env.B_ID]]"
#     } 
#   ]
# })


#// 17
#//
#// User [admin] revokes the [share] permission of [mcp_server_a] from user [c]
POST /resources/mcp-server/$[[env.MCP_A_ID]]/share
{
  "revokes": [
    {
      "id": "$[[grant_user_a_mcp_a_user_c]]",
      "principal_type": "user",
      "principal_id": "$[[env.C_ID]]",
      "permission": $[[env.SHARE_ACCESS]]
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
#       "resource_id": "$[[env.MCP_A_ID]]",
#       "permission": $[[env.SHARE_ACCESS]],
#       "principal_id": "$[[env.C_ID]]"
#     } 
#   ]
# })

#// 18
#//
#// User [b] cannot see mcp server [mcp_server_a]
GET /mcp_server/_search?&from=0&size=100&query=mcp_server
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

#// 19
#//
#// User [b] cannot see mcp server [mcp_server_a]
GET /mcp_server/_search?&from=0&size=100&query=mcp_server
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


#// 20
#//
#// User [c] cannot see mcp server [mcp_server_a]
GET /mcp_server/_search?&from=0&size=100&query=mcp_server
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


#// 21
#//
#// User [d] cannot see mcp server [mcp_server_a]
GET /mcp_server/_search?&from=0&size=100&query=mcp_server
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