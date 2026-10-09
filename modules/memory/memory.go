/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	llmmodule "infini.sh/coco/modules/llm"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"

	"github.com/tmc/langchaingo/llms"
)

// Long-term memory, confirmation-gated (W6): every memory starts pending,
// only the owner confirms or rejects it, and only confirmed memories
// recall into prompts or the search_memory tool. The distillation helper
// turns a conversation excerpt into PENDING records — the same human
// gate the governance queue enforces for knowledge.

const (
	Category       = "memory"
	Resource       = "memory"
	distillMaxLens = 4000 // conversation excerpt cap handed to the model
)

func init() {
	orm.MustRegisterSchemaWithIndexName(core.MemoryRecord{}, "memory")
	registerRoutes()
}

func registerRoutes() {
	handler := &APIHandler{}
	readPermission := security.GetSimplePermission(Category, Resource, string(security.Read))
	createPermission := security.GetSimplePermission(Category, Resource, string(security.Create))
	updatePermission := security.GetSimplePermission(Category, Resource, string(security.Update))
	deletePermission := security.GetSimplePermission(Category, Resource, string(security.Delete))
	security.GetOrInitPermissionKeys(readPermission, createPermission, updatePermission, deletePermission)

	// memory reads are login-gated and strictly own-scoped — the
	// permission keys exist for role wiring, the handler enforces the
	// owner check regardless
	api.HandleUIMethod(api.GET, "/memory/_mine", handler.mine, api.RequireLogin(), api.RequirePermission(readPermission))
	api.HandleUIMethod(api.POST, "/memory/", handler.create, api.RequireLogin(), api.RequirePermission(createPermission))
	api.HandleUIMethod(api.POST, "/memory/_distill", handler.distill, api.RequireLogin(), api.RequirePermission(createPermission))
	api.HandleUIMethod(api.PUT, "/memory/:id/_confirm", handler.confirm, api.RequireLogin(), api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.PUT, "/memory/:id/_reject", handler.reject, api.RequireLogin(), api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.DELETE, "/memory/:id", handler.remove, api.RequireLogin(), api.RequirePermission(deletePermission))

	// recall: confirmed memories only — resident kinds always, recalled
	// kinds keyword-matched; the MCP face of the same contract
	api.HandleUIMethod(api.GET, "/memory/_search", handler.search,
		api.RequireLogin(), api.RequirePermission(readPermission),
		api.MCPTool("search_memory", "Search the calling user's confirmed long-term memories (profile/preference/fact/task/interest). Pass a query to recall facts and tasks; resident kinds (profile/preference) return regardless. Only confirmed memories exist here — pending ones await the owner's review."))
}

type APIHandler struct {
	api.Handler
}

/* ---------------- helpers ---------------- */

func currentUser(r *http.Request) (string, bool) {
	u, err := security.GetUserFromRequest(r)
	if err != nil || u == nil || u.UserID == "" {
		return "", false
	}
	return u.UserID, true
}

func loadOwned(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (*core.MemoryRecord, bool) {
	userID, ok := currentUser(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, false
	}
	ctx := orm.NewContextWithParent(r.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.MemoryRecord{})
	rec := core.MemoryRecord{}
	rec.ID = ps.ByName("id")
	exists, err := orm.GetV2(ctx, &rec)
	if err != nil || !exists {
		w.WriteHeader(http.StatusNotFound)
		return nil, false
	}
	// strict owner scope: no cross-user reads, admins included (memories
	// are personal context, not shared knowledge)
	if rec.UserID != userID {
		w.WriteHeader(http.StatusForbidden)
		return nil, false
	}
	return &rec, true
}

/* ---------------- CRUD ---------------- */

func (h *APIHandler) create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userID, ok := currentUser(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	body := util.MapStr{}
	if err := h.DecodeJSON(r, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	rec := &core.MemoryRecord{
		Kind:    str(body, "kind"),
		Content: strings.TrimSpace(str(body, "content")),
		Status:  core.MemoryStatusPending,
		UserID:  userID,
	}
	if v, ok := body["context"].(map[string]interface{}); ok {
		rec.Context = v
	}
	if !core.ValidMemoryKind(rec.Kind) || rec.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("kind must be one of profile/preference/fact/task/interest; content required"))
		return
	}
	ctx := orm.NewContextWithParent(r.Context())
	ctx.Refresh = orm.WaitForRefresh
	if err := orm.Create(ctx, rec); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	writeJSON(w, util.MapStr{"_id": rec.ID, "status": rec.Status})
}

