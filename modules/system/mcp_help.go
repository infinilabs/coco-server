/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"

	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
)

type mcpHelpTool struct {
	Name        string
	Description string
}

type mcpHelpData struct {
	Endpoint string
	Tools    []mcpHelpTool
}

var mcpHelpTemplate = template.Must(template.New("mcp-help").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Endpoint}} — Coco AI MCP Server</title>
<style>
  :root { color-scheme: light dark; }
  * { box-sizing: border-box; }
  body {
    margin: 0; padding: 40px 20px; font: 14px/1.6 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    background: #f5f6f8; color: #1f2329;
  }
  @media (prefers-color-scheme: dark) {
    body { background: #17171a; color: #e8e8e9; }
    .card { background: #202024 !important; border-color: #34343a !important; }
    code, pre { background: #17171a !important; }
  }
  .wrap { max-width: 860px; margin: 0 auto; }
  h1 { font-size: 22px; margin: 0 0 4px; }
  .sub { color: #6b6f76; margin-bottom: 28px; }
  .card {
    background: #fff; border: 1px solid #e4e6eb; border-radius: 10px;
    padding: 20px 24px; margin-bottom: 20px;
  }
  .card h2 { font-size: 15px; margin: 0 0 12px; }
  code {
    background: #f0f1f3; border-radius: 4px; padding: 2px 6px;
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 13px;
  }
  pre {
    background: #f0f1f3; border-radius: 8px; padding: 14px 16px; overflow-x: auto; margin: 0;
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12.5px; line-height: 1.55;
  }
  table { border-collapse: collapse; width: 100%; }
  td { padding: 7px 10px 7px 0; vertical-align: top; }
  td.name { white-space: nowrap; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 13px; }
  .hint { color: #6b6f76; font-size: 13px; margin-top: 10px; }
  .badge { display: inline-block; background: #e8f2ff; color: #0066cc; border-radius: 4px; padding: 1px 8px; font-size: 12px; margin-left: 8px; }
  @media (prefers-color-scheme: dark) { .badge { background: #10233f; color: #66aaff; } }
</style>
</head>
<body>
<div class="wrap">
  <h1>Coco AI MCP Server</h1>
  <div class="sub">Operate Coco AI from any MCP client — search, assistants, wiki and more.</div>

  <div class="card">
    <h2>Endpoint<span class="badge">streamable-http</span></h2>
    <pre>{{.Endpoint}}</pre>
    <div class="hint">Stateless JSON-RPC 2.0 over HTTP POST. This page is the browser-friendly view; MCP clients should POST to the URL above.</div>
  </div>

  <div class="card">
    <h2>Authentication</h2>
    <p style="margin:0 0 8px">Every request must carry a credential — any one of:</p>
    <table>
      <tr><td class="name">X-API-TOKEN</td><td>an API token created in the console (Security → API Token). Recommended for MCP clients.</td></tr>
      <tr><td class="name">Authorization: Bearer</td><td>a JWT access token from <code>POST /account/login</code>.</td></tr>
      <tr><td class="name">Cookie</td><td>the console login session (what you are using to read this page).</td></tr>
    </table>
    <div class="hint">Visible tools are filtered by the caller's permissions — the same permissions the console enforces.</div>
  </div>

  <div class="card">
    <h2>Client configuration</h2>
    <pre>{
  "mcpServers": {
    "coco": {
      "url": "{{.Endpoint}}",
      "headers": { "X-API-TOKEN": "&lt;your-api-token&gt;" }
    }
  }
}</pre>
    <div class="hint">Works with Claude Desktop, Cursor, ZCode and any streamable-http MCP client.</div>
  </div>

  <div class="card">
    <h2>Try it with curl</h2>
    <pre># initialize
curl -s {{.Endpoint}} \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -H 'X-API-TOKEN: &lt;your-api-token&gt;' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"curl","version":"0.0.0"}}}'

# list tools
curl -s {{.Endpoint}} \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -H 'X-API-TOKEN: &lt;your-api-token&gt;' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'</pre>
  </div>

  <div class="card">
    <h2>Available tools</h2>
    <table>
      {{range .Tools}}
      <tr><td class="name">{{.Name}}</td><td>{{.Description}}</td></tr>
      {{else}}
      <tr><td>No MCP tools registered.</td></tr>
      {{end}}
    </table>
  </div>
</div>
</body>
</html>`))

// serveMCPHelpPage renders a browser-friendly guide for the /mcp endpoint.
// The streamable-HTTP MCP server itself only answers JSON-RPC POSTs; a human
// opening the URL in a browser deserves connection instructions instead of an
// error page. Registered in the router tree, which takes precedence over the
// framework mux fallback that serves the MCP protocol.
func (h *APIHandler) serveMCPHelpPage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	endpoint := fmt.Sprintf("%s://%s%s", scheme, r.Host, mcpBasePath())

	tools := []mcpHelpTool{}
	api.WalkMCPAutoUIMethodRoutes(func(route api.RegisteredUIMethodRoute) {
		name := fmt.Sprintf("%s_%s", strings.ToLower(string(route.Route.Method)), strings.Trim(route.Route.Path, "/"))
		description := fmt.Sprintf("Call %s %s", route.Route.Method, route.Route.Path)
		if route.Options != nil && route.Options.Labels != nil {
			if v, ok := route.Options.Labels[api.MCPToolName]; ok {
				if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
					name = s
				}
			}
			if v, ok := route.Options.Labels[api.MCPToolDescription]; ok {
				if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
					description = s
				}
			}
		}
		tools = append(tools, mcpHelpTool{Name: name, Description: description})
	})
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = mcpHelpTemplate.Execute(w, mcpHelpData{Endpoint: endpoint, Tools: tools})
}

func mcpBasePath() string {
	cfg := global.Env().SystemConfig.WebAppConfig
	if cfg.MCP == (config.MCPConfig{}) {
		return "/mcp"
	}
	path := strings.TrimSpace(cfg.MCP.BasePath)
	if path == "" {
		return "/mcp"
	}
	return path
}
