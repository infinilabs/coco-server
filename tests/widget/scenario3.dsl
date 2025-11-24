#// Scenario 3:
#//
#// Users with [edit] permission can modify widget settings


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
#//  1. User [admin] grants user [a] [edit] permission to [widget_a]
#//  2. User [admin] grants user [b] [view] permission to [widget_a]
#//  3. User [admin] grants user [c] [share] permission to [widget_a]
#//  4. Users [a, b, c] have the corresponding permission
#//  5. [widget_a] is invisible to user [d]
#//
#//----------------------------------------------------------------------------


#// 6
#//
#// User [admin] grants user [a] [edit] permission to [widget_a]
POST /resources/integration/$[[env.WIDGET_A_ID]]/share
{
	"shares": [
		{
			"resource_type": "integration",
			"resource_id": "$[[env.WIDGET_A_ID]]",
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
#       "resource_id": "$[[env.WIDGET_A_ID]]",
#       "principal_id": "$[[env.A_ID]]",
#       "permission": $[[env.EDIT_ACCESS]]
#     }
#   ]
# })


#// 7
#//
#// User [admin] grants user [b] [view] permission to [widget_a]
POST /resources/integration/$[[env.WIDGET_A_ID]]/share
{
	"shares": [
		{
			"resource_type": "integration",
			"resource_id": "$[[env.WIDGET_A_ID]]",
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
#       "resource_id": "$[[env.WIDGET_A_ID]]",
#       "principal_id": "$[[env.B_ID]]",
#       "permission": $[[env.VIEW_ACCESS]]
#     }
#   ]
# })


#// 8
#//
#// User [admin] grants user [c] [share] permission to [widget_a]
POST /resources/integration/$[[env.WIDGET_A_ID]]/share
{
	"shares": [
		{
			"resource_type": "integration",
			"resource_id": "$[[env.WIDGET_A_ID]]",
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
#       "resource_id": "$[[env.WIDGET_A_ID]]",
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
		"resource_id": "$[[env.WIDGET_A_ID]]",
		"resource_type": "integration"
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
#     "resource_id": "$[[env.WIDGET_A_ID]]",
#     "principal_id": "$[[env.A_ID]]",
#     "permission": $[[env.EDIT_ACCESS]]
#   },
#   {
#     "resource_id": "$[[env.WIDGET_A_ID]]",
#     "principal_id": "$[[env.B_ID]]",
#     "permission": $[[env.VIEW_ACCESS]]
#   },
#   {
#     "resource_id": "$[[env.WIDGET_A_ID]]",
#     "principal_id": "$[[env.C_ID]]",
#     "permission": $[[env.SHARE_ACCESS]]
#   }
# ])


#// 10
#//
#// User [d] cannot see widget [widget_a]
GET /integration/_search?filter=type:any(embedded%2Cfloating%2Call%2Cpage%2Cmodal)&from=0&size=100&query=widget_a&t=1763624116683
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
#//  1. User [a] renames [widget_a] [widget_1]
#//  2. User [b, c] can see [widget_1]
#//  3. User [d] cannot see [widget_1]
#//
#//----------------------------------------------------------------------------


#// 11
#//
#// User [a] renames [widget_a] [widget_1]
PUT /integration/$[[env.WIDGET_A_ID]]
{
	"name": "  widget_1",
	"enabled": true,
	"guest": {
		"enabled": false
	},
	"appearance": {
		"theme": "auto",
		"language": "zh-CN"
	},
	"cors": {
		"enabled": true,
		"allowed_origins": [
			"*"
		]
	},
	"searchbox_mode": "embedded",
	"hotkey": "ctrl+/",
	"enabled_module": {
		"search": {
			"enabled": true,
			"datasource": [
				"*"
			],
			"placeholder": "Search whatever you want..."
		},
		"ai_chat": {
			"enabled": true,
			"placeholder": "Ask whatever you want...",
			"assistants": [
				"d4cnic28sig62phqk9o0"
			],
			"start_page_config": {
				"enabled": false,
				"logo": {
					"light": "",
					"dark": ""
				}
			}
		},
		"features": [
			"search_active",
			"think_active"
		]
	},
	"start_page": {
		"enabled": false
	},
	"options": {
		"embedded_placeholder": "Search..."
	},
	"type": "embedded"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"_id":"$[[env.WIDGET_A_ID]]","result":"updated"})


#// 12
#//
#// User [b] can see [widget_1]
GET /integration/_search?filter=type:any(embedded%2Cfloating%2Call%2Cpage%2Cmodal)&from=0&size=100&query=widget_1&t=1763624116684
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
#     { "_source.name": "  widget_1" }
#   ]
# })


#// 13
#//
#// User [c] can see [widget_1]
GET /integration/_search?filter=type:any(embedded%2Cfloating%2Call%2Cpage%2Cmodal)&from=0&size=100&query=widget_1&t=1763624116685
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
#     { "_source.name": "  widget_1" }
#   ]
# })


#// 14
#//
#// User [d] cannot see [widget_1]
GET /integration/_search?filter=type:any(embedded%2Cfloating%2Call%2Cpage%2Cmodal)&from=0&size=100&query=widget_1&t=1763624116686
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
#//  1. User [admin] deletes [widget_1]
#//  2. [widget_1] is invisible to users [a, b, c, d]
#//
#//----------------------------------------------------------------------------


#// 15
#//
#// User [admin] deletes [widget_1]
DELETE /integration/$[[env.WIDGET_A_ID]]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"_id":"$[[env.WIDGET_A_ID]]","result":"deleted"})


#// 16
#//
#// User [a] cannot see [widget_1]
GET /integration/_search?filter=type:any(embedded%2Cfloating%2Call%2Cpage%2Cmodal)&from=0&size=100&query=widget_1&t=1763624116687
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
#// User [b] cannot see [widget_1]
GET /integration/_search?filter=type:any(embedded%2Cfloating%2Call%2Cpage%2Cmodal)&from=0&size=100&query=widget_1&t=1763624116688
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
#// User [c] cannot see [widget_1]
GET /integration/_search?filter=type:any(embedded%2Cfloating%2Call%2Cpage%2Cmodal)&from=0&size=100&query=widget_1&t=1763624116689
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
#// User [d] cannot see [widget_1]
GET /integration/_search?filter=type:any(embedded%2Cfloating%2Call%2Cpage%2Cmodal)&from=0&size=100&query=widget_1&t=1763624116690
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
