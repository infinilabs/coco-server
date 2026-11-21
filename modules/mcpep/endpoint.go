/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package mcpep

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

// MCP endpoint tokens (W7/P5): bearer tokens for external agents against
// the MCP server. The framework's RegisterMCPToolAuthorizer hook is the
// single authorization point — when a request carries
// `Authorization: Bearer mcp_...`, the token MUST resolve to an enabled
// endpoint and pass its rate limit; requests without a bearer keep the
// legacy console path (session auth, unchanged). The plaintext is shown
// exactly once — only the SHA-256 lands in the store.

const (
	Category        = "mcp"
	Resource        = "endpoint"
	tokenPrefix     = "mcp_"
	defaultRate     = 60
	lateUseDebounce = 30 * time.Second // last_used_at write cadence
)

func init() {
	orm.MustRegisterSchemaWithIndexName(core.MCPEndpoint{}, "mcp-endpoint")
	registerRoutes()
	api.RegisterMCPToolAuthorizer(authorize)
}

func registerRoutes() {
	handler := &APIHandler{}
	readPermission := security.GetSimplePermission(Category, Resource, string(security.Read))
	createPermission := security.GetSimplePermission(Category, Resource, string(security.Create))
	deletePermission := security.GetSimplePermission(Category, Resource, string(security.Delete))
	security.GetOrInitPermissionKeys(readPermission, createPermission, deletePermission)

	api.HandleUIMethod(api.GET, "/mcp_endpoint/_list", handler.list, api.RequireLogin(), api.RequirePermission(readPermission))
	api.HandleUIMethod(api.POST, "/mcp_endpoint/", handler.create, api.RequireLogin(), api.RequirePermission(createPermission))
	api.HandleUIMethod(api.POST, "/mcp_endpoint/:id/_rotate", handler.rotate, api.RequireLogin(), api.RequirePermission(createPermission))
	api.HandleUIMethod(api.PUT, "/mcp_endpoint/:id/_toggle", handler.toggle, api.RequireLogin(), api.RequirePermission(createPermission))
	api.HandleUIMethod(api.DELETE, "/mcp_endpoint/:id", handler.remove, api.RequireLogin(), api.RequirePermission(deletePermission))
}

type APIHandler struct {
	api.Handler
}

/* ---------------- token plumbing ---------------- */

func generateToken() (plaintext, hash string) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		// crypto-random failure must not produce a predictable token — the
		// caller surfaces the error instead of issuing anything
		return "", ""
	}
	plaintext = tokenPrefix + hex.EncodeToString(raw)
	return plaintext, hashToken(plaintext)
}

func hashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// endpointByToken resolves a bearer plaintext to its endpoint row. A
// broken store denies (fail-closed) — the MCP gate must never panic.
func endpointByToken(plaintext string) (row *core.MCPEndpoint) {
	defer func() {
		if r := recover(); r != nil {
			log.Warnf("mcp endpoint: token lookup failed (store unavailable): %v", r)
			row = nil
		}
	}()
	return endpointByTokenLookup(plaintext)
}

func endpointByTokenLookup(plaintext string) *core.MCPEndpoint {
	octx := orm.NewContext()
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.MCPEndpoint{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("token_hash", hashToken(plaintext))))
	if err != nil {
		return nil
	}
	row := decodeOne(res)
	return row
}

