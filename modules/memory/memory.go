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
	// Category must stay "coco": permission keys render as "<category>#<resource>/<action>"
	// and the frontend gates the memory pages on "coco#memory/read|update" — a module-local
	// prefix here (e.g. "memory") silently locks the UI out of its own data
	Category       = "coco"
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
	// "all" is the UI's explicit no-filter choice — treat it like an absent
	// param instead of matching a literal status value that never exists
	if status != "" && status != "all" {
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
	// the human gate owns prompt-visible state: confirming a replacement
	// is the moment the displaced record leaves the context
	if status == core.MemoryStatusConfirmed && rec.SupersedeOf != "" {
		if ownerID, ok := currentUser(r); ok {
			supersedeTarget(ctx, ownerID, rec.SupersedeOf)
		}
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
	var overflow []core.MemoryRecord // non-resident, no keyword match
	for _, rec := range recs {
		if limit > 0 && len(out) >= limit {
			break
		}
		if isResidentKind(rec.Kind) {
			out = append(out, rec)
			continue
		}
		// recalled kinds: keyword overlap ranks topical matches first —
		// but nothing is DROPPED: non-matching context can still matter
		// (the model judges relevance better than a lexical filter), and
		// RenderSection's char cap bounds the final prompt size
		if q != "" && keywordOverlap(strings.ToLower(rec.Content), q) {
			out = append(out, rec)
		} else {
			overflow = append(overflow, rec)
		}
	}
	for _, rec := range overflow {
		if limit > 0 && len(out) >= limit {
			break
		}
		out = append(out, rec)
	}
	return out
}

func isResidentKind(kind string) bool {
	return kind == core.MemoryKindProfile || kind == core.MemoryKindPreference
}

// PromptSectionMaxChars caps the rendered memory prompt section (bytes;
// ~800 CJK chars — small against the context window, enough for dozens
// of memories).
const PromptSectionMaxChars = 2400

// RenderSection renders recalled records as the prompt-injection text.
// Exported so every reply path (simple mode, deep_search, ...) injects
// the SAME section format. Recalled kinds (fact/task/interest) carry an
// as-of month so the model can read change/trend over time; resident
// state kinds stay undated (always current).
func RenderSection(recs []core.MemoryRecord, maxChars int) string {
	if len(recs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\nLong-term Memory (confirmed facts about this user — respect preferences, use facts when relevant; dated entries are observations — newer supersedes older):\n")
	used := 0
	for _, rec := range recs {
		var line string
		if isResidentKind(rec.Kind) {
			line = fmt.Sprintf("- [%s] %s\n", rec.Kind, rec.Content)
		} else {
			line = fmt.Sprintf("- [%s] (as of %s) %s\n", rec.Kind, monthOf(rec.Updated), rec.Content)
		}
		if used+len(line) > maxChars {
			break
		}
		sb.WriteString(line)
		used += len(line)
	}
	return sb.String()
}

// monthOf renders the as-of month "YYYY-MM" of an ORM timestamp,
// "unknown" when absent.
func monthOf(ts *time.Time) string {
	if ts == nil {
		return "unknown"
	}
	return ts.Format("2006-01")
}

func keywordOverlap(content, query string) bool {
	for _, tok := range strings.Fields(query) {
		if len(tok) >= 2 && strings.Contains(content, tok) {
			return true
		}
	}
	// CJK fallback: Chinese text has no spaces, so Fields yields ONE giant
	// token that never appears in any memory — match on character bigrams
	// instead (≥2 shared bigrams ≈ topically related)
	return bigramOverlap(content, query)
}

func bigramOverlap(content, query string) bool {
	runes := []rune(query)
	hits := 0
	for i := 0; i+2 <= len(runes); i++ {
		if strings.Contains(content, string(runes[i:i+2])) {
			hits++
			if hits >= 2 {
				return true
			}
		}
	}
	return false
}

/* ---------------- distillation ---------------- */

const distillSystemPrompt = `You extract durable long-term memories from a conversation excerpt.
Return ONLY a JSON array (possibly empty) of objects {"kind": "...", "content": "..."} where kind is one of:
"profile" (who the user is — CURRENT state: role, gender, age, height, weight, location, team...),
"preference" (how the user wants replies), "fact" (a dated observation worth keeping as history —
prefix the content with its time, e.g. "2026-11: weighed 82kg"), "task" (something the user is working on),
"interest" (a recurring topic).
State vs event: a changeable current attribute (weight, city, role) is "profile" — keep ONE record with the
latest value. Historical observations worth tracking over time (measurements, milestones, moves) are "fact"
with a time prefix — NEVER merge or overwrite those.
Updating state: if the excerpt CHANGES a state that already exists in EXISTING MEMORIES below (e.g. a new
weight), include "supersedes": "<that record's id>" on the new item. Only supersede, never restate an
unchanged memory.
Rules: standalone sentences, the user's language, no conversation references, no duplicates of things
already obvious. Skip anything transient. Max 5 items. No explanations, just the JSON array.`

// distillRateLimiter: the MANUAL distill endpoint's per-user rate limit
// (once per window). The automatic path (AutoDistill) uses the per-user
// merge buffer (B1) instead; the flush also stamps this so a manual call
// right after an automatic one is still throttled.
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
	} else {
		log.Infof("memory: manual distill extracted nothing for user %s", userID)
	}
	writeJSON(w, util.MapStr{"created": created, "status": "pending — confirm via /memory/:id/_confirm"})
}

