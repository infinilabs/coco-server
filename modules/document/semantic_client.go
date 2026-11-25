/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// semanticRoute names how the semantic leg of a search is executed.
//
// The engine-side semantic query is the native path, but it only works when
// the engine itself has an embedding service configured. When it doesn't,
// Coco still owns an embedding model (Settings → Default Model) and the
// documents already carry vectors written at pipeline time — so the semantic
// leg degrades to a client-side route: BM25 recall, then cosine rerank with
// the query embedding computed by Coco. Only when neither head exists does
// the leg collapse to keyword search.
type semanticRoute string

const (
	semanticRouteEngine semanticRoute = "engine" // engine semantic query (query_text)
	semanticRouteClient semanticRoute = "client" // BM25 recall + Coco-side cosine rerank
	semanticRouteNone   semanticRoute = "none"   // degrade to keyword
)

// semanticPlan is the resolved execution plan for the semantic leg.
type semanticPlan struct {
	Route          semanticRoute          `json:"route"`
	Engine         EngineVectorCapability `json:"engine"`
	EmbeddingModel *core.ModelId          `json:"embedding_model,omitempty"`
	Reason         string                 `json:"reason,omitempty"` // set when Route is none
}

var (
	// resolveEmbeddingModelFn is swapped out in tests.
	resolveEmbeddingModelFn = resolveDefaultEmbeddingModel
)

// planSemantic decides how the semantic leg runs right now:
// engine query if the engine can, client rerank if Coco has a default
// embedding model, keyword otherwise.
func planSemantic(ctx context.Context) semanticPlan {
	engineCap := engineVectorCapability(ctx)
	if engineCap.Available {
		return semanticPlan{Route: semanticRouteEngine, Engine: engineCap}
	}
	if m := resolveEmbeddingModelFn(); m != nil {
		return semanticPlan{
			Route:          semanticRouteClient,
			Engine:         engineCap,
			EmbeddingModel: m,
			Reason:         fmt.Sprintf("engine semantic query unavailable (%s), using client-side rerank", engineCap.Reason),
		}
	}
	return semanticPlan{
		Route:  semanticRouteNone,
		Engine: engineCap,
		Reason: fmt.Sprintf("engine semantic query unavailable (%s) and no default embedding model is configured, degrading to keyword", engineCap.Reason),
	}
}

// resolveDefaultEmbeddingModel returns the default embedding ModelId when it
// is fully configured and its provider exists.
func resolveDefaultEmbeddingModel() *core.ModelId {
	m := llmmodule.ResolveModel(core.LLMTypeEmbedding, nil)
	if m == nil {
		return nil
	}
	if _, err := common.GetModelProvider(m.ProviderID); err != nil {
		return nil
	}
	return m
}

// embeddingClient is the subset of llms.Model that generates embeddings; both
// the OpenAI-compatible and Ollama clients implement it.
type embeddingClient interface {
	CreateEmbedding(ctx context.Context, texts []string) ([][]float32, error)
}

var (
	// newEmbeddingLLMFn is swapped out in tests.
	newEmbeddingLLMFn = langchain.GetEmbeddingLLM
)

// embedQueryText computes the query embedding with Coco's default embedding
// model, enforcing the dimension the document vector fields are mapped with.
func embedQueryText(ctx context.Context, model *core.ModelId, text string) ([]float32, error) {
	provider, err := common.GetModelProvider(model.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("get embedding model provider %s: %w", model.ProviderID, err)
	}
	client, ok := newEmbeddingLLMFn(provider.BaseURL, provider.APIType, model.ID, provider.APIKey, core.RequiredEmbeddingDimension).(embeddingClient)
	if !ok {
		return nil, fmt.Errorf("model %s (%s) does not support embeddings", model.ID, provider.APIType)
	}
	return embedTextWith(ctx, client, text)
}

// embedTextWith runs one text through an embedding client and enforces the
// dimension the document vector fields are mapped with.
func embedTextWith(ctx context.Context, client embeddingClient, text string) ([]float32, error) {
	vectors, err := client.CreateEmbedding(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vectors) != 1 {
		return nil, fmt.Errorf("embed query: expected 1 vector, got %d", len(vectors))
	}
	if len(vectors[0]) != core.RequiredEmbeddingDimension {
		return nil, fmt.Errorf("embed query: dimension mismatch, got %d, want %d", len(vectors[0]), core.RequiredEmbeddingDimension)
	}
	return vectors[0], nil
}

