/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

// Dedup tiers, best (most certain) first. The tier drives the label the
// reviewer sees; nothing is ever deleted automatically — the scan only
// recommends, humans decide (same contract as D1's governance rule).
const (
	dedupTierExact   = 1 // identical normalized content
	dedupTierNear    = 2 // simhash hamming distance <= 3
	dedupTierSimilar = 3 // simhash hamming distance 4..8
	dedupTierVersion = 4 // same title, different content: suspected old/new versions
)

// dedupDocMeta is what the scan loads per document.
type dedupDocMeta struct {
	ID         string
	Title      string
	Content    string
	SourceID   string
	SourceName string
	Type       string
	Size       int
	Updated    int64
	Disabled   bool
}

// dedupMember is one document inside a reported group.
type dedupMember struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	SourceID   string `json:"source_id"`
	SourceName string `json:"source_name"`
	Type       string `json:"type,omitempty"`
	Size       int    `json:"size,omitempty"`
	Updated    int64  `json:"updated,omitempty"`
	Disabled   bool   `json:"disabled,omitempty"`
}

// dedupGroup is one connected cluster of suspected duplicates.
type dedupGroup struct {
	Key        string        `json:"key"`
	Tier       int           `json:"tier"`
	TierLabel  string        `json:"tier_label"`
	Similarity float64       `json:"similarity"`
	KeepID     string        `json:"keep_id"`
	Members    []dedupMember `json:"members"`
}

// dedupReport is the full scan result.
type dedupReport struct {
	Scanned       int          `json:"scanned"`
	Fingerprinted int          `json:"fingerprinted"`
	GroupCount    int          `json:"group_count"`
	Dismissed     int          `json:"dismissed_pairs"`
	GeneratedAt   int64        `json:"generated_at"`
	Groups        []dedupGroup `json:"groups"`
}

func dedupTierLabel(tier int) string {
	switch tier {
	case dedupTierExact:
		return "identical_content"
	case dedupTierNear:
		return "near_identical"
	case dedupTierSimilar:
		return "highly_similar"
	case dedupTierVersion:
		return "suspected_versions"
	}
	return "unknown"
}

// dedupGroupKey is the stable identity of a group: md5 of its sorted ids.
func dedupGroupKey(ids []string) string {
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	return util.MD5digest(strings.Join(sorted, "|"))
}

// dedupKeeperLess orders duplicate members best-keeper-first: newest update,
// then larger size, then stable id — shared by the scan and the cache patch.
func dedupKeeperLess(a, b dedupMember) bool {
	if a.Updated != b.Updated {
		return a.Updated > b.Updated
	}
	if a.Size != b.Size {
		return a.Size > b.Size
	}
	return a.ID < b.ID
}

