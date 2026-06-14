package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/mark3labs/mcp-go/client"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/memory"
	langchaingoTools "github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/tools/duckduckgo"
	"github.com/tmc/langchaingo/tools/scraper"
	"github.com/tmc/langchaingo/tools/wikipedia"
	"infini.sh/coco/core"
	common2 "infini.sh/coco/modules/assistant/common"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

const conversationalToolPromptSuffix = `Begin!

Previous conversation history:
{{.history}}

New input: {{.input}}

Thought:{{.agent_scratchpad}}`

func CallLLMTools(ctx context.Context, reqMsg *core.ChatMessage, replyMsg *core.ChatMessage, params *common2.RAGContext, inputValues map[string]any, sender core.MessageSender) (string, error) {
	if params == nil || params.AssistantCfg == nil {
		//return nil
		panic("invalid assistant config, skip")
	}

	// Resolve the picking-tool model: MCPConfig.Model override -> settings
	// PickingToolModel -> settings LanguageModel; if still nothing, fall back
	// to the answering model (which is guaranteed resolved at this point).
	override := &core.ModelId{}
	if params.AssistantCfg.MCPConfig.Enabled && params.AssistantCfg.MCPConfig.Model != nil {
		override.ProviderID = params.AssistantCfg.MCPConfig.Model.ProviderID
		override.ID = params.AssistantCfg.MCPConfig.Model.Name
	}
	resolvedTool := llmmodule.ResolveAssistantModel(core.AssistantModelUsePickingTool, override)
	var providerID, modelName string
	if resolvedTool != nil {
		providerID = resolvedTool.ProviderID
		modelName = resolvedTool.ID
	} else {
		answering := params.MustGetAnsweringModel()
		providerID = answering.ProviderID
		modelName = answering.Name
	}

	llm, err := langchain.SimplyGetLLM(providerID, modelName, "")
	if err != nil {
		panic(err)
	}

	agentTools := []langchaingoTools.Tool{}

	if params.AssistantCfg.ToolsConfig.Enabled {
		webAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

		if params.AssistantCfg.ToolsConfig.BuiltinTools.Calculator {
			agentTools = append(agentTools, langchaingoTools.Calculator{})
		}

		if params.AssistantCfg.ToolsConfig.BuiltinTools.Wikipedia {
			wp := wikipedia.New(webAgent)
			agentTools = append(agentTools, wp)
		}

		if params.AssistantCfg.ToolsConfig.BuiltinTools.Duckduckgo {
			ddg, err := duckduckgo.New(50, webAgent)
			if err == nil && ddg != nil {
				agentTools = append(agentTools, ddg)
			}
		}

		if params.AssistantCfg.ToolsConfig.BuiltinTools.Scraper {
			scr, err := scraper.New()
			if err == nil && scr != nil {
				agentTools = append(agentTools, scr)
			}
		}
	}

	mcpClients := []*client.Client{}
	defer func() {
		for _, f := range mcpClients {
			_ = f.Close()
		}
	}()

	log.Debug("found total ", len(params.MCPServers), " mcp servers")

	for _, id := range params.MCPServers {
		v, err := common.GetMPCServer(id)
		if err != nil || v == nil {
			log.Errorf("Failed to get MPC Server [%s]: %v", id, err)
			continue
		}

		log.Tracef("start init mcp server: %v, %v", v.Name, v.Type)

		if !v.Enabled {
			continue
		}

		var mcpClient *client.Client
		switch v.Type {
		case common.StreamableHTTP:
			bytes := util.MustToJSONBytes(v.Config)
			cfg := core.StreamableHttpConfig{}
			err := util.FromJSONBytes(bytes, &cfg)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("convert from json fail: %v", err)
				}
				continue
			}

			if !util.IsValidURL(cfg.URL) {
				if global.Env().IsDebug {
					log.Errorf("invalid url: %v", cfg.URL)
				}
				continue
			}

			mcpClient, err = client.NewStreamableHttpClient(cfg.URL)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("NewStreamableHttpClient fail: %v", err)
				}
				continue
			}
		case common.SSE:
			bytes := util.MustToJSONBytes(v.Config)
			cfg := core.SSEConfig{}
			err := util.FromJSONBytes(bytes, &cfg)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("convert from json fail: %v", err)
				}
				continue
			}

			mcpClient, err = client.NewSSEMCPClient(cfg.URL)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("NewSSEMCPClient fail: %v", err)
				}
				continue
			}
			if err := mcpClient.Start(context.Background()); err != nil {
				if global.Env().IsDebug {
					log.Errorf("start client fail: %v", err)
				}
				continue
			}

		case common.Stdio:
			bytes := util.MustToJSONBytes(v.Config)

			cfg := core.StdioConfig{}
			err := util.FromJSONBytes(bytes, &cfg)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("convert from json fail: %v", err)
				}
				continue
			}
			envs := []string{}
			if len(cfg.Env) > 0 {
				for k, v := range cfg.Env {
					envs = append(envs, fmt.Sprintf("%v=%v", k, v))
				}
			}
			mcpClient, err = client.NewStdioMCPClient(cfg.Command, envs, cfg.Args...)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("NewStdioMCPClient fail: %v", err)
				}
				continue
			}
			if err := mcpClient.Start(ctx); err != nil {
				if global.Env().IsDebug {
					log.Errorf("start client fail: %v", err)
				}
				continue
			}
		default:
			if global.Env().IsDebug {
				log.Errorf("invalid type: %v", v.Type)
			}
			continue
		}

		if mcpClient != nil {
			mcpClients = append(mcpClients, mcpClient)
			mcpAdapter, err := langchain.New(mcpClient)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("error on new langchain client: %v", err)
				}
				continue
			}

			mcpTools, err := mcpAdapter.Tools()
			log.Tracef("get %v tools from mcp server: %v", v.Name)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("error get %v tools from mcp server: %v", v.Name, err)
				}
				continue
			}
			agentTools = append(agentTools, mcpTools...)
		}

		log.Tracef("end init mcp server: %v", v.Name)
	}

	if len(agentTools) <= 0 {
		log.Debug("total get ", len(agentTools), " tools")
		return "", nil
	}

	buffer := memory.NewConversationBuffer()
	if params.ChatHistory != nil {
		buffer.ChatHistory = params.ChatHistory
	}

	toolsOutputBuffer := strings.Builder{}
	// Store tool-call records in Description because they are display-oriented
	// Markdown text, while details.payload is mapped as an object in Elasticsearch.
	persistToolsOutput := func() {
		toolsOutput := toolsOutputBuffer.String()
		if toolsOutput != "" {
			replyMsg.Details = append(replyMsg.Details, core.ProcessingDetails{
				Order:       15,
				Type:        common.Tools,
				Description: toolsOutput,
			})
		}
	}
	callback := langchain.LogHandler{}
	toolsSeq := 0
	emitToolCall := func(toolName, arguments, result string) {
		chunk := formatToolCallChunk(toolName, arguments, result)
		// The frontend appends tools chunks verbatim, so preserve the same blank-line
		// separator that is later persisted in the accumulated Description.
		if toolsOutputBuffer.Len() > 0 {
			chunk = "\n\n" + chunk
		}
		toolsOutputBuffer.WriteString(chunk)

		sendErr := sender.SendChunkMessage(core.MessageTypeAssistant, common.Tools, chunk, toolsSeq)
		if sendErr != nil {
			panic(sendErr)
		}
		toolsSeq++
	}
	// Wrap tools instead of forwarding LLM streaming tokens. Agent callback text can
	// include Thought or final-answer content, and MCP tools do not consistently
	// emit tool-end callbacks; wrapping Tool.Call keeps the stream limited to the
	// actual tool name, arguments, and output.
	agentTools = wrapToolCallReporters(agentTools, emitToolCall)

	executor, err := agents.Initialize(
		llm,
		agentTools,
		agents.ConversationalReactDescription,
		//agents.WithReturnIntermediateSteps(),
		agents.WithMaxIterations(params.AssistantCfg.MCPConfig.MaxIterations),
		agents.WithPromptSuffix(conversationalToolPromptSuffixWithCurrentTime()),
		agents.WithCallbacksHandler(&callback),
		agents.WithMemory(buffer),
		agents.WithParserErrorHandler(agents.NewParserErrorHandler(func(err string) string {
			return "Your output format was incorrect. You must respond in one of the following two formats:\n" +
				"1. To use a tool:\n" +
				"Action: <tool_name>\nAction Input: <input>\n" +
				"2. To give a final answer:\n" +
				"AI: <your answer>\n" +
				"Do NOT include the tool result in your output. Wait for the Observation."
		})),
	)
	if err != nil {
		persistToolsOutput()
		return toolsOutputBuffer.String(), fmt.Errorf("error on executor: %w", err)
	}

	log.Debugf("start call LLM tools")
	answer, err := chains.Run(ctx, executor, reqMsg.Message)
	if err != nil {
		persistToolsOutput()
		return toolsOutputBuffer.String(), fmt.Errorf("error running chains: %w", err)
	}

	log.Debug("MCP call answer:", answer)

	persistToolsOutput()

	return answer, nil
}

