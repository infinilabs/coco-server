#// Case 4: 
#//   1. admin grants:
#//      1. user [a] [view] permission to datasource [datasource_1]
#//      2. user [a] [share] permission to document [datasource_1:file_a]
#//   2. admin revokes the permission to datasource [datasource_1] from user [a]
#//   3. User [a] has [share] permission to document [datasource_1:file_a]
#//   4. Document [datasource_1:file_a] is the only document in [datasource_1] 
#//      that user [a] can see


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


#//----------------------------------------------------------------------------
#//
#// Test
#//
#//----------------------------------------------------------------------------

#// 3
#//
#// [Admin] grants user [a] [view] permission to datasource [datasource_1]
POST /resources/datasource/$[[env.DATASOURCE_1_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_1_ID]]",
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
#   disable_header_names_normalizing: true
# },
#
# register: [
#   { grant_user_admin_datasource_1_user_a: "_ctx.response.body_json.created.0.id" },
# ],
#
# assert: (200, {
#   "created": [
#     { 
#       "resource_id": "$[[env.DATASOURCE_1_ID]]",
#       "principal_id": "$[[env.A_ID]]",
#       "permission": $[[env.VIEW_ACCESS]]  
#     } 
#   ]
# })


#// 4
#//
#// [Admin] grants user [a] [Share] permission to document [datasource_1:file_a]
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


#// 5
#//
#// User [admin] revokes the [view] permission to datasource [datasource_1] from user [a]
POST /resources/datasource/$[[env.DATASOURCE_1_ID]]/share
{
  "revokes": [
    {
      "id": "$[[grant_user_admin_datasource_1_user_a]]",
      "principal_type": "user",
      "principal_id": "$[[env.A_ID]]",
      "permission": $[[env.VIEW_ACCESS]]
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
#       "resource_id": "$[[env.DATASOURCE_1_ID]]",
#       "permission": $[[env.VIEW_ACCESS]],
#       "principal_id": "$[[env.A_ID]]"
#     } 
#   ]
# })


#// 6
#//
#// List all permissions granted to [datasource_1:file_a]
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


#// 7
#//
#// User [a] can only see [datasource_1:file_a]
GET /document/_search?filter=source.id:any($[[env.DATASOURCE_1_ID]])&from=0&size=100&query=file
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: (200, {
#   "hits.total.value": 1,
#   "hits.hits": [ { "_source.title": "file_a"  } ]
# })