// buildDedupReport is the pure grouping core: fingerprints every doc, builds
// duplicate edges tier by tier, merges them with union-find and picks the
// recommended keeper (newest update wins, then larger size, then stable id).
// Dismissed pair keys never become edges.
func buildDedupReport(docs []dedupDocMeta, dismissed map[string]bool) *dedupReport {
	report := &dedupReport{Groups: []dedupGroup{}, Dismissed: len(dismissed), GeneratedAt: time.Now().UnixMilli()}

	type fp struct {
		meta dedupDocMeta
		hash string
		sim  uint64
	}
	fps := make([]fp, 0, len(docs))
	byHash := map[string][]int{}
	for _, d := range docs {
		report.Scanned++
		text := d.Content
		if text == "" {
			text = d.Title
		}
		hash, sim, ok := contentFingerprint(text)
		if !ok {
			continue
		}
		report.Fingerprinted++
		idx := len(fps)
		fps = append(fps, fp{meta: d, hash: hash, sim: sim})
		byHash[hash] = append(byHash[hash], idx)
	}

	// union-find over document indexes; edges are collected first and
	// aggregated per FINAL root once all unions are done (a root changes as
	// groups merge, so per-edge bookkeeping keyed on roots mid-flight lies)
	parent := make([]int, len(fps))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	union := func(a, b int) { parent[find(a)] = find(b) }

	type dedupEdge struct {
		a, b, tier int
		sim        float64
	}
	edges := make([]dedupEdge, 0, 64)
	edge := func(a, b, tier int, sim float64) {
		if dismissed[dedupPairKey(fps[a].meta.ID, fps[b].meta.ID)] {
			return
		}
		edges = append(edges, dedupEdge{a: a, b: b, tier: tier, sim: sim})
	}

	// tier 1: exact hash groups
	for _, idxs := range byHash {
		if len(idxs) < 2 {
			continue
		}
		for i := 1; i < len(idxs); i++ {
			edge(idxs[0], idxs[i], dedupTierExact, 1)
		}
	}

	// tier 2/3: pairwise simhash hamming. Block-bucket nominating cannot
	// cover radius > 3 (the pigeonhole only holds for one clean block among
	// four), and the scan is capped at a few tens of thousands of docs — a
	// direct O(n^2) pass is a few hundred million cheap ops at the cap and
	// misses nothing. Simhash only nominates; the exact bigram overlap
	// decides — short unrelated texts routinely land within hamming 8 and
	// must not become edges.
	const simhashMaxNear, simhashMaxSimilar = 3, 8
	for a := 0; a < len(fps); a++ {
		for c := a + 1; c < len(fps); c++ {
			if fps[a].hash == fps[c].hash {
				continue // already tier 1
			}
			dist := hammingDistance64(fps[a].sim, fps[c].sim)
			if dist > simhashMaxSimilar {
				continue
			}
			jac := bigramJaccard(fps[a].meta.Content, fps[c].meta.Content)
			if jac < 0.5 {
				continue
			}
			if dist <= simhashMaxNear {
				edge(a, c, dedupTierNear, jac)
			} else {
				edge(a, c, dedupTierSimilar, jac)
			}
		}
	}

	// tier 4: same normalized title, different content — suspected versions
	byTitle := map[string][]int{}
	for i, f := range fps {
		title := normalizeFingerprintText(f.meta.Title)
		if len([]rune(title)) < 6 {
			continue
		}
		byTitle[title] = append(byTitle[title], i)
	}
	for _, idxs := range byTitle {
		if len(idxs) < 2 {
			continue
		}
		for i := 1; i < len(idxs); i++ {
			edge(idxs[0], idxs[i], dedupTierVersion, 0.5)
		}
	}

	// union strong edges (tiers 1-3, content-based) first; a tier-4
	// same-title edge is a weak signal (sim 0.5) and must never chain
	// documents into a group whose members are already duplicates by
	// content. A mixed group wears its best edge's tier/similarity — two
	// identical pairs of different data sharing a filename would read as
	// "identical_content 100%" (the products.json case). Weak edges only
	// gather documents that no strong edge claimed.
	var strongEdges, weakEdges []dedupEdge
	for _, e := range edges {
		if e.tier == dedupTierVersion {
			weakEdges = append(weakEdges, e)
		} else {
			strongEdges = append(strongEdges, e)
		}
	}
	for _, e := range strongEdges {
		union(e.a, e.b)
	}
	strongRoot := make(map[int]bool, len(strongEdges))
	for _, e := range strongEdges {
		strongRoot[find(e.a)] = true
	}
	joined := strongEdges
	for _, e := range weakEdges {
		if strongRoot[find(e.a)] || strongRoot[find(e.b)] {
			continue
		}
		union(e.a, e.b)
		joined = append(joined, e)
	}

	// aggregate edges and members per final root (roots no longer change)
	edgeTier := map[int]int{}
	edgeSim := map[int]float64{}
	for _, e := range joined {
		root := find(e.a)
		if t, ok := edgeTier[root]; !ok || e.tier < t {
			edgeTier[root] = e.tier
		}
		if e.sim > edgeSim[root] {
			edgeSim[root] = e.sim
		}
	}
	groupDocs := map[int][]dedupDocMeta{}
	for i, f := range fps {
		root := find(i)
		if _, has := edgeTier[root]; has {
			groupDocs[root] = append(groupDocs[root], f.meta)
		}
	}
	for root, metas := range groupDocs {
		if len(metas) < 2 {
			continue
		}
		g := dedupGroup{
			Tier:       edgeTier[root],
			TierLabel:  dedupTierLabel(edgeTier[root]),
			Similarity: edgeSim[root],
			Members:    make([]dedupMember, 0, len(metas)),
		}
		ids := make([]string, 0, len(metas))
		for _, m := range metas {
			g.Members = append(g.Members, toDedupMember(m))
			ids = append(ids, m.ID)
		}
		sort.Slice(g.Members, func(a, b int) bool { return dedupKeeperLess(g.Members[a], g.Members[b]) })
		g.KeepID = g.Members[0].ID
		g.Key = dedupGroupKey(ids)
		report.Groups = append(report.Groups, g)
	}
	sort.Slice(report.Groups, func(a, b int) bool {
		if report.Groups[a].Tier != report.Groups[b].Tier {
			return report.Groups[a].Tier < report.Groups[b].Tier
		}
		return len(report.Groups[a].Members) > len(report.Groups[b].Members)
	})
	report.GroupCount = len(report.Groups)
	return report
}

