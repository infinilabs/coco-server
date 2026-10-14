/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// EngineVectorCapability reports whether the backing engine can execute the
// semantic query right now. The engine-side semantic query needs an embedding
// service configured in the engine itself; when that service is missing the
// engine answers 500 (null pointer on a nil embedding request) instead of a
// usable error, so Coco probes once and caches the verdict instead of burning
// a failed search on every request.
type EngineVectorCapability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
	CheckedAt int64  `json:"checked_at"` // unix milliseconds
}

const engineCapabilityTTL = time.Minute

var (
	engineCapabilityMu     sync.Mutex
	engineCapabilityCached *EngineVectorCapability
	engineCapabilityUntil  time.Time

	// probeEngineSemanticFn is swapped out in tests.
	probeEngineSemanticFn = probeEngineSemantic
)

// The three definitive probe reasons — classify returns them with
// definitive=true and isDefinitiveCapability caches exactly these strings;
// sharing the constants is what keeps the two switches from drifting apart
// (a missing case there once left "pipeline is not defined" uncached and
// re-probed on every search).
const (
	engineReasonNoEmbedding = "engine embedding service is not configured"
	engineReasonRejected    = "engine rejected the semantic query"
	engineReasonNoPipeline  = "engine search pipeline is not defined (sync the Engine AI settings)"
)

// classifyEngineSemanticError sorts engine errors into a capability verdict
// plus whether the verdict is definitive (cacheable) — anything else
// (connectivity) is transient and must be retried on the next search.
func classifyEngineSemanticError(errText string) (EngineVectorCapability, bool) {
	l := strings.ToLower(errText)
	switch {
	case l == "":
		return EngineVectorCapability{Available: true}, true
	case strings.Contains(l, "embeddingrequest"), strings.Contains(l, "null_pointer_exception"):
		return EngineVectorCapability{Available: false, Reason: engineReasonNoEmbedding}, true
	case strings.Contains(l, "[semantic]"), strings.Contains(l, "query does not support"), strings.Contains(l, "parsing_exception"), strings.Contains(l, "unknown token"):
		return EngineVectorCapability{Available: false, Reason: engineReasonRejected}, true
	case strings.Contains(l, "is not defined"):
		return EngineVectorCapability{Available: false, Reason: engineReasonNoPipeline}, true
	default:
		return EngineVectorCapability{Available: false, Reason: trimReason(errText)}, false
	}
}

func trimReason(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}

// engineSearchPipelineName returns the search pipeline the semantic leg
// should run through: set only when Engine AI is enabled and its model
// resolves — an enabled-but-unsynced config would fail every semantic query
// with "pipeline is not defined", which the probe then reports honestly.
func engineSearchPipelineName() string {
	cfg := common.AppConfig().EngineAI
	if cfg == nil || !cfg.Enabled {
		return ""
	}
	if plan, err := common.ResolveEngineAIPlan(cfg); err == nil && plan.Model != nil {
		return core.EngineSearchPipelineName
	}
	return ""
}

// probeEngineSemantic runs a size-0 semantic query against the document index
// and reports what the engine said. It bypasses document permission filters:
// no hits are fetched and no user data is returned, this is a pure engine
// capability check. When Engine AI is enabled the canary runs through the
// managed search pipeline, matching how production searches execute.
func probeEngineSemantic(ctx context.Context) EngineVectorCapability {
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})
	if name := engineSearchPipelineName(); name != "" {
		orm.WithQueryArgs(octx, &[]util.KV{{Key: "search_pipeline", Value: name}})
	}

	builder := orm.NewQuery().Size(0)
	builder.Must(orm.SemanticQuery(documentEmbeddingField(), "coco engine capability probe", 0, ""))

	if _, err := orm.SearchV2(octx, builder); err != nil {
		var definitive bool
		cap, definitive := classifyEngineSemanticError(err.Error())
		if !definitive {
			log.Tracef("search: engine semantic probe failed (transient, not cached): %v", err)
		}
		return cap
	}
	return EngineVectorCapability{Available: true}
}

