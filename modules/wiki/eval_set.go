/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

// Per-KB validation set + replay (LLM-wiki P3, design §4.2 G4): the
// search-eval (D9) pattern moved down to one knowledge base — golden
// queries scored against the KB-scoped published-page retrieval that
// mirrors the chat knowledge tier's wiki leg (wikiRoute's direct-index
// branch: same boosts, same published/noise gate). Skills gate on the
// trend: applying a skill proposal in a KB that has cases fires an
// async replay whose scores land in the evolution log next to the skill
// it gated. Answer-level (LLM) scoring and automatic rollback on
// regression stay progressive per the design; these numbers measure
// retrieval, and say so.

const (
	// wikiEvalCaseCap bounds one run; validation sets are curated.
	wikiEvalCaseCap = 200
	// wikiEvalWindow is the scoring window per case.
	wikiEvalWindow = 10
	// wikiEvalTopK is the hit window that counts as correct.
	wikiEvalTopK = 4
)

/* ---------------- cases ---------------- */

// listWikiEvalCasesHandler serves GET /wiki/kb/:id/_evalset.
func (h *APIHandler) listWikiEvalCasesHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	cases, err := wikiEvalCases(req.Context(), kbID, wikiEvalCaseCap)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteJSON(w, util.MapStr{"total": len(cases), "data": cases}, http.StatusOK)
}

type wikiEvalCaseBody struct {
	Query          string   `json:"query"`
	ExpectedIDs    []string `json:"expected_ids"`
	ExpectedTitles []string `json:"expected_titles"`
	Note           string   `json:"note"`
}

// upsertWikiEvalCaseHandler serves POST /wiki/kb/:id/_evalset — upsert
// keyed on (kb, normalized query): re-annotating a question replaces its
// expectation instead of stacking a twin.
func (h *APIHandler) upsertWikiEvalCaseHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	body := wikiEvalCaseBody{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	query := normalizeEvalQuery(body.Query)
	if query == "" {
		h.WriteError(w, "query is required", http.StatusBadRequest)
		return
	}
	expected := normalizeEvalIDList(body.ExpectedIDs)
	if len(expected) == 0 {
		h.WriteError(w, "expected_ids is required (at least one wiki page the query must return)", http.StatusBadRequest)
		return
	}

	wctx := orm.NewContextWithParent(req.Context())
	wctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	wctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	// operator-paced read-after-write: "add a case, hit run" must see
	// the case in the run's snapshot
	wctx.Refresh = orm.WaitForRefresh
	orm.WithModel(wctx, &core.WikiEvalCase{})

	existing := wikiEvalCaseByQuery(req.Context(), kbID, query)
	entry := &core.WikiEvalCase{
		KbID:           kbID,
		Query:          query,
		ExpectedIDs:    expected,
		ExpectedTitles: normalizeEvalTitleList(body.ExpectedTitles, len(expected)),
		Note:           strings.TrimSpace(body.Note),
	}
	if existing != nil {
		entry.SetID(existing.ID)
		if err := orm.Update(wctx, entry); err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if err := orm.Create(wctx, entry); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteJSON(w, entry, http.StatusOK)
}

// deleteWikiEvalCaseHandler serves DELETE /wiki/kb/:id/_evalset/:cid.
func (h *APIHandler) deleteWikiEvalCaseHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	entry := &core.WikiEvalCase{}
	entry.SetID(ps.ByName("cid"))
	dctx := orm.NewContextWithParent(req.Context())
	dctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	dctx.Refresh = orm.WaitForRefresh
	orm.WithModel(dctx, &core.WikiEvalCase{})
	if err := orm.Delete(dctx, entry); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteJSON(w, util.MapStr{"deleted": entry.ID}, http.StatusOK)
}

/* ---------------- replay ---------------- */

// runWikiEvalHandler serves POST /wiki/kb/:id/_evalset/_run: every case
// replays through KB-scoped published-page retrieval and the run is
// scored and stored. Bounded by the curated set size, so synchronous is
// honest here — the operator asked for the number.
func (h *APIHandler) runWikiEvalHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	run, err := RunWikiEvalReplay(req.Context(), ps.ByName("id"), "manual")
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "empty") {
			status = http.StatusBadRequest
		}
		h.WriteError(w, err.Error(), status)
		return
	}
	h.WriteJSON(w, run, http.StatusOK)
}

