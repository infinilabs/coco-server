/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
)

// Post-fusion rerank leg (D4): RRF orders candidates by consensus of the
// recall routes, not by how well each document actually answers the query.
// When a reranker model is configured (Settings → Default Model → rerank),
// the fused top window is re-scored by a cross-encoder-style API and the
// order becomes the reranker's verdict. No model or a failing call degrades
// to pure RRF — the same fallback discipline as the semantic leg.

const (
	// rerankCandidateWindow bounds how many fused hits are re-scored;
	// rerankers score query-document pairs, so the window is the cost dial.
	rerankCandidateWindow = 50
	// rerankTimeout caps one rerank call; past it the leg degrades to RRF
	// instead of stalling the search.
	rerankTimeout = 10 * time.Second

	// D4.5 composite weights: the model verdict dominates but the RRF
	// consensus and the configured source priority keep a say — a slightly
	// lower-scored hit from the authoritative source outranks a twin from
	// an unranked one.
	rerankWeightModel  = 0.6
	rerankWeightRRF    = 0.3
	rerankWeightSource = 0.1

	// MMR (D4.5): greedy selection scoring 0.7×relevance − 0.3×redundancy
	// (max Jaccard of title+summary token sets against already-selected
	// hits), so a near-duplicate of a picked hit sinks below a diverse one.
	mmrLambda     = 0.7
	mmrRedundancy = 0.3
)

// rerankPlan is the resolved rerank configuration for one request.
type rerankPlan struct {
	Model *core.ModelId // nil = leg not configured, degrade silently
	Note  string        // why the leg is off or degraded
}

var (
	// resolveRerankModelFn is swapped out in tests (touches provider store).
	resolveRerankModelFn = resolveRerankModel
	// rerankHTTPPostFn is swapped out in tests.
	rerankHTTPPostFn = rerankHTTPPost
	// getRerankProviderFn is swapped out in tests (touches provider store).
	getRerankProviderFn = common.GetModelProvider
)

// planRerank resolves the reranker: default rerank model, provider must
// exist and speak an OpenAI-compatible API (the /rerank endpoint convention
// Jina, Cohere, vLLM and compatible gateways share). Other API types have
// no rerank convention — the leg reports why and stays off instead of
// throwing at query time.
func planRerank() rerankPlan {
	m := llmmodule.ResolveModel(core.LLMTypeRerank, nil)
	if m == nil {
		return rerankPlan{Note: "no rerank model configured, fused order kept"}
	}
	provider, err := common.GetModelProvider(m.ProviderID)
	if err != nil {
		return rerankPlan{Note: fmt.Sprintf("rerank provider %s unavailable: %v", m.ProviderID, err)}
	}
	if !strings.EqualFold(provider.APIType, "openai") {
		return rerankPlan{Note: fmt.Sprintf("rerank needs an OpenAI-compatible provider, %s uses %s", provider.Name, provider.APIType)}
	}
	return rerankPlan{Model: m}
}

// resolveRerankModel returns the configured rerank ModelId, nil when absent.
func resolveRerankModel() *core.ModelId {
	return llmmodule.ResolveModel(core.LLMTypeRerank, nil)
}

// rerankHTTPPost posts JSON to url with the provider key and returns the
// response body.
func rerankHTTPPost(ctx context.Context, url, apiKey string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	client := &http.Client{Timeout: rerankTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		message := string(out)
		if len(message) > 200 {
			message = message[:200]
		}
		return nil, fmt.Errorf("rerank API returned %s: %s", resp.Status, message)
	}
	return out, nil
}

// rerankRequest/rerankResponse follow the Jina/Cohere/vLLM /rerank shape:
// documents in, per-document relevance scores with original indexes out.
type rerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