// Distill runs the excerpt through the default language model and files
// every extracted memory as PENDING. The user's existing pending/confirmed
// memories join the prompt so the model can supersede changed state
// instead of creating duplicates.
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
		llms.TextParts(llms.ChatMessageTypeHuman, buildDistillInput(userID, excerpt)),
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
	// distill runs in the background — no session user, so the create
	// hooks need the write bypass to pass
	octx.Set(orm.DirectWriteWithoutPermissionCheck, true)
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
		if supersedes := strings.TrimSpace(item.Supersedes); supersedes != "" {
			// record the link only — the old record stays confirmed and
			// keeps injecting until the replacement is CONFIRMED; the
			// human gate owns every prompt-visible state change
			rec.SupersedeOf = validateSupersedeTarget(octx, userID, supersedes)
		}
		if err := orm.Create(octx, rec); err != nil {
			log.Warnf("memory: filing distilled record failed: %v", err)
			continue
		}
		created++
	}
	return created, nil
}

// buildDistillInput renders the model input: the exchange excerpt plus a
// compact inventory of the user's existing memories (id + kind + content)
// so the model can tell duplicates from updates.
func buildDistillInput(userID, excerpt string) string {
	existing := listActiveMemories(userID)
	if len(existing) == 0 {
		return excerpt
	}
	var sb strings.Builder
	sb.WriteString("EXISTING MEMORIES:\n")
	for _, rec := range existing {
		sb.WriteString(fmt.Sprintf("- %s [%s] %s\n", rec.ID, rec.Kind, rec.Content))
	}
	sb.WriteString("\nCONVERSATION EXCERPT:\n")
	sb.WriteString(excerpt)
	return sb.String()
}

// listActiveMemories loads the user's pending+confirmed records (the ones
// a new distillation can duplicate or supersede), most recently updated
// first, capped to keep the prompt bounded.
func listActiveMemories(userID string) []core.MemoryRecord {
	octx := orm.NewContextWithParent(context.Background())
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.MemoryRecord{})
	builder := orm.NewQuery().Size(50).
		Filter(
			orm.TermQuery("user_id", userID),
			orm.TermsQuery("status", []string{core.MemoryStatusPending, core.MemoryStatusConfirmed}),
		).
		SortBy(orm.Sort{Field: "updated", SortType: orm.DESC})
	res, err := orm.SearchV2(octx, builder)
	if err != nil {
		log.Warnf("memory: loading existing memories for distill failed: %v", err)
		return nil
	}
	recs, _, err := decodeRecords(res)
	if err != nil {
		log.Warnf("memory: decoding existing memories for distill failed: %v", err)
		return nil
	}
	return recs
}

// validateSupersedeTarget checks the referenced record exists, belongs to
// the user and is still active — the link is stored on the pending
// replacement; the actual status flip happens when the replacement is
// confirmed (supersedeTarget).
func validateSupersedeTarget(parent context.Context, userID, supersedesID string) string {
	if supersedesID == "" {
		return ""
	}
	octx := orm.NewContextWithParent(parent)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.MemoryRecord{})
	old := core.MemoryRecord{}
	old.ID = supersedesID
	exists, err := orm.GetV2(octx, &old)
	if err != nil || !exists {
		log.Debugf("memory: supersedes target %s not found", supersedesID)
		return ""
	}
	if old.UserID != userID {
		log.Warnf("memory: supersedes target %s belongs to another user, ignored", supersedesID)
		return ""
	}
	if old.Status != core.MemoryStatusPending && old.Status != core.MemoryStatusConfirmed {
		return ""
	}
	return supersedesID
}

