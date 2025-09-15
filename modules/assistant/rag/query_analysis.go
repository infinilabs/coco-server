/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rag

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/chains"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/util"
	"regexp"
	"strings"
)

type QueryIntent struct {
	Category   string   `json:"category"`
	Intent     string   `json:"intent"`
	Query      []string `json:"query"`
	Keyword    []string `json:"keyword"`
	Suggestion []string `json:"suggestion"`

	NeedPlanTasks     bool `json:"need_plan_tasks"`     //if it is not a simple task
	NeedCallTools     bool `json:"need_call_tools"`     //if it is necessary
	NeedNetworkSearch bool `json:"need_network_search"` //if need external data sources
}

func QueryAnalysisFromString(str string) (*QueryIntent, error) {
	log.Trace("input:", str)
	jsonContent := extractJSON(str)
	obj := QueryIntent{}
	err := util.FromJSONBytes([]byte(jsonContent), &obj)
	if err != nil {
		return nil, err
	}
	return &obj, nil
}

var jsonBlockTag = regexp.MustCompile(`(?m)(.*?)\<JSON\>([\w\W]+)\<\/JSON\>(.*?)`)
var jsonMarkdownTag = regexp.MustCompile(`(?m)(.*?[\x60]{3,})json([\w\W]+)([\x60]{3,})(.*?)`)

func extractJSON(input string) string {
	matches := jsonMarkdownTag.FindAllStringSubmatch(input, -1)
	if len(matches) > 0 {
		if len(matches[0]) > 2 {
			return strings.TrimSpace(matches[0][2])
		}
	}

	matches = jsonBlockTag.FindAllStringSubmatch(input, -1)
	if len(matches) > 0 {
		if len(matches[0]) > 2 {
			return strings.TrimSpace(matches[0][2])
		}
	}

	return ""
}

func ProcessQueryIntent(ctx context.Context, sessionID string, provider *common.ModelProvider, cfg *common.ModelConfig, reqMsg, replyMsg *common.ChatMessage, assistant *common.Assistant, inputValues map[string]any, sender common.MessageSender) (*QueryIntent, error) {
	// Initialize the LLM
	llm := langchain.GetLLM(provider.BaseURL, provider.APIType, cfg.Name, provider.APIKey, assistant.Keepalive)

	// Create the prompt template
	promptTemplate, err := GetPromptTemplate(cfg, common.QueryIntentPromptTemplate, []string{"history", "query"}, inputValues)
	if err != nil {
		return nil, err
	}

	// Create the LLM chain
	llmChain := chains.NewLLMChain(llm, promptTemplate)

	var chunkSeq = 0
	temperature := langchain.GetTemperature(cfg, provider, 0.8)
	maxTokens := langchain.GetMaxTokens(cfg, provider, 1024)

	// Execute the chain
	output, err := chains.Call(ctx, llmChain, inputValues, chains.WithTemperature(temperature),
		chains.WithMaxTokens(maxTokens),
		chains.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			if len(chunk) > 0 {
				chunkSeq++
				//queryIntentBuffer.Write(chunk)
				fmt.Println(string(chunk))
				msg := common.NewMessageChunk(sessionID, replyMsg.ID, common.MessageTypeAssistant, reqMsg.ID, common.QueryIntent, string(chunk), chunkSeq)
				err := sender.SendMessage(msg)
				if err != nil {
					_ = log.Error(err)
					return err
				}
			}
			return nil
		}))
	if err != nil {
		return nil, fmt.Errorf("error executing LLM chain: %w", err)
	}

	// Extract the generated text
	generatedText, ok := output["text"].(string)
	if !ok {
		return nil, fmt.Errorf("unexpected output type: %T", output["text"])
	}

	// Parse the generated text to extract the JSON
	queryIntent, err := QueryAnalysisFromString(generatedText)
	if err != nil {
		return nil, fmt.Errorf("error parsing query intent: %w", err)
	}

	// Attach the query intent to the reply message
	replyMsg.Details = append(replyMsg.Details, common.ProcessingDetails{
		Order:   10,
		Type:    common.QueryIntent,
		Payload: queryIntent,
	})

	return queryIntent, nil
}
