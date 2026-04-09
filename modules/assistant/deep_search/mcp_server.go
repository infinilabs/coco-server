/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package deep_search

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"infini.sh/coco/core"
	common2 "infini.sh/coco/modules/assistant/common"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

// GetAssistantFn is a function that retrieves an assistant by ID.
//
// It exists because we need to use it to break the import cycle between
// deep_search and service packages.
type GetAssistantFn func(ctx context.Context, assistantID string) (*core.Assistant, bool, error)

// userClaimsKey is the context key for storing authenticated user claims (*security.UserClaims).
type userClaimsKey struct{}

// authErrorKey is the context key for storing the authentication error message.
type authErrorKey struct{}

// assistantIDKey is the context key for storing the assistant ID extracted from the URL path.
type assistantIDKey struct{}

// newContextFunc creates an HTTPContextFunc that extracts both auth and
// assistant_id from the request.  assistant_id is parsed from the URL
// path: basePath/{assistant_id}
func newContextFunc(basePath string) server.HTTPContextFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		/*
		   Extract assistant_id from URL path.
		   Note: validation of assistant_id (existence check) is deferred to the
		   tool handler, because HTTPContextFunc can only return a context — it
		   cannot return an error or write an HTTP error response.
		*/
		path := strings.TrimPrefix(r.URL.Path, basePath)
		path = strings.Trim(path, "/")
		if segments := strings.SplitN(path, "/", 2); len(segments) > 0 && segments[0] != "" {
			ctx = context.WithValue(ctx, assistantIDKey{}, segments[0])
		}

		/*
		   authenticate via X-API-TOKEN
		*/
		claims, err := core.ValidateLoginByAPITokenHeader(nil, r)
		if err != nil {
			log.Warn("MCP auth failed: ", err)
			return context.WithValue(ctx, authErrorKey{}, err.Error())
		}
		ctx = context.WithValue(ctx, userClaimsKey{}, claims)
		// Also register the user in the framework's standard context key so that
		// downstream components (e.g. enterprise_search, internal_search) that
		// call security.MustGetUserFromContext(ctx) can find the authenticated user.
		ctx = security.AddUserToContext(ctx, claims.UserSessionInfo)
		return ctx
	}
}

func newSearchToolHandler(getAssistant GetAssistantFn) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fmt.Printf("DBG: mcp server executed")

		// Extract user from context
		claims, ok := ctx.Value(userClaimsKey{}).(*security.UserClaims)
		if !ok {
			if errMsg, hasErr := ctx.Value(authErrorKey{}).(string); hasErr {
				return mcp.NewToolResultError(fmt.Sprintf("authentication failed: %s", errMsg)), nil
			} else {
				// X-API-TOKEN is not provided
				return mcp.NewToolResultError("authentication required: provide X-API-TOKEN header"), nil
			}
		}
		userID := claims.MustGetUserID()

		// Extract tool arguments
		query, _ := request.GetArguments()["query"].(string)
		if query == "" {
			return mcp.NewToolResultError("missing required argument: query"), nil
		}

		size := 10 // default value
		if v, ok := request.GetArguments()["size"].(float64); ok && v > 0 {
			size = int(v)
		}

		// Get assistant_id from context (extracted from URL path)
		assistantID, _ := ctx.Value(assistantIDKey{}).(string)
		if assistantID == "" {
			return mcp.NewToolResultError("missing assistant_id in URL path"), nil
		}

		// Get assistant config
		assistant, exists, err := getAssistant(ctx, assistantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get assistant: %v", err)), nil
		}
		if !exists {
			return mcp.NewToolResultError(fmt.Sprintf("assistant not found: %s", assistantID)), nil
		}
		if assistant.DeepThinkConfig == nil {
			return mcp.NewToolResultError("assistant does not have DeepThinkConfig configured"), nil
		}

		// build RAGContext
		params := &common2.RAGContext{
			SearchDB:     true,
			MCP:          assistant.MCPConfig.Enabled,
			Datasource:   strings.Join(assistant.Datasource.GetIDs(), ","),
			AssistantCfg: assistant,
			InputValues: map[string]any{
				"query": query,
				// Needed: search pipeline failed: var [history] required, but was not found
				"history": "</empty>",
			},
		}
		if assistant.MCPConfig.Enabled {
			params.MCPServers = assistant.MCPConfig.GetIDs()
		}

		// build messages
		reqMsg := &core.ChatMessage{Message: query}
		reqMsg.ID = util.GetUUID()

		replyMsg := &core.ChatMessage{}
		replyMsg.ID = util.GetUUID()

		sender := &common2.MemoryMessageSender{}

		// run search pipeline
		docs, err := RunSearchPipeline(ctx, userID, params, assistant, reqMsg, replyMsg, sender)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search pipeline failed: %v", err)), nil
		}

		if len(docs) == 0 {
			return mcp.NewToolResultText("no documents found"), nil
		}

		if size < len(docs) {
			docs = docs[:size]
		}

		result := util.MustToJSON(docs)
		return mcp.NewToolResultText(result), nil
	}
}

// NewMCPHandler creates an MCP server http.Handler for the search tool.
// URL pattern: basePath/{assistant_id}, e.g. /search/mcp/{assistant_id}
func NewMCPHandler(basePath string, getAssistant GetAssistantFn) http.Handler {
	basePath = strings.TrimSuffix(basePath, "/") + "/"

	mcpServer := server.NewMCPServer(
		"coco-search",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	mcpServer.AddTool(
		mcp.NewTool("search",
			mcp.WithDescription("Search documents using Coco's deep search pipeline with intent analysis"),
			mcp.WithString("query",
				mcp.Description("The search query string"),
				mcp.Required(),
			),
			mcp.WithNumber("size",
				mcp.Description("Maximum number of documents to return (default: 10)"),
			),
		),
		newSearchToolHandler(getAssistant),
	)

	return server.NewStreamableHTTPServer(mcpServer,
		server.WithHTTPContextFunc(newContextFunc(basePath)),
	)
}
