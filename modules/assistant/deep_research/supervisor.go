package deep_research

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"golang.org/x/sync/errgroup"
	"infini.sh/framework/core/global"

	"infini.sh/coco/modules/assistant/langchain"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
)

// CreateSupervisorGraph creates the supervisor subgraph for managing research delegation
func CreateSupervisorGraph(ctx context.Context, config *core.DeepResearchConfig, researcherGraph *graph.MessageGraph) (*graph.MessageGraph, error) {
	workflow := graph.NewMessageGraph()

	// Set up schema with reducers
	schema := graph.NewMapSchema()
	schema.RegisterReducer("supervisor_messages", graph.AppendReducer)
	schema.RegisterReducer("notes", graph.AppendReducer)
	schema.RegisterReducer("raw_notes", graph.AppendReducer)
	workflow.SetSchema(schema)

	// Supervisor node - delegates research tasks
	workflow.AddNode("supervisor", "supervisor", func(ctx context.Context, state interface{}) (interface{}, error) {
		mState, ok := state.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid state type")
		}

		messages, ok := mState["supervisor_messages"].([]llms.MessageContent)
		if !ok {
			return nil, fmt.Errorf("supervisor_messages not found")
		}

		researchBrief, _ := mState["research_brief"].(string)
		researchIterations, _ := mState["research_iterations"].(int)

		log.Infof("[Supervisor] Starting iteration %d with %d existing messages", researchIterations+1, len(messages))

		// Check iteration limit
		if researchIterations >= config.MaxResearcherIterations {
			log.Infof("[Supervisor] Reached max research iterations (%d), completing research", config.MaxResearcherIterations)

			// Create completion message
			completeMsg := llms.MessageContent{
				Role: llms.ChatMessageTypeAI,
				Parts: []llms.ContentPart{
					llms.ToolCall{
						ID: "complete_1",
						FunctionCall: &llms.FunctionCall{
							Name:      "ResearchComplete",
							Arguments: `{"complete": true}`,
						},
					},
				},
			}

			return map[string]interface{}{
				"supervisor_messages": []llms.MessageContent{completeMsg},
			}, nil
		}

		// Prepare system prompt
		systemPrompt := GetSupervisorSystemPrompt(config.MaxResearcherIterations, config.MaxConcurrentResearchUnits)

		// Build message list properly
		var msgs []llms.MessageContent

		// Always start with system message
		msgs = append(msgs, llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt))

		// Add research brief as initial human message only if no conversation history
		if len(messages) == 0 {
			msgs = append(msgs, llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("研究简报：%s\n\n请分析这份研究简报并决定如何委派研究。首先使用 think_tool 规划你的方法，然后调用 ConductResearch 委派任务。", researchBrief)))
		} else {
			// For subsequent iterations, append the conversation history
			msgs = append(msgs, messages...)
		}

		// Define tools for supervisor
		toolDefs := []llms.Tool{
			{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        "ConductResearch",
					Description: "将研究任务委派给专门的子代理。提供详细的研究主题。",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"research_topic": map[string]interface{}{
								"type":        "string",
								"description": "要研究的主题。应该是一个详细描述的单一主题（至少一段）。",
							},
						},
						"required": []string{"research_topic"},
					},
				},
			},
			{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        "ResearchComplete",
					Description: "当研究完成且你有足够信息时调用此工具。",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"complete": map[string]interface{}{
								"type":        "boolean",
								"description": "当研究完成时设置为 true",
							},
						},
						"required": []string{"complete"},
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

		var resp *llms.ContentResponse
		var allChunks strings.Builder

		// Streaming approach with proper tool call reassembly
		resp, err := langchain.DirectGenerate(ctx, &config.ResearchModel, msgs, nil, func(chunk []byte, seq int) {
			allChunks.Write(chunk)
		}, llms.WithTools(toolDefs))
		if err != nil {
			return nil, fmt.Errorf("report generation failed: %w", err)
		}

		//finalContent := allChunks.String()
		//log.Info("[Supervisor] Streaming content fragments:", finalContent)

		// CRITICAL FIX: The response contains broken tool call fragments from streaming
		// We need to find the actual tool calls vs fragments
		choice := resp.Choices[0]

		aiMsg := llms.MessageContent{Role: llms.ChatMessageTypeAI}

		if choice.Content != "" {
			aiMsg.Parts = append(aiMsg.Parts, llms.TextPart(choice.Content))
		}

		for i, tc := range choice.ToolCalls {
			log.Tracef("[Supervisor] Adding tool[%d]: ID=%s, Name=%s, Args=%s",
				i, tc.ID, tc.FunctionCall.Name, tc.FunctionCall.Arguments)
			aiMsg.Parts = append(aiMsg.Parts, tc)
		}

		return map[string]interface{}{
			"supervisor_messages": []llms.MessageContent{aiMsg},
			"research_iterations": researchIterations + 1,
		}, nil
	})

	// Supervisor tools node - executes research delegation
	workflow.AddNode("supervisor_tools", "supervisor_tools", func(ctx context.Context, state interface{}) (interface{}, error) {
		mState := state.(map[string]interface{})
		messages := mState["supervisor_messages"].([]llms.MessageContent)
		lastMsg := messages[len(messages)-1]

		var toolMessages []llms.MessageContent
		var allRawNotes []string
		var allNotes []string

		toolCalls := GetToolCalls(lastMsg)

		//log.Info("[Supervisor] Total tool calls found:", len(toolCalls))
		//for i, tc := range toolCalls {
		//	log.Infof("[Supervisor] Tool[%d]: ID=%s, Name=%s, Args=%s", i, tc.ID, tc.FunctionCall.Name, tc.FunctionCall.Arguments)
		//}

		// Separate tool calls by type
		conductResearchCalls := ExtractToolCallsByName(toolCalls, "ConductResearch")
		thinkCalls := ExtractToolCallsByName(toolCalls, "think_tool")

		// Handle think tool calls
		thinkTool := &ThinkToolImpl{}
		for _, tc := range thinkCalls {
			args, _ := ParseToolArguments(tc)
			reflection, _ := args["reflection"].(string)
			result, _ := thinkTool.Call(ctx, reflection)
			toolMsg := CreateToolMessage(tc.ID, tc.FunctionCall.Name, result)
			toolMessages = append(toolMessages, toolMsg)
		}

		// CRITICAL FIX: Reconstruct fragmented tool calls from streaming
		// The streaming response breaks tool calls into fragments - we need to collect them properly
		//log.Info("[Supervisor] Reconstructing tool calls from streaming fragments...")

		// Find all the fragments and reconstruct actual tool calls
		//log.Info("[Supervisor] Tool calls before reconstruction:", len(conductResearchCalls)+len(thinkCalls))

		actualToolCalls := []llms.ToolCall{}
		reconstructedArgs := ""
		currentCallName := ""
		currentCallID := ""

		// Filter fragments to get actual tool calls with proper function names
		for i, tc := range toolCalls {
			log.Tracef("[Supervisor] Processing fragment[%d]: Name=%s, Args=%s", i, tc.FunctionCall.Name, tc.FunctionCall.Arguments)

			if tc.FunctionCall.Name != "" {
				// This is the start of a real tool call
				if reconstructedArgs != "" {
					// Add the previous reconstructed call
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

		//log.Info("[Supervisor] Tool calls after reconstruction: %d", len(actualToolCalls))
		//for i, tc := range actualToolCalls {
		//	log.Infof("[Supervisor] Reconstructed tool[%d]: Name=%s, Args=%q", i, tc.FunctionCall.Name, tc.FunctionCall.Arguments)
		//}

		// Use the reconstructed calls instead of the fragmented ones if we have valid ones
		if len(actualToolCalls) > 0 {
			toolCalls = actualToolCalls
			conductResearchCalls = ExtractToolCallsByName(toolCalls, "ConductResearch")
			thinkCalls = ExtractToolCallsByName(toolCalls, "think_tool")
		}

		log.Info("[Supervisor] Final tool counts:", len(conductResearchCalls), " research calls,", len(thinkCalls), " think calls")

		// Continue with normal tool execution but fix the fragmented tool calls
		// Handle ConductResearch calls in parallel (up to max concurrent)
		maxConcurrent := config.MaxConcurrentResearchUnits
		if len(conductResearchCalls) > maxConcurrent {
			log.Debugf("[Supervisor] Limiting research tasks from %d to %d", len(conductResearchCalls), maxConcurrent)

			// Add error messages for overflow
			for _, tc := range conductResearchCalls[maxConcurrent:] {
				errMsg := CreateToolMessage(
					tc.ID,
					tc.FunctionCall.Name,
					fmt.Sprintf("错误：超过了最大并发研究单元数 (%d)。请尝试减少任务数量。", maxConcurrent),
				)
				toolMessages = append(toolMessages, errMsg)
			}

			conductResearchCalls = conductResearchCalls[:maxConcurrent]
		}

		resultsChan := make(chan researchResult, len(conductResearchCalls))
		g, ctx := errgroup.WithContext(ctx)

		for _, tc := range conductResearchCalls {
			toolCall := tc
			g.Go(func() error {
				result, err := runResearch(ctx, researcherGraph, toolCall)
				if err != nil {
					return err
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case resultsChan <- result:
					return nil
				}
			})
		}

		// Wait for all research tasks to complete
		go func() {
			g.Wait()
			close(resultsChan)
		}()

		// Collect results
		for res := range resultsChan {
			var content string
			if res.err != nil {
				content = fmt.Sprintf("Research error: %v", res.err)
			} else {
				content = res.result
				allNotes = append(allNotes, res.result)
				allRawNotes = append(allRawNotes, res.rawNotes...)
			}

			toolMsg := CreateToolMessage(res.toolCallID, "ConductResearch", content)
			toolMessages = append(toolMessages, toolMsg)
		}

		return map[string]interface{}{
			"supervisor_messages": toolMessages,
			"notes":               allNotes,
			"raw_notes":           allRawNotes,
		}, nil
	})

	// Define edges
	workflow.SetEntryPoint("supervisor")

	workflow.AddConditionalEdge("supervisor", func(ctx context.Context, state interface{}) string {
		mState := state.(map[string]interface{})
		messages := mState["supervisor_messages"].([]llms.MessageContent)
		lastMsg := messages[len(messages)-1]

		toolCalls := GetToolCalls(lastMsg)

		// Check if ResearchComplete was called
		for _, tc := range toolCalls {
			if tc.FunctionCall.Name == "ResearchComplete" {
				return graph.END
			}
		}

		// If there are other tool calls, go to tools node
		if len(toolCalls) > 0 {
			return "supervisor_tools"
		}

		return graph.END
	})

	workflow.AddEdge("supervisor_tools", "supervisor")

	return workflow, nil
}

// Execute research tasks in parallel
type researchResult struct {
	toolCallID string
	result     string
	rawNotes   []string
	err        error
}

func runResearch(ctx context.Context, researcherGraph *graph.MessageGraph, toolCall llms.ToolCall) (researchResult, error) {

	var rr researchResult
	rr.toolCallID = toolCall.ID

	// Unified cancel check
	if isCanceled(ctx) {
		return rr, ctx.Err()
	}

	// Parse tool arguments
	args, err := ParseToolArguments(toolCall)
	if err != nil {
		rr.err = err
		return rr, err
	}

	researchTopic, _ := args["research_topic"].(string)

	// Compile researcher graph
	researcherRunnable, err := researcherGraph.Compile()
	if err != nil {
		rr.err = fmt.Errorf("failed to compile researcher: %w", err)
		return rr, rr.err
	}

	// Prepare state
	researcherState := map[string]interface{}{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, researchTopic),
		},
		"research_topic":       researchTopic,
		"tool_call_iterations": 0,
		"raw_notes":            []string{},
	}

	// Invoke subgraph
	result, err := researcherRunnable.Invoke(ctx, researcherState)
	if err != nil {
		rr.err = fmt.Errorf("researcher execution failed: %w", err)
		return rr, rr.err
	}

	// Check again before reading state
	if isCanceled(ctx) {
		return rr, ctx.Err()
	}

	resultState := result.(map[string]interface{})
	rr.result, _ = resultState["compressed_research"].(string)
	rr.rawNotes, _ = resultState["raw_notes"].([]string)

	return rr, nil
}

