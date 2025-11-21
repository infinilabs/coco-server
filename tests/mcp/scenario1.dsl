#// Scenario 1:
#//
#// Owner (admin) grants other users different permissions to MCP servers


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
#// Case 1
#//   1. User [admin] grants user [a] [view] permission to [mcp_server_a]
#//   2. User [admin] grants user [b] [edit] permission to [mcp_server_b]
#//   3. User [admin] grants user [c] [share] permission to [mcp_server_c]
#//   4. User [admin] grants user [d] [view] permission to [mcp_server_d]
#//   5. Users [a, b, c, d] have the corresponding permissions
#//
#//----------------------------------------------------------------------------


#// 6
#//
#// User [admin] grants user [a] [view] permission to [mcp_server_a]
POST /resources/mcp-server/$[[env.MCP_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "mcp-server",
      "resource_id": "$[[env.MCP_A_ID]]",
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
#       "resource_id": "$[[env.MCP_A_ID]]", 
#       "principal_id": "$[[env.A_ID]]", 
#       "permission": $[[env.VIEW_ACCESS]] 
#     } 
#   ]
# })


#// 7
#//
#// User [admin] grants user [b] [edit] permission to [mcp_server_b]
POST /resources/mcp-server/$[[env.MCP_B_ID]]/share
{
  "shares": [
    {
      "resource_type": "mcp-server",
      "resource_id": "$[[env.MCP_B_ID]]",
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
#       "resource_id": "$[[env.MCP_B_ID]]", 
#       "principal_id": "$[[env.B_ID]]", 
#       "permission": $[[env.EDIT_ACCESS]] 
#     } 
#   ]
# })


#// 8
#//
#// User [admin] grants user [c] [share] permission to [mcp_server_c]
POST /resources/mcp-server/$[[env.MCP_C_ID]]/share
{
  "shares": [
    {
      "resource_type": "mcp-server",
      "resource_id": "$[[env.MCP_C_ID]]",
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
#       "resource_id": "$[[env.MCP_C_ID]]", 
#       "principal_id": "$[[env.C_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]] 
#     } 
#   ]
# })


#// 9
#//
#// User [admin] grants user [d] [view] permission to [mcp_server_d]
POST /resources/mcp-server/$[[env.MCP_D_ID]]/share
{
  "shares": [
    {
      "resource_type": "mcp-server",
      "resource_id": "$[[env.MCP_D_ID]]",
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
#       "resource_id": "$[[env.MCP_D_ID]]", 
#       "principal_id": "$[[env.D_ID]]", 
#       "permission": $[[env.VIEW_ACCESS]] 
#     } 
#   ]
# })


#// 10
#//
#// Users [a, b, c, d] have the corresponding permissions
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.MCP_A_ID]]",
    "resource_type": "mcp-server"
  },
  {
    "resource_id": "$[[env.MCP_B_ID]]",
    "resource_type": "mcp-server"
  },
  {
    "resource_id": "$[[env.MCP_C_ID]]",
    "resource_type": "mcp-server"
  },
  {
    "resource_id": "$[[env.MCP_D_ID]]",
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
# assert: (200,  
# [ 
#   {
#     "permission": $[[env.VIEW_ACCESS]],
#     "principal_id": "$[[env.A_ID]]"
#   },
#   {
#     "permission": $[[env.EDIT_ACCESS]],
#     "principal_id": "$[[env.B_ID]]"
#   },
#   {
#     "permission": $[[env.SHARE_ACCESS]],
#     "principal_id": "$[[env.C_ID]]"
#   },
#   {
#     "permission": $[[env.VIEW_ACCESS]],
#     "principal_id": "$[[env.D_ID]]"
#   }
# ])