// listWikiEvalRunsHandler serves GET /wiki/kb/:id/_evalset/_runs — the
// score trend, newest first.
func (h *APIHandler) listWikiEvalRunsHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	size := h.GetIntOrDefault(req, "size", 20)
	if size < 1 || size > 100 {
		size = 20
	}
	octx := orm.NewContextWithParent(req.Context())
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiEvalRun{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(size).
		Filter(orm.TermQuery("kb_id", ps.ByName("id"))).
		SortBy(orm.Sort{Field: "created", SortType: orm.DESC}))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	runs, _, derr := elastic.DecodeHits[core.WikiEvalRun](res)
	if derr != nil {
		h.WriteError(w, derr.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteJSON(w, util.MapStr{"total": len(runs), "data": runs}, http.StatusOK)
}

// RunWikiEvalReplay scores the KB's whole validation set against
// KB-scoped published-page retrieval and persists the run. Caller-visible
// because the skill gate tie-in reuses it.
func RunWikiEvalReplay(parent context.Context, kbID, trigger string) (*core.WikiEvalRun, error) {
	if kbID == "" {
		return nil, context.Canceled
	}
	cases, err := wikiEvalCases(parent, kbID, wikiEvalCaseCap)
	if err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, errEmptyEvalSet(kbID)
	}

	results := make([]core.WikiEvalCaseResult, 0, len(cases))
	for i := range cases {
		results = append(results, executeWikiEvalCase(parent, kbID, &cases[i]))
	}
	run := &core.WikiEvalRun{KbID: kbID, Trigger: trigger, Cases: results}
	run.TotalCases, run.Top4Hits, run.MRR, run.AvgTookMS = scoreWikiEvalRun(results)
	if run.TotalCases > 0 {
		run.Top4Rate = float64(run.Top4Hits) / float64(run.TotalCases)
	}

	wctx := orm.NewContextWithParent(parent)
	wctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(wctx, &core.WikiEvalRun{})
	if err := orm.Create(wctx, run); err != nil {
		log.Warnf("wiki: eval run persist failed for kb [%s]: %v", kbID, err)
	}
	return run, nil
}

type evalSetEmptyError struct{ kbID string }

func (e evalSetEmptyError) Error() string {
	return "validation set is empty for kb " + e.kbID + " — add cases first"
}

func errEmptyEvalSet(kbID string) error { return evalSetEmptyError{kbID: kbID} }

// executeWikiEvalCase replays one golden query through the KB-scoped
// published-page retrieval — the same query shape, boosts and noise gate
// as the chat knowledge tier's wiki leg (wikiRoute direct-index branch).
func executeWikiEvalCase(parent context.Context, kbID string, c *core.WikiEvalCase) core.WikiEvalCaseResult {
	started := time.Now()
	result := core.WikiEvalCaseResult{Query: c.Query}
	expected := map[string]bool{}
	for _, id := range c.ExpectedIDs {
		expected[id] = true
	}

	octx := orm.NewContextWithParent(parent)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiArticle{})
	builder := orm.NewQuery().From(0).Size(wikiEvalWindow)
	builder.Query(c.Query)
	builder.DefaultQueryField("title^20", "title.pinyin^8", "summary^4", "tags^2", "combined_fulltext")
	builder.Filter(
		orm.TermQuery("kb_id", kbID),
		orm.TermQuery("status", core.WikiArticlePublished),
		orm.MustNotQuery(orm.TermQuery("confidence", "low")),
	)
	builder.Exclude("content", "linked_pages")
	res, err := orm.SearchV2(octx, builder)
	if err != nil {
		result.Error = err.Error()
		result.TookMS = time.Since(started).Milliseconds()
		return result
	}
	articles, _, derr := elastic.DecodeHits[core.WikiArticle](res)
	if derr != nil {
		result.Error = derr.Error()
		result.TookMS = time.Since(started).Milliseconds()
		return result
	}
	result.Total = len(articles)
	for i := range articles {
		if i < wikiEvalTopK {
			result.TopIDs = append(result.TopIDs, articles[i].ID)
			result.TopTitles = append(result.TopTitles, articles[i].Title)
		}
		if expected[articles[i].ID] && result.HitRank == 0 {
			result.HitRank = i + 1
		}
	}
	result.TookMS = time.Since(started).Milliseconds()
	return result
}