// fetchDocumentVectors loads the stored 1024-dim vectors for the given
// document IDs. Missing vectors (docs indexed before the vectorize stage)
// are simply absent from the map.
func fetchDocumentVectors(ctx context.Context, ids []string) (map[string][]float32, error) {
	out := make(map[string][]float32, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})

	builder := orm.NewQuery().Size(len(ids))
	builder.Must(orm.TermsQuery("id", ids))
	builder.Include("id", documentEmbeddingField())

	var docs []core.Document
	if err, _ := elastic.SearchV2WithResultItemMapper(octx, &docs, builder, nil); err != nil {
		return nil, err
	}
	for _, doc := range docs {
		if len(doc.AiInsights.Embedding.Embedding1024) > 0 {
			out[doc.ID] = doc.AiInsights.Embedding.Embedding1024
		}
	}
	return out, nil
}

// cosineSimilarityF32 returns the cosine similarity of two equal-length
// vectors; zero vectors return 0.
func cosineSimilarityF32(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// chunkRefineCap bounds how many head documents get the best-chunk
// refinement: chunk vectors are fetched per doc (10 chunks × 1024 floats
// each on typical docs), refining the whole recall window would move
// megabytes per query. Ranks below the top-50 barely move RRF anyway.
const chunkRefineCap = 50

// semanticQuoteRunes bounds the chunk excerpt carried on the hit for the
// citation UI (W10/W17 consume it).
const semanticQuoteRunes = 300

// semanticChunkVec is one retrievable chunk with its vector and locating
// payload, as loaded for the best-chunk refinement.
type semanticChunkVec struct {
	Vector     []float32
	Text       string // full chunk text — neighbor expansion needs it whole
	Breadcrumb string
	StartPage  int
	EndPage    int
}

// fetchDocumentChunkVectors loads chunk-level vectors for the given doc
// ids (W3/D14 option a). Docs whose chunks carry no vectors are simply
// absent — the doc-level rerank already scored them.
var fetchDocumentChunkVectorsFn = func(ctx context.Context, ids []string) (map[string][]semanticChunkVec, error) {
	out := make(map[string][]semanticChunkVec, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})

	builder := orm.NewQuery().Size(len(ids))
	builder.Must(orm.TermsQuery("id", ids))
	builder.Include("id", "document_chunk")

	var docs []core.Document
	if err, _ := elastic.SearchV2WithResultItemMapper(octx, &docs, builder, nil); err != nil {
		return nil, err
	}
	for _, doc := range docs {
		var chunks []semanticChunkVec
		for _, c := range doc.Chunks {
			if len(c.Embedding.Embedding1024) == 0 {
				continue
			}
			chunks = append(chunks, semanticChunkVec{
				Vector:     c.Embedding.Embedding1024,
				Text:       c.Text,
				Breadcrumb: c.Breadcrumb,
				StartPage:  c.Range.Start,
				EndPage:    c.Range.End,
			})
		}
		if len(chunks) > 0 {
			out[doc.ID] = chunks
		}
	}
	return out, nil
}

func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

// neighborExpandTrigger/neighborExpandTarget carry the D4.5 rule: a best
// chunk shorter than 350 runes gets its adjacent chunks' edges appended,
// bounded to ~850 runes total — the passage the reader needs, not the
// whole section.
const (
	neighborExpandTrigger = 350
	neighborExpandTarget  = 850
)