func toDedupMember(m dedupDocMeta) dedupMember {
	return dedupMember{
		ID: m.ID, Title: m.Title, SourceID: m.SourceID, SourceName: m.SourceName,
		Type: m.Type, Size: m.Size, Updated: m.Updated, Disabled: m.Disabled,
	}
}

// ---- scan runner (ES reads) ----

const (
	dedupScanPageSize   = 500
	dedupScanDefaultMax = 5000
	dedupScanTimeout    = 10 * time.Minute
	dedupReportTTL      = time.Minute
	// one-shot load bound for the dismissal set; query-time filtering is
	// the scaled fix, this cap keeps the in-memory map bounded until then
	dedupDismissalCap = 10000
)

// dedupReportResponse is what the report endpoint returns: the cached report
// (possibly stale, served while a refresh scans in the background) overlaid
// with the live scan progress so the UI renders a progress bar, not a spinner.
type dedupReportResponse struct {
	Scanned       int          `json:"scanned"`
	Fingerprinted int          `json:"fingerprinted"`
	GroupCount    int          `json:"group_count"`
	Dismissed     int          `json:"dismissed_pairs"`
	GeneratedAt   int64        `json:"generated_at"`
	Groups        []dedupGroup `json:"groups"`
	Scanning      bool         `json:"scanning"`
	Stage         string       `json:"stage,omitempty"` // fetch | group
	ProgressDone  int          `json:"progress_scanned,omitempty"`
	ProgressTotal int          `json:"total,omitempty"`
	Error         string       `json:"error,omitempty"`
}

// dedupMu guards the report cache and the background-scan lifecycle together:
// a scan writes the cache at its end, report reads consult both.
var (
	dedupMu          sync.Mutex
	dedupCache       *dedupReport
	dedupCacheAt     time.Time
	dedupScanRunning bool
	dedupScanStage   string // fetch | group | ""
	dedupScanDone    int
	dedupScanTotal   int
	dedupScanErr     string

	// dismissal-set cache for retrieval-time folding (W12): the fold runs
	// on every search, loading the dismissals per query would be absurd;
	// 60s staleness is fine — a dismissal takes at most a minute to stop
	// folding its pair
	dismissedMu    sync.Mutex
	dismissedCache map[string]bool
	dismissedAt    time.Time
)

// dismissedPairsCached serves the fold with a short-TTL copy of the
// dismissal set; load failures yield nil (fold by hash alone — the
// pre-W12 behavior, never a failed search).
func dismissedPairsCached(ctx context.Context) map[string]bool {
	dismissedMu.Lock()
	defer dismissedMu.Unlock()
	if dismissedCache != nil && time.Since(dismissedAt) < time.Minute {
		return dismissedCache
	}
	set, err := loadDedupDismissals(ctx)
	if err != nil {
		return nil
	}
	dismissedCache = set
	dismissedAt = time.Now()
	return set
}

