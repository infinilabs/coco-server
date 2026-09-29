/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"fmt"
	"net/http"
	"strings"

	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
)

/* ---------------- knowledge execution: POST /wiki/article/_from_chat ---------------- */

const (
	fromChatMaxContentBytes = 100 * 1024 // 100 KB of markdown is far beyond any answer
	fromChatMaxSources      = 50
	fromChatMaxTitleRunes   = 200
	fromChatMaxSummaryRunes = 500
)

type fromChatSource struct {
	DocID   string `json:"doc_id"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Excerpt string `json:"excerpt"`
}

type fromChatRequest struct {
	KbID      string           `json:"kb_id"`
	Title     string           `json:"title"`
	Summary   string           `json:"summary"`
	Content   string           `json:"content"`
	Sources   []fromChatSource `json:"sources"`
	MessageID string           `json:"message_id"`
	SessionID string           `json:"session_id"`
}

// createArticleFromChat lands an assistant answer into a knowledge base as a
// draft article — the chat-side twin of the KM generation pipeline. The client
// supplies the content and citations it already rendered (streamed replies are
// not guaranteed to know their server-side message id); the server re-validates
// and stamps the provenance. D1 is inherited from createDraftArticle: the
// result is a draft, publishing stays a human action.
func (h *APIHandler) createArticleFromChat(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := fromChatRequest{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.Error400(w, err.Error())
		return
	}

	body.Title = strings.TrimSpace(body.Title)
	if body.KbID == "" || body.Title == "" || strings.TrimSpace(body.Content) == "" {
		h.Error400(w, "kb_id, title and content are required")
		return
	}
	if len([]rune(body.Title)) > fromChatMaxTitleRunes {
		h.Error400(w, fmt.Sprintf("title exceeds %d characters", fromChatMaxTitleRunes))
		return
	}
	if len(body.Content) > fromChatMaxContentBytes {
		h.Error400(w, fmt.Sprintf("content exceeds %d bytes", fromChatMaxContentBytes))
		return
	}
	if len(body.Sources) > fromChatMaxSources {
		h.Error400(w, fmt.Sprintf("sources exceed %d items", fromChatMaxSources))
		return
	}

	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiKnowledgeBase{})

	var kb core.WikiKnowledgeBase
	kb.SetID(body.KbID)
	exists, err := orm.GetV2(ctx, &kb)
	if err != nil {
		if !strings.Contains(err.Error(), "record not found") {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		exists = false // the sqlite backend reports a miss as an error
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, body.KbID)
		return
	}

	sources := make([]core.WikiSourceReference, 0, len(body.Sources))
	for _, s := range body.Sources {
		if s.DocID == "" {
			continue // the doc_id backlink is the D2 contract; no id, no reference
		}
		excerpt := s.Excerpt
		if len(excerpt) > 200 {
			excerpt = excerpt[:200]
		}
		sources = append(sources, core.WikiSourceReference{
			DocID:   s.DocID,
			Title:   s.Title,
			URL:     s.URL,
			Excerpt: excerpt,
		})
	}

	summary := strings.TrimSpace(body.Summary)
	if len([]rune(summary)) > fromChatMaxSummaryRunes {
		summary = string([]rune(summary)[:fromChatMaxSummaryRunes])
	}

	article, err := createDraftArticle(&kb, body.Title, summary, body.Content,
		"concept", "", sources, "",
		fmt.Sprintf("saved from chat answer (%d citations)", len(sources)),
		fmt.Sprintf("Chat answer %q saved as draft to knowledge base %s", body.Title, kb.Name))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteCreatedOKJSON(w, article.ID)
}