type rerankResponse struct {
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"results"`
}

// rerankFusedHits re-scores the top fused hits with the reranker and returns
// them in the composite order (D4.5: 0.6 model + 0.3 RRF + 0.1 source
// priority), then MMR-diversifies the scored window. Ties keep RRF order —
// the sort is stable and starts from the fused sequence. Hits beyond the
// candidate window keep their RRF order behind the re-scored ones, so
// nothing is dropped.
func rerankFusedHits(ctx context.Context, plan rerankPlan, query string, sourcePriority []string, fused []elastic.DocumentWithMeta[core.Document]) ([]elastic.DocumentWithMeta[core.Document], []rerankScoreChange, error) {
	if plan.Model == nil || len(fused) == 0 {
		return fused, nil, nil
	}
	provider, err := getRerankProviderFn(plan.Model.ProviderID)
	if err != nil {
		return fused, nil, err
	}

	window := len(fused)
	if window > rerankCandidateWindow {
		window = rerankCandidateWindow
	}
	documents := make([]string, window)
	for i := 0; i < window; i++ {
		documents[i] = rerankDocumentText(&fused[i])
	}

	body, err := util.ToJSONBytes(rerankRequest{
		Model:     plan.Model.ID,
		Query:     query,
		Documents: documents,
		TopN:      window,
	})
	if err != nil {
		return fused, nil, err
	}
	url := strings.TrimSuffix(provider.BaseURL, "/") + "/rerank"
	raw, err := rerankHTTPPostFn(ctx, url, provider.APIKey, body)
	if err != nil {
		return fused, nil, err
	}
	var parsed rerankResponse
	if err := util.FromJSONBytes(raw, &parsed); err != nil {
		return fused, nil, err
	}

	// capture the RRF scores of the window before any reordering — the
	// composite needs them normalized
	rrfNorm := minMaxNormalizeF32(scoresOfWindow(fused, window))

	// composite per scored hit: 0.6×model(norm) + 0.3×rrf(norm) + 0.1×source
	modelRaw := make([]float64, 0, len(parsed.Results))
	idxOf := map[int]int{} // hit index → position in modelRaw
	for i := range parsed.Results {
		idx := parsed.Results[i].Index
		if idx < 0 || idx >= window {
			continue
		}
		if _, dup := idxOf[idx]; dup {
			continue
		}
		idxOf[idx] = len(modelRaw)
		modelRaw = append(modelRaw, parsed.Results[i].RelevanceScore)
	}
	modelNorm := minMaxNormalizeF64(modelRaw)

	scored := make([]rerankComposite, 0, len(idxOf))
	for idx, pos := range idxOf {
		scored = append(scored, rerankComposite{
			idx:       idx,
			score:     rerankWeightModel*modelNorm[pos] + rerankWeightRRF*rrfNorm[idx] + rerankWeightSource*sourceWeightFor(fused[idx].Source.Source.ID, sourcePriority),
			relevance: parsed.Results[idxOf[idx]].RelevanceScore,
		})
	}
	sort.SliceStable(scored, func(a, b int) bool { return scored[a].score > scored[b].score })

	// MMR diversification over the reranker texts of the scored hits
	mmrTexts := make([]string, len(scored))
	for i, c := range scored {
		mmrTexts[i] = documents[c.idx]
	}
	mmrOrder(scored, mmrTexts)

	out := make([]elastic.DocumentWithMeta[core.Document], 0, len(fused))
	changes := make([]rerankScoreChange, 0, len(scored))
	scoredIndex := map[int]bool{}
	for newRank, c := range scored {
		scoredIndex[c.idx] = true
		hit := fused[c.idx]
		changes = append(changes, rerankScoreChange{
			ID:             hit.ID,
			RRFRank:        c.idx + 1,
			RerankRank:     newRank + 1,
			RelevanceScore: c.relevance,
			CompositeScore: c.score,
		})
		out = append(out, hit)
	}
	// in-window hits the reranker didn't score keep RRF order behind the
	// scored ones, then the beyond-window tail — nothing is dropped
	for i := 0; i < len(fused); i++ {
		if i < window && scoredIndex[i] {
			continue
		}
		out = append(out, fused[i])
	}
	return out, changes, nil
}

// scoresOfWindow copies the fused scores of the candidate window.
func scoresOfWindow(fused []elastic.DocumentWithMeta[core.Document], window int) []float32 {
	out := make([]float32, window)
	for i := 0; i < window; i++ {
		out[i] = fused[i].Score
	}
	return out
}

func minMaxNormalizeF32(v []float32) []float64 {
	if len(v) == 0 {
		return nil
	}
	min, max := v[0], v[0]
	for _, x := range v {
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
	}
	out := make([]float64, len(v))
	span := float64(max - min)
	for i, x := range v {
		if span == 0 {
			out[i] = 1
			continue
		}
		out[i] = float64(x-min) / span
	}
	return out
}

func minMaxNormalizeF64(v []float64) []float64 {
	if len(v) == 0 {
		return nil
	}
	min, max := v[0], v[0]
	for _, x := range v {
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
	}
	out := make([]float64, len(v))
	span := max - min
	for i, x := range v {
		if span == 0 {
			out[i] = 1
			continue
		}
		out[i] = (x - min) / span
	}
	return out
}

// sourceWeightFor maps a hit's datasource to the 0..1 source weight (D4.5):
// the built-in knowledge source tops at 1.0, listed sources decay linearly
// from 1.0 toward 0.5 by their priority position, unlisted sit at 0.5 — a
// recommendation, never a veto.
func sourceWeightFor(sourceID string, priority []string) float64 {
	if sourceID == "wiki" {
		return 1.0
	}
	for i, s := range priority {
		if s == sourceID && len(priority) > 0 {
			return 1.0 - 0.5*float64(i)/float64(len(priority))
		}
	}
	return 0.5
}

// rerankComposite is one scored hit's composite verdict (D4.5).
type rerankComposite struct {
	idx       int     // index into the fused window
	score     float64 // composite: 0.6 model + 0.3 rrf + 0.1 source
	relevance float64 // raw reranker relevance, kept for reporting
}

// mmrOrder diversifies an already score-ordered list in place (greedy MMR,
// D4.5): each pick maximizes λ×score − (1−λ)×max Jaccard against the
// already-picked texts — a near-duplicate of a picked hit sinks below a
// diverse hit with slightly lower relevance.
func mmrOrder(scored []rerankComposite, texts []string) {
	if len(scored) <= 1 {
		return
	}
	picked := make([]int, 0, len(scored))
	remaining := make([]int, 0, len(scored))
	for i := range scored {
		remaining = append(remaining, i)
	}
	for len(remaining) > 0 {
		best, bestScore := 0, 0.0
		first := true
		for _, r := range remaining {
			redundancy := 0.0
			for _, p := range picked {
				if j := tokenJaccard(texts[r], texts[p]); j > redundancy {
					redundancy = j
				}
			}
			v := mmrLambda*scored[r].score - mmrRedundancy*redundancy
			if first || v > bestScore {
				best, bestScore, first = r, v, false
			}
		}
		picked = append(picked, best)
		for i, r := range remaining {
			if r == best {
				remaining = append(remaining[:i], remaining[i+1:]...)
				break
			}
		}
	}
	reordered := make([]rerankComposite, len(scored))
	reorderedTexts := make([]string, len(texts))
	for i, p := range picked {
		reordered[i] = scored[p]
		if p < len(texts) {
			reorderedTexts[i] = texts[p]
		}
	}
	copy(scored, reordered)
	copy(texts, reorderedTexts)
}

// tokenJaccard is the Jaccard similarity of two texts' token sets.
func tokenJaccard(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	as := tokenSet(a)
	bs := tokenSet(b)
	if len(as) == 0 || len(bs) == 0 {
		return 0
	}
	inter := 0
	for t := range as {
		if bs[t] {
			inter++
		}
	}
	union := len(as) + len(bs) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func tokenSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, f := range strings.Fields(strings.ToLower(s)) {
		out[f] = true
	}
	return out
}

// rerankDocumentText is what the reranker reads per hit: title plus summary
// (full content is not carried on search hits).
func rerankDocumentText(hit *elastic.DocumentWithMeta[core.Document]) string {
	text := strings.TrimSpace(hit.Source.Title)
	if summary := strings.TrimSpace(hit.Source.Summary); summary != "" {
		text += "\n" + summary
	}
	if text == "" {
		text = hit.ID
	}
	return text
}

// rerankScoreChange records one hit's journey through the reranker; the
// studio returns the list so the RRF order, the reranker order and each
// delta are visible side by side.
type rerankScoreChange struct {
	ID             string  `json:"id"`
	RRFRank        int     `json:"rrf_rank"`
	RerankRank     int     `json:"rerank_rank"`
	RelevanceScore float64 `json:"relevance_score"`
	CompositeScore float64 `json:"composite_score"`
}

// applyRerank is the production hook: re-score the fused list when a
// reranker is configured, degrade to the fused order on any failure. The
// returned note travels as a response header so callers can tell applied
// from degraded.
func applyRerank(ctx context.Context, query string, fused []elastic.DocumentWithMeta[core.Document]) ([]elastic.DocumentWithMeta[core.Document], string) {
	return applyRerankWithPlan(ctx, planRerank(), query, fused)
}

// applyRerankWithPlan is applyRerank with an explicit plan, so tests can
// exercise the degrade path without the provider store.
func applyRerankWithPlan(ctx context.Context, plan rerankPlan, query string, fused []elastic.DocumentWithMeta[core.Document]) ([]elastic.DocumentWithMeta[core.Document], string) {
	if plan.Model == nil {
		return fused, plan.Note
	}
	reranked, _, err := rerankFusedHits(ctx, plan, query, loadCompileRules(ctx).SourcePriority, fused)
	if err != nil {
		log.Warnf("hybrid_rrf: rerank failed, keeping RRF order: %v", err)
		return fused, fmt.Sprintf("rerank degraded: %v", err)
	}
	return reranked, fmt.Sprintf("rerank applied (%s/%s)", plan.Model.ProviderID, plan.Model.ID)
}