// engineVectorCapability returns the cached probe verdict, refreshing it when
// older than engineCapabilityTTL. Only definitive verdicts are cached, so a
// flaky engine never gets pinned as permanently unavailable — but a healthy
// engine is re-probed at most once a minute, which also picks up an embedding
// service being configured while Coco runs.
func engineVectorCapability(ctx context.Context) EngineVectorCapability {
	engineCapabilityMu.Lock()
	defer engineCapabilityMu.Unlock()
	if engineCapabilityCached != nil && time.Now().Before(engineCapabilityUntil) {
		return *engineCapabilityCached
	}
	cap := probeEngineSemanticFn(ctx)
	cap.CheckedAt = time.Now().UnixMilli()
	if isDefinitiveCapability(cap) {
		engineCapabilityCached = &cap
		engineCapabilityUntil = time.Now().Add(engineCapabilityTTL)
	} else {
		engineCapabilityCached = nil
		engineCapabilityUntil = time.Time{}
	}
	return cap
}

// isDefinitiveCapability reports whether a verdict may be cached: the reasons
// emitted by classifyEngineSemanticError's definitive branches.
func isDefinitiveCapability(cap EngineVectorCapability) bool {
	if cap.Available {
		return true
	}
	switch cap.Reason {
	case engineReasonNoEmbedding, engineReasonRejected, engineReasonNoPipeline:
		return true
	}
	return false
}

// resetEngineCapabilityCache drops the cached probe verdict so the next
// search re-probes immediately (used after a pipeline sync).
func resetEngineCapabilityCache() {
	engineCapabilityMu.Lock()
	engineCapabilityCached = nil
	engineCapabilityUntil = time.Time{}
	engineCapabilityMu.Unlock()
}

// engineSemanticReady tells whether the engine-side semantic query can run.
func engineSemanticReady(ctx context.Context) bool {
	return engineVectorCapability(ctx).Available
}

// engineCapabilityReport answers "what can the search stack do right now":
// the resolved semantic route, the engine probe behind it, the configured
// embedding model, and how many documents actually carry vectors. The search
// studio reads this so tuning happens against the truth, not an assumption.
func (h *APIHandler) engineCapabilityReport(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	plan := planSemantic(req.Context())

	total, withVectors := countDocuments(req.Context())

	model := util.MapStr{}
	if plan.EmbeddingModel != nil {
		model = util.MapStr{"provider_id": plan.EmbeddingModel.ProviderID, "id": plan.EmbeddingModel.ID}
	}
	h.WriteJSON(w, util.MapStr{
		"semantic": util.MapStr{
			"route":           string(plan.Route),
			"reason":          plan.Reason,
			"engine":          plan.Engine,
			"embedding_model": model,
		},
		"vector_field":         documentEmbeddingField(),
		"search_pipeline":      engineSearchPipelineName(),
		"server_side_collapse": serverSideCollapse(req.Context()),
		"documents": util.MapStr{
			"total":              total,
			"with_vectors":       withVectors,
			"by_embedding_model": embeddingModelDistribution(req.Context()),
		},
	}, http.StatusOK)
}

// embeddingModelDistribution aggregates the embedding_model stamps (W1) so a
// default-model change can see exactly how many documents would keep stale
// vectors and offer a batch reprocess instead of silently mixing models.
func embeddingModelDistribution(ctx context.Context) map[string]int64 {
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})

	builder := orm.NewQuery().Size(0).
		AddAgg("models", &orm.TermsAggregation{Field: "embedding_model", Size: 10})
	res, err := orm.Aggregate(octx, builder)
	if err != nil || res == nil {
		return nil
	}
	node := res.Aggs["models"]
	if node == nil {
		return nil
	}
	out := map[string]int64{}
	for _, b := range node.Buckets {
		key := b.Key
		if key == "" {
			key = "(unstamped)"
		}
		out[key] = b.DocCount
	}
	return out
}

