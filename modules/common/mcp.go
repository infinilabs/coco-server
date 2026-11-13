/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/framework/core/util"
)

// MCPQueryEnvelopeSchema builds an MCP tool input schema for GET-style routes.
// The framework MCP bridge decodes tool arguments into a
// {path_params, query, headers, body, raw_body} envelope, so the typed
// parameters a caller sees must live under `query` (or `path_params`) — a flat
// {query: "..."} schema would fail envelope decoding.
func MCPQueryEnvelopeSchema(params util.MapStr, required []string) string {
	query := util.MapStr{
		"type":        "object",
		"description": "Query string parameters.",
		"properties":  params,
	}
	if len(required) > 0 {
		query["required"] = required
	}
	return util.ToJson(util.MapStr{
		"type": "object",
		"properties": util.MapStr{
			"query": query,
		},
		"required":             []string{"query"},
		"additionalProperties": false,
	}, false)
}

// MCPBodyEnvelopeSchema builds an MCP tool input schema for POST/PUT-style
// routes where the typed parameters are sent as the JSON request body.
func MCPBodyEnvelopeSchema(params util.MapStr, required []string) string {
	body := util.MapStr{
		"type":       "object",
		"properties": params,
	}
	if len(required) > 0 {
		body["required"] = required
	}
	return util.ToJson(util.MapStr{
		"type": "object",
		"properties": util.MapStr{
			"body": body,
		},
		"required":             []string{"body"},
		"additionalProperties": false,
	}, false)
}
