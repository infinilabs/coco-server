/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package service

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"infini.sh/coco/core"
	common2 "infini.sh/coco/modules/assistant/common"
	"infini.sh/coco/modules/assistant/deep_research"
	"infini.sh/coco/modules/assistant/deep_search"
	"infini.sh/coco/modules/assistant/tools"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

func CreateAssistantReplyMessage(sessionID, assistantID, requestMessageID string) *core.ChatMessage {
	msg := &core.ChatMessage{
		SessionID:      sessionID,
		MessageType:    core.MessageTypeAssistant,
		ReplyMessageID: requestMessageID,
		AssistantID:    assistantID,
	}
	now := time.Now()
	msg.Created = &now
	msg.ID = util.GetUUID()

	return msg
}

// save response and send END signal to receiver
func finalizeProcessing(ctx context.Context, sessionID string, msg *core.ChatMessage, sender core.MessageSender) {

	ctx1 := orm.NewContextWithParent(ctx)

	if err := orm.Save(ctx1, msg); err != nil {
		_ = log.Errorf("Failed to save assistant message: %v", err)
	}

	_ = sender.SendMessage(core.NewMessageChunk(
		sessionID, msg.ID, core.MessageTypeSystem, msg.ReplyMessageID,
		common.ReplyEnd, "Processing completed", 0,
	))
}

func ProcessMessageAsync(ctx context.Context, userID string, reqMsg *core.ChatMessage, params *common2.RAGContext, sender core.MessageSender) error {
	log.Debugf("Starting async processing for session: %v", params.SessionID)

	replyMsg := CreateAssistantReplyMessage(params.SessionID, reqMsg.AssistantID, reqMsg.ID)

	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}
				msg := fmt.Sprintf("⚠️ error in async processing message reply, %v", v)
				if replyMsg.Message == "" {
					replyMsg.Message = msg
					_ = sender.SendMessage(core.NewMessageChunk(
						params.SessionID, replyMsg.ID, core.MessageTypeSystem, reqMsg.ID,
						common.Response, msg, 0,
					))
				}
				_ = log.Error(msg)
			}
		}
		finalizeProcessing(ctx, params.SessionID, replyMsg, sender)
		// clear the inflight message task
		taskID := GetReplyMessageTaskID(params.SessionID, reqMsg.ID)
		InflightMessages.Delete(taskID)

		log.Info("finished async processing message")
	}()

	reqMsg.Details = make([]core.ProcessingDetails, 0)

	// Prepare input values
	if params.InputValues == nil {
		params.InputValues = map[string]any{}
	}

	params.InputValues["query"] = reqMsg.Message

	// Processing pipeline
	if params.AssistantCfg.ChatSettings.HistoryMessage.Number > 0 {
		params.ChatHistory, params.InputValues["history"], _ = FetchSessionHistory(ctx, reqMsg, params.AssistantCfg.ChatSettings.HistoryMessage.Number)
	} else {
		params.InputValues["history"] = "</empty>"
	}

	var err error
	switch params.AssistantCfg.Type {
	case core.AssistantTypeDeepThink:
		return deep_search.RunDeepSearchTask(
			ctx,
			userID,
			params,
			params.AssistantCfg,
			reqMsg,
			replyMsg,
			sender,
		)
	case core.AssistantTypeDeepResearch:
		log.Info("start running deep research")
		err = deep_research.RunDeepResearch(ctx, reqMsg.Message, params.AssistantCfg, reqMsg,
			replyMsg,
			sender)
		log.Info("end running deep research")
		break
	default:
		//simple mode
		var toolsMayHavePromisedResult = false
		if params.MCP && ((params.AssistantCfg.MCPConfig.Enabled && len(params.MCPServers) > 0) || params.AssistantCfg.ToolsConfig.Enabled) {
			//process LLM tools / functions
			answer, err := tools.CallLLMTools(ctx, reqMsg, replyMsg, params, params.InputValues, sender)
			if err != nil {
				log.Error(answer, err)
			}

			if answer != "" {
				if params.AssistantCfg.DeepThinkConfig != nil && params.AssistantCfg.DeepThinkConfig.ToolsPromisedResultSize > 0 && len(answer) > params.AssistantCfg.DeepThinkConfig.ToolsPromisedResultSize {
					toolsMayHavePromisedResult = true
				}
				params.InputValues["tools_output"] = answer
			}
		}

		if params.SearchDB && !toolsMayHavePromisedResult && params.AssistantCfg.Datasource.Enabled && len(params.AssistantCfg.Datasource.GetIDs()) > 0 {
			var fetchSize = 10
			docs, _ := tools.InitialDocumentBriefSearch(ctx, userID, reqMsg, replyMsg, params, fetchSize, sender)
			params.InputValues["references"] = docs
		}

		err = langchain.GenerateFinalResponse(ctx, reqMsg, replyMsg, params, params.InputValues, sender)
		log.Info("async reply task done for query:", reqMsg.Message)
		break
	}
	return err
}

func FetchSessionHistory(ctx context.Context, reqMsg *core.ChatMessage, size int) (*memory.ChatMessageHistory, string, error) {
	var historyStr = strings.Builder{}

	chatHistory := memory.NewChatMessageHistory(memory.WithPreviousMessages([]llms.ChatMessage{}))

	//get chat history
	history, err := GetChatHistoryBySessionInternal(reqMsg.SessionID, size)
	if err != nil {
		return nil, "", err
	}

	if len(history) <= 1 {
		return nil, "", nil
	}

	historyStr.WriteString("<conversation>\n")

	for i := len(history) - 1; i >= 0; i-- {
		v := history[i]
		msgText := util.SubStringWithSuffix(v.Message, 1000, "...")
		switch v.MessageType {
		case core.MessageTypeSystem:
			msg := llms.SystemChatMessage{Content: msgText}
			_ = chatHistory.AddMessage(ctx, msg)
			break
		case core.MessageTypeAssistant:
			msg := llms.AIChatMessage{Content: msgText}
			_ = chatHistory.AddMessage(ctx, msg)
			break
		case core.MessageTypeUser:
			msg := llms.HumanChatMessage{Content: msgText}
			_ = chatHistory.AddMessage(ctx, msg)
			break
		}

		historyStr.WriteString(v.MessageType + ": " + msgText)
		if v.DownVote > 0 {
			historyStr.WriteString(fmt.Sprintf("(%v people up voted this answer)", v.UpVote))
		}
		if v.DownVote > 0 {
			historyStr.WriteString(fmt.Sprintf("(%v people down voted this answer)", v.DownVote))
		}
		historyStr.WriteString("\n\n")
	}
	historyStr.WriteString("</conversation>")

	return chatHistory, historyStr.String(), nil
}
