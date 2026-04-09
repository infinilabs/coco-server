package deep_search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"infini.sh/coco/core"
	common2 "infini.sh/coco/modules/assistant/common"
	"infini.sh/coco/modules/assistant/tools"
	"infini.sh/coco/modules/datasource"
	"infini.sh/coco/modules/llm"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/framework/core/util"
)

// RunSearchPipeline executes the full search pipeline: intent analysis, optional
// tool calling, initial document search, and re-pick with in-depth fetch.
func RunSearchPipeline(ctx context.Context, userID string, params *common2.RAGContext, cfg *core.Assistant,
	reqMsg, replyMsg *core.ChatMessage, sender core.MessageSender) ([]core.Document, error) {

	if cfg.DeepThinkConfig == nil {
		return nil, fmt.Errorf("invalid deep think config")
	}

	if cfg.DeepThinkConfig.PickDatasource {
		var datasourceStr = strings.Builder{}
		if len(params.AssistantCfg.Datasource.GetIDs()) > 0 {
			ds, err := datasource.GetDatasourceByID(params.AssistantCfg.Datasource.GetIDs())
			if err == nil && ds != nil {
				for _, v := range ds {
					datasourceStr.WriteString(fmt.Sprintf("ID: %v, Name: %v, Description: %v \n", v.ID, v.Name, v.Description))
				}
			}
		}
		params.InputValues["network_sources"] = datasourceStr.String()
	}

	if cfg.DeepThinkConfig.PickTools {
		var mcpServers = strings.Builder{}
		if len(params.AssistantCfg.MCPConfig.GetIDs()) > 0 {
			ds, err := llm.GetMCPServersByID(params.AssistantCfg.MCPConfig.GetIDs())
			if err == nil && ds != nil {
				for _, v := range ds {
					mcpServers.WriteString(fmt.Sprintf("Name: %v, Desc: %v \n", v.Name, v.Description))
				}
			}
		}

		params.InputValues["tool_list"] = mcpServers.String()
	}

	intentStart := time.Now()
	queryIntent, err := langchain.ProcessQueryIntent(ctx, params.SessionID, &cfg.DeepThinkConfig.IntentAnalysisModel, reqMsg, replyMsg, params.AssistantCfg, params.InputValues, sender)
	if err != nil {
		log.Error("error on processing query intent analysis: ", err)
		return nil, err
	}
	fmt.Printf("[SearchPipeline] ProcessQueryIntent took %v\n", time.Since(intentStart))

	params.InputValues["intent"] = util.MustToJSON(params.QueryIntent)

	var toolsMayHavePromisedResult = false
	if params.MCP && ((params.AssistantCfg.MCPConfig.Enabled && len(params.MCPServers) > 0) || params.AssistantCfg.ToolsConfig.Enabled) {
		if !(cfg.DeepThinkConfig.PickTools && !queryIntent.NeedCallTools) {
			//call tools
			//process LLM tools / functions
			answer, err := tools.CallLLMTools(ctx, reqMsg, replyMsg, params, params.InputValues, sender)
			if err != nil {
				log.Error(answer, err)
				return nil, err
			}

			if answer != "" {
				if params.AssistantCfg.DeepThinkConfig != nil && params.AssistantCfg.DeepThinkConfig.ToolsPromisedResultSize > 0 && len(answer) > params.AssistantCfg.DeepThinkConfig.ToolsPromisedResultSize {
					toolsMayHavePromisedResult = true
				}
				params.InputValues["tools_output"] = answer
			}
		} else {
			log.Info("intent analyzer decided to skip call LLM tools")
		}
	} else {
		log.Info("LLM tools not enabled, skip call LLM tools")
	}

	var docs []core.Document
	if params.SearchDB && !toolsMayHavePromisedResult && params.AssistantCfg.Datasource.Enabled && len(params.AssistantCfg.Datasource.GetIDs()) > 0 {
		if !(cfg.DeepThinkConfig.PickDatasource && !queryIntent.NeedNetworkSearch) {
			var fetchSize = 50
			searchStart := time.Now()
			docs, _ = tools.InitialDocumentBriefSearch(ctx, userID, reqMsg, replyMsg, params, 0, fetchSize, sender)
			fmt.Printf("[SearchPipeline] InitialDocumentBriefSearch took %v, returned %d docs\n", time.Since(searchStart), len(docs))
			params.InputValues["references"] = util.MustToJSON(docs)

			if len(docs) > 10 {
				//re-pick top docs
				pickStart := time.Now()
				docs, _ = tools.PickingDocuments(ctx, reqMsg, replyMsg, params, docs, sender)
				fmt.Printf("[SearchPipeline] PickingDocuments took %v, returned %d docs\n", time.Since(pickStart), len(docs))
				fetchStart := time.Now()
				_ = tools.FetchDocumentInDepth(ctx, reqMsg, replyMsg, params, docs, params.InputValues, sender)
				fmt.Printf("[SearchPipeline] FetchDocumentInDepth took %v\n", time.Since(fetchStart))
			}
		}
	}

	return docs, nil
}

func RunDeepSearchTask(ctx context.Context, userID string, params *common2.RAGContext, cfg *core.Assistant,
	reqMsg, replyMsg *core.ChatMessage, sender core.MessageSender) error {

	_, err := RunSearchPipeline(ctx, userID, params, cfg, reqMsg, replyMsg, sender)
	if err != nil {
		return err
	}

	err = langchain.GenerateFinalResponse(ctx, reqMsg, replyMsg, params, params.InputValues, sender)
	log.Info("async reply task done for query:", reqMsg.Message)
	return err
}
