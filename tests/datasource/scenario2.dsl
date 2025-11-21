#// Scenario 2:
#//
#// Users that are granted share permission can share datasource/document


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
#//   1. admin grants:
#//      1. user [a] [share] permission to [datasource_2:file_a]
#//      2. user [b] [share] permission to [datasource_1:file_b]
#//      3. user [c] [share] permission to [datasource_2:file_c]
#//   2. Users [a, b, c] have the corresponding permission
#//   3. Users [d] don't have access to [datasource_1, datasource_2]
#//
#//----------------------------------------------------------------------------

#// 6
#//
#// Admin grants user [a] [Share] permission to [datasource_2:file_a]
POST /resources/document/$[[env.DATASOURCE_2_DOCUMENT_FILE_A_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_A_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_2_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_2_DOCUMENT_FILE_A_PATH]]",
      "resource_is_folder": false,
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
#       "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_A_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]],
#       "principal_id": "$[[env.A_ID]]" 
#     } 
#   ]   
# })


#// 7
#//
#// Admin grants user [b] [Share] access of [datasource_1:file_b]
POST /resources/document/$[[env.DATASOURCE_1_DOCUMENT_FILE_B_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_B_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_1_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_1_DOCUMENT_FILE_B_PATH]]",
      "resource_is_folder": false,
      "principal_type": "user",
      "principal_id": "$[[env.B_ID]]",
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
#       "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_B_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]],
#       "principal_id": "$[[env.B_ID]]" 
#     } 
#   ]   
# })


#// 8
#//
#// [Admin] grants user [c] [Share] access of [datasource_2:file_c]
POST /resources/document/$[[env.DATASOURCE_2_DOCUMENT_FILE_C_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_C_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_2_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_2_DOCUMENT_FILE_C_PATH]]",
      "resource_is_folder": false,
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
#       "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_C_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]],
#       "principal_id": "$[[env.C_ID]]" 
#     } 
#   ]   
# })


#// 9
#//
#// User a can share [datasource_2:file_a]
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_A_ID]]",
    "resource_type": "document",
    "resource_category_type": "datasource",
    "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
    "resource_parent_path": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]"
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


#// 10
#//
#// User b can share [datasource_1:file_b]
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_B_ID]]",
    "resource_type": "document",
    "resource_category_type": "datasource",
    "resource_category_id": "$[[env.DATASOURCE_1_ID]]",
    "resource_parent_path": "$[[env.DATASOURCE_1_PATH_WITH_TAILING_SLASH]]"
  }
]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[b_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# 
# assert: (200,  
# [ 
#   {
#     "permission": $[[env.SHARE_ACCESS]],
#     "principal_id": "$[[env.B_ID]]"
#   }
# ])


#// 11
#//
#// User c can share [datasource_2:file_c]
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_C_ID]]",
    "resource_type": "document",
    "resource_category_type": "datasource",
    "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
    "resource_parent_path": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]"
  }
]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[c_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# 
# assert: (200,  
# [ 
#   {
#     "permission": $[[env.SHARE_ACCESS]],
#     "principal_id": "$[[env.C_ID]]"
#   }
# ])


#// 12
#//
#// User [d] cannot see [datasource_1, datasource_2]
GET /datasource/_search
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"hits.total.value": 0} )


#//----------------------------------------------------------------------------
#//
#// Case 2: 
#//   1. User [a] grants:
#//      1. user [b] [view] permission to [datasource_2:file_a]
#//      2. user [c] [edit] permission to [datasource_2:file_a]
#//   2. Users [b, c] have the corresponding permission
#//   3. Users [d] don't have access to [datasource_2:file_a]
#//
#//----------------------------------------------------------------------------

#// 13
#//
#// User a can grant [view] permission to [datasource_2:file_a] to b
POST /resources/document/$[[env.DATASOURCE_2_DOCUMENT_FILE_A_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_A_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_2_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_2_DOCUMENT_FILE_A_PATH]]",
      "resource_is_folder": false,
      "principal_type": "user",
      "principal_id": "$[[env.B_ID]]",
      "permission": $[[env.VIEW_ACCESS]]
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
# assert: (200, {
#   "created": [
#     { 
#       "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_A_ID]]", 
#       "permission": $[[env.VIEW_ACCESS]],
#       "principal_id": "$[[env.B_ID]]" 
#     } 
#   ]   
# })