// engineAIStatusResponse is the read-back (回显) surface: what Coco wants,
// what the engine actually has, and whether they match.
func (h *APIHandler) engineAIStatusResponse(ctx context.Context) util.MapStr {
	cfg := common.AppConfig().EngineAI
	plan, err := common.ResolveEngineAIPlan(cfg)
	if err != nil {
		return util.MapStr{"error": err.Error(), "enabled": cfg != nil && cfg.Enabled}
	}
	actual := common.ReadEnginePipelines(ctx, plan)
	drift := common.CompareEnginePipelines(plan, actual)

	status := util.MapStr{
		"enabled":  plan.Enabled,
		"config":   plan.Config,
		"warnings": plan.Warnings,
		"ingest_pipeline": util.MapStr{
			"name":    core.EngineIngestPipelineName,
			"desired": common.MaskEngineAISecret(plan.IngestPipeline),
			"actual":  actual.IngestPipeline,
			"in_sync": drift.IngestInSync,
		},
		"search_pipeline": util.MapStr{
			"name":    core.EngineSearchPipelineName,
			"desired": common.MaskEngineAISecret(plan.SearchPipeline),
			"actual":  actual.SearchPipeline,
			"in_sync": drift.SearchInSync,
		},
		"document_index": util.MapStr{
			"name":             plan.DocumentIndex,
			"default_pipeline": actual.DefaultPipeline,
			"expected":         core.EngineIngestPipelineName,
			"in_sync":          drift.DefaultInSync,
		},
		"in_sync": drift.Applied,
	}
	if len(actual.Errors) > 0 {
		status["engine_errors"] = actual.Errors
	}
	if plan.Model != nil {
		status["model"] = util.MapStr{
			"provider_id":  plan.Model.ProviderID,
			"id":           plan.Model.ID,
			"vendor":       plan.Vendor,
			"url":          plan.EmbeddingURL,
			"text_field":   plan.TextField,
			"vector_field": plan.VectorField,
		}
	}
	return status
}

// engineAIStatus answers "what is deployed and does it match the config".
func (h *APIHandler) engineAIStatus(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	h.WriteJSON(w, h.engineAIStatusResponse(req.Context()), http.StatusOK)
}

// engineAISync pushes the resolved plan to the engine and answers with the
// fresh status — configure, update and echo-back in one call.
func (h *APIHandler) engineAISync(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	cfg := common.AppConfig().EngineAI
	plan, err := common.ResolveEngineAIPlan(cfg)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if plan.Model == nil {
		h.WriteError(w, "no embedding model configured", http.StatusBadRequest)
		return
	}
	if err := common.ApplyEnginePipelines(req.Context(), plan); err != nil {
		log.Warnf("engine_ai: manual sync failed: %v", err)
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// The probe caches a verdict for up to a minute; a fresh sync flips the
	// engine to available immediately, so drop the cache.
	resetEngineCapabilityCache()
	h.WriteJSON(w, h.engineAIStatusResponse(req.Context()), http.StatusOK)
}

// countDocuments returns the document total and how many carry the semantic
// vector. Cheap size-0 searches; errors are reported as -1 rather than failing
// the whole report.
func countDocuments(ctx context.Context) (int64, int64) {
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})

	count := func(builder *orm.QueryBuilder) int64 {
		err, res := elastic.SearchV2WithResultItemMapper(octx, nil, builder, nil)
		if err != nil || res == nil {
			return -1
		}
		return res.Total
	}
	total := count(orm.NewQuery().Size(0))
	withVectors := count(orm.NewQuery().Size(0).Filter(orm.ExistsQuery(documentEmbeddingField())))
	return total, withVectors
}

var (
	collapseMu     sync.Mutex
	collapseCached *bool
	collapseUntil  time.Time
)

// serverSideCollapse probes (60s TTL) whether the engine honors the ES-style
// collapse parameter on the document index (W12): a true verdict means the
// fold COULD move server-side and stay correct across pages; false keeps the
// proven per-page client fold. The report surfaces the capability honestly —
// the fold itself stays client-side until the switch is deliberate.
func serverSideCollapse(ctx context.Context) bool {
	collapseMu.Lock()
	defer collapseMu.Unlock()
	if collapseCached != nil && time.Now().Before(collapseUntil) {
		return *collapseCached
	}
	supported := false
	if client := elastic.GetClientNoPanic(global.MustLookupString(elastic.GlobalSystemElasticsearchID)); client != nil {
		body := util.MapStr{
			"size":     1,
			"collapse": util.MapStr{"field": "content_hash"},
			"query":    util.MapStr{"match_all": util.MapStr{}},
		}
		if _, err := client.SearchWithRawQueryDSL(orm.GetIndexName(&core.Document{}), util.MustToJSONBytes(body)); err == nil {
			supported = true
		} else {
			log.Tracef("search: server-side collapse probe negative: %v", err)
		}
	}
	collapseCached = &supported
	collapseUntil = time.Now().Add(engineCapabilityTTL)
	return supported
}
