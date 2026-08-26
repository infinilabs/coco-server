#// Scenario 4:
#//
#// The public widget wrapper API works by both document ID and tenant:alias,
#// and the (tenant, alias) pair stays unique


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
	"login": "admin@mail.com",
	"password": "$[[env.ADMIN_PASSWORD]]"
}
# assert: (200, {status: "ok"}),
#
# register: [
#   { admin_token: "_ctx.response.body_json.access_token" },
# ]


#//----------------------------------------------------------------------------
#//
#// Case 1:
#//  1. User [admin] sets alias [infinilabs:coco-website-searchbox] on the
#//     built-in [full-screen-widget-default] widget
#//  2. The alias pair is persisted on the widget
#//  3. Another widget cannot take the same alias pair
#//
#//----------------------------------------------------------------------------


#// 2
#//
#// User [admin] sets tenant and alias on the built-in fullscreen widget
PUT /integration/$[[env.WIDGET_FS_ID]]
{
	"tenant": "$[[env.WIDGET_FS_TENANT]]",
	"alias": "$[[env.WIDGET_FS_ALIAS]]",
	"enabled": true
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"_id":"$[[env.WIDGET_FS_ID]]","result":"updated"})


#// 3
#//
#// The alias pair is persisted on the widget
GET /integration/$[[env.WIDGET_FS_ID]]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {
#   "_id": "$[[env.WIDGET_FS_ID]]",
#   "_source.tenant": "$[[env.WIDGET_FS_TENANT]]",
#   "_source.alias": "$[[env.WIDGET_FS_ALIAS]]"
# })


#// 4
#//
#// Another widget cannot take the same (tenant, alias) pair
POST /integration/
{
	"name": "widget_alias_dup",
	"type": "embedded",
	"enabled": false,
	"tenant": "$[[env.WIDGET_FS_TENANT]]",
	"alias": "$[[env.WIDGET_FS_ALIAS]]"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: {
#   _ctx.response.status: 400,
# }


#//----------------------------------------------------------------------------
#//
#// Case 2:
#//  1. The wrapper API serves the widget JS by document ID
#//  2. The wrapper API serves the same widget JS by tenant:alias
#//  3. An unknown alias is rejected with not_found
#//
#//----------------------------------------------------------------------------


#// 5
#//
#// Wrapper API by document ID returns the widget JS
GET /integration/$[[env.WIDGET_FS_ID]]/widget
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.header.content-type: "application/javascript",
# }


#// 6
#//
#// Wrapper API by tenant:alias returns the same widget JS
GET /integration/$[[env.WIDGET_FS_TENANT]]:$[[env.WIDGET_FS_ALIAS]]/widget
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.header.content-type: "application/javascript",
# }


#// 7
#//
#// An unknown alias is rejected with not_found
GET /integration/$[[env.WIDGET_FS_TENANT]]:alias-not-exist/widget
# assert: {
#   _ctx.response.status: 404,
#   _ctx.response.body_json.result: "not_found",
# }


#//----------------------------------------------------------------------------
#//
#// Case 3:
#//  1. User [admin] clears the alias pair on the widget
#//  2. The stored tenant/alias are empty (an omitted empty string in the
#//     partial update would silently keep the old alias)
#//  3. The by-alias wrapper no longer resolves, the by-ID wrapper still does
#//
#//----------------------------------------------------------------------------


#// 8
#//
#// User [admin] clears tenant and alias on the widget
PUT /integration/$[[env.WIDGET_FS_ID]]
{
	"tenant": "",
	"alias": "",
	"enabled": true
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: (200, {"_id":"$[[env.WIDGET_FS_ID]]","result":"updated"})


#// 9
#//
#// The stored tenant and alias are empty
GET /integration/$[[env.WIDGET_FS_ID]]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
#
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json._source.tenant: "",
#   _ctx.response.body_json._source.alias: "",
# }


#// 10
#//
#// The by-alias wrapper no longer resolves the widget
GET /integration/$[[env.WIDGET_FS_TENANT]]:$[[env.WIDGET_FS_ALIAS]]/widget
# assert: {
#   _ctx.response.status: 404,
#   _ctx.response.body_json.result: "not_found",
# }


#// 11
#//
#// The by-ID wrapper still serves the widget JS
GET /integration/$[[env.WIDGET_FS_ID]]/widget
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.header.content-type: "application/javascript",
# }
