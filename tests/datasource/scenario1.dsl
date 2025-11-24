#// Scenario 1:
#//
#// Owner (admin) grants other users permission, then other users will have the 
#// corresponding permission.


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
#// Admin grants user [a] [Share] access to [datasource_1:file_a]
POST /resources/document/$[[env.DATASOURCE_1_DOCUMENT_FILE_A_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_A_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_1_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_1_DOCUMENT_FILE_A_PATH]]",
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
#       "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_A_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]],
#       "principal_id": "$[[env.A_ID]]" 
#     } 
#   ]   
# })

#// 3
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



#// 4
#//
#// Admin grants user [c] [Share] access of [datasource_1:file_c]
POST /resources/document/$[[env.DATASOURCE_1_DOCUMENT_FILE_C_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_C_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_1_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_1_DOCUMENT_FILE_C_PATH]]",
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
#       "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_C_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]],
#       "principal_id": "$[[env.C_ID]]" 
#     } 
#   ]   
# })



#// 5
#//
#// Admin grants user [d] [Share] access of [datasource_1:file_d]
POST /resources/document/$[[env.DATASOURCE_1_DOCUMENT_FILE_D_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_D_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_1_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_1_DOCUMENT_FILE_D_PATH]]",
      "resource_is_folder": false,
      "principal_type": "user",
      "principal_id": "$[[env.D_ID]]",
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
#       "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_D_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]],
#       "principal_id": "$[[env.D_ID]]" 
#     } 
#   ]   
# })



#// 6
#//
#// Admin grants user [a] [Share] access to [datasource_2:file_a]
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
#// Admin grants [Share] access to [datasource_2:file_b] to b
POST /resources/document/$[[env.DATASOURCE_2_DOCUMENT_FILE_B_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_B_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_2_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_2_DOCUMENT_FILE_B_PATH]]",
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
#       "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_B_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]],
#       "principal_id": "$[[env.B_ID]]" 
#     } 
#   ]   
# })



#// 8
#//
#// Admin grants user [c] [Share] access to [datasource_2:file_c]
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
#// Admin grants user [d] [Share] access to [datasource_2:file_d]
POST /resources/document/$[[env.DATASOURCE_2_DOCUMENT_FILE_D_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "datasource",
      "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
      "resource_type": "document",
      "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_D_ID]]",
      "resource_parent_path": "$[[env.DATASOURCE_2_PATH]]",
      "resource_full_path": "$[[env.DATASOURCE_2_DOCUMENT_FILE_D_PATH]]",
      "resource_is_folder": false,
      "principal_type": "user",
      "principal_id": "$[[env.D_ID]]",
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
#       "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_D_ID]]", 
#       "permission": $[[env.SHARE_ACCESS]],
#       "principal_id": "$[[env.D_ID]]" 
#     } 
#   ]   
# })

#// 10
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

#// 11
#//
#// User a can share [datasource_1:file_a]
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_A_ID]]",
    "resource_type": "document",
    "resource_category_type": "datasource",
    "resource_category_id": "$[[env.DATASOURCE_1_ID]]",
    "resource_parent_path": "$[[env.DATASOURCE_1_PATH_WITH_TAILING_SLASH]]"
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


#// 12
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


#// 13
#//
#// Search using account a
POST /query/_search?query=file_a&size=100
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {
#   "hits.total.value": 2, 
#   "hits.hits": [
#     {"_source.title": "file_a"},
#     {"_source.title": "file_a"}
#   ]
# })


#// 14
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


#// 15
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


#// 16
#//
#// User b can share [datasource_2:file_b]
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_B_ID]]",
    "resource_type": "document",
    "resource_category_type": "datasource",
    "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
    "resource_parent_path": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]"
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


#// 17
#//
#// Search using account b
POST /query/_search?query=file_a&size=100
# request: {
#   headers: [
#     {Authorization: "Bearer $[[b_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {
#   "hits.total.value": 2, 
#   "hits.hits": [
#     {"_source.title": "file_b"},
#     {"_source.title": "file_b"}
#   ]
# })


#// 18
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


#// 19
#//
#// User c can share [datasource_1:file_c]
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_C_ID]]",
    "resource_type": "document",
    "resource_category_type": "datasource",
    "resource_category_id": "$[[env.DATASOURCE_1_ID]]",
    "resource_parent_path": "$[[env.DATASOURCE_1_PATH_WITH_TAILING_SLASH]]"
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


#// 20
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


#// 21
#//
#// Search using account c
POST /query/_search?query=file_a&size=100
# request: {
#   headers: [
#     {Authorization: "Bearer $[[c_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {
#   "hits.total.value": 2, 
#   "hits.hits": [
#     {"_source.title": "file_c"},
#     {"_source.title": "file_c"}
#   ]
# })


#// 22
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


#// 23
#//
#// User d can share [datasource_1:file_d]
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.DATASOURCE_1_DOCUMENT_FILE_D_ID]]",
    "resource_type": "document",
    "resource_category_type": "datasource",
    "resource_category_id": "$[[env.DATASOURCE_1_ID]]",
    "resource_parent_path": "$[[env.DATASOURCE_1_PATH_WITH_TAILING_SLASH]]"
  }
]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# 
# assert: (200,  
# [ 
#   {
#     "permission": $[[env.SHARE_ACCESS]],
#     "principal_id": "$[[env.D_ID]]"
#   }
# ])


#// 24
#//
#// User d can share [datasource_2:file_d]
POST /resources/shares/_batch_get
[
  {
    "resource_id": "$[[env.DATASOURCE_2_DOCUMENT_FILE_D_ID]]",
    "resource_type": "document",
    "resource_category_type": "datasource",
    "resource_category_id": "$[[env.DATASOURCE_2_ID]]",
    "resource_parent_path": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]"
  }
]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# 
# assert: (200,  
# [ 
#   {
#     "permission": $[[env.SHARE_ACCESS]],
#     "principal_id": "$[[env.D_ID]]"
#   }
# ])


#// 25
#//
#// Search using account d
POST /query/_search?query=file_a&size=100
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {
#   "hits.total.value": 2, 
#   "hits.hits": [
#     {"_source.title": "file_d"},
#     {"_source.title": "file_d"}
#   ]
# })