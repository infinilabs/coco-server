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
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
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

// classifyEngineSemanticError maps an engine error text to a capability
// verdict. Definitive verdicts (engine missing its embedding service, engine
// not understanding the query) are safe to cache; anything else (timeouts,
// connectivity) is transient and must be retried on the next search.
func classifyEngineSemanticError(errText string) (EngineVectorCapability, bool) {
	l := strings.ToLower(errText)
	switch {
	case l == "":
		return EngineVectorCapability{Available: true}, true
	case strings.Contains(l, "embeddingrequest"), strings.Contains(l, "null_pointer_exception"):
		return EngineVectorCapability{Available: false, Reason: "engine embedding service is not configured"}, true
	case strings.Contains(l, "[semantic]"), strings.Contains(l, "query does not support"), strings.Contains(l, "parsing_exception"), strings.Contains(l, "unknown token"):
		return EngineVectorCapability{Available: false, Reason: "engine rejected the semantic query"}, true
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

// probeEngineSemantic runs a size-0 semantic query against the document index
// and reports what the engine said. It bypasses document permission filters:
// no hits are fetched and no user data is returned, this is a pure engine
// capability check.
func probeEngineSemantic(ctx context.Context) EngineVectorCapability {
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})

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
	case "engine embedding service is not configured", "engine rejected the semantic query":
		return true
	}
	return false
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
		"vector_field": documentEmbeddingField(),
		"documents": util.MapStr{
			"total":        total,
			"with_vectors": withVectors,
		},
	}, http.StatusOK)
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
