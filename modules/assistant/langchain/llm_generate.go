/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package langchain

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
	common2 "infini.sh/coco/modules/assistant/common"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/util"
)

func GenerateResponse(taskCtx context.Context, provider *core.ModelProvider, modelConfig *core.ModelConfig,
	reqMsg, replyMsg *core.ChatMessage, sessionID string, rolePrompt string,
	inputValues map[string]any, sender core.MessageSender) error {

	err := sender.SendChunkMessage(core.MessageTypeAssistant, common.Response, string(""), 0)
	if err != nil {
		panic(err)
	}

	// Prepare the system message
	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, rolePrompt),
	}

	//response
	reasoningBuffer := strings.Builder{}
	messageBuffer := strings.Builder{}
	// note: we use defer to ensure that the response message is saved after processing
	// even if user cancels the task or if an error occurs
	defer func() {
		//save response message to system
		if messageBuffer.Len() > 0 {
			replyMsg.Message = messageBuffer.String()
		} else {
			log.Warnf("seems empty reply for query: %v", replyMsg)
		}
		if reasoningBuffer.Len() > 0 {
			detail := core.ProcessingDetails{Order: 50, Type: common.Think, Description: reasoningBuffer.String()}
			replyMsg.Details = append(replyMsg.Details, detail)
		}
	}()
	chunkSeq := 0

	llm, err := SimplyGetLLM(modelConfig.ProviderID, modelConfig.Name, modelConfig.Keepalive)
	if err != nil {
		panic(err)
	}

	options := GetLLOptions(modelConfig)

	if modelConfig.Settings.Reasoning {
		options = append(options, llms.WithStreamingReasoningFunc(func(ctx context.Context, reasoningChunk []byte, chunk []byte) error {
			log.Trace(string(reasoningChunk), ",", string(chunk))
			// Use taskCtx here to check for cancellation or other context-specific logic
			select {
			case <-ctx.Done(): // Check if the task has been canceled or has expired
				log.Warnf("Task for message %v canceled", reqMsg.ID)
				return taskCtx.Err() // Return the context error (canceled or deadline exceeded)
			case <-taskCtx.Done(): // Check if the task has been canceled or has expired
				log.Warnf("Task for message %v canceled", reqMsg.ID)
				return taskCtx.Err() // Return the context error (canceled or deadline exceeded)
			default:

				//Handle the <Think> part
				if len(reasoningChunk) > 0 {
					chunkSeq += 1
					reasoningBuffer.Write(reasoningChunk)
					err = sender.SendChunkMessage(core.MessageTypeAssistant, common.Think, string(reasoningChunk), chunkSeq)
					if err != nil {
						panic(err)
					}
					return nil
				}

				//Handle response
				if len(chunk) > 0 {
					chunkSeq += 1

					err = sender.SendChunkMessage(core.MessageTypeAssistant, common.Response, string(chunk), chunkSeq)
					if err != nil {
						panic(err)
					}

					//log.Debug(msg)
					messageBuffer.Write(chunk)
				}

				return nil
			}

		}))
	} else {
		//this part works for ollama
		options = append(options, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			if len(chunk) > 0 {
				log.Trace(string(chunk))
				chunkSeq += 1
				err = sender.SendChunkMessage(core.MessageTypeAssistant, common.Response, string(chunk), chunkSeq)
				if err != nil {
					panic(err)
				}
				messageBuffer.Write(chunk)
			}
			return nil
		}))
	}

	contextPrompt := ``

	if v, ok := inputValues["history"]; ok {
		text, ok := v.(string)
		if ok {
			//if params.AssistantCfg.ChatSettings.HistoryMessage.CompressionThreshold > 0 && len(text) > params.AssistantCfg.ChatSettings.HistoryMessage.CompressionThreshold {
			//	//log.Error("history is too large: %v, compressing, target size: %v", len(text), params.AssistantCfg.ChatSettings.HistoryMessage.CompressionThreshold)
			//	//TODO compress history
			//}
			contextPrompt += fmt.Sprintf("\nConversation:\n%v\n", text)
		}
	}

	if v, ok := inputValues["references"]; ok {
		contextPrompt += util.SubString(fmt.Sprintf("\nReferences:\n%v\n", v), 0, 4096*2) //TODO
	}

	if v, ok := inputValues["tools_output"]; ok {
		contextPrompt += fmt.Sprintf("\nTools Output:\n%v\n", v)
	}

	inputValues["context"] = contextPrompt

	template := common.GenerateAnswerPromptTemplate
	if modelConfig.PromptConfig != nil && modelConfig.PromptConfig.PromptTemplate != "" {
		template = modelConfig.PromptConfig.PromptTemplate
	}

	// Create the prompt template
	finalPrompt, err := GetPromptStringByTemplateArgs(modelConfig, template, []string{"query", "context"}, inputValues)
	if err != nil {
		panic(err)
	}

	// Append the user's message
	content = append(content, llms.TextParts(llms.ChatMessageTypeHuman, finalPrompt))

	log.Info(content)

	completion, err := llm.GenerateContent(taskCtx, content, options...)
	if err != nil {
		log.Error(err)
		return err
	}
	_ = completion

	chunkSeq += 1

	return nil
}

