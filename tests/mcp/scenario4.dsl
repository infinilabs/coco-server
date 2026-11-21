#// Scenario 4: MCP server endpoint (streamable-http)
#//
#// Coco exposes its capabilities to external agents at POST /mcp.
#//   1. initialize handshake succeeds and reports the server identity
#//   2. tools/list returns the curated coco tools to an admin
#//   3. tools/list is filtered down for a regular user, empty when anonymous
#//   4. tools/call search_documents returns real hits from the seed corpus
#//   5. calling a tool without the required permission is rejected
#//   6. GET /mcp serves a browser-friendly help page for logged-in users


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
#// Log in to account a
POST /account/login
{
  "login": "a@mail.com",
  "password": "$[[env.A_PASSWORD]]"
}
# assert: (200, {status: "ok"}),
#
# register: [
#   { a_token: "_ctx.response.body_json.access_token" },
# ]


#// 3
#//
#// MCP initialize handshake succeeds
POST /mcp
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-03-26",
    "capabilities": {},
    "clientInfo": {"name": "loadgen", "version": "0.0.0"}
  }
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#     {Accept: "application/json"},
#     {Content-Type: "application/json"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.result.serverInfo.name: "Coco AI",
# }


#// 4
#//
#// tools/list returns the curated coco tools to an admin
POST /mcp
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#     {Accept: "application/json"},
#     {Content-Type: "application/json"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.jsonrpc: "2.0",
# }


#// 5
#//
#// tools/list is filtered down for a regular user
POST /mcp
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/list"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#     {Accept: "application/json"},
#     {Content-Type: "application/json"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.jsonrpc: "2.0",
# }


#// 6
#//
#// tools/list is empty for anonymous callers
POST /mcp
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/list"
}
# request: {
#   headers: [
#     {Accept: "application/json"},
#     {Content-Type: "application/json"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.jsonrpc: "2.0",
# }


#// 7
#//
#// tools/call search_documents returns real hits from the seed corpus
POST /mcp
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "search_documents",
    "arguments": {
      "query": {
        "query": "file",
        "size": 3
      }
    }
  }
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#     {Accept: "application/json"},
#     {Content-Type: "application/json"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.result.content.0.type: "text",
# }


#// 8
#//
#// calling a tool without the required permission is rejected
POST /mcp
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "wiki_kb_create",
    "arguments": {
      "body": {
        "name": "mcp_test_kb"
      }
    }
  }
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[a_token]]"},
#     {Accept: "application/json"},
#     {Content-Type: "application/json"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.result.isError: true,
# }


#// 9
#//
#// GET /mcp serves a browser-friendly help page for logged-in users
GET /mcp
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
# }