#// 14
#//
#// User [a] can grant user [c] [edit] permission to [datasource_2:file_a]
POST /resources/document/$[[env.DATASOURCE_2_DOCUMENT_FILE_A_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_A_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_2_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_2_DOCUMENT_FILE_A_PATH]]",
      "resource_is_folder": false,
      "principal_type": "user",
      "principal_id": "$[[env.C_ID]]",
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
# assert: (200, {
#   "created": [
#     { 
#       "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_A_ID]]", 
#       "permission": $[[env.EDIT_ACCESS]],
#       "principal_id": "$[[env.C_ID]]" 
#     } 
#   ]   
# })


#// 15
#//
#// Permission set for [datasource_2:file_a]:
#// 1. user a: share
#// 2. user b: view
#// 3. user b: edit
#//
#// This request uses user [a]'s token
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_A_ID]]",
    "resource_type": "document",
    "resource_category_type": "datasource",
    "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
    "resource_parent_path": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]"
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
#     "permission": $[[env.VIEW_ACCESS]],
#     "principal_id": "$[[env.B_ID]]"
#   },
#   {
#     "permission": $[[env.EDIT_ACCESS]],
#     "principal_id": "$[[env.C_ID]]"
#   }
# ])


#// 16
#//
#// User [d] cannot see [datasource_1, datasource_2]
GET /datasource/_search
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"hits.total.value": 0} )


#// 17
#//
#// User [d] cannot see any document from [datasource_2]
GET /document/_search?filter=source.id:any($[[env.DATASOURCE_2_ID]])&from=0&size=100&query=file
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: (200, { "hits.total.value": 0 })


#//----------------------------------------------------------------------------
#//
#// Case 3: 
#//   1. User [b] grants:
#//      1. user [a] [view] permission to [datasource_1:file_b]
#//      2. user [c] [edit] permission to [datasource_1:file_b]
#//   2. Users [a, c] have the corresponding permission
#//   3. Users [d] don't have access to [datasource_1:file_b]
#//
#//----------------------------------------------------------------------------


#// 18
#//
#// User [b] can grant user [a] [view] permission to [datasource_1:file_b]
POST /resources/document/$[[env.DATASOURCE_1_DOCUMENT_FILE_B_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_B_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_1_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_1_DOCUMENT_FILE_B_PATH]]",
      "resource_is_folder": false,
      "principal_type": "user",
      "principal_id": "$[[env.A_ID]]",
      "permission": $[[env.VIEW_ACCESS]]
    }
  ],
  "revokes": []
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[b_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {
#   "created": [
#     { 
#       "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_B_ID]]", 
#       "permission": $[[env.VIEW_ACCESS]],
#       "principal_id": "$[[env.A_ID]]" 
#     } 
#   ]   
# })


#// 19
#//
#// User [b] can grant user [c] [edit] permission to [datasource_1:file_b]
POST /resources/document/$[[env.DATASOURCE_1_DOCUMENT_FILE_B_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_B_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_1_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_1_DOCUMENT_FILE_B_PATH]]",
      "resource_is_folder": false,
      "principal_type": "user",
      "principal_id": "$[[env.C_ID]]",
      "permission": $[[env.EDIT_ACCESS]]
    }
  ],
  "revokes": []
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[b_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {
#   "created": [
#     { 
#       "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_B_ID]]", 
#       "permission": $[[env.EDIT_ACCESS]],
#       "principal_id": "$[[env.C_ID]]" 
#     } 
#   ]   
# })


#// 20
#//
#// Permission set for [datasource_1:file_b]:
#// 1. user b: share
#// 2. user a: view
#// 3. user c: edit
#//
#// This request uses user [b]'s token
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_B_ID]]",
    "resource_type": "document",
    "resource_category_type": "datasource",
    "resource_category_id": "$[[env.DATASOURCE_1_ID]]",
    "resource_parent_path": "$[[env.DATASOURCE_1_PATH_WITH_TAILING_SLASH]]"
  }
]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[b_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# 
# assert: (200,  
# [ 
#   {
#     "permission": $[[env.SHARE_ACCESS]],
#     "principal_id": "$[[env.B_ID]]"
#   },
#   {
#     "permission": $[[env.VIEW_ACCESS]],
#     "principal_id": "$[[env.A_ID]]"
#   },
#   {
#     "permission": $[[env.EDIT_ACCESS]],
#     "principal_id": "$[[env.C_ID]]"
#   }
# ])


#// 21
#//
#// User [d] cannot see [datasource_1, datasource_2]
GET /datasource/_search
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"hits.total.value": 0} )


