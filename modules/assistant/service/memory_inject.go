/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package service

import (
	"context"
	"fmt"
	"strings"

	"infini.sh/coco/modules/memory"
)

// Long-term memory injection (W6): confirmed memories for the calling
// user join the assistant context — resident kinds (profile/preference)
// every turn, fact/task kinds recalled against the current query.
// PENDING memories never inject (the confirmation gate in the memory
// module enforces this; this helper just consumes Recall's contract).
//
// Injected as a dedicated prompt section so the model can distinguish
// remembered context from the current conversation.

const memorySectionMaxChars = 1500

// memoryRecallFn is swappable in tests.
var memoryRecallFn = memory.Recall

// buildMemorySection recalls the user's confirmed memories and renders
// them as a prompt section. Empty when the user has no applicable
// memories (no section — no noise).
func buildMemorySection(userID, query string) string {
	recs := memoryRecallFn(context.Background(), userID, query, 20)
	if len(recs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\nLong-term Memory (confirmed facts about this user — respect preferences, use facts when relevant):\n")
	used := 0
	for _, rec := range recs {
		line := fmt.Sprintf("- [%s] %s\n", rec.Kind, rec.Content)
		if used+len(line) > memorySectionMaxChars {
			break
		}
		sb.WriteString(line)
		used += len(line)
	}
	return sb.String()
}
