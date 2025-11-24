#// Scenario 5:
#//
#// Test datasource owner could 
#//   1. edit datasources
#//   2. delete datasources and documents
#//
#// And the changes are visible to other users.


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
#// User [admin] grants users [a,b,c,d] [view] permission to 
#// [datasource_1,datasource_2], so that all the datasources/documents are visible
#// to these users.
#//
#//----------------------------------------------------------------------------

#// 6
#//
#// User [admin] grants users [a,b,c,d] [view] permission to [datasource_1] 
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
    },
    {
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_1_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.B_ID]]",
      "permission": $[[env.VIEW_ACCESS]]
    },
    {
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_1_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.C_ID]]",
      "permission": $[[env.VIEW_ACCESS]]
    },
    {
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_1_ID]]",
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
#   disable_header_names_normalizing: true
# },
#
# register: [
#   { grant_user_admin_datasource_1_user_a: "_ctx.response.body_json.created.0.id" },
#   { grant_user_admin_datasource_1_user_b: "_ctx.response.body_json.created.1.id" },
#   { grant_user_admin_datasource_1_user_c: "_ctx.response.body_json.created.2.id" },
#   { grant_user_admin_datasource_1_user_d: "_ctx.response.body_json.created.3.id" }
# ],
#
# assert: (200, {
#   "created": [
#     { 
#       "resource_id": "$[[env.DATASOURCE_1_ID]]",
#       "principal_id": "$[[env.A_ID]]",
#       "permission": $[[env.VIEW_ACCESS]]  
#     },
#     { 
#       "resource_id": "$[[env.DATASOURCE_1_ID]]",
#       "principal_id": "$[[env.B_ID]]",
#       "permission": $[[env.VIEW_ACCESS]]  
#     },
#     { 
#       "resource_id": "$[[env.DATASOURCE_1_ID]]",
#       "principal_id": "$[[env.C_ID]]",
#       "permission": $[[env.VIEW_ACCESS]]  
#     },
#     { 
#       "resource_id": "$[[env.DATASOURCE_1_ID]]",
#       "principal_id": "$[[env.D_ID]]",
#       "permission": $[[env.VIEW_ACCESS]]  
#     } 
#   ]
# })


#// 7
#//
#// User [admin] grants users [a,b,c,d] [view] permission to [datasource_2] 
POST /resources/datasource/$[[env.DATASOURCE_2_ID]]/share
{
  "shares": [
    {
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_2_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.A_ID]]",
      "permission": $[[env.VIEW_ACCESS]]
    },
    {
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_2_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.B_ID]]",
      "permission": $[[env.VIEW_ACCESS]]
    },
    {
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_2_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.C_ID]]",
      "permission": $[[env.VIEW_ACCESS]]
    },
    {
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_2_ID]]",
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
#   disable_header_names_normalizing: true
# },
#
# register: [
#   { grant_user_admin_datasource_2_user_a: "_ctx.response.body_json.created.0.id" },
#   { grant_user_admin_datasource_2_user_b: "_ctx.response.body_json.created.1.id" },
#   { grant_user_admin_datasource_2_user_c: "_ctx.response.body_json.created.2.id" },
#   { grant_user_admin_datasource_2_user_d: "_ctx.response.body_json.created.3.id" }
# ],
#
# assert: (200, {
#   "created": [
#     { 
#       "resource_id": "$[[env.DATASOURCE_2_ID]]",
#       "principal_id": "$[[env.A_ID]]",
#       "permission": $[[env.VIEW_ACCESS]]  
#     },
#     { 
#       "resource_id": "$[[env.DATASOURCE_2_ID]]",
#       "principal_id": "$[[env.B_ID]]",
#       "permission": $[[env.VIEW_ACCESS]]  
#     },
#     { 
#       "resource_id": "$[[env.DATASOURCE_2_ID]]",
#       "principal_id": "$[[env.C_ID]]",
#       "permission": $[[env.VIEW_ACCESS]]  
#     },
#     { 
#       "resource_id": "$[[env.DATASOURCE_2_ID]]",
#       "principal_id": "$[[env.D_ID]]",
#       "permission": $[[env.VIEW_ACCESS]]  
#     } 
#   ]
# })


#//----------------------------------------------------------------------------
#//
#// Case 1:
#//   1. User [admin] disables datasource [datasource_1]
#//   2. Search using account [a], the search results only include documents from 
#//      [datasource_2]
#//   3. Search using account [b], the search results only include documents from 
#//      [datasource_2]
#//   4. Search using account [c], the search results only include documents from 
#//      [datasource_2]
#//   5. Search using account [d], the search results only include documents from 
#//      [datasource_2]
#//
#//----------------------------------------------------------------------------

