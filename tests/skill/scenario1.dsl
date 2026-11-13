#// Scenario 1: skill lifecycle — seeded builtins, CRUD, builtin protection
#// and the export endpoints for external agents.


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


#// 2
#//
#// Seeded builtin skills are present after startup
GET /skill/_search?size=50
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
# register: [
#   { builtin_skill_id: "_ctx.response.body_json.hits.hits.0._id" },
# ],
# assert: {
#   _ctx.response.status: 200,
# }


#// 3
#//
#// Get the seeded retrieval-expert skill with its instructions
GET /skill/$[[builtin_skill_id]]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
# }


#// 4
#//
#// Create a user skill (builtin flag in input must be ignored)
POST /skill/
{
  "name": "scenario-skill",
  "title": "Scenario Skill",
  "description": "created by integration test",
  "instructions": "Always answer with a haiku.",
  "builtin": true,
  "enabled": true
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
# register: [
#   { user_skill_id: "_ctx.response.body_json._id" },
# ],
# assert: {
#   _ctx.response.status: 200,
# }


#// 5
#//
#// The created skill is not builtin
GET /skill/$[[user_skill_id]]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json._source.builtin: false,
# }


#// 6
#//
#// Update the user skill
PUT /skill/$[[user_skill_id]]
{
  "title": "Scenario Skill v2"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
# }


#// 7
#//
#// Builtin skills cannot be deleted
DELETE /skill/$[[builtin_skill_id]]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
# assert: {
#   _ctx.response.status: 403,
# }


#// 8
#//
#// User skills can be deleted
DELETE /skill/$[[user_skill_id]]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
# }


#// 9
#//
#// Export a drop-in MCP client configuration
GET /skill/_export/mcp_json
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.mcpServers.coco.url: "http://127.0.0.1:9000/mcp",
# }


#// 10
#//
#// Export SKILLS.md generated from real skill records
GET /skill/_export/skills_md
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
# }
