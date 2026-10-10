/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	httprouter "infini.sh/framework/core/api/router"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Near-duplicate group confirmation (W12): exact hash groups fold in
// search results automatically; simhash/phash candidate groups only fold
// after the operator confirms them here — a wrong fold is worse than a
// visible duplicate. Confirmation persists keyed by the sorted member
// ids' hash, so re-scans produce the same key and survive scan churn.

// DedupGroupKey hashes the sorted member ids into the stable group key.
func DedupGroupKey(memberIDs []string) string {
	sorted := append([]string(nil), memberIDs...)
	sort.Strings(sorted)
	sum := sha256.Sum256([]byte(strings.Join(sorted, "|")))
	return hex.EncodeToString(sum[:16])
}

var (
	confirmedMu     sync.Mutex
	confirmedCache  map[string][]string // group_key → sorted member ids
	confirmedCached bool
	confirmedAt     time.Time
)

// invalidateConfirmedGroups drops the cache; called after every write.
func invalidateConfirmedGroups() {
	confirmedMu.Lock()
	confirmedCache = nil
	confirmedCached = false
	confirmedMu.Unlock()
}

// dedupGroupConfirmHandler persists a confirmed near-duplicate group:
// POST /document/dedup/confirm_group {"ids": [...], "tier": "near-duplicate"}
// Idempotent — re-confirming the same member set overwrites in place.
func (h *APIHandler) dedupGroupConfirmHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body := util.MapStr{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	ids := stringSliceOf(body["ids"])
	if len(ids) < 2 {
		h.WriteError(w, "ids (≥2) required", http.StatusBadRequest)
		return
	}
	tier, _ := body["tier"].(string)
	sort.Strings(ids)
	key := DedupGroupKey(ids)

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.DocumentDedupGroup{})

	rec := core.DocumentDedupGroup{}
	rec.ID = key
	rec.GroupKey = key
	rec.MemberIDs = ids
	rec.Tier = tier
	if err := orm.Update(ctx, &rec); err != nil {
		log.Warnf("dedup: group confirm failed for %d members: %v", len(ids), err)
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	invalidateConfirmedGroups()
	h.WriteOKJSON(w, util.MapStr{"group_key": key, "members": len(ids)})
}

// dedupGroupRemoveHandler deletes a confirmed group:
// POST /document/dedup/confirm_group/_remove {"group_key": "..."} — un-folds.
// (Path-param form would add a :key wildcard under /document/dedup/, which
// conflicts with /document/:doc_id's subtree in httprouter.)
func (h *APIHandler) dedupGroupRemoveHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body := util.MapStr{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	key, _ := body["group_key"].(string)
	if key == "" {
		h.WriteError(w, "group_key required", http.StatusBadRequest)
		return
	}
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.DocumentDedupGroup{})

	rec := core.DocumentDedupGroup{}
	rec.ID = key
	if err := orm.Delete(ctx, &rec); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	invalidateConfirmedGroups()
	h.WriteOKJSON(w, util.MapStr{"result": "deleted", "group_key": key})
}

// confirmedGroupsCached serves the search fold with a short-TTL copy of
// the confirmed groups; load failure yields nil (no folding by group —
// hash folding still runs, never a failed search).
func confirmedGroupsCached(ctx context.Context) map[string][]string {
	confirmedMu.Lock()
	defer confirmedMu.Unlock()
	if confirmedCached && confirmedCache != nil && time.Since(confirmedAt) < time.Minute {
		return confirmedCache
	}
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.DocumentDedupGroup{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(1000))
	if err != nil {
		return nil
	}
	recs, _, derr := elastic.DecodeHits[core.DocumentDedupGroup](res)
	if derr != nil {
		return nil
	}
	out := make(map[string][]string, len(recs))
	for _, r := range recs {
		out[r.GroupKey] = r.MemberIDs
	}
	confirmedCache = out
	confirmedCached = true
	confirmedAt = time.Now()
	return out
}

// confirmedGroupForDoc returns the confirmed group key whose members
// contain docID (used by the fold to attach group metadata to hits).
func confirmedGroupForDoc(groups map[string][]string, docID string) (string, []string, bool) {
	for key, members := range groups {
		for _, m := range members {
			if m == docID {
				return key, members, true
			}
		}
	}
	return "", nil, false
}

var _ = context.Background
