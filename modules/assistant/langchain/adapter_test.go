// Package langchaingo_mcp_adapter_test implements tests for the MCP adapter.
package langchain

import (
	"context"
	"testing"

	adapter "github.com/i2y/langchaingo-mcp-adapter"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMCPClient is a mock implementation of the MCPClient.
type MockMCPClient struct {
	mock.Mock
}

// Initialize mocks the Initialize method of the MCPClient.
func (m *MockMCPClient) Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*mcp.InitializeResult), args.Error(1)
}

// Ping mocks the Ping method of the MCPClient.
func (m *MockMCPClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// ListResources mocks the ListResources method of the MCPClient.
func (m *MockMCPClient) ListResources(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*mcp.ListResourcesResult), args.Error(1)
}

// ListResourceTemplates mocks the ListResourceTemplates method of the MCPClient.
func (m *MockMCPClient) ListResourceTemplates(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*mcp.ListResourceTemplatesResult), args.Error(1)
}

// ReadResource mocks the ReadResource method of the MCPClient.
func (m *MockMCPClient) ReadResource(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*mcp.ReadResourceResult), args.Error(1)
}

// Subscribe mocks the Subscribe method of the MCPClient.
func (m *MockMCPClient) Subscribe(ctx context.Context, request mcp.SubscribeRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

// Unsubscribe mocks the Unsubscribe method of the MCPClient.
func (m *MockMCPClient) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

// ListPrompts mocks the ListPrompts method of the MCPClient.
func (m *MockMCPClient) ListPrompts(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*mcp.ListPromptsResult), args.Error(1)
}

// GetPrompt mocks the GetPrompt method of the MCPClient.
func (m *MockMCPClient) GetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*mcp.GetPromptResult), args.Error(1)
}

// ListTools mocks the ListTools method of the MCPClient.
func (m *MockMCPClient) ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*mcp.ListToolsResult), args.Error(1)
}

// CallTool mocks the CallTool method of the MCPClient.
func (m *MockMCPClient) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*mcp.CallToolResult), args.Error(1)
}

// SetLevel mocks the SetLevel method of the MCPClient.
func (m *MockMCPClient) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

// Complete mocks the Complete method of the MCPClient.
func (m *MockMCPClient) Complete(ctx context.Context, request mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*mcp.CompleteResult), args.Error(1)
}

// Close mocks the Close method of the MCPClient.
func (m *MockMCPClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// OnNotification mocks the OnNotification method of the MCPClient.
func (m *MockMCPClient) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
	m.Called(handler)
}

// TestNew tests the New function of the adapter.
func TestNew(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockMCPClient)
		expectError    bool
		expectedErrMsg string
	}{
		{
			name: "successful initialization",
			setupMock: func(m *MockMCPClient) {
				m.On("Initialize", mock.Anything, mock.Anything).Return(&mcp.InitializeResult{
					ServerInfo: mcp.Implementation{
						Name:    "test-server",
						Version: "1.0.0",
					},
				}, nil)
			},
			expectError: false,
		},
		{
			name: "initialization error",
			setupMock: func(m *MockMCPClient) {
				m.On("Initialize", mock.Anything, mock.Anything).Return(
					&mcp.InitializeResult{}, assert.AnError)
			},
			expectError:    true,
			expectedErrMsg: "initialize: assert.AnError general error for testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMCPClient{}
			tt.setupMock(mockClient)

			adapter, err := adapter.New(mockClient)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, adapter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, adapter)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// TestTools tests the Tools method of the adapter.
func TestTools(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockMCPClient)
		expectError    bool
		expectedErrMsg string
		expectedTools  int
	}{
		{
			name: "successful tools retrieval",
			setupMock: func(m *MockMCPClient) {
				m.On("Initialize", mock.Anything, mock.Anything).Return(&mcp.InitializeResult{
					ServerInfo: mcp.Implementation{
						Name:    "test-server",
						Version: "1.0.0",
					},
				}, nil)

				m.On("ListTools", mock.Anything, mock.Anything).Return(&mcp.ListToolsResult{
					Tools: []mcp.Tool{
						{
							Name:        "tool1",
							Description: "Test Tool 1",
							InputSchema: mcp.ToolInputSchema{
								Properties: map[string]any{
									"param1": map[string]any{
										"type":        "string",
										"description": "Parameter 1",
									},
								},
							},
						},
						{
							Name:        "tool2",
							Description: "Test Tool 2",
							InputSchema: mcp.ToolInputSchema{
								Properties: map[string]any{
									"param1": map[string]any{
										"type":        "number",
										"description": "Parameter 1",
									},
								},
							},
						},
					},
				}, nil)
			},
			expectError:   false,
			expectedTools: 2,
		},
		{
			name: "list tools error",
			setupMock: func(m *MockMCPClient) {
				m.On("Initialize", mock.Anything, mock.Anything).Return(&mcp.InitializeResult{
					ServerInfo: mcp.Implementation{
						Name:    "test-server",
						Version: "1.0.0",
					},
				}, nil)

				m.On("ListTools", mock.Anything, mock.Anything).Return(
					&mcp.ListToolsResult{}, assert.AnError)
			},
			expectError:    true,
			expectedErrMsg: "list tools: assert.AnError general error for testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMCPClient{}
			tt.setupMock(mockClient)

			a, err := adapter.New(mockClient)
			if err != nil && !tt.expectError {
				t.Fatalf("Failed to create adapter: %v", err)
			}

			if tt.expectError {
				if err == nil {
					tools, err := a.Tools()
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
					assert.Nil(t, tools)
				}
			} else {
				tools, err := a.Tools()
				assert.NoError(t, err)
				assert.Len(t, tools, tt.expectedTools)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// TestToolCall tests the Call method of the created tools.
func TestToolCall(t *testing.T) {
	tests := []struct {
		name           string
		toolName       string
		toolDesc       string
		inputSchema    map[string]any
		setupMock      func(*MockMCPClient)
		input          string
		expectError    bool
		expectedErrMsg string
		expectedOutput string
	}{
		{
			name:        "successful tool call",
			toolName:    "test-tool",
			toolDesc:    "A test tool",
			inputSchema: map[string]any{"param1": map[string]any{"type": "string"}},
			setupMock: func(m *MockMCPClient) {
				m.On("CallTool", mock.Anything, mock.MatchedBy(func(req mcp.CallToolRequest) bool {
					return req.Params.Name == "test-tool"
				})).Return(&mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "tool execution result",
						},
					},
				}, nil)
			},
			input:          `{"param1": "test value"}`,
			expectError:    false,
			expectedOutput: "tool execution result",
		},
		{
			name:           "invalid json input",
			toolName:       "test-tool",
			toolDesc:       "A test tool",
			inputSchema:    map[string]any{"param1": map[string]any{"type": "string"}},
			input:          `invalid json`,
			expectError:    true,
			expectedErrMsg: "unmarshal input",
		},
		{
			name:        "tool call error",
			toolName:    "test-tool",
			toolDesc:    "A test tool",
			inputSchema: map[string]any{"param1": map[string]any{"type": "string"}},
			setupMock: func(m *MockMCPClient) {
				m.On("CallTool", mock.Anything, mock.Anything).Return(
					&mcp.CallToolResult{}, assert.AnError)
			},
			input:          `{"param1": "test value"}`,
			expectError:    true,
			expectedErrMsg: "call the tool: assert.AnError general error for testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockMCPClient{}

			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			tool, err := adapter.NewToolForTesting(tt.toolName, tt.toolDesc, tt.inputSchema, mockClient)
			require.NoError(t, err)

			result, err := tool.Call(context.Background(), tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, result)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