type ChunkBufferCollector func(chunk []byte, seq int)

func GetPromptMessages(modelConfig *core.ModelConfig, systemPrompt, userPrompt string, requiredInput []string, inputValues map[string]interface{}) []llms.MessageContent {

	content := []llms.MessageContent{}

	if systemPrompt != "" {
		content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt))
	}

	// Create the prompt template
	finalPrompt, err := GetPromptStringByTemplateArgs(modelConfig, userPrompt, requiredInput, inputValues)
	if err != nil {
		panic(err)
	}

	// Append the user's message
	content = append(content, llms.TextParts(llms.ChatMessageTypeHuman, finalPrompt))
	return content
}

func DirectGenerate(taskCtx context.Context, modelConfig *core.ModelConfig, msgs []llms.MessageContent, reasoningBuffer, messageBuffer ChunkBufferCollector, extraOptions ...llms.CallOption) (*llms.ContentResponse, error) {
	llm, err := SimplyGetLLM(modelConfig.ProviderID, modelConfig.Name, modelConfig.Keepalive)
	if err != nil {
		panic(err)
	}

	options := GetLLOptions(modelConfig)
	chunkSeq := 0
	if modelConfig.Settings.Reasoning {
		options = append(options, llms.WithStreamingReasoningFunc(func(ctx context.Context, reasoningChunk []byte, chunk []byte) error {
			log.Trace(string(reasoningChunk), ",", string(chunk))
			// Use taskCtx here to check for cancellation or other context-specific logic
			select {
			case <-ctx.Done(): // Check if the task has been canceled or has expired
				//log.Warnf("Task for message %v canceled", reqMsg.ID)
				return taskCtx.Err() // Return the context error (canceled or deadline exceeded)
			case <-taskCtx.Done(): // Check if the task has been canceled or has expired
				//log.Warnf("Task for message %v canceled", reqMsg.ID)
				return taskCtx.Err() // Return the context error (canceled or deadline exceeded)
			default:

				//Handle the <Think> part
				if len(reasoningChunk) > 0 {
					chunkSeq += 1
					if reasoningBuffer != nil {
						reasoningBuffer(reasoningChunk, chunkSeq)
					}
					//fmt.Print(string(chunk))
					//reasoningBuffer.Write(reasoningChunk)
					//msg := core.NewMessageChunk(sessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.Think, string(reasoningChunk), chunkSeq)
					//log.Info(util.MustToJSON(msg))
					//err = sender.SendMessage(msg)
					//if err != nil {
					//	panic(err)
					//}
					return nil
				}

				//Handle response
				if len(chunk) > 0 {
					chunkSeq += 1

					//msg := core.NewMessageChunk(sessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.Response, string(chunk), chunkSeq)
					//err = sender.SendMessage(msg)
					//if err != nil {
					//	panic(err)
					//}

					//log.Debug(msg)
					//messageBuffer.Write(chunk)

					if messageBuffer != nil {
						messageBuffer(chunk, chunkSeq)
					}
					//fmt.Print(string(chunk))
				}

				return nil
			}

		}))
	} else {
		//this part works for ollama
		options = append(options, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			if len(chunk) > 0 {
				log.Trace(string(chunk))
				chunkSeq += 1
				//msg := core.NewMessageChunk(sessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.Response, string(chunk), chunkSeq)
				//err = sender.SendMessage(msg)
				//messageBuffer.Write(chunk)

				if messageBuffer != nil {
					messageBuffer(chunk, chunkSeq)
				}
				//fmt.Print(string(chunk))
			}
			return nil
		}))
	}

	options = append(options, extraOptions...)

	completion, err := llm.GenerateContent(taskCtx, msgs, options...)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	_ = completion

	chunkSeq += 1
	return completion, err
}

