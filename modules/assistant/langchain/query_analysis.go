/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package langchain

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/chains"
	"infini.sh/coco/core"
	common2 "infini.sh/coco/modules/assistant/common"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/util"
)

func QueryAnalysisFromString(str string) (*common2.QueryIntent, error) {
	log.Trace("input:", str)
	jsonContent := extractJSON(str)
	obj := common2.QueryIntent{}
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

func ProcessQueryIntent(ctx context.Context, sessionID string, model *core.ModelConfig, reqMsg, replyMsg *core.ChatMessage, assistant *core.Assistant, inputValues map[string]any, sender core.MessageSender) (*common2.QueryIntent, error) {
	// maxAttempts limits the number of LLM calls for query intent parsing.
	// When the LLM returns malformed JSON (e.g. nested arrays instead of
	// flat string arrays), the error is fed back and the LLM gets another
	// chance to self-correct. Capped at 3 to avoid runaway retries.
	const maxAttempts = 3

	// Initialize the LLM
	llm, err := SimplyGetLLM(model.ProviderID, model.Name, assistant.Keepalive)
	if err != nil {
		return nil, err
	}

	// Create the prompt template
	promptTemplate, err := GetPromptTemplate(model, common.QueryIntentPromptTemplate, []string{"history", "query"}, inputValues)
	if err != nil {
		return nil, err
	}

	// Create the LLM chain
	llmChain := chains.NewLLMChain(llm, promptTemplate)

	var chunkSeq = 0

	// Copy inputValues to avoid mutating the original on retries
	currentInputs := make(map[string]any, len(inputValues))
	for k, v := range inputValues {
		currentInputs[k] = v
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// On retry, append the error feedback to the query so the LLM can self-correct
		if attempt > 0 {
			log.Warnf("query intent parse failed (attempt %d/%d): %v, retrying", attempt, maxAttempts, lastErr)
			currentInputs["query"] = fmt.Sprintf("%v\n\n"+
				"[Your previous output had a formatting error: %s]\n"+
				"Please fix the JSON format and try again. "+
				"Make sure keyword, query, and suggestion are flat arrays of strings (e.g. [\"a\", \"b\"]), not nested arrays.",
				inputValues["query"], lastErr.Error())
		}

		// Execute the chain
		// Only stream chunks to the client on the first attempt to avoid
		// sending duplicated/garbled content when retrying.
		callOpts := []chains.ChainCallOption{
			chains.WithTemperature(util.GetFloat64OrDefault(model.Settings.Temperature, 0.9)),
			chains.WithMaxTokens(util.GetIntOrDefault(model.Settings.MaxTokens, 1024)),
		}
		if attempt == 0 {
			callOpts = append(callOpts, chains.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				if len(chunk) > 0 {
					chunkSeq++
					if sendErr := sender.SendChunkMessage(core.MessageTypeAssistant, common.QueryIntent, string(chunk), chunkSeq); sendErr != nil {
						_ = log.Error(sendErr)
						return sendErr
					}
				}
				return nil
			}))
		}
		output, err := chains.Call(ctx, llmChain, currentInputs, callOpts...)
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
			lastErr = fmt.Errorf("error parsing query intent: %w", err)
			continue
		}

		// Attach the query intent to the reply message
		replyMsg.Details = append(replyMsg.Details, core.ProcessingDetails{
			Order:   10,
			Type:    common.QueryIntent,
			Payload: queryIntent,
		})

		return queryIntent, nil
	}

	return nil, lastErr
}