// loadDedupDismissals returns the dismissed pair keys. Sorted newest-first
// so the cap, when it bites, drops the stalest dismissals; at the cap it
// warns — a silently-truncated set re-recommends dismissed pairs, which
// reads like a dedup bug but is a load limit.
func loadDedupDismissals(ctx context.Context) (map[string]bool, error) {
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.DocumentDedupDismissal{})
	var items []core.DocumentDedupDismissal
	builder := orm.NewQuery().Size(dedupDismissalCap).
		SortBy(orm.Sort{Field: "created", SortType: orm.DESC})
	err, _ := elastic.SearchV2WithResultItemMapper(octx, &items, builder, nil)
	if err != nil {
		return nil, err
	}
	if len(items) >= dedupDismissalCap {
		log.Warnf("dedup: dismissal set hit the %d-row load cap — oldest dismissals fell out, dismissed pairs may reappear in the report", dedupDismissalCap)
	}
	out := make(map[string]bool, len(items))
	for _, item := range items {
		out[item.PairKey] = true
	}
	return out, nil
}

// scanDocuments walks the document index and returns the metadata the pure
// grouping core needs. Only the fields fingerprints require are fetched.
// onPage reports (fetched, total) after every page so a background scan can
// publish progress; total comes from the first page's hit count, capped at max.
func scanDocuments(ctx context.Context, max int, onPage func(fetched, total int)) ([]dedupDocMeta, error) {
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})

	docs := make([]dedupDocMeta, 0, max)
	total := 0
	for from := 0; from < max; from += dedupScanPageSize {
		size := dedupScanPageSize
		if from+size > max {
			size = max - from
		}
		builder := orm.NewQuery().From(from).Size(size)
		builder.SortBy(orm.Sort{Field: "updated", SortType: orm.DESC})
		builder.Include("id", "title", "content", "source.id", "source.name", "type", "size", "updated", "disabled")
		var page []core.Document
		err, res := elastic.SearchV2WithResultItemMapper(octx, &page, builder, nil)
		if err != nil {
			return nil, err
		}
		if res != nil && res.Total > 0 {
			total = int(res.Total)
			if total > max {
				total = max
			}
		}
		for _, d := range page {
			docs = append(docs, dedupDocMeta{
				ID: d.ID, Title: d.Title, Content: d.Content,
				SourceID: d.Source.ID, SourceName: d.Source.Name,
				Type: d.Type, Size: d.Size,
				Updated: updatedAtMillis(d.Updated), Disabled: d.Disabled,
			})
		}
		if onPage != nil {
			onPage(len(docs), total)
		}
		if len(page) < size {
			break
		}
	}
	return docs, nil
}

func updatedAtMillis(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixMilli()
}

// startDedupScan launches the scan in the background (single-flight: an
// already-running scan wins) and returns immediately — the HTTP request
// never blocks on it; clients poll the report endpoint for progress. The
// goroutine owns its context: the triggering request may be long gone.
func startDedupScan(max int) {
	dedupMu.Lock()
	if dedupScanRunning {
		dedupMu.Unlock()
		return
	}
	dedupScanRunning = true
	dedupScanStage = "fetch"
	dedupScanDone = 0
	dedupScanTotal = 0
	dedupScanErr = ""
	dedupMu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), dedupScanTimeout)
		defer cancel()

		report, err := runDedupScanTracked(ctx, max)

		dedupMu.Lock()
		dedupScanRunning = false
		dedupScanStage = ""
		if err != nil {
			dedupScanErr = err.Error()
			log.Warnf("dedup: background scan failed: %v", err)
		} else {
			dedupCache = report
			dedupCacheAt = time.Now()
			dedupScanErr = ""
		}
		dedupMu.Unlock()
	}()
}

// runDedupScanTracked executes a full scan, publishing stage/progress into
// the scan state as it goes, and returns the report for the caller to cache.
func runDedupScanTracked(ctx context.Context, max int) (*dedupReport, error) {
	if max <= 0 {
		max = dedupScanDefaultMax
	}
	dismissed, err := loadDedupDismissals(ctx)
	if err != nil {
		log.Warnf("dedup: load dismissals failed: %v", err)
		dismissed = map[string]bool{}
	}
	onPage := func(fetched, total int) {
		dedupMu.Lock()
		dedupScanDone = fetched
		if total > dedupScanTotal {
			dedupScanTotal = total
		}
		dedupMu.Unlock()
	}
	docs, err := scanDocuments(ctx, max, onPage)
	if err != nil {
		return nil, err
	}
	dedupMu.Lock()
	dedupScanStage = "group"
	dedupMu.Unlock()
	return buildDedupReport(docs, dismissed), nil
}

