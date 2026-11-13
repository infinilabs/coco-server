/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

// The raw experience layer (LLM-wiki P1, the WikiSkill raw/ analog):
// immutable, expiring traces of how answers went in the wild. Capture is
// opt-in per KB (ExperienceCapture) because traces contain user questions.
// The distiller (distill.go) is the only reader.

// experienceRetentionDefault keeps raw traces for 90 days — enough for the
// distiller's lookback window, short enough to bound the privacy surface.
const experienceRetentionDefault = 90 * 24 * time.Hour

const experienceListDefaultSize = 50
const experienceListMaxSize = 200
const experienceMaxText = 8000

// RecordExperience writes one trace. Fire-and-forget by contract: capture
// must never fail the work signal that produced it. No-op unless the KB
// exists with ExperienceCapture on — the opt-in gate lives here so every
// capture path shares it.
func RecordExperience(parent *orm.Context, exp *core.WikiExperience) {
	if exp == nil || exp.KbID == "" {
		return
	}
	defer func() {
		if r := recover(); r == nil {
			return
		} else {
			log.Warnf("wiki: experience capture recovered: %v", r)
		}
	}()
	kctx := orm.NewContextWithParent(parent)
	kctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(kctx, &core.WikiKnowledgeBase{})
	var kb core.WikiKnowledgeBase
	kb.SetID(exp.KbID)
	exists, err := orm.GetV2(kctx, &kb)
	if err != nil || !exists || !kb.ExperienceCapture {
		return
	}
	if exp.Retention == nil {
		t := time.Now().Add(experienceRetentionDefault)
		exp.Retention = &t
	}
	exp.Question = util.SubString(exp.Question, 0, experienceMaxText)
	exp.Answer = util.SubString(exp.Answer, 0, experienceMaxText)
	exp.Note = util.SubString(exp.Note, 0, experienceMaxText)

	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	if err := orm.Create(ctx, exp); err != nil {
		log.Warnf("wiki: experience capture failed for kb [%s]: %v", exp.KbID, err)
	}
}

// KBsByAssistant returns the ids of experience-capturing KBs bound to an
// assistant — the vote/API capture path resolves its targets this way.
func KBsByAssistant(parent *orm.Context, assistantID string) []string {
	if assistantID == "" {
		return nil
	}
	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiKnowledgeBase{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(50).
		Filter(orm.TermQuery("assistant_id", assistantID),
			orm.TermQuery("experience_capture", true)))
	if err != nil || res == nil {
		return nil
	}
	kbs, _, derr := elastic.DecodeHits[core.WikiKnowledgeBase](res)
	if derr != nil {
		return nil
	}
	ids := make([]string, 0, len(kbs))
	for i := range kbs {
		ids = append(ids, kbs[i].ID)
	}
	return ids
}

// kbBindingCache short-circuits the datasource→KB lookup: the enrich
// pipeline resolves it once per document, and datasource bindings change
// at human speed. Entries live for the process lifetime — a rebinding
// applies after restart, which is fine for a pipeline default.
var (
	kbBindingCacheMu sync.RWMutex
	kbBindingCache   = map[string]string{}
)

// ResolveKBByDatasource returns the id of the wiki KB whose datasource
// binding contains datasourceID — the entity-extraction target resolution
// (a global enrich pipeline serves several KBs). Empty when no KB binds
// this datasource.
func ResolveKBByDatasource(parent *orm.Context, datasourceID string) string {
	if datasourceID == "" {
		return ""
	}
	kbBindingCacheMu.RLock()
	if cached, ok := kbBindingCache[datasourceID]; ok {
		kbBindingCacheMu.RUnlock()
		return cached
	}
	kbBindingCacheMu.RUnlock()

	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiKnowledgeBase{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("datasource_ids", datasourceID)))
	if err != nil || res == nil {
		return ""
	}
	kbs, _, derr := elastic.DecodeHits[core.WikiKnowledgeBase](res)
	if derr != nil || len(kbs) == 0 {
		return ""
	}
	kbID := kbs[0].ID
	kbBindingCacheMu.Lock()
	kbBindingCache[datasourceID] = kbID
	kbBindingCacheMu.Unlock()
	return kbID
}

// expireExperiences deletes expired traces for a KB — called from the
// distill run so the raw layer never grows without bound.
func expireExperiences(ctx *orm.Context, kbID string) int {
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiExperience{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(500).
		Filter(orm.TermQuery("kb_id", kbID)))
	if err != nil || res == nil {
		return 0
	}
	experiences, _, derr := elastic.DecodeHits[core.WikiExperience](res)
	if derr != nil {
		return 0
	}
	now := time.Now()
	removed := 0
	for i := range experiences {
		if experiences[i].Retention == nil || !experiences[i].Retention.Before(now) {
			continue
		}
		if err := deleteByID(ctx, &core.WikiExperience{}, experiences[i].ID); err == nil {
			removed++
		}
	}
	return removed
}

/* ---------------- API ---------------- */