// toolCallReportingTool decorates a langchaingo tool with a completion callback
// while preserving the original tool contract used by the agent executor.
type toolCallReportingTool struct {
	tool       langchaingoTools.Tool
	onComplete func(toolName, arguments, result string)
}

// Name returns the wrapped tool name so agent planning still sees the original
// tool identity.
func (reportingTool toolCallReportingTool) Name() string {
	return reportingTool.tool.Name()
}

// Description returns the wrapped tool description so prompt construction is not
// changed by the reporting layer.
func (reportingTool toolCallReportingTool) Description() string {
	return reportingTool.tool.Description()
}

// Call executes the wrapped tool and reports a display record containing the
// tool name, the raw input arguments, and either the output or the execution
// error.
// When the underlying tool returns an error, the error is converted into
// an observation string and a nil error is returned so the agent loop continues
// — the LLM sees the failure and can decide to retry, pick another tool, or
// produce a final answer without aborting the conversation.
func (reportingTool toolCallReportingTool) Call(ctx context.Context, input string) (string, error) {
	output, err := reportingTool.tool.Call(ctx, input)
	result := output
	if err != nil {
		// Context lifecycle errors must propagate so the agent loop stops
		// immediately when the caller cancels or the deadline expires.
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if result != "" {
			result += "\n"
		}
		result += fmt.Sprintf("Error: %v", err)
	}
	if reportingTool.onComplete != nil {
		reportingTool.onComplete(reportingTool.Name(), input, result)
	}
	// Return the error text as the observation instead of propagating it, so the
	// agent executor does not abort and the LLM can react to the failure.
	return result, nil
}