#// 22
#//
#// User [d] cannot see any document from [datasource_1]
GET /document/_search?filter=source.id:any($[[env.DATASOURCE_1_ID]])&from=0&size=100&query=file
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: (200, { "hits.total.value": 0 })


#//----------------------------------------------------------------------------
#//
#// Case 4: 
#//   1. User [c] grants:
#//      1. user [a] [view] permission to [datasource_2:file_c]
#//      2. user [b] [edit] permission to [datasource_2:file_c]
#//   2. Users [a, b] have the corresponding permission
#//   3. Users [d] don't have access to [datasource_2:file_c]
#//
#//----------------------------------------------------------------------------


#// 23
#//
#// User [c] can grant user [a] [view] permission to [datasource_2:file_c]
POST /resources/document/$[[env.DATASOURCE_2_DOCUMENT_FILE_C_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_C_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_2_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_2_DOCUMENT_FILE_C_PATH]]",
      "resource_is_folder": false,
      "principal_type": "user",
      "principal_id": "$[[env.A_ID]]",
      "permission": $[[env.VIEW_ACCESS]]
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
# assert: (200, {
#   "created": [
#     { 
#       "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_C_ID]]", 
#       "permission": $[[env.VIEW_ACCESS]],
#       "principal_id": "$[[env.A_ID]]" 
#     } 
#   ]   
# })


#// 24
#//
#// User [c] can grant user [b] [edit] permission to [datasource_2:file_c]
POST /resources/document/$[[env.DATASOURCE_2_DOCUMENT_FILE_C_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_C_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_2_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_2_DOCUMENT_FILE_C_PATH]]",
      "resource_is_folder": false,
      "principal_type": "user",
      "principal_id": "$[[env.B_ID]]",
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
# assert: (200, {
#   "created": [
#     { 
#       "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_C_ID]]", 
#       "permission": $[[env.EDIT_ACCESS]],
#       "principal_id": "$[[env.B_ID]]" 
#     } 
#   ]   
# })


#// 25
#//
#// Permission set for [datasource_2:file_c]:
#// 1. user c: share
#// 2. user a: view
#// 3. user b: edit
#//
#// This request uses user [c]'s token
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_C_ID]]",
    "resource_type": "document",
    "resource_category_type": "datasource",
    "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
    "resource_parent_path": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]"
  }
]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[c_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# 
# assert: (200,  
# [ 
#   {
#     "permission": $[[env.SHARE_ACCESS]],
#     "principal_id": "$[[env.C_ID]]"
#   },
#   {
#     "permission": $[[env.VIEW_ACCESS]],
#     "principal_id": "$[[env.A_ID]]"
#   },
#   {
#     "permission": $[[env.EDIT_ACCESS]],
#     "principal_id": "$[[env.B_ID]]"
#   }
# ])

#// 26
#//
#// User [d] cannot see [datasource_1, datasource_2]
GET /datasource/_search
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"hits.total.value": 0} )

#// 27
#//
#// User [d] cannot see any document from [datasource_2]
GET /document/_search?filter=source.id:any($[[env.DATASOURCE_2_ID]])&from=0&size=100&query=file
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: (200, { "hits.total.value": 0 })