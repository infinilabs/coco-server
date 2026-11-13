/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"fmt"
	"strings"

	"infini.sh/coco/core"
)

// Deterministic entity description: a zero-hallucination summary built
// from provenance alone. Entities without an LLM-written Description
// (legacy rows, extraction summaries unavailable) still get a usable
// one-liner for cards, tooltips and AI references instead of a bare
// name — the wiki layer's answer to "实体没有详情".

const entityExcerptMaxRunes = 80

// EntityDeterministicDescription renders "在 N 篇文档中被提及" plus the
// first verbatim source excerpt. Returns "" when the entity has no
// provenance at all (nothing honest to say).
func EntityDeterministicDescription(e *core.WikiEntity) string {
	if e == nil || len(e.Sources) == 0 {
		return ""
	}
	var excerpt, title string
	for i := range e.Sources {
		s := strings.TrimSpace(e.Sources[i].Excerpt)
		if s != "" {
			excerpt = s
			title = strings.TrimSpace(e.Sources[i].Title)
			break
		}
	}
	head := fmt.Sprintf("在 %d 篇文档中被提及", len(e.Sources))
	if excerpt == "" {
		return head + "。"
	}
	runes := []rune(excerpt)
	if len(runes) > entityExcerptMaxRunes {
		excerpt = string(runes[:entityExcerptMaxRunes]) + "…"
	}
	if title != "" {
		return fmt.Sprintf("%s，如《%s》：「%s」", head, title, excerpt)
	}
	return fmt.Sprintf("%s：「%s」", head, excerpt)
}

// EntityDisplayDescription prefers the stored (LLM-written) description
// and falls back to the deterministic one — display layers should never
// need to know which one they got.
func EntityDisplayDescription(e *core.WikiEntity) string {
	if e == nil {
		return ""
	}
	if d := strings.TrimSpace(e.Description); d != "" {
		return d
	}
	return EntityDeterministicDescription(e)
}
