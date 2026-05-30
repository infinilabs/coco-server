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
	deep_research2 "infini.sh/coco/modules/assistant/deep_research_v2"
	"infini.sh/coco/modules/assistant/deep_search"
	"infini.sh/coco/modules/assistant/tools"
	attachmentmod "infini.sh/coco/modules/attachment"

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

	_ = sender.SendChunkMessage(core.MessageTypeSystem,
		common.ReplyEnd, "Processing completed", 0,
	)
}

func ProcessMessageAsync(ctx context.Context, userID string, reqMsg, replyMsg *core.ChatMessage, params *common2.RAGContext, sender core.MessageSender) error {
	log.Debugf("Starting async processing for session: %v", params.SessionID)

	var err error
	//messageBuffer := strings.Builder{}
	_ = sender.SendChunkMessage(core.MessageTypeSystem,
		common.ReplyStart, "", 0,
	)

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
				if replyMsg.Message != "" {
					replyMsg.Message += "\n\n"
				}
				replyMsg.Message += msg
				_ = sender.SendChunkMessage(core.MessageTypeSystem,
					common.Response, msg, 0,
				)
				_ = log.Error(msg)
			}
		}

		if err != nil {
			log.Errorf("Failed to process message reply: %v", err)
			replyMsg.Message += err.Error()
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

	// Wait for any attachments referenced by this request to finish their
	// initial parsing before invoking the LLM. The stream stays open during
	// the wait; the client remains in "thinking" state until either the
	// attachments are ready or the wait fails/times out.
	if err = waitAndAttachAttachments(ctx, reqMsg, params, sender); err != nil {
		return err
	}

	params.InputValues["query"] = reqMsg.Message

	// Processing pipeline
	if params.AssistantCfg.ChatSettings.HistoryMessage.Number > 0 {
		params.ChatHistory, params.InputValues["history"], _ = FetchSessionHistory(ctx, reqMsg, params.AssistantCfg.ChatSettings.HistoryMessage.Number)
	} else {
		params.InputValues["history"] = "</empty>"
	}

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
		err = deep_research2.RunDeepResearchV2(ctx, reqMsg.Message, params.AssistantCfg.DeepResearchConfig, reqMsg,
			replyMsg,
			sender)
		//err = deep_research.RunDeepResearch(ctx, reqMsg.Message, params.AssistantCfg.DeepResearchConfig, reqMsg,
		//	replyMsg,
		//	sender)
		log.Info("end running deep research")
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
			docs, _ := tools.InitialDocumentBriefSearch(ctx, userID, reqMsg, replyMsg, params, 0, fetchSize, sender)
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

// attachmentWaitTimeout bounds how long ProcessMessageAsync will block waiting
// for attachment initial-parsing to complete before failing the chat. The
// outer HTTP context (e.g. sendChatMessageV2) currently uses a 5 minute total
// budget; keeping the wait well below that leaves room for the actual LLM call.
const attachmentWaitTimeout = 90 * time.Second

// attachmentWaitHeartbeat controls how frequently a keepalive chunk is emitted
// while waiting. This is the cap of the underlying polling backoff so the UI
// receives a steady-but-not-noisy progress signal.
const attachmentWaitHeartbeat = 5 * time.Second

// waitAndAttachAttachments blocks until every attachment referenced by reqMsg
// reaches a terminal initial_parsing state. On success it stores the loaded
// attachments into params.InputValues["attachments"] so downstream prompt
// assembly can inject their text.
//
// Behavior:
//   - empty / missing attachment list: no-op.
//   - all referenced attachments missing-or-deleted: skip wait, no-op.
//   - any attachment in failed/canceled state: surface a system error chunk
//     and abort the chat without calling the LLM.
//   - context canceled (e.g. client disconnect): propagate the cancellation;
//     finalize will not emit additional chunks on a broken stream.
//   - wait timeout: surface a timeout system chunk and abort the chat.
func waitAndAttachAttachments(ctx context.Context, reqMsg *core.ChatMessage, params *common2.RAGContext, sender core.MessageSender) error {
	if reqMsg == nil || len(reqMsg.Attachments) == 0 {
		return nil
	}

	// Filter out deleted / missing attachments up front so we don't block on
	// IDs that will never have stats written for them.
	live := attachmentmod.LoadAttachmentsForChat(reqMsg.Attachments)
	if len(live) == 0 {
		log.Debugf("attachment wait: no live attachments to wait for in session [%s]", params.SessionID)
		return nil
	}
	liveIDs := make([]string, 0, len(live))
	for _, a := range live {
		if a != nil {
			liveIDs = append(liveIDs, a.ID)
		}
	}

	heartbeat := func(pending []string) {
		// The chunk content is informational; the main signal is its arrival,
		// which keeps reverse proxies from idling out the chunked response.
		_ = sender.SendChunkMessage(core.MessageTypeSystem,
			common.AttachmentWaiting,
			fmt.Sprintf("waiting for %d attachment(s) to finish processing", len(pending)),
			0,
		)
	}

	failedIDs, waitErr := attachmentmod.WaitForAttachmentsCompletion(ctx, liveIDs, attachmentWaitTimeout, heartbeat)
	if waitErr != nil {
		if ctx.Err() != nil {
			// Outer context canceled (client disconnect / explicit stop): just
			// propagate so the caller's finalize flow can run.
			return ctx.Err()
		}
		// Wait-scoped timeout.
		msg := fmt.Sprintf("attachment processing timed out after %s", attachmentWaitTimeout)
		_ = sender.SendChunkMessage(core.MessageTypeSystem, common.Response, msg, 0)
		return fmt.Errorf("%s", msg)
	}
	if len(failedIDs) > 0 {
		msg := fmt.Sprintf("attachment processing failed for: %s", strings.Join(failedIDs, ", "))
		_ = sender.SendChunkMessage(core.MessageTypeSystem, common.Response, msg, 0)
		return fmt.Errorf("%s", msg)
	}

	// Re-read metadata so that Attachment.Text reflects whatever the processor
	// wrote during the wait. The underlying store is eventually consistent;
	// a missing Text at this point is not an error, the prompt formatter will
	// substitute a placeholder.
	refreshed := attachmentmod.LoadAttachmentsForChat(liveIDs)
	if len(refreshed) == 0 {
		// Attachments disappeared (deleted between wait start and now): fall
		// back to whatever we loaded up front rather than failing the chat.
		refreshed = live
	}
	params.InputValues["attachments"] = refreshed
	return nil
}