func (h *APIHandler) mine(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userID, ok := currentUser(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	kind := r.URL.Query().Get("kind")
	status := r.URL.Query().Get("status")

	ctx := orm.NewContextWithParent(r.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.MemoryRecord{})
	builder := orm.NewQuery().Size(200).
		Filter(orm.TermQuery("user_id", userID)).
		SortBy(orm.Sort{Field: "created", SortType: orm.DESC})
	if kind != "" {
		builder.Filter(orm.TermQuery("kind", kind))
	}
	if status != "" {
		builder.Filter(orm.TermQuery("status", status))
	}
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	recs, _, _ := decodeRecords(res)
	writeJSON(w, util.MapStr{"items": recs, "total": len(recs)})
}

func (h *APIHandler) confirm(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	setStatus(w, r, ps, core.MemoryStatusConfirmed)
}

func (h *APIHandler) reject(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	setStatus(w, r, ps, core.MemoryStatusRejected)
}

func setStatus(w http.ResponseWriter, r *http.Request, ps httprouter.Params, status string) {
	rec, ok := loadOwned(w, r, ps)
	if !ok {
		return
	}
	if rec.Status != core.MemoryStatusPending {
		writeJSON(w, util.MapStr{"_id": rec.ID, "status": rec.Status, "note": "already settled"})
		return
	}
	rec.Status = status
	ctx := orm.NewContextWithParent(r.Context())
	ctx.Refresh = orm.WaitForRefresh
	if err := orm.Update(ctx, rec); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	writeJSON(w, util.MapStr{"_id": rec.ID, "status": rec.Status})
}

func (h *APIHandler) remove(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	rec, ok := loadOwned(w, r, ps)
	if !ok {
		return
	}
	ctx := orm.NewContextWithParent(r.Context())
	ctx.Refresh = orm.WaitForRefresh
	if err := orm.Delete(ctx, rec); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	writeJSON(w, util.MapStr{"_id": rec.ID, "result": "deleted"})
}

/* ---------------- recall + MCP ---------------- */

// search returns the caller's CONFIRMED memories: resident kinds always
// (length-capped by the caller), recalled kinds keyword-matched against
// the query. Pending/rejected never surface here.
func (h *APIHandler) search(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userID, ok := currentUser(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	items := Recall(r.Context(), userID, query, 20)
	writeJSON(w, util.MapStr{"items": items, "total": len(items)})
}

// Recall is the prompt-injection contract: confirmed memories only.
func Recall(ctx context.Context, userID, query string, limit int) []core.MemoryRecord {
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.MemoryRecord{})
	builder := orm.NewQuery().Size(200).
		Filter(
			orm.TermQuery("user_id", userID),
			orm.TermQuery("status", core.MemoryStatusConfirmed),
		).
		SortBy(orm.Sort{Field: "updated", SortType: orm.DESC})
	res, err := orm.SearchV2(octx, builder)
	if err != nil {
		return nil
	}
	recs, _, err := decodeRecords(res)
	if err != nil {
		return nil
	}

	out := make([]core.MemoryRecord, 0, limit)
	q := strings.ToLower(query)
	for _, rec := range recs {
		if limit > 0 && len(out) >= limit {
			break
		}
		if isResidentKind(rec.Kind) {
			out = append(out, rec)
			continue
		}
		// recalled kinds: naive keyword overlap (vector recall needs an
		// embedding per memory — worth it once memories number in the
		// hundreds, not at introduction)
		if q != "" && keywordOverlap(strings.ToLower(rec.Content), q) {
			out = append(out, rec)
		}
	}
	return out
}

func isResidentKind(kind string) bool {
	return kind == core.MemoryKindProfile || kind == core.MemoryKindPreference
}

func keywordOverlap(content, query string) bool {
	for _, tok := range strings.Fields(query) {
		if len(tok) >= 2 && strings.Contains(content, tok) {
			return true
		}
	}
	return false
}

/* ---------------- distillation ---------------- */

const distillSystemPrompt = `You extract durable long-term memories from a conversation excerpt.
Return ONLY a JSON array (possibly empty) of objects {"kind": "...", "content": "..."} where kind is one of:
"profile" (who the user is), "preference" (how the user wants replies), "fact" (a stable fact worth remembering),
"task" (something the user is working on / asked to be reminded of), "interest" (a recurring topic).
Rules: standalone sentences, the user's language, no conversation references, no duplicates of things already obvious.
Skip anything transient. Max 5 items. No explanations, just the JSON array.`

// distillRateLimiter: one distillation per user per window (W6: 5 min).
var distillMu sync.Map // userID → time.Time

func (h *APIHandler) distill(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userID, ok := currentUser(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if last, loaded := distillMu.Load(userID); loaded {
		if t := last.(time.Time); time.Since(t) < 5*time.Minute {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("distillation rate-limited to once per 5 minutes"))
			return
		}
	}

	body := util.MapStr{}
	if err := h.DecodeJSON(r, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	excerpt := str(body, "conversation")
	if len(excerpt) > distillMaxLens {
		excerpt = excerpt[:distillMaxLens]
	}
	if strings.TrimSpace(excerpt) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("conversation required"))
		return
	}

	created, err := Distill(r.Context(), userID, excerpt, str(body, "session_id"))
	if err != nil {
		log.Warnf("memory: distillation failed: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("distillation model unavailable: " + err.Error()))
		return
	}
	distillMu.Store(userID, time.Now())
	if created > 0 {
		NotifyPendingMemories(userID, created)
	}
	writeJSON(w, util.MapStr{"created": created, "status": "pending — confirm via /memory/:id/_confirm"})
}

// Distill runs the excerpt through the default language model and files
// every extracted memory as PENDING.
func Distill(ctx context.Context, userID, excerpt, sessionID string) (int, error) {
	m := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if m == nil {
		return 0, fmt.Errorf("no default language model configured")
	}
	llm, err := langchain.SimplyGetLLM(m.ProviderID, m.ID, "")
	if err != nil {
		return 0, err
	}
	messages := []llms.MessageContent{
		langchain.SystemTextParts(distillSystemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, excerpt),
	}
	resp, err := llm.GenerateContent(ctx, messages, llms.WithMaxTokens(600))
	if err != nil {
		return 0, err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return 0, fmt.Errorf("empty distillation response")
	}
	parsed, err := parseDistillOutput(resp.Choices[0].Content)
	if err != nil {
		return 0, err
	}

	octx := orm.NewContextWithParent(ctx)
	octx.Refresh = orm.WaitForRefresh
	created := 0
	for _, item := range parsed {
		if !core.ValidMemoryKind(item.Kind) || strings.TrimSpace(item.Content) == "" {
			continue
		}
		rec := &core.MemoryRecord{
			Kind:    item.Kind,
			Content: strings.TrimSpace(item.Content),
			Status:  core.MemoryStatusPending,
			UserID:  userID,
			Context: map[string]interface{}{"source": "distill", "session_id": sessionID, "model": m.ProviderID + "/" + m.ID},
		}
		if err := orm.Create(octx, rec); err != nil {
			log.Warnf("memory: filing distilled record failed: %v", err)
			continue
		}
		created++
	}
	return created, nil
}

type distillItem struct {
	Kind    string `json:"kind"`
	Content string `json:"content"`
}

// parseDistillOutput finds the JSON array in the model answer.
func parseDistillOutput(s string) ([]distillItem, error) {
	fenced := regexpFind(s, "(?s)```[a-zA-Z]*\\n?(.*?)```")
	if fenced != "" {
		s = fenced
	}
	start, end := strings.Index(s, "["), strings.LastIndex(s, "]")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON array in distillation output")
	}
	var items []distillItem
	if err := json.Unmarshal([]byte(s[start:end+1]), &items); err != nil {
		return nil, err
	}
	if len(items) > 5 {
		items = items[:5]
	}
	return items, nil
}

/* ---------------- misc ---------------- */

func str(m util.MapStr, k string) string {
	v, _ := m[k].(string)
	return v
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

var distillFenceRe = regexp.MustCompile("(?s)```[a-zA-Z]*\n?(.*?)```")

func regexpFind(s, pattern string) string {
	m := distillFenceRe.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func decodeRecords(res *orm.SearchResult) ([]core.MemoryRecord, int64, error) {
	out := &struct {
		Hits struct {
			Total int64 `json:"value"`
			Hits  []struct {
				Source core.MemoryRecord `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}{}
	if raw, ok := res.Payload.([]byte); ok {
		if err := json.Unmarshal(raw, out); err != nil {
			return nil, 0, err
		}
	}
	recs := make([]core.MemoryRecord, 0, len(out.Hits.Hits))
	for _, h := range out.Hits.Hits {
		recs = append(recs, h.Source)
	}
	return recs, out.Hits.Total, nil
}
