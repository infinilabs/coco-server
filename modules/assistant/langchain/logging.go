// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package langchain

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

// LogHandler is a callback handler that prints to the standard output.
type LogHandler struct {
	CustomWriteFunc func(chunk string)
}

func (l *LogHandler) HandleLLMGenerateContentStart(_ context.Context, ms []llms.MessageContent) {
	log.Trace("Entering LLM with messages:")
	for _, m := range ms {
		// TODO: Implement logging of other content types
		var buf strings.Builder
		for _, t := range m.Parts {
			if t, ok := t.(llms.TextContent); ok {
				buf.WriteString(t.Text)
			}
		}
		log.Trace("Role:", m.Role)
		log.Trace("Text:", buf.String())
	}
}

func (l *LogHandler) HandleLLMGenerateContentEnd(_ context.Context, res *llms.ContentResponse) {
	log.Trace("Exiting LLM with response:")
	for _, c := range res.Choices {
		if c.Content != "" {
			log.Trace("Content:", c.Content)
		}
		if c.StopReason != "" {
			log.Trace("StopReason:", c.StopReason)
		}
		if len(c.GenerationInfo) > 0 {
			log.Trace("GenerationInfo:")
			for k, v := range c.GenerationInfo {
				fmt.Printf("%20s: %v\n", k, v)
			}
		}
		if c.FuncCall != nil {
			log.Trace("FuncCall: ", c.FuncCall.Name, c.FuncCall.Arguments)
		}
	}
}

func (l *LogHandler) HandleStreamingFunc(_ context.Context, chunk []byte) {
	if l.CustomWriteFunc != nil {
		l.CustomWriteFunc(string(chunk))
	}
}

func (l *LogHandler) HandleText(_ context.Context, text string) {
	log.Trace(text)
}

func (l *LogHandler) HandleLLMStart(_ context.Context, prompts []string) {
	log.Trace("Entering LLM with prompts:", prompts)
}

func (l *LogHandler) HandleLLMError(_ context.Context, err error) {
	log.Trace("Exiting LLM with error:", err)
}

func (l *LogHandler) HandleChainStart(_ context.Context, inputs map[string]any) {
	log.Debug("Entering chain with inputs:", formatChainValues(inputs))
}

func (l *LogHandler) HandleChainEnd(_ context.Context, outputs map[string]any) {
	log.Debug("Exiting chain with outputs:", formatChainValues(outputs))
}

func (l *LogHandler) HandleChainError(_ context.Context, err error) {
	log.Trace("Exiting chain with error:", err)
}

func (l *LogHandler) HandleToolStart(_ context.Context, input string) {
	log.Trace("Entering tool with input:", removeNewLines(input))
}

func (l *LogHandler) HandleToolEnd(_ context.Context, output string) {
	log.Trace("Exiting tool with output:", removeNewLines(output))
}

func (l *LogHandler) HandleToolError(_ context.Context, err error) {
	log.Trace("Exiting tool with error:", err)
}

func (l *LogHandler) HandleAgentAction(_ context.Context, action schema.AgentAction) {
	log.Trace("Agent selected action:", formatAgentAction(action))
}

func (l *LogHandler) HandleAgentFinish(_ context.Context, finish schema.AgentFinish) {
	fmt.Printf("Agent finish: %v \n", finish)
}

func (l *LogHandler) HandleRetrieverStart(_ context.Context, query string) {
	log.Trace("Entering retriever with query:", removeNewLines(query))
}

func (l *LogHandler) HandleRetrieverEnd(_ context.Context, query string, documents []schema.Document) {
	log.Trace("Exiting retriever with documents for query:", documents, query)
}

func formatChainValues(values map[string]any) string {
	output := ""
	for key, value := range values {
		output += fmt.Sprintf("\"%s\" : \"%s\", ", removeNewLines(key), removeNewLines(value))
	}

	return output
}

func formatAgentAction(action schema.AgentAction) string {
	return fmt.Sprintf("\"%s\" with input \"%s\"", removeNewLines(action.Tool), removeNewLines(action.ToolInput))
}

func removeNewLines(s any) string {
	return strings.ReplaceAll(fmt.Sprint(s), "\n", " ")
}
