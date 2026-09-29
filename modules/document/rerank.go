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
// them in the reranker's order (ties keep RRF order — the sort is stable and
// starts from the fused sequence). Hits beyond the candidate window keep
// their RRF order behind the re-scored ones, so nothing is dropped.
func rerankFusedHits(ctx context.Context, plan rerankPlan, query string, fused []elastic.DocumentWithMeta[core.Document]) ([]elastic.DocumentWithMeta[core.Document], []rerankScoreChange, error) {
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

	// rank by relevance desc, ties keep RRF order (stable sort on a slice
	// already in RRF order)
	order := make([]int, 0, len(parsed.Results))
	seen := map[int]bool{}
	for i := range parsed.Results {
		idx := parsed.Results[i].Index
		if idx < 0 || idx >= window || seen[idx] {
			continue
		}
		seen[idx] = true
		order = append(order, i)
	}
	sort.SliceStable(order, func(a, b int) bool {
		return parsed.Results[order[a]].RelevanceScore > parsed.Results[order[b]].RelevanceScore
	})

	out := make([]elastic.DocumentWithMeta[core.Document], 0, len(fused))
	changes := make([]rerankScoreChange, 0, len(order))
	scoredIndex := map[int]bool{}
	for newRank, resultIdx := range order {
		idx := parsed.Results[resultIdx].Index
		scoredIndex[idx] = true
		hit := fused[idx]
		changes = append(changes, rerankScoreChange{
			ID:             hit.ID,
			RRFRank:        idx + 1,
			RerankRank:     newRank + 1,
			RelevanceScore: parsed.Results[resultIdx].RelevanceScore,
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
	reranked, _, err := rerankFusedHits(ctx, plan, query, fused)
	if err != nil {
		log.Warnf("hybrid_rrf: rerank failed, keeping RRF order: %v", err)
		return fused, fmt.Sprintf("rerank degraded: %v", err)
	}
	return reranked, fmt.Sprintf("rerank applied (%s/%s)", plan.Model.ProviderID, plan.Model.ID)
}