// patchDedupCache applies a review action to the cached report in place —
// the UI sees the effect on the very next read, no full re-scan. The TTL
// resets because the patch is the fresh truth; the next natural re-scan
// reconciles anything the surgical patch cannot express.
func patchDedupCache(patch func(r *dedupReport)) {
	dedupMu.Lock()
	defer dedupMu.Unlock()
	if dedupCache == nil {
		return
	}
	patch(dedupCache)
	dedupCacheAt = time.Now()
}

// reelectDedupKeeper re-picks the recommended keeper of a group after a
// member was removed — same rule as the scan: newest update, then larger
// size, then stable id.
func reelectDedupKeeper(g *dedupGroup) {
	if len(g.Members) == 0 {
		return
	}
	sort.Slice(g.Members, func(a, b int) bool { return dedupKeeperLess(g.Members[a], g.Members[b]) })
	g.KeepID = g.Members[0].ID
}

// patchDedupCacheOnAction folds exclude/delete decisions into the cached
// report: deleted members leave their groups (a group under two members
// dissolves), excluded members stay but read as disabled.
func patchDedupCacheOnAction(ids []string, action string) {
	gone := make(map[string]bool, len(ids))
	for _, id := range ids {
		gone[id] = true
	}
	patchDedupCache(func(r *dedupReport) {
		for gi := range r.Groups {
			g := &r.Groups[gi]
			kept := g.Members[:0]
			for _, m := range g.Members {
				if !gone[m.ID] {
					kept = append(kept, m)
					continue
				}
				if action == "delete" {
					continue
				}
				m.Disabled = true
				kept = append(kept, m)
			}
			g.Members = kept
			if !memberIn(g, g.KeepID) {
				reelectDedupKeeper(g)
			}
		}
		groups := r.Groups[:0]
		for _, g := range r.Groups {
			if len(g.Members) >= 2 {
				groups = append(groups, g)
			}
		}
		r.Groups = groups
		r.GroupCount = len(groups)
	})
}

func memberIn(g *dedupGroup, id string) bool {
	for _, m := range g.Members {
		if m.ID == id {
			return true
		}
	}
	return false
}

// patchDedupCacheOnDismiss drops the dismissed group from the cached report
// (the UI dismisses whole groups — every pair among the ids is now dismissed,
// so the group has no edges left).
func patchDedupCacheOnDismiss(ids []string, pairs int) {
	key := dedupGroupKey(ids)
	patchDedupCache(func(r *dedupReport) {
		groups := r.Groups[:0]
		for _, g := range r.Groups {
			if g.Key != key {
				groups = append(groups, g)
			}
		}
		r.Groups = groups
		r.GroupCount = len(groups)
		r.Dismissed += pairs
	})
}

// ---- review API ----

// dedupReportHandler serves the cached report and kicks off a background
// scan when none is fresh — stale results are served while it runs
// (stale-while-revalidate), progress rides along in the response. The
// system only reports; every removal happens through an explicit human action.
func (h *APIHandler) dedupReportHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	refresh := h.GetParameterOrDefault(req, "refresh", "") != ""
	max := h.GetIntOrDefault(req, "max", dedupScanDefaultMax)
	if max <= 0 || max > 50000 {
		max = dedupScanDefaultMax
	}

	dedupMu.Lock()
	fresh := dedupCache != nil && time.Since(dedupCacheAt) < dedupReportTTL
	running := dedupScanRunning
	cache := dedupCache
	dedupMu.Unlock()
	if refresh || (!fresh && !running) {
		startDedupScan(max)
	}

	resp := dedupReportResponse{Groups: []dedupGroup{}}
	if cache != nil {
		resp.Scanned = cache.Scanned
		resp.Fingerprinted = cache.Fingerprinted
		resp.GroupCount = cache.GroupCount
		resp.Dismissed = cache.Dismissed
		resp.GeneratedAt = cache.GeneratedAt
		resp.Groups = cache.Groups
		if resp.Groups == nil {
			resp.Groups = []dedupGroup{}
		}
	}
	dedupMu.Lock()
	resp.Scanning = dedupScanRunning
	resp.Stage = dedupScanStage
	resp.ProgressDone = dedupScanDone
	resp.ProgressTotal = dedupScanTotal
	resp.Error = dedupScanErr
	dedupMu.Unlock()
	h.WriteJSON(w, resp, http.StatusOK)
}