// scoreWikiEvalRun folds case results into the run's headline numbers:
// top-K hit rate (a hit within the first wikiEvalTopK hits) and mean
// reciprocal rank over the scoring window.
func scoreWikiEvalRun(results []core.WikiEvalCaseResult) (total, topKHits int, mrr float64, avgTookMS int64) {
	if len(results) == 0 {
		return 0, 0, 0, 0
	}
	var tookTotal int64
	var rrSum float64
	for i := range results {
		tookTotal += results[i].TookMS
		if results[i].HitRank > 0 && results[i].HitRank <= wikiEvalTopK {
			topKHits++
		}
		if results[i].HitRank > 0 {
			rrSum += 1 / float64(results[i].HitRank)
		}
	}
	total = len(results)
	mrr = rrSum / float64(len(results))
	avgTookMS = tookTotal / int64(len(results))
	return total, topKHits, mrr, avgTookMS
}

/* ---------------- skill gate tie-in ---------------- */

// ScheduleSkillEvalReplay fires the post-apply replay for a KB whose
// validation set exists (design §4.2 G4: scores land in the evolution
// log next to the skill). Best-effort by contract — the apply already
// succeeded, the replay is bookkeeping; recover-guarded because it rides
// a bare goroutine.
func ScheduleSkillEvalReplay(kbID, actor string) {
	if kbID == "" {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Warnf("wiki: skill eval replay recovered: %v", r)
			}
		}()
		ctx := orm.NewContext()
		ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
		ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
		cases, err := wikiEvalCases(ctx, kbID, 1)
		if err != nil || len(cases) == 0 {
			return // progressive capability: no set, no replay
		}
		run, err := RunWikiEvalReplay(ctx, kbID, "skill_applied")
		if err != nil || run == nil {
			return
		}
		AppendEvolutionEntry(ctx, kbID, core.WikiEvolutionEvalRun, actor, "eval_run", run.ID,
			"验证集回放: hit@4 "+fmt.Sprintf("%.1f", run.Top4Rate*100)+"%, MRR "+fmt.Sprintf("%.3f", run.MRR),
			map[string]interface{}{
				"run_id":   run.ID,
				"hit_rate": run.Top4Rate,
				"mrr":      run.MRR,
				"cases":    run.TotalCases,
				"trigger":  "skill_applied",
			})
	}()
}

/* ---------------- helpers ---------------- */

func wikiEvalCases(parent context.Context, kbID string, size int) ([]core.WikiEvalCase, error) {
	if kbID == "" {
		return nil, evalSetEmptyError{kbID: kbID}
	}
	if size < 1 || size > wikiEvalCaseCap {
		size = wikiEvalCaseCap
	}
	octx := orm.NewContextWithParent(parent)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiEvalCase{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(size).
		Filter(orm.TermQuery("kb_id", kbID)).
		SortBy(orm.Sort{Field: "created", SortType: orm.DESC}))
	if err != nil {
		return nil, err
	}
	cases, _, derr := elastic.DecodeHits[core.WikiEvalCase](res)
	if derr != nil {
		return nil, derr
	}
	return cases, nil
}

func wikiEvalCaseByQuery(parent context.Context, kbID, query string) *core.WikiEvalCase {
	octx := orm.NewContextWithParent(parent)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiEvalCase{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("kb_id", kbID), orm.TermQuery("query", query)))
	if err != nil {
		return nil
	}
	existing, _, derr := elastic.DecodeHits[core.WikiEvalCase](res)
	if derr != nil || len(existing) == 0 {
		return nil
	}
	return &existing[0]
}

func normalizeEvalQuery(query string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(query))), " ")
}

func normalizeEvalIDList(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func normalizeEvalTitleList(titles []string, n int) []string {
	out := make([]string, 0, n)
	seen := map[string]bool{}
	for _, t := range titles {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
		if len(out) >= n {
			break
		}
	}
	return out
}