#// 8
#//
#// User [admin] disables datasource [datasource_1]
PUT /datasource/$[[env.DATASOURCE_1_ID]]
{
  "_system": {
    "owner_id": "$[[env.ADMIN_ID]]"
  },
  "connector": {
    "id": "local_fs"
  },
  "enabled": false,
  "enrichment_pipeline": null,
  "id": "$[[env.DATASOURCE_1_ID]]",
  "name": "$[[env.DATASOURCE_1_NAME]]",
  "sync": {
    "enabled": false,
    "interval": "1s",
    "page_size": 0,
    "strategy": "interval"
  },
  "type": "connector",
  "webhook": {
    "enabled": false
  },
  "_index": "$[[env.DATASOURCE_INDEX]]",
  "_type": "_doc",
  "shares": [
    {
      "id": "$[[grant_user_admin_datasource_1_user_a]]",
      "_system": {
        "owner_id": "$[[env.ADMIN_ID]]"
      },
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_parent_path": "/",
      "resource_parent_path_reversed": "/",
      "principal_type": "user",
      "principal_id": "$[[env.A_ID]]",
      "permission": $[[env.VIEW_ACCESS]],
      "entity": {
        "type": "user",
        "id": "$[[env.A_ID]]",
        "icon": "circle-user",
        "title": "$[[env.A_NAME]]",
        "subtitle": "$[[env.A_MAIL]]"
      }
    },
    {
      "id": "$[[grant_user_admin_datasource_1_user_b]]",
      "_system": {
        "owner_id": "$[[env.ADMIN_ID]]"
      },
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_parent_path": "/",
      "resource_parent_path_reversed": "/",
      "principal_type": "user",
      "principal_id": "$[[env.B_ID]]",
      "permission": $[[env.VIEW_ACCESS]],
      "entity": {
        "type": "user",
        "id": "$[[env.B_ID]]",
        "icon": "circle-user",
        "title": "$[[env.B_NAME]]",
        "subtitle": "$[[env.B_MAIL]]"
      }
    },
    {
      "id": "$[[grant_user_admin_datasource_1_user_c]]",
      "_system": {
        "owner_id": "$[[env.ADMIN_ID]]"
      },
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_parent_path": "/",
      "resource_parent_path_reversed": "/",
      "principal_type": "user",
      "principal_id": "$[[env.C_ID]]",
      "permission": $[[env.VIEW_ACCESS]],
      "entity": {
        "type": "user",
        "id": "$[[env.C_ID]]",
        "icon": "circle-user",
        "title": "$[[env.C_NAME]]",
        "subtitle": "$[[env.C_MAIL]]"
      }
    },
    {
      "id": "$[[grant_user_admin_datasource_1_user_d]]",
      "_system": {
        "owner_id": "$[[env.ADMIN_ID]]"
      },
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_parent_path": "/",
      "resource_parent_path_reversed": "/",
      "principal_type": "user",
      "principal_id": "$[[env.D_ID]]",
      "permission": $[[env.VIEW_ACCESS]],
      "entity": {
        "type": "user",
        "id": "$[[env.D_ID]]",
        "icon": "circle-user",
        "title": "$[[env.D_NAME]]",
        "subtitle": "$[[env.D_MAIL]]"
      }
    }
  ],
  "owner": {
    "type": "user",
    "id": "$[[env.ADMIN_ID]]",
    "icon": "circle-user",
    "title": "$[[env.ADMIN_NAME]]",
    "subtitle": "$[[env.ADMIN_MAIL]]"
  },
  "editor": {
    "type": "user",
    "id": "$[[env.ADMIN_ID]]",
    "icon": "circle-user",
    "title": "$[[env.ADMIN_NAME]]",
    "subtitle": "$[[env.ADMIN_MAIL]]"
  }
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: { "_ctx.response.status": 200 }


#// 9
#//
#// Search using account [a]
POST /query/_search?query=file_a&size=100
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {
#   "hits.total.value": 4, 
#   "hits.hits": [
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" },
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" },
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" },
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" }
#   ]
# })


#// 10
#//
#// Search using account [b]
POST /query/_search?query=file_a&size=100
# request: {
#   headers: [
#     {Authorization: "Bearer $[[b_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {
#   "hits.total.value": 4, 
#   "hits.hits": [
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" },
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" },
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" },
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" }
#   ]
# })


#// 11
#//
#// Search using account [c]
POST /query/_search?query=file_a&size=100
# request: {
#   headers: [
#     {Authorization: "Bearer $[[c_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {
#   "hits.total.value": 4, 
#   "hits.hits": [
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" },
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" },
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" },
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" }
#   ]
# })


#// 12
#//
#// Search using account [d]
POST /query/_search?query=file_a&size=100
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {
#   "hits.total.value": 4, 
#   "hits.hits": [
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" },
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" },
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" },
#     {"_source.category": "$[[env.DATASOURCE_2_PATH_WITH_TAILING_SLASH]]" }
#   ]
# })


#//----------------------------------------------------------------------------
#//
#// Case 2:
#//   1. User [admin] renames datasource [datasource_1] [datasource_A]
#//   2. Users [a,b,c,d] now see [datasource_A]
#//
#//----------------------------------------------------------------------------


#// 13
#//
#// [admin] renames datasource [datasource_1] [datasource_A]
PUT /datasource/$[[env.DATASOURCE_1_ID]]
{
  "_system": {
    "owner_id": "$[[env.ADMIN_ID]]"
  },
  "connector": {
    "id": "local_fs"
  },
  "enabled": false,
  "enrichment_pipeline": null,
  "id": "$[[env.DATASOURCE_1_ID]]",
  "name": "datasource_A",
  "sync": {
    "enabled": false,
    "interval": "1s",
    "page_size": 0,
    "strategy": "interval"
  },
  "type": "connector",
  "webhook": {
    "enabled": false
  },
  "_index": "$[[env.DATASOURCE_INDEX]]",
  "_type": "_doc",
  "shares": [
    {
      "id": "$[[grant_user_admin_datasource_1_user_a]]",
      "_system": {
        "owner_id": "$[[env.ADMIN_ID]]"
      },
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_parent_path": "/",
      "resource_parent_path_reversed": "/",
      "principal_type": "user",
      "principal_id": "$[[env.A_ID]]",
      "permission": $[[env.VIEW_ACCESS]],
      "entity": {
        "type": "user",
        "id": "$[[env.A_ID]]",
        "icon": "circle-user",
        "title": "$[[env.A_NAME]]",
        "subtitle": "$[[env.A_MAIL]]"
      }
    },
    {
      "id": "$[[grant_user_admin_datasource_1_user_b]]",
      "_system": {
        "owner_id": "$[[env.ADMIN_ID]]"
      },
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_parent_path": "/",
      "resource_parent_path_reversed": "/",
      "principal_type": "user",
      "principal_id": "$[[env.B_ID]]",
      "permission": $[[env.VIEW_ACCESS]],
      "entity": {
        "type": "user",
        "id": "$[[env.B_ID]]",
        "icon": "circle-user",
        "title": "$[[env.B_NAME]]",
        "subtitle": "$[[env.B_MAIL]]"
      }
    },
    {
      "id": "$[[grant_user_admin_datasource_1_user_c]]",
      "_system": {
        "owner_id": "$[[env.ADMIN_ID]]"
      },
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_parent_path": "/",
      "resource_parent_path_reversed": "/",
      "principal_type": "user",
      "principal_id": "$[[env.C_ID]]",
      "permission": $[[env.VIEW_ACCESS]],
      "entity": {
        "type": "user",
        "id": "$[[env.C_ID]]",
        "icon": "circle-user",
        "title": "$[[env.C_NAME]]",
        "subtitle": "$[[env.C_MAIL]]"
      }
    },
    {
      "id": "$[[grant_user_admin_datasource_1_user_d]]",
      "_system": {
        "owner_id": "$[[env.ADMIN_ID]]"
      },
      "resource_category_type": "connector",
      "resource_category_id": "local_fs",
      "resource_type": "datasource",
      "resource_id": "$[[env.DATASOURCE_1_ID]]",
      "resource_parent_path": "/",
      "resource_parent_path_reversed": "/",
      "principal_type": "user",
      "principal_id": "$[[env.D_ID]]",
      "permission": $[[env.VIEW_ACCESS]],
      "entity": {
        "type": "user",
        "id": "$[[env.D_ID]]",
        "icon": "circle-user",
        "title": "$[[env.D_NAME]]",
        "subtitle": "$[[env.D_MAIL]]"
      }
    }
  ],
  "owner": {
    "type": "user",
    "id": "$[[env.ADMIN_ID]]",
    "icon": "circle-user",
    "title": "$[[env.ADMIN_NAME]]",
    "subtitle": "$[[env.ADMIN_MAIL]]"
  },
  "editor": {
    "type": "user",
    "id": "$[[env.ADMIN_ID]]",
    "icon": "circle-user",
    "title": "$[[env.ADMIN_NAME]]",
    "subtitle": "$[[env.ADMIN_MAIL]]"
  }
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: { "_ctx.response.status": 200 }


#// 14
#//
#// User [a] sees [datasource_A]
GET /datasource/_search
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
#     { "_source.name": "$[[env.DATASOURCE_2_NAME]]" },
#     { "_source.name": "datasource_A" }, 
#   ]
#} )


#// 15
#//
#// User [b] sees [datasource_A]
GET /datasource/_search
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
#     { "_source.name": "$[[env.DATASOURCE_2_NAME]]" },
#     { "_source.name": "datasource_A" }, 
#   ]
#} )


#// 16
#//
#// User [c] sees [datasource_A]
GET /datasource/_search
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
#     { "_source.name": "$[[env.DATASOURCE_2_NAME]]" },
#     { "_source.name": "datasource_A" }, 
#   ]
#} )


#// 17
#//
#// User [d] sees [datasource_A]
GET /datasource/_search
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
#     { "_source.name": "$[[env.DATASOURCE_2_NAME]]" },
#     { "_source.name": "datasource_A" }, 
#   ]
#} )


#//----------------------------------------------------------------------------
#//
#// Case 3:
#//   1. User [admin] deletes document [datasource_2:file_a]
#//   2. Users [a,b,c,d] cannot see [datasource_2:file_a] any longer
#//
#//----------------------------------------------------------------------------

#// 18
#//
#// User [admin] deletes document [datasource_2:file_a]
DELETE /document/$[[env.DATASOURCE_2_DOCUMENT_FILE_A_ID]]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: { "_ctx.response.status": 200 }

#// 19
#//
#// User [a] cannot see [datasource_2:file_a]
GET /document/_search?filter=source.id:any($[[env.DATASOURCE_2_ID]])&from=0&size=100&query=file
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: (200, {
#   "hits.total.value": 3,
#   "hits.hits": [ 
#     { "_source.title": "file_b"  },
#     { "_source.title": "file_c"  },
#     { "_source.title": "file_d"  } 
#   ]
# })


#// 20
#//
#// User [b] cannot see [datasource_2:file_a]
GET /document/_search?filter=source.id:any($[[env.DATASOURCE_2_ID]])&from=0&size=100&query=file
# request: {
#   headers: [
#     {Authorization: "Bearer $[[b_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: (200, {
#   "hits.total.value": 3,
#   "hits.hits": [ 
#     { "_source.title": "file_b"  },
#     { "_source.title": "file_c"  },
#     { "_source.title": "file_d"  } 
#   ]
# })


#// 21
#//
#// User [c] cannot see [datasource_2:file_a]
GET /document/_search?filter=source.id:any($[[env.DATASOURCE_2_ID]])&from=0&size=100&query=file
# request: {
#   headers: [
#     {Authorization: "Bearer $[[c_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: (200, {
#   "hits.total.value": 3,
#   "hits.hits": [ 
#     { "_source.title": "file_b"  },
#     { "_source.title": "file_c"  },
#     { "_source.title": "file_d"  } 
#   ]
# })


#// 22
#//
#// User [d] cannot see [datasource_2:file_a]
GET /document/_search?filter=source.id:any($[[env.DATASOURCE_2_ID]])&from=0&size=100&query=file
# request: {
#   headers: [
#     {Authorization: "Bearer $[[d_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: (200, {
#   "hits.total.value": 3,
#   "hits.hits": [ 
#     { "_source.title": "file_b"  },
#     { "_source.title": "file_c"  },
#     { "_source.title": "file_d"  } 
#   ]
# })


#//----------------------------------------------------------------------------
#//
#// Case 4:
#//   1. User [admin] deletes datasource [datasource_2]
#//   2. Users [a,b,c,d] cannot see [datasource_2] any longer
#//
#//----------------------------------------------------------------------------

#// 23
#//
#// User [admin] deletes datasource [datasource_2]
DELETE /datasource/$[[env.DATASOURCE_2_ID]]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"_id":"$[[env.DATASOURCE_2_ID]]","result":"deleted"} )


#// 24
#//
#// User [a] cannot see [datasource_2]
GET /datasource/_search
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
#     { "_source.name": "datasource_A" }, 
#   ]
#} )


#// 25
#//
#// User [b] cannot see [datasource_2]
GET /datasource/_search
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
#     { "_source.name": "datasource_A" }, 
#   ]
#} )


#// 26
#//
#// User [c] cannot see [datasource_2]
GET /datasource/_search
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
#     { "_source.name": "datasource_A" }, 
#   ]
#} )


#// 27
#//
#// User [d] cannot see [datasource_2]
GET /datasource/_search
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
#     { "_source.name": "datasource_A" }, 
#   ]
#} )