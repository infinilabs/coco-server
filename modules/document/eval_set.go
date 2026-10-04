/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

// Golden-query evaluation set (D9): a human-annotated set of query →
// expected-document pairs, scored against the live search pipeline (the
// same queryWithRRF production runs, rerank included). Top-4 hit rate and
// MRR per run; every run is stored so tuning shows its trend. The rule the
// set enforces: no recall-stack change ships without a before/after run.
// Cases with expected documents but zero recall also file a governance
// proposal — retrieval that cannot find documents known to exist is a gap
// worth queueing (D5), recommendation only.

const (
	// evalCaseCap bounds one run; golden sets are curated, not harvested.
	evalCaseCap = 200
	// evalDefaultTopN is how many fused hits each case scores over.
	evalDefaultTopN = 10
	// evalTopK is the hit window that counts as a correct answer.
	evalTopK = 4
)

// searchEvalCases serves GET /search/studio/eval/_cases.
func (h *APIHandler) searchEvalCases(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	size := h.GetIntOrDefault(req, "size", 200)
	if size < 1 || size > 500 {
		size = 200
	}
	octx := orm.NewContextWithParent(req.Context())
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.SearchEvalCase{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(size).
		SortBy(orm.Sort{Field: "created", SortType: orm.DESC}))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cases, _, err := elastic.DecodeHits[core.SearchEvalCase](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteJSON(w, util.MapStr{"total": len(cases), "data": cases}, http.StatusOK)
}

type searchEvalCaseBody struct {
	Query          string   `json:"query"`
	ExpectedIDs    []string `json:"expected_ids"`
	ExpectedTitles []string `json:"expected_titles"`
	Datasource     string   `json:"datasource"`
	Note           string   `json:"note"`
}

// createSearchEvalCase serves POST /search/studio/eval/_cases — an upsert
// keyed on the normalized query: re-annotating a question replaces its
// expectation instead of duplicating the case.
func (h *APIHandler) createSearchEvalCase(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := searchEvalCaseBody{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	query := util.CleanUserQuery(body.Query)
	if query == "" {
		h.WriteError(w, "query is required", http.StatusBadRequest)
		return
	}
	expected := normalizeIDList(body.ExpectedIDs)
	if len(expected) == 0 {
		h.WriteError(w, "expected_ids is required (at least one document the query must return)", http.StatusBadRequest)
		return
	}

	wctx := orm.NewContextWithParent(req.Context())
	wctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	wctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(wctx, &core.SearchEvalCase{})

	normalized := normalizeSearchQuery(query)
	rctx := orm.NewContextWithParent(req.Context())
	rctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(rctx, &core.SearchEvalCase{})
	res, err := orm.SearchV2(rctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("query", normalized)))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	existing, _, derr := elastic.DecodeHits[core.SearchEvalCase](res)
	if derr != nil {
		h.WriteError(w, derr.Error(), http.StatusInternalServerError)
		return
	}

	entry := &core.SearchEvalCase{
		Query:          normalized,
		ExpectedIDs:    expected,
		ExpectedTitles: normalizeTitleList(body.ExpectedTitles, len(expected)),
		Datasource:     strings.TrimSpace(body.Datasource),
		Note:           strings.TrimSpace(body.Note),
	}
	if len(existing) > 0 {
		entry.SetID(existing[0].ID)
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

// deleteSearchEvalCase serves DELETE /search/studio/eval/_cases/:id.
func (h *APIHandler) deleteSearchEvalCase(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	dctx := orm.NewContextWithParent(req.Context())
	dctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(dctx, &core.SearchEvalCase{})
	entry := &core.SearchEvalCase{}
	entry.SetID(id)
	if err := orm.Delete(dctx, entry); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteJSON(w, util.MapStr{"deleted": id}, http.StatusOK)
}

// listSearchEvalRuns serves GET /search/studio/eval/_runs — the score trend.
func (h *APIHandler) listSearchEvalRuns(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	size := h.GetIntOrDefault(req, "size", 50)
	if size < 1 || size > 100 {
		size = 50
	}
	octx := orm.NewContextWithParent(req.Context())
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.SearchEvalRun{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(size).
		SortBy(orm.Sort{Field: "created", SortType: orm.DESC}))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	runs, _, err := elastic.DecodeHits[core.SearchEvalRun](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteJSON(w, util.MapStr{"total": len(runs), "data": runs}, http.StatusOK)
}

type searchEvalRunBody struct {
	// TopN bounds the fused window each case scores over (default 10).
	TopN int `json:"top_n"`
}

// runSearchEval serves POST /search/studio/eval/_run: every case runs
// through the production fused pipeline (permissions included) and the run
// is scored and stored. Long by design — a run is an explicit operator
// action, not a request-path side effect.
func (h *APIHandler) runSearchEval(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := searchEvalRunBody{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	topN := body.TopN
	if topN < 1 || topN > maxRRFWindow {
		topN = evalDefaultTopN
	}

	octx := orm.NewContextWithParent(req.Context())
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.SearchEvalCase{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(evalCaseCap).
		SortBy(orm.Sort{Field: "created", SortType: orm.ASC}))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cases, _, err := elastic.DecodeHits[core.SearchEvalCase](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(cases) == 0 {
		h.WriteError(w, "evaluation set is empty — add cases first", http.StatusBadRequest)
		return
	}

	started := time.Now()
	results := make([]core.SearchEvalCaseResult, 0, len(cases))
	for i := range cases {
		results = append(results, h.executeEvalCase(req, &cases[i], topN))
	}

	run := &core.SearchEvalRun{Cases: results}
	run.TotalCases, run.Top4Hits, run.MRR, run.AvgTookMS = scoreEvalRun(results)
	if run.TotalCases > 0 {
		run.Top4Rate = float64(run.Top4Hits) / float64(run.TotalCases)
	}

	wctx := orm.NewContextWithParent(req.Context())
	wctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(wctx, &core.SearchEvalRun{})
	if err := orm.Create(wctx, run); err != nil {
		log.Warnf("eval run persist failed: %v", err)
	}

	// zero-recall cases feed the gap queue (D5) — the set asserts these
	// documents exist, retrieval just cannot find them
	for i := range results {
		if results[i].Total == 0 && results[i].Error == "" {
			fileEvalKnowledgeGap(req.Context(), &cases[i])
		}
	}

	h.WriteJSON(w, util.MapStr{
		"run":       run,
		"took_ms":   time.Since(started).Milliseconds(),
		"top4_rate": run.Top4Rate,
		"mrr":       run.MRR,
	}, http.StatusOK)
}

// executeEvalCase runs one golden query through the production fused
// pipeline. The synthetic GET request keeps the caller's headers
// (permissions travel with them) while pinning from/size to the scoring
// window; the case's datasource scope rides the explicit parameter.
func (h *APIHandler) executeEvalCase(req *http.Request, c *core.SearchEvalCase, topN int) core.SearchEvalCaseResult {
	started := time.Now()
	result := core.SearchEvalCaseResult{Query: c.Query}

	expected := map[string]bool{}
	for _, id := range c.ExpectedIDs {
		expected[id] = true
	}

	evalReq := req.Clone(req.Context())
	evalReq.Method = http.MethodGet
	evalReq.Body = http.NoBody
	u := *req.URL
	params := url.Values{}
	params.Set("from", "0")
	params.Set("size", strconv.Itoa(topN))
	u.RawQuery = params.Encode()
	evalReq.URL = &u

	resp, _, _, err := h.queryWithRRF(evalReq, c.Query, c.Datasource, "", "", "", "", 3)
	result.TookMS = time.Since(started).Milliseconds()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Total = resp.GetTotal()
	for i := range resp.Hits.Hits {
		hit := &resp.Hits.Hits[i]
		if i < evalTopK {
			result.TopIDs = append(result.TopIDs, hit.ID)
			result.TopTitles = append(result.TopTitles, hit.Source.Title)
		}
		if expected[hit.ID] && result.HitRank == 0 {
			result.HitRank = i + 1
		}
	}
	return result
}

// scoreEvalRun folds case results into the run's headline numbers:
// top-K hit rate (a hit within the first evalTopK fused hits) and mean
// reciprocal rank over the scoring window.
func scoreEvalRun(results []core.SearchEvalCaseResult) (total, top4Hits int64, mrr float64, avgTookMS int64) {
	if len(results) == 0 {
		return 0, 0, 0, 0
	}
	var tookTotal int64
	var rrSum float64
	for i := range results {
		tookTotal += results[i].TookMS
		if results[i].HitRank > 0 && results[i].HitRank <= evalTopK {
			top4Hits++
		}
		if results[i].HitRank > 0 {
			rrSum += 1 / float64(results[i].HitRank)
		}
	}
	total = int64(len(results))
	mrr = rrSum / float64(len(results))
	avgTookMS = tookTotal / int64(len(results))
	return total, top4Hits, mrr, avgTookMS
}

// fileEvalKnowledgeGap files (or refreshes) a knowledge-gap proposal for a
// golden query whose recall came back empty. Same idempotency anchor as the
// zero-hit loop (ArticleID = query hash), with evidence naming the eval set
// as the source — a reviewer sees the query, why it matters and where it
// came from. Recommendation only, never an edit.
func fileEvalKnowledgeGap(ctx context.Context, c *core.SearchEvalCase) {
	query := normalizeSearchQuery(c.Query)
	gapID := searchQueryHash(query)

	wctx := orm.NewContextWithParent(ctx)
	wctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	wctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(wctx, &core.WikiGovernanceProposal{})

	rctx := orm.NewContextWithParent(ctx)
	rctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(rctx, &core.WikiGovernanceProposal{})
	res, err := orm.SearchV2(rctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("article_id", gapID), orm.TermQuery("type", core.WikiGovernanceKnowledgeGap)))
	if err == nil {
		existing, _, derr := elastic.DecodeHits[core.WikiGovernanceProposal](res)
		if derr == nil && len(existing) > 0 {
			if existing[0].Status == core.WikiGovernanceOpen {
				existing[0].Evidence = evalGapEvidence(query, c)
				_ = orm.Update(wctx, &existing[0])
			}
			return
		}
	}

	proposal := &core.WikiGovernanceProposal{
		ArticleID:    gapID,
		ArticleTitle: query,
		Type:         core.WikiGovernanceKnowledgeGap,
		Status:       core.WikiGovernanceOpen,
		Reason:       "golden query returns zero hits — the eval set asserts these documents exist, recall cannot find them",
		Evidence:     evalGapEvidence(query, c),
	}
	if err := orm.Create(wctx, proposal); err != nil {
		log.Warnf("eval knowledge gap proposal create failed: %v", err)
	}
}

func evalGapEvidence(query string, c *core.SearchEvalCase) util.MapStr {
	ev := util.MapStr{"query": query, "source": "eval_set"}
	if len(c.ExpectedIDs) > 0 {
		ev["expected_ids"] = c.ExpectedIDs
	}
	if len(c.ExpectedTitles) > 0 {
		ev["expected_titles"] = c.ExpectedTitles
	}
	return ev
}

// normalizeIDList deduplicates and trims expected document ids; empty
// entries drop out.
func normalizeIDList(ids []string) []string {
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

// normalizeTitleList pads or truncates display titles to the id count so
// the two lists never disagree in length.
func normalizeTitleList(titles []string, n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		title := ""
		if i < len(titles) {
			title = strings.TrimSpace(titles[i])
		}
		out = append(out, title)
	}
	return out
}