func GenerateFinalResponse(taskCtx context.Context, reqMsg, replyMsg *core.ChatMessage, params *common2.RAGContext,
	inputValues map[string]any, sender core.MessageSender) error {
	_ = sender.SendChunkMessage(core.MessageTypeAssistant, common.Response, string(""), 0)

	// Prepare the system message
	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, params.AssistantCfg.RolePrompt),
	}

	//response
	reasoningBuffer := strings.Builder{}
	messageBuffer := strings.Builder{}
	// note: we use defer to ensure that the response message is saved after processing
	// even if user cancels the task or if an error occurs
	defer func() {
		//save response message to system
		if messageBuffer.Len() > 0 {
			log.Error("set message buffer:", messageBuffer.String())
			replyMsg.Message = messageBuffer.String()
		} else {
			log.Warnf("seems empty reply for query: %v", replyMsg)
		}
		if reasoningBuffer.Len() > 0 {
			detail := core.ProcessingDetails{Order: 50, Type: common.Think, Description: reasoningBuffer.String()}
			replyMsg.Details = append(replyMsg.Details, detail)
		}
	}()
	chunkSeq := 0
	var err error

	llm, err := SimplyGetLLM(params.AssistantCfg.AnsweringModel.ProviderID, params.AssistantCfg.AnsweringModel.Name, "")
	if err != nil {
		panic(err)
	}

	options := []llms.CallOption{}
	maxTokens := GetMaxTokens(params.MustGetAnsweringModel(), 1024)
	temperature := GetTemperature(params.MustGetAnsweringModel(), 0.8)
	maxLength := GetMaxLength(params.MustGetAnsweringModel(), 0)
	options = append(options, llms.WithMaxTokens(maxTokens))
	options = append(options, llms.WithMaxLength(maxLength))
	options = append(options, llms.WithTemperature(temperature))

	if params.MustGetAnsweringModel().Settings.Reasoning {
		options = append(options, llms.WithStreamingReasoningFunc(func(ctx context.Context, reasoningChunk []byte, chunk []byte) error {
			log.Trace(string(reasoningChunk), ",", string(chunk))

			// Use taskCtx here to check for cancellation or other context-specific logic
			select {
			case <-ctx.Done(): // Check if the task has been canceled or has expired
				log.Warnf("Task for message %v canceled (ctx=%v)", reqMsg.ID, ctx.Err())
				return taskCtx.Err() // Return the context error (canceled or deadline exceeded)
			case <-taskCtx.Done(): // Check if the task has been canceled or has expired
				log.Warnf("Task for message %v canceled (taskCtx=%v)", reqMsg.ID, taskCtx.Err())
				return taskCtx.Err() // Return the context error (canceled or deadline exceeded)
			default:

				//Handle the <Think> part
				if len(reasoningChunk) > 0 {
					chunkSeq += 1
					reasoningBuffer.Write(reasoningChunk)
					err = sender.SendChunkMessage(core.MessageTypeAssistant, common.Think, string(reasoningChunk), chunkSeq)
					if err != nil {
						panic(err)
					}
					return nil
				}

				//Handle response
				if len(chunk) > 0 {
					chunkSeq += 1

					err = sender.SendChunkMessage(core.MessageTypeAssistant, common.Response, string(chunk), chunkSeq)
					if err != nil {
						panic(err)
					}

					messageBuffer.Write(chunk)
				}

				return nil
			}

		}))
	} else {
		//this part works for ollama
		options = append(options, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			if len(chunk) > 0 {
				log.Trace(string(chunk))
				chunkSeq += 1
				err = sender.SendChunkMessage(core.MessageTypeAssistant, common.Response, string(chunk), chunkSeq)
				messageBuffer.Write(chunk)
			}
			return nil
		}))
	}

	//for context is not set
	if _, ok := inputValues["context"]; !ok {
		contextPrompt := ``

		if v, ok := inputValues["history"]; ok {
			text, ok := v.(string)
			if ok {
				if params.AssistantCfg.ChatSettings.HistoryMessage.CompressionThreshold > 0 && len(text) > params.AssistantCfg.ChatSettings.HistoryMessage.CompressionThreshold {
					//log.Error("history is too large: %v, compressing, target size: %v", len(text), params.AssistantCfg.ChatSettings.HistoryMessage.CompressionThreshold)
					//TODO compress history
				}
				contextPrompt += fmt.Sprintf("\nConversation:\n%v\n", text)
			}
		}

		if v, ok := inputValues["references"]; ok {
			contextPrompt += util.SubString(fmt.Sprintf("\nReferences:\n%v\n", v), 0, 4096*2) //TODO
		}

		if v, ok := inputValues["tools_output"]; ok {
			contextPrompt += fmt.Sprintf("\nTools Output:\n%v\n", v)
		}

		inputValues["context"] = contextPrompt
	}

	template := common.GenerateAnswerPromptTemplate
	if params.AssistantCfg.AnsweringModel.PromptConfig != nil && params.AssistantCfg.AnsweringModel.PromptConfig.PromptTemplate != "" {
		template = params.AssistantCfg.AnsweringModel.PromptConfig.PromptTemplate
	}

	// Create the prompt template
	finalPrompt, err := GetPromptStringByTemplateArgs(params.MustGetAnsweringModel(), template, []string{"query", "context"}, inputValues)
	if err != nil {
		panic(err)
	}

	// Append the user's message
	content = append(content, llms.TextParts(llms.ChatMessageTypeHuman, finalPrompt))

	log.Info(content)

	completion, err := llm.GenerateContent(taskCtx, content, options...)
	if err != nil {
		log.Error(err)
		return err
	}
	_ = completion

	chunkSeq += 1

	return nil
}
