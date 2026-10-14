/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package skill

import (
	"fmt"
	"net/http"
	"strings"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

// mcpEndpointFromRequest derives the absolute /mcp URL the way the help page
// does, so exported client configs are copy-paste ready.
func mcpEndpointFromRequest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	path := strings.TrimSpace(global.Env().SystemConfig.WebAppConfig.MCP.BasePath)
	if path == "" {
		path = "/mcp"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, path)
}

// exportMCPJSON returns a drop-in client configuration for the coco MCP
// server (Claude Desktop / Cursor / ZCode, any streamable-http client).
func (h *APIHandler) exportMCPJSON(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	cfg := util.MapStr{
		"mcpServers": util.MapStr{
			"coco": util.MapStr{
				"url":     mcpEndpointFromRequest(r),
				"headers": util.MapStr{"X-API-TOKEN": "<your-api-token>"},
			},
		},
	}
	h.WriteJSON(w, cfg, http.StatusOK)
}

// exportSkillsMD renders a SKILLS.md for external agents: the enabled coco
// skills as one markdown skill file, generated from the user's real records
// (not a hardcoded catalog).
func (h *APIHandler) exportSkillsMD(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	skills, err := GetEnabledSkills()
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var sb strings.Builder
	sb.WriteString("---\nname: coco\ndescription: ")
	sb.WriteString("Operate Coco AI - enterprise search, assistants, wiki and more - from this agent via the coco MCP server")
	sb.WriteString("\n---\n\n")
	sb.WriteString("# Coco AI Skills\n\n")
	sb.WriteString(fmt.Sprintf("Connect to Coco AI over MCP first (endpoint: `%s`), then apply the skills below when relevant.\n\n", mcpEndpointFromRequest(r)))

	for _, s := range skills {
		sb.WriteString("## ")
		sb.WriteString(s.Title)
		sb.WriteString("\n\n")
		if s.Description != "" {
			sb.WriteString(s.Description)
			sb.WriteString("\n\n")
		}
		sb.WriteString(s.Instructions)
		sb.WriteString("\n\n")
	}
	if len(skills) == 0 {
		sb.WriteString("(no skills enabled - enable skills in the Coco console to export them here)\n")
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=SKILLS.md")
	_, _ = w.Write([]byte(sb.String()))
}