// captureRequest is the generic capture payload for API/MCP callers (the
// chat vote button lands here once wired; corrections go through their own
// hook below).
type captureRequest struct {
	Kind      string   `json:"kind,omitempty"`
	SessionID string   `json:"session_id,omitempty"`
	MessageID string   `json:"message_id,omitempty"`
	Question  string   `json:"question,omitempty"`
	Answer    string   `json:"answer,omitempty"`
	Feedback  string   `json:"feedback,omitempty"`
	Note      string   `json:"note,omitempty"`
	Outcome   string   `json:"outcome,omitempty"`
	CitedIDs  []string `json:"cited_doc_ids,omitempty"`
}

func (h *APIHandler) captureExperience(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	body := captureRequest{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Question) == "" && strings.TrimSpace(body.Note) == "" {
		h.Error400(w, "question or note is required")
		return
	}
	if user, err := security.GetUserFromRequest(req); err != nil || user == nil {
		h.WriteError(w, "login required", http.StatusUnauthorized)
		return
	}
	exp := &core.WikiExperience{
		KbID:      kbID,
		Kind:      body.Kind,
		SessionID: body.SessionID,
		MessageID: body.MessageID,
		Question:  strings.TrimSpace(body.Question),
		Answer:    strings.TrimSpace(body.Answer),
		Feedback:  body.Feedback,
		Note:      strings.TrimSpace(body.Note),
		Outcome:   body.Outcome,
	}
	for _, docID := range body.CitedIDs {
		exp.Citations = append(exp.Citations, core.WikiSourceReference{DocID: docID})
	}
	RecordExperience(orm.NewContextWithParent(req.Context()), exp)
	h.WriteCreatedOKJSON(w, "accepted")
}

func (h *APIHandler) listExperience(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	from, _ := strconv.Atoi(req.FormValue("from"))
	size, _ := strconv.Atoi(req.FormValue("size"))
	if size <= 0 {
		size = experienceListDefaultSize
	}
	if size > experienceListMaxSize {
		size = experienceListMaxSize
	}
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiExperience{})
	builder := orm.NewQuery()
	builder.From(from)
	builder.Size(size)
	builder.Filter(orm.TermQuery("kb_id", kbID))
	builder.SortBy(orm.Sort{Field: "created", SortType: "desc"})
	res, err := orm.SearchV2(ctx, builder)
	if err != nil || res == nil {
		h.WriteError(w, fmt.Sprintf("experience query failed: %v", err), http.StatusInternalServerError)
		return
	}
	entries, total, derr := elastic.DecodeHits[core.WikiExperience](res)
	if derr != nil {
		h.WriteError(w, derr.Error(), http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []core.WikiExperience{}
	}
	// the loop's schedule facts ride along: when the next automatic distill
	// fires and when this KB's last one ran (nil = never)
	lastRun := distillLastRun(kbID)
	h.WriteOKJSON(w, util.MapStr{
		"total":   total,
		"entries": entries,
		"distill": util.MapStr{
			"interval": distillScheduleInterval(),
			"last_run": lastRun,
		},
	})
}

// voteRequest is the assistant-side capture payload: the chat UI knows the
// assistant (the session's), not the KB — targets resolve through the
// assistant↔KB binding, and each KB's ExperienceCapture opt-in still gates
// inside RecordExperience.
type voteRequest struct {
	AssistantID string `json:"assistant_id"`
	SessionID   string `json:"session_id,omitempty"`
	MessageID   string `json:"message_id,omitempty"`
	Question    string `json:"question,omitempty"`
	Answer      string `json:"answer,omitempty"`
	Feedback    string `json:"feedback"` // positive | negative
	Note        string `json:"note,omitempty"`
}

// captureExperienceVote lands a thumbs-up/down from the chat surface.
// POST /wiki/experience/_vote. Always 200 with a count — vote state must
// never surface whether a capturing KB exists.
func (h *APIHandler) captureExperienceVote(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := voteRequest{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	feedback := strings.TrimSpace(body.Feedback)
	if feedback != core.ExperienceFeedbackPositive && feedback != core.ExperienceFeedbackNegative {
		h.Error400(w, "feedback must be positive or negative")
		return
	}
	if strings.TrimSpace(body.AssistantID) == "" {
		h.Error400(w, "assistant_id is required")
		return
	}
	if user, err := security.GetUserFromRequest(req); err != nil || user == nil {
		h.WriteError(w, "login required", http.StatusUnauthorized)
		return
	}
	ctx := orm.NewContextWithParent(req.Context())
	kbIDs := KBsByAssistant(ctx, body.AssistantID)
	recorded := 0
	for _, kbID := range kbIDs {
		exp := &core.WikiExperience{
			KbID:      kbID,
			Kind:      "chat",
			SessionID: body.SessionID,
			MessageID: body.MessageID,
			Question:  strings.TrimSpace(body.Question),
			Answer:    strings.TrimSpace(body.Answer),
			Feedback:  feedback,
			Note:      strings.TrimSpace(body.Note),
			Outcome:   core.ExperienceOutcomeResolved,
		}
		RecordExperience(ctx, exp)
		recorded++
	}
	h.WriteOKJSON(w, util.MapStr{"recorded": recorded})
}
