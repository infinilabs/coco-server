// Package langchaingo_mcp_adapter provides an adapter between LangChain Go and MCP servers.
package langchain

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/util"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	langchaingoTools "github.com/tmc/langchaingo/tools"
)

// mcpTool implements the langchaingoTools.Tool interface for MCP tools.
type mcpTool struct {
	name        string
	description string
	inputSchema []byte
	client      client.MCPClient
}

// Name returns the name of the tool.
func (t *mcpTool) Name() string {
	return t.name
}

// Description returns the description of the tool along with its input schema.
func (t *mcpTool) Description() string {
	return t.description + "\n The input schema is: " + string(t.inputSchema)
}

// Call invokes the MCP tool with the given input and returns the result.
func (t *mcpTool) Call(ctx context.Context, input string) (string, error) {

	log.Debugf("call action: %v, args: %v", t.name, input)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
	}
	req.Params.Name = t.name

	var args map[string]interface{}
	err := json.Unmarshal([]byte(input), &args)
	if err != nil {
		return "", fmt.Errorf("unmarshal input: %w", err)
	}
	req.Params.Arguments = args

	log.Debugf("call action: %v, args: %v", t.name, util.MustToJSON(req))

	res, err := t.client.CallTool(ctx, req)
	if err != nil {
		return "", fmt.Errorf("call the tool: %w", err)
	}

	return res.Content[0].(mcp.TextContent).Text, nil
}

// MCPAdapter adapts an MCP client to the LangChain Go tools interface.
type MCPAdapter struct {
	client client.MCPClient
}

// New creates a new MCPAdapter instance with the given MCP client.
// It initializes the connection with the MCP server.
func New(client client.MCPClient) (*MCPAdapter, error) {
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "langchaingo-mcp-adapter",
		Version: "1.0.0",
	}

	initResult, err := client.Initialize(context.Background(), initRequest)
	if err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	slog.Debug(
		"Initialized with server",
		"name",
		initResult.ServerInfo.Name,
		"version",
		initResult.ServerInfo.Version,
	)

	return &MCPAdapter{
		client: client,
	}, nil
}

// Tools returns a list of all available tools from the MCP server.
// Each tool is wrapped as a langchaingoTools.Tool.
func (a *MCPAdapter) Tools() ([]langchaingoTools.Tool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	toolsRequest := mcp.ListToolsRequest{}
	tools, err := a.client.ListTools(ctx, toolsRequest)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	var mcpTools []langchaingoTools.Tool

	for _, tool := range tools.Tools {
		log.Debug("tool name:", tool.Name, ", description:", tool.Description)

		mcpTool, err := newLangchaingoTool(tool.Name, tool.Description, tool.InputSchema.Properties, a.client)
		if err != nil {
			return nil, fmt.Errorf("new langchaingo tool: %w", err)
		}
		mcpTools = append(mcpTools, mcpTool)
	}

	return mcpTools, nil
}

// newLangchaingoTool creates a new langchaingo tool from MCP tool information.
func newLangchaingoTool(name, description string, inputSchema map[string]any, client client.MCPClient) (langchaingoTools.Tool, error) {
	jsonSchema, err := json.Marshal(inputSchema)
	if err != nil {
		return nil, fmt.Errorf("marshal input schema: %w", err)
	}

	return &mcpTool{
		name:        name,
		description: description,
		inputSchema: jsonSchema,
		client:      client,
	}, nil
}

// NewToolForTesting creates an mcpTool instance for testing purposes.
// This function is for testing only and should not be used in production applications.
func NewToolForTesting(name, description string, inputSchema map[string]any, client client.MCPClient) (langchaingoTools.Tool, error) {
	return newLangchaingoTool(name, description, inputSchema, client)
}