func decodeOne(res *orm.SearchResult) *core.MCPEndpoint {
	if res == nil {
		return nil
	}
	out := &struct {
		Hits struct {
			Hits []struct {
				Source core.MCPEndpoint `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}{}
	if raw, ok := res.Payload.([]byte); ok {
		if err := util.FromJSONBytes(raw, out); err != nil {
			return nil
		}
	}
	if len(out.Hits.Hits) == 0 {
		return nil
	}
	return &out.Hits.Hits[0].Source
}

/* ---------------- rate limiting (in-process sliding window) ---------------- */

type rateWindow struct {
	mu    sync.Mutex
	hits  []time.Time
	limit int
}

var rateWindows sync.Map // endpoint ID → *rateWindow

func allowAndTouch(ep *core.MCPEndpoint) bool {
	limit := ep.RateLimitPerMinute
	if limit <= 0 {
		limit = defaultRate
	}
	wAny, _ := rateWindows.LoadOrStore(ep.ID, &rateWindow{limit: limit})
	w := wAny.(*rateWindow)
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	kept := w.hits[:0]
	for _, t := range w.hits {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	w.hits = kept
	if len(w.hits) >= w.limit {
		return false
	}
	w.hits = append(w.hits, now)
	return true
}

// lastUsedFlusher batches last_used_at writes.
var (
	lastUsedMu   sync.Mutex
	lastUsedSeen = map[string]time.Time{}
)

func markUsed(ep *core.MCPEndpoint) {
	lastUsedMu.Lock()
	if t, ok := lastUsedSeen[ep.ID]; ok && time.Since(t) < lateUseDebounce {
		lastUsedMu.Unlock()
		return
	}
	lastUsedSeen[ep.ID] = time.Now()
	lastUsedMu.Unlock()

	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ep.LastUsedAt = time.Now().UnixMilli()
	if err := orm.Update(ctx, ep); err != nil {
		log.Tracef("mcp endpoint: last_used_at update failed for [%s]: %v", ep.ID, err)
	}
}

/* ---------------- the authorizer hook ---------------- */

// bearerFromHeaders extracts `Authorization: Bearer mcp_...` (empty when
// the request is not token-bearing).
func bearerFromHeaders(h http.Header) string {
	auth := strings.TrimSpace(h.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return ""
	}
	token := strings.TrimSpace(auth[len("bearer "):])
	if !strings.HasPrefix(token, tokenPrefix) {
		return "" // foreign bearer schemes pass through untouched
	}
	return token
}

// authorize is registered into the framework's MCP tool gate. Requests
// without an mcp_ bearer keep the legacy path (return true — the
// pre-existing permission model applies). Requests WITH one must resolve
// to an enabled endpoint under their rate limit.
func authorize(h http.Header, _ *api.HandlerOptions) bool {
	token := bearerFromHeaders(h)
	if token == "" {
		return true
	}
	ep := endpointByToken(token)
	if ep == nil || !ep.Enabled {
		return false
	}
	if !allowAndTouch(ep) {
		log.Warnf("mcp endpoint [%s]: rate limit exceeded, request denied", ep.Name)
		return false
	}
	go markUsed(ep)
	return true
}

/* ---------------- CRUD ---------------- */

func (h *APIHandler) create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	body := util.MapStr{}
	if err := h.DecodeJSON(r, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(str(body, "name"))
	if name == "" {
		h.WriteError(w, "name required", http.StatusBadRequest)
		return
	}
	plaintext, hash := generateToken()
	if plaintext == "" {
		h.WriteError(w, "token randomness unavailable, try again", http.StatusServiceUnavailable)
		return
	}
	ep := &core.MCPEndpoint{
		Name:               name,
		Enabled:            true,
		TokenHash:          hash,
		DatasourceIDs:      strSlice(body, "datasource_ids"),
		ToolGroups:         strSlice(body, "tool_groups"),
		RateLimitPerMinute: intOf(body, "rate_limit_per_minute"),
	}
	ctx := orm.NewContextWithParent(r.Context())
	ctx.Refresh = orm.WaitForRefresh
	if err := orm.Create(ctx, ep); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, util.MapStr{
		"_id":     ep.ID,
		"token":   plaintext,
		"note":    "store this token now — it is shown exactly once",
		"created": ep,
	})
}

func (h *APIHandler) rotate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ep := loadOne(w, r, ps)
	if ep == nil {
		return
	}
	plaintext, hash := generateToken()
	if plaintext == "" {
		h.WriteError(w, "token randomness unavailable, try again", http.StatusServiceUnavailable)
		return
	}
	ep.TokenHash = hash
	ctx := orm.NewContextWithParent(r.Context())
	ctx.Refresh = orm.WaitForRefresh
	if err := orm.Update(ctx, ep); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rateWindows.Delete(ep.ID) // old window dies with the old token
	h.WriteOKJSON(w, util.MapStr{"_id": ep.ID, "token": plaintext, "note": "old token is dead; store the new one now"})
}

func (h *APIHandler) toggle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ep := loadOne(w, r, ps)
	if ep == nil {
		return
	}
	ep.Enabled = !ep.Enabled
	ctx := orm.NewContextWithParent(r.Context())
	ctx.Refresh = orm.WaitForRefresh
	if err := orm.Update(ctx, ep); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, util.MapStr{"_id": ep.ID, "enabled": ep.Enabled})
}

func (h *APIHandler) remove(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ep := loadOne(w, r, ps)
	if ep == nil {
		return
	}
	ctx := orm.NewContextWithParent(r.Context())
	ctx.Refresh = orm.WaitForRefresh
	if err := orm.Delete(ctx, ep); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rateWindows.Delete(ep.ID)
	h.WriteOKJSON(w, util.MapStr{"_id": ep.ID, "result": "deleted"})
}

func (h *APIHandler) list(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := orm.NewContextWithParent(r.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.MCPEndpoint{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(100).
		SortBy(orm.Sort{Field: "created", SortType: orm.DESC}))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows := decodeList(res)
	// token hashes never leave the server
	for i := range rows {
		rows[i].TokenHash = ""
	}
	h.WriteOKJSON(w, util.MapStr{"items": rows, "total": len(rows)})
}

func loadOne(w http.ResponseWriter, r *http.Request, ps httprouter.Params) *core.MCPEndpoint {
	ctx := orm.NewContextWithParent(r.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.MCPEndpoint{})
	ep := core.MCPEndpoint{}
	ep.ID = ps.ByName("id")
	exists, err := orm.GetV2(ctx, &ep)
	if err != nil || !exists {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	return &ep
}

func decodeList(res *orm.SearchResult) []core.MCPEndpoint {
	out := &struct {
		Hits struct {
			Hits []struct {
				Source core.MCPEndpoint `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}{}
	if raw, ok := res.Payload.([]byte); ok {
		_ = util.FromJSONBytes(raw, out)
	}
	rows := make([]core.MCPEndpoint, 0, len(out.Hits.Hits))
	for _, h := range out.Hits.Hits {
		rows = append(rows, h.Source)
	}
	return rows
}

/* ---------------- misc ---------------- */

func str(m util.MapStr, k string) string {
	v, _ := m[k].(string)
	return v
}

func strSlice(m util.MapStr, k string) []string {
	raw, ok := m[k].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func intOf(m util.MapStr, k string) int {
	switch v := m[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}
