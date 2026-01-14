package deep_research

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/assistant/langchain"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
)

// CreateResearcherGraph creates the researcher subgraph for conducting focused research
func CreateResearcherGraph(ctx context.Context, config *core.DeepResearchConfig) (*graph.MessageGraph, error) {
	workflow := graph.NewMessageGraph()

	// Set up schema with reducers
	schema := graph.NewMapSchema()
	schema.RegisterReducer("messages", graph.AppendReducer)
	schema.RegisterReducer("raw_notes", graph.AppendReducer)
	workflow.SetSchema(schema)

	// Initialize search tool
	searchTool := &TavilySearchTool{APIKey: "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"}
	enterpriseSearchTool := &EnterpriseSearchTool{}
	thinkTool := &ThinkToolImpl{}

	// Researcher node - conducts research using search tools
	workflow.AddNode("researcher", "researcher", func(ctx context.Context, state interface{}) (interface{}, error) {
		mState, ok := state.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid state type")
		}

		messages, ok := mState["messages"].([]llms.MessageContent)
		if !ok {
			return nil, fmt.Errorf("messages not found in state")
		}

		toolCallIterations, _ := mState["tool_call_iterations"].(int)

		//log.Error("config.MaxToolCallIterations:", config.MaxToolCallIterations)

		// Check iteration limit
		if toolCallIterations >= config.MaxToolCallIterations {
			log.Infof("[Researcher] Reached max tool call iterations (%d), ending research", config.MaxToolCallIterations)
			return map[string]interface{}{
				"messages": []llms.MessageContent{
					llms.TextParts(llms.ChatMessageTypeAI, "研究完成 - 达到迭代限制。"),
				},
			}, nil
		}

		// Prepare messages with system prompt
		systemPrompt := GetResearcherSystemPrompt(config.MaxToolCallIterations)

		var msgs []llms.MessageContent
		// Always start with system message
		msgs = append(msgs, llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt))
		// Then add conversation history
		msgs = append(msgs, messages...)

		// Define tools for the model
		toolDefs := []llms.Tool{
			{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        "enterprise_search",
					Description: "用于进行内部企业网络的数据搜索以收集信息。输入应该是搜索查询字符串。",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "搜索查询",
							},
						},
						"required": []string{"query"},
					},
				},
			}, {
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        "tavily_search",
					Description: "在网络上搜索信息。输入应该是搜索查询字符串。",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "搜索查询",
							},
						},
						"required": []string{"query"},
					},
				},
			},
			{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        "think_tool",
					Description: "使用此工具反思你的进度并规划下一步。",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"reflection": map[string]interface{}{
								"type":        "string",
								"description": "你对当前状态和下一步的反思",
							},
						},
						"required": []string{"reflection"},
					},
				},
			},
		}

		// Try non-streaming first, then fallback to streaming with fragment reconstruction
		var resp *llms.ContentResponse
		var researcherChunks strings.Builder

		resp, err := langchain.DirectGenerate(ctx, &config.ResearchModel, msgs, nil, func(chunk []byte, seq int) {
			researcherChunks.Write(chunk)
		}, llms.WithTools(toolDefs))
		if err != nil {
			return nil, fmt.Errorf("researcher call failed: %w", err)
		}

		choice := resp.Choices[0]
		aiMsg := llms.MessageContent{Role: llms.ChatMessageTypeAI}

		if choice.Content != "" {
			log.Tracef("Tool content：", choice.Content)
			aiMsg.Parts = append(aiMsg.Parts, llms.TextPart(choice.Content))
		}

		// Add tool calls to message - these should have proper structure with non-streaming
		for i, tc := range choice.ToolCalls {
			log.Tracef("Tool[%d] name: %s, ID: %s", i, tc.FunctionCall.Name, tc.ID)
			aiMsg.Parts = append(aiMsg.Parts, tc)
		}

		return map[string]interface{}{
			"messages":             []llms.MessageContent{aiMsg},
			"tool_call_iterations": toolCallIterations + 1,
		}, nil
	})

	// Researcher tools node - executes tool calls
	workflow.AddNode("researcher_tools", "researcher_tools", func(ctx context.Context, state interface{}) (interface{}, error) {
		mState, ok := state.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid state type")
		}

		messages := mState["messages"].([]llms.MessageContent)
		lastMsg := messages[len(messages)-1]

		var toolMessages []llms.MessageContent
		var rawNotes []string

		toolCalls := GetToolCalls(lastMsg)

		log.Trace("[Researcher] Tool calls found:", len(toolCalls))
		for i, tc := range toolCalls {
			log.Tracef("[Researcher] Tool[%d]: ID=%s, Name=%s, Args=%q", i, tc.ID, tc.FunctionCall.Name, tc.FunctionCall.Arguments)
		}

		actualToolCalls := []llms.ToolCall{}
		reconstructedArgs := ""
		currentCallName := ""
		currentCallID := ""

		for _, tc := range toolCalls {
			if tc.FunctionCall.Name != "" {
				// This is the start of a real tool call
				if reconstructedArgs != "" {
					actualToolCalls = append(actualToolCalls, llms.ToolCall{
						ID: currentCallID,
						FunctionCall: &llms.FunctionCall{
							Name:      currentCallName,
							Arguments: reconstructedArgs,
						},
					})
				}
				// Start new reconstruction
				currentCallID = tc.ID
				currentCallName = tc.FunctionCall.Name
				reconstructedArgs = tc.FunctionCall.Arguments
			} else if tc.FunctionCall.Name == "" && tc.FunctionCall.Arguments != "" {
				// This is a fragment of the previous call
				reconstructedArgs += tc.FunctionCall.Arguments
			}
		}

		// Add the last reconstructed call
		if currentCallName != "" && reconstructedArgs != "" {
			actualToolCalls = append(actualToolCalls, llms.ToolCall{
				ID: currentCallID,
				FunctionCall: &llms.FunctionCall{
					Name:      currentCallName,
					Arguments: reconstructedArgs,
				},
			})
		}

		log.Tracef("[Researcher] Tool calls after reconstruction: %d", len(actualToolCalls))
		for i, tc := range actualToolCalls {
			log.Tracef("[Researcher] Reconstructed tool[%d]: Name=%s, Args=%q", i, tc.FunctionCall.Name, tc.FunctionCall.Arguments)
		}

		// Use reconstructed calls if we have them
		if len(actualToolCalls) > 0 {
			toolCalls = actualToolCalls
		}

		// Show final tool calls after reconstruction
		log.Tracef("[Researcher] Final tool calls after reconstruction:", len(toolCalls))
		for i, tc := range toolCalls {
			log.Tracef("[Researcher] Final tool[%d]: ID=%s, Name=%s, Args=%q", i, tc.ID, tc.FunctionCall.Name, tc.FunctionCall.Arguments)
		}

		for _, tc := range toolCalls {
			args, err := ParseToolArguments(tc)
			if err != nil {
				log.Infof("[Researcher Tools] Failed to parse args: %v", err)
				continue
			}

			log.Trace("start call function: ", tc.FunctionCall.Name)

			var result string
			switch tc.FunctionCall.Name {
			case "enterprise_search":
				query, _ := args["query"].(string)
				result, err = enterpriseSearchTool.Call(ctx, query)
				if err != nil {
					result = fmt.Sprintf("Search error: %v", err)
				} else {
					// Store raw search results
					rawNotes = append(rawNotes, result)
				}
			case "tavily_search":
				query, _ := args["query"].(string)
				result, err = searchTool.Call(ctx, query)
				if err != nil {
					result = fmt.Sprintf("Search error: %v", err)
				} else {
					// Store raw search results
					rawNotes = append(rawNotes, result)
				}

			case "think_tool":
				reflection, _ := args["reflection"].(string)
				result, _ = thinkTool.Call(ctx, reflection)

			default:
				result = fmt.Sprintf("Unknown tool: %s", tc.FunctionCall.Name)
			}

			toolMsg := CreateToolMessage(tc.ID, tc.FunctionCall.Name, result)
			toolMessages = append(toolMessages, toolMsg)
		}

		return map[string]interface{}{
			"messages":  toolMessages,
			"raw_notes": rawNotes,
		}, nil
	})

	// Compress research node - summarizes findings
	workflow.AddNode("compress_research", "compress_research", func(ctx context.Context, state interface{}) (interface{}, error) {
		mState := state.(map[string]interface{})

		researchTopic, _ := mState["research_topic"].(string)
		rawNotes, _ := mState["raw_notes"].([]string)

		if len(rawNotes) == 0 {
			return map[string]interface{}{
				"compressed_research": "没有研究结果可压缩。",
			}, nil
		}

		// Create compression prompt
		prompt := GetCompressionPrompt(researchTopic, JoinNotes(rawNotes))

		resp, err := langchain.DirectGenerate(ctx, &config.ResearchModel, []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, prompt),
		}, nil, func(chunk []byte, seq int) {
			//allChunks.Write(chunk)
		}, llms.WithMaxTokens(config.CompressionModelMaxTokens))
		if err != nil {
			return nil, fmt.Errorf("report generation failed: %w", err)
		}

		compressed := resp.Choices[0].Content
		log.Trace("[Researcher] Compression result length:", len(compressed))

		return map[string]interface{}{
			"compressed_research": compressed,
			"raw_notes":           rawNotes,
		}, nil
	})

	// Define edges
	workflow.SetEntryPoint("researcher")

	// Conditional edge from researcher
	workflow.AddConditionalEdge("researcher", func(ctx context.Context, state interface{}) string {
		mState := state.(map[string]interface{})
		messages := mState["messages"].([]llms.MessageContent)
		lastMsg := messages[len(messages)-1]

		if HasToolCalls(lastMsg) {
			return "researcher_tools"
		}
		return "compress_research"
	})

	workflow.AddEdge("researcher_tools", "researcher")
	workflow.AddEdge("compress_research", graph.END)

	return workflow, nil
}