type dedupDismissBody struct {
	// IDs is the pair (or the whole group) being marked as NOT duplicates.
	IDs    []string `json:"ids"`
	Reason string   `json:"reason,omitempty"`
}

// dedupDismissHandler persists "not duplicates" verdicts: every pair among
// the submitted ids is dismissed and never reported again.
func (h *APIHandler) dedupDismissHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body := dedupDismissBody{}
	if err := h.DecodeJSON(req, &body); err != nil || len(body.IDs) < 2 {
		h.WriteError(w, "ids (at least two) is required", http.StatusBadRequest)
		return
	}
	sort.Strings(body.IDs)

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh
	created := 0
	for i := 0; i < len(body.IDs); i++ {
		for j := i + 1; j < len(body.IDs); j++ {
			dismissal := &core.DocumentDedupDismissal{
				PairKey: dedupPairKey(body.IDs[i], body.IDs[j]),
				Reason:  body.Reason,
			}
			dismissal.ID = util.GetUUID()
			if err := orm.Create(ctx, dismissal); err != nil {
				log.Warnf("dedup: dismiss %s failed: %v", dismissal.PairKey, err)
				continue
			}
			created++
		}
	}
	invalidate := false
	if created > 0 {
		patchDedupCacheOnDismiss(body.IDs, created)
		invalidate = true
	}
	if invalidate {
		// keep the retrieval-time fold in sync too
		dismissedMu.Lock()
		dismissedCache = nil
		dismissedMu.Unlock()
	}
	h.WriteJSON(w, util.MapStr{"dismissed": created}, http.StatusOK)
}

type dedupActionBody struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"` // exclude | delete
}

// dedupActionHandler applies the reviewer's decision to whole documents:
// "exclude" flips the disabled flag (kept, out of retrieval — reversible),
// "delete" removes them (checked against the delete permission separately).
func (h *APIHandler) dedupActionHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body := dedupActionBody{}
	if err := h.DecodeJSON(req, &body); err != nil || len(body.IDs) == 0 {
		h.WriteError(w, "ids is required", http.StatusBadRequest)
		return
	}

	if body.Action == "delete" {
		deletePermission := security.GetSimplePermission(Category, Resource, string(security.Delete))
		reqUser, err := security.GetUserFromRequest(req)
		if err != nil || reqUser == nil || !reqUser.UserAssignedPermission.ValidateFor(security.GetOrInitPermissionKey(deletePermission)) {
			h.WriteError(w, "delete requires document delete permission", http.StatusForbidden)
			return
		}
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	// the reviewer acts with route-level document:update (delete is checked
	// again above); the orm-level owner/sharing check would reject exactly
	// the cross-datasource documents a dedup cleanup is meant to reach
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh

	acted, failed := 0, 0
	for _, id := range body.IDs {
		doc := core.Document{}
		doc.ID = id
		exists, err := orm.GetV2(ctx, &doc)
		if err != nil || !exists {
			failed++
			continue
		}
		if body.Action == "delete" {
			if err := orm.Delete(ctx, &doc); err != nil {
				failed++
				continue
			}
			acted++
			continue
		}
		// exclude: keep the document, drop it out of retrieval
		doc.Disabled = true
		if err := orm.Save(ctx, &doc); err != nil {
			failed++
			continue
		}
		acted++
	}
	// patch the cached report so the UI reflects the decision instantly;
	// the fold cache follows for excluded docs (their fingerprints fold
	// differently once disabled)
	patchDedupCacheOnAction(body.IDs, body.Action)
	h.WriteJSON(w, util.MapStr{"action": body.Action, "acted": acted, "failed": failed}, http.StatusOK)
}