// expandNeighbors returns the tail of the previous chunk and the head of
// the next chunk around cs[best], sized to fit the target budget. Chunks
// already carry overlap prefixes, so the edges stitch naturally.
func expandNeighbors(cs []semanticChunkVec, best int) (before, after string) {
	bestLen := len([]rune(cs[best].Text))
	if bestLen >= neighborExpandTrigger {
		return "", ""
	}
	budget := neighborExpandTarget - bestLen
	half := budget / 2

	if best > 0 {
		prev := []rune(cs[best-1].Text)
		n := minInt(half, len(prev))
		before = string(prev[len(prev)-n:])
	}
	if best+1 < len(cs) {
		next := []rune(cs[best+1].Text)
		n := minInt(budget-half, len(next))
		after = string(next[:n])
	}
	return before, after
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// refineHitsByBestChunk upgrades the head hits to chunk granularity: each
// refined doc's score becomes cosine(query, best chunk)+1 and its metadata
// carries semantic_chunk (index/breadcrumb/page range/quote) so citations
// can land on the block. Hits without chunk vectors keep their doc-level
// scores; the list is re-sorted so refined scores take effect.
func refineHitsByBestChunk(chunks map[string][]semanticChunkVec, queryVec []float32, hits []elastic.DocumentWithMeta[core.Document]) int {
	refined := 0
	for i := range hits {
		if refined >= chunkRefineCap {
			break
		}
		cs, ok := chunks[hits[i].ID]
		if !ok || len(cs) == 0 {
			continue
		}
		best := -1
		bestScore := math.Inf(-1)
		for j, c := range cs {
			if s := cosineSimilarityF32(queryVec, c.Vector); s > bestScore {
				bestScore, best = s, j
			}
		}
		if best < 0 {
			continue
		}
		bc := cs[best]
		hits[i].Score = float32(bestScore + 1)
		if hits[i].Source.Metadata == nil {
			hits[i].Source.Metadata = util.MapStr{}
		}
		before, after := expandNeighbors(cs, best)
		hits[i].Source.Metadata["semantic_chunk"] = util.MapStr{
			"index":      best,
			"breadcrumb": bc.Breadcrumb,
			"pages":      util.MapStr{"start": bc.StartPage, "end": bc.EndPage},
			"quote":      truncateRunes(bc.Text, semanticQuoteRunes),
			"neighbors":  util.MapStr{"before": before, "after": after},
		}
		refined++
	}
	sortHitsByScoreStable(hits)
	return refined
}

// sortHitsByScoreStable orders hits by descending score, NaN scores
// (unscored tail) keeping their relative order at the end.
func sortHitsByScoreStable(hits []elastic.DocumentWithMeta[core.Document]) {
	sort.SliceStable(hits, func(i, j int) bool {
		a, b := float64(hits[i].Score), float64(hits[j].Score)
		aNaN, bNaN := math.IsNaN(a), math.IsNaN(b)
		if aNaN || bNaN {
			return false
		}
		return a > b
	})
}

// paginateHits slices a reranked hit list to the requested page in place.
func paginateHits(result *elastic.SearchResponseWithMeta[core.Document], from, size int) {
	end := from + size
	if end > len(result.Hits.Hits) {
		end = len(result.Hits.Hits)
	}
	if from > len(result.Hits.Hits) {
		from = len(result.Hits.Hits)
	}
	result.Hits.Hits = result.Hits.Hits[from:end]
	if len(result.Hits.Hits) > 0 {
		result.Hits.MaxScore = result.Hits.Hits[0].Score
	}
}

// errSemanticSkipped is returned by the semantic leg when no route can run;
// callers treat it like a failed route (single-route degrade), with the
// reason carried in the message.
var errSemanticSkipped = errors.New("semantic route skipped")

// rerankHitsBySemantic reorders hits by cosine similarity between the query
// embedding and each hit's stored vector, in place. Score becomes cosine+1
// (0..2, order-preserving, non-negative like other engine scores). Hits
// without a vector keep their BM25 order at the tail — they were recalled,
// they just can't be scored semantically.
func rerankHitsBySemantic(vectors map[string][]float32, queryVec []float32, hits []elastic.DocumentWithMeta[core.Document]) {
	type scored struct {
		hit   elastic.DocumentWithMeta[core.Document]
		score float64
	}
	scoredHits := make([]scored, len(hits))
	for i, hit := range hits {
		scoredHits[i] = scored{hit: hit, score: math.NaN()}
		if v, ok := vectors[hit.ID]; ok {
			scoredHits[i].score = cosineSimilarityF32(queryVec, v) + 1
		}
	}
	sort.SliceStable(scoredHits, func(i, j int) bool {
		a, b := scoredHits[i].score, scoredHits[j].score
		aNaN, bNaN := math.IsNaN(a), math.IsNaN(b)
		if aNaN && bNaN {
			return false // stable: keep BM25 order at the tail
		}
		if aNaN {
			return false
		}
		if bNaN {
			return true
		}
		if a != b {
			return a > b
		}
		return false
	})
	for i, sh := range scoredHits {
		hits[i] = sh.hit
		if !math.IsNaN(sh.score) {
			hits[i].Score = float32(sh.score)
		}
	}
}

// clientSemanticRecall runs the client-side semantic route: BM25 recall with
// the caller's builder (permissions and filters already applied by
// QueryDocuments), then fetches the stored vectors, embeds the query with
// Coco's default embedding model and reranks by cosine. It returns the
// reranked response plus a human-readable note of what actually ran. When the
// embedding call itself fails, the un-reranked BM25 recall is returned with a
// note instead of an error — recall already succeeded, so the user still gets
// results.
func clientSemanticRecall(ctx context.Context, builder *orm.QueryBuilder, query, datasource, integrationID, category, subcategory, richCategory string, fuzziness int) (*elastic.SearchResponseWithMeta[core.Document], string, error) {
	// Recall only docs that carry a vector: the rerank below can only score
	// those, so widening recall to un-vectorized docs would just pad the tail.
	builder.Filter(orm.ExistsQuery(documentEmbeddingField()))

	resp, err := QueryDocuments(ctx, builder, query, datasource, integrationID, category, subcategory, richCategory, "keyword", fuzziness, nil)
	if err != nil {
		return nil, "", err
	}
	out := &elastic.SearchResponseWithMeta[core.Document]{}
	if len(resp.Raw) > 0 {
		util.MustFromJSONBytes(resp.Raw, out)
	}

	plan := planSemantic(ctx)
	if plan.EmbeddingModel == nil {
		return out, "client semantic: no default embedding model, returning BM25 recall", nil
	}
	ids := make([]string, 0, len(out.Hits.Hits))
	for _, hit := range out.Hits.Hits {
		ids = append(ids, hit.ID)
	}
	vectors, err := fetchDocumentVectors(ctx, ids)
	if err != nil {
		log.Warnf("search: client semantic vector fetch failed: %v", err)
		return out, "client semantic: vector fetch failed, returning BM25 recall", nil
	}
	queryVec, err := embedQueryText(ctx, plan.EmbeddingModel, query)
	if err != nil {
		log.Warnf("search: client semantic query embedding failed: %v", err)
		return out, "client semantic: query embedding failed, returning BM25 recall", nil
	}

	rerankHitsBySemantic(vectors, queryVec, out.Hits.Hits)

	// W3/D14 option a — chunk-level refinement of the head: fetch chunk
	// vectors for the top docs only (bounded), rescore each by its best
	// chunk and attach the block locator for citations. Failure degrades
	// to the doc-level rerank that already ran.
	refineIDs := make([]string, 0, chunkRefineCap)
	for i, hit := range out.Hits.Hits {
		if i >= chunkRefineCap {
			break
		}
		refineIDs = append(refineIDs, hit.ID)
	}
	note := fmt.Sprintf("client-side semantic rerank (BM25 recall + cosine, model %s/%s)", plan.EmbeddingModel.ProviderID, plan.EmbeddingModel.ID)
	if chunkVecs, cerr := fetchDocumentChunkVectorsFn(ctx, refineIDs); cerr != nil {
		log.Debugf("search: chunk vector fetch failed, keeping doc-level rerank: %v", cerr)
	} else if refined := refineHitsByBestChunk(chunkVecs, queryVec, out.Hits.Hits); refined > 0 {
		note = fmt.Sprintf("client-side semantic rerank (BM25 recall + best-chunk cosine on %d docs, model %s/%s)", refined, plan.EmbeddingModel.ProviderID, plan.EmbeddingModel.ID)
	}

	if len(out.Hits.Hits) > 0 {
		out.Hits.MaxScore = out.Hits.Hits[0].Score
	}
	return out, note, nil
}
