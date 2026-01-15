package tools

import (
	"context"
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
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

func CallLLMTools(ctx context.Context, reqMsg *core.ChatMessage, replyMsg *core.ChatMessage, params *common2.RAGContext, inputValues map[string]any, sender core.MessageSender) (string, error) {
	if params == nil || params.AssistantCfg == nil {
		//return nil
		panic("invalid assistant config, skip")
	}

	//get llm for mcp, use answering model if not mcp specified model
	providerID := params.MustGetAnsweringModel().ProviderID
	modelName := params.MustGetAnsweringModel().Name
	if params.AssistantCfg.MCPConfig.Enabled {
		if params.AssistantCfg.MCPConfig.Model != nil {
			if params.AssistantCfg.MCPConfig.Model.Name != "" {
				modelName = params.AssistantCfg.MCPConfig.Model.Name
				providerID = params.AssistantCfg.MCPConfig.Model.ProviderID
			}
		}
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
			break
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

			break
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
			break
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

	answerBuffer := strings.Builder{}
	callback := langchain.LogHandler{}
	toolsSeq := 0
	callback.CustomWriteFunc = func(chunk string) {
		if chunk != "" {
			answerBuffer.WriteString(chunk)
			echoMsg := core.NewMessageChunk(params.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.Tools, chunk, toolsSeq)
			_ = sender.SendMessage(echoMsg)
		}
		toolsSeq++
	}

	executor, err := agents.Initialize(
		llm,
		agentTools,
		agents.ConversationalReactDescription,
		//agents.WithReturnIntermediateSteps(),
		agents.WithMaxIterations(params.AssistantCfg.MCPConfig.MaxIterations),
		agents.WithCallbacksHandler(&callback),
		agents.WithMemory(buffer),
	)
	if err != nil {
		return answerBuffer.String(), fmt.Errorf("error on executor: %w", err)
	}

	log.Debugf("start call LLM tools")
	answer, err := chains.Run(ctx, executor, reqMsg.Message)
	if err != nil {
		return answerBuffer.String(), fmt.Errorf("error running chains: %w", err)
	}

	log.Debug("MCP call answer:", answer)

	return answer, nil
}