// wrapToolCallReporters applies toolCallReportingTool to every available tool so
// built-in tools and MCP tools produce the same streaming and persistence shape.
func wrapToolCallReporters(agentTools []langchaingoTools.Tool, onComplete func(toolName, arguments, result string)) []langchaingoTools.Tool {
	wrappedTools := make([]langchaingoTools.Tool, 0, len(agentTools))
	for _, agentTool := range agentTools {
		wrappedTools = append(wrappedTools, toolCallReportingTool{
			tool:       agentTool,
			onComplete: onComplete,
		})
	}
	return wrappedTools
}

// conversationalToolPromptSuffixWithCurrentTime mirrors langchaingo's default
// conversational-agent suffix and prepends the shared current-time context. The
// default suffix is unexported upstream, but overriding only the suffix keeps the
// default tool descriptions and tool-use instructions intact.
func conversationalToolPromptSuffixWithCurrentTime() string {
	return langchain.PromptWithCurrentTime("") + "\n\n" + conversationalToolPromptSuffix
}

// formatToolCallChunk renders one tool invocation as a single Markdown bullet
// section that the frontend can append directly and persist as Description text.
func formatToolCallChunk(toolName, arguments, result string) string {
	argumentsLanguage, argumentsText := formatToolArguments(arguments)
	resultText := strings.TrimSpace(result)
	if resultText == "" {
		resultText = "(empty)"
	}

	return fmt.Sprintf(
		"* %s\n\n  Arguments:\n\n%s\n\n  Output:\n\n%s",
		strings.TrimSpace(toolName),
		indentMarkdownBlock(markdownCodeBlock(argumentsLanguage, argumentsText)),
		indentMarkdownBlock(markdownCodeBlock("", resultText)),
	)
}

// formatToolArguments normalizes the agent-provided tool input for display. JSON
// arguments are pretty-printed and marked with the json language for Markdown;
// non-JSON inputs are preserved as plain text.
func formatToolArguments(arguments string) (string, string) {
	argumentsText := strings.TrimSpace(arguments)
	if argumentsText == "" {
		return "", "(empty)"
	}

	var argumentsValue interface{}
	if err := json.Unmarshal([]byte(argumentsText), &argumentsValue); err == nil {
		formattedArguments, err := json.MarshalIndent(argumentsValue, "", "  ")
		if err == nil {
			return "json", string(formattedArguments)
		}
	}

	return "", argumentsText
}

// markdownCodeBlock wraps arbitrary tool text in a fenced Markdown code block.
// The fence is expanded when needed so nested backticks in tool output do not
// prematurely close the block.
func markdownCodeBlock(language, content string) string {
	fence := "```"
	for strings.Contains(content, fence) {
		fence += "`"
	}
	if language == "" {
		return fmt.Sprintf("%s\n%s\n%s", fence, content, fence)
	}
	return fmt.Sprintf("%s%s\n%s\n%s", fence, language, content, fence)
}

// indentMarkdownBlock nests a multi-line Markdown block under the current tool
// call bullet, keeping arguments and output visually grouped as one invocation.
func indentMarkdownBlock(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}
