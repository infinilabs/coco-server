/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package langchain

import (
	"fmt"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
)

const currentTimePromptFormat = "The current time is %s. Use this value as the authoritative current date and time. If the user asks about the current date or time, answer directly from this value and ignore earlier conversation turns that claim the current time is unavailable."

// PromptWithCurrentTime appends the current local time to a prompt so every LLM
// call has a consistent temporal reference for time-sensitive questions and
// tool decisions.
func PromptWithCurrentTime(prompt string) string {
	return promptWithTime(prompt, time.Now())
}

// SystemPromptWithCurrentTime appends the current local time to a system prompt.
func SystemPromptWithCurrentTime(prompt string) string {
	return PromptWithCurrentTime(prompt)
}

// SystemTextParts builds a system-role message with the shared current-time
// context attached. Use it instead of calling llms.TextParts directly for system
// prompts.
func SystemTextParts(prompt string) llms.MessageContent {
	return llms.TextParts(llms.ChatMessageTypeSystem, SystemPromptWithCurrentTime(prompt))
}

func promptWithTime(prompt string, now time.Time) string {
	timePrompt := fmt.Sprintf(currentTimePromptFormat, now.Format("January 02, 2006 15:04"))
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return timePrompt
	}
	return prompt + "\n\n" + timePrompt
}