func isCanceled(ctx context.Context) bool {
	if global.ShuttingDown() {
		return true
	}
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

//func runResearch(ctx context.Context, call llms.ToolCall) (researchResult, error) {
//	defer wg.Done()
//
//	args, err := ParseToolArguments(toolCall)
//	if err != nil {
//		resultsChan <- researchResult{
//			toolCallID: toolCall.ID,
//			err:        err,
//		}
//		return
//	}
//
//	researchTopic, _ := args["research_topic"].(string)
//
//	// Compile and invoke researcher subgraph
//	researcherRunnable, err := researcherGraph.Compile()
//	if err != nil {
//		resultsChan <- researchResult{
//			toolCallID: toolCall.ID,
//			err:        fmt.Errorf("failed to compile researcher: %w", err),
//		}
//		return
//	}
//
//	// Prepare researcher state
//	researcherState := map[string]interface{}{
//		"messages": []llms.MessageContent{
//			llms.TextParts(llms.ChatMessageTypeHuman, researchTopic),
//		},
//		"research_topic":       researchTopic,
//		"tool_call_iterations": 0,
//		"raw_notes":            []string{},
//	}
//
//	result, err := researcherRunnable.Invoke(ctx, researcherState)
//	if err != nil {
//		resultsChan <- researchResult{
//			toolCallID: toolCall.ID,
//			err:        fmt.Errorf("researcher execution failed: %w", err),
//		}
//		return
//	}
//
//	resultState := result.(map[string]interface{})
//	compressed, _ := resultState["compressed_research"].(string)
//	rawNotes, _ := resultState["raw_notes"].([]string)
//
//	log.Info("[Supervisor] Research result:", compressed)
//	log.Info("[Supervisor] Raw notes count:", len(rawNotes))
//
//	resultsChan <- researchResult{
//		toolCallID: toolCall.ID,
//		result:     compressed,
//		rawNotes:   rawNotes,
//	}
//}