// supersedeTarget flips the displaced record to superseded — called when
// its replacement is CONFIRMED. Best-effort: failures log only.
func supersedeTarget(parent context.Context, userID, id string) {
	octx := orm.NewContextWithParent(parent)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	octx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.MemoryRecord{})
	old := core.MemoryRecord{}
	old.ID = id
	exists, err := orm.GetV2(octx, &old)
	if err != nil || !exists || old.UserID != userID {
		return
	}
	if old.Status != core.MemoryStatusPending && old.Status != core.MemoryStatusConfirmed {
		return
	}
	old.Status = core.MemoryStatusSuperseded
	if err := orm.Update(octx, &old); err != nil {
		log.Warnf("memory: marking %s superseded failed: %v", id, err)
		return
	}
	log.Infof("memory: record %s superseded by %s (replacement confirmed)", id, userID)
}

type distillItem struct {
	Kind      string `json:"kind"`
	Content   string `json:"content"`
	Supersedes string `json:"supersedes,omitempty"`
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

// B1: during the cooldown window excerpts are NOT dropped — they
// accumulate in a per-user buffer and are distilled together when the
// window closes. The flush is timer-driven, so the last buffered turns
// survive even if the user stops chatting mid-window.

type distillBuffer struct {
	mu        sync.Mutex
	excerpts  []string
	sessionID string
	timer     *time.Timer
}

var distillBuffers sync.Map // userID → *distillBuffer

func AutoDistill(userID, excerpt, sessionID string) {
	if userID == "" || strings.TrimSpace(excerpt) == "" {
		return
	}
	bufI, _ := distillBuffers.LoadOrStore(userID, &distillBuffer{})
	buf := bufI.(*distillBuffer)
	buf.mu.Lock()
	if buf.timer != nil {
		// merge window already open: just accumulate (B1)
		buf.excerpts = append(buf.excerpts, excerpt)
		if sessionID != "" {
			buf.sessionID = sessionID
		}
		n := len(buf.excerpts)
		buf.mu.Unlock()
		log.Infof("memory: auto-distill cooldown active, buffered excerpt #%d for user %s (flushes with the window)", n, userID)
		return
	}
	// first turn of the window: open it; this excerpt goes into the
	// buffer too so the single flush covers the whole window
	buf.excerpts = append(buf.excerpts, excerpt)
	if sessionID != "" {
		buf.sessionID = sessionID
	}
	buf.timer = time.AfterFunc(distillCooldown, func() { flushDistillBuffer(userID) })
	buf.mu.Unlock()
	log.Infof("memory: auto-distill accepted excerpt for user %s, merge window open for %s", userID, distillCooldown)
}

func flushDistillBuffer(userID string) {
	bufI, loaded := distillBuffers.LoadAndDelete(userID)
	if !loaded {
		return
	}
	buf := bufI.(*distillBuffer)
	buf.mu.Lock()
	excerpts := buf.excerpts
	sessionID := buf.sessionID
	buf.timer = nil
	buf.mu.Unlock()
	if len(excerpts) == 0 {
		return
	}

	// merge: cap each turn, then drop oldest turns first if the combined
	// excerpt would exceed the model budget
	const perExcerptCap = 2000
	parts := make([]string, 0, len(excerpts))
	for _, e := range excerpts {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if len(e) > perExcerptCap {
			e = e[:perExcerptCap]
		}
		parts = append(parts, e)
	}
	for len(parts) > 1 && len(strings.Join(parts, "\n\n---\n\n")) > distillMaxLens {
		parts = parts[1:]
	}
	merged := strings.Join(parts, "\n\n---\n\n")

	// keep the manual-distill endpoint's rate limiter consistent with
	// the automatic flush
	distillMu.Store(userID, time.Now())

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Warnf("memory: auto-distill panicked: %v", r)
			}
		}()
		created, err := Distill(context.Background(), userID, merged, sessionID)
		if err != nil {
			log.Warnf("memory: auto-distill failed (%d buffered turn(s)): %v", len(excerpts), err)
			return
		}
		if created > 0 {
			log.Infof("memory: auto-distill produced %d pending record(s) from %d buffered turn(s) for user %s", created, len(excerpts), userID)
			NotifyPendingMemories(userID, created)
		} else {
			log.Infof("memory: auto-distill extracted nothing from %d buffered turn(s) for user %s", len(excerpts), userID)
		}
	}()
}

// distillCooldown is the per-user auto-distill merge window (B1: turns
// inside the window are buffered and distilled together, not dropped).
const distillCooldown = 5 * time.Minute
