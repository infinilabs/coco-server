/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"net/http"
	"sync"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/rsa"
	"infini.sh/framework/core/elastic"
)

// rsaRoute runs the federated remote-source leg (DESIGN-RSA.md): every
// enabled source is queried directly — under its own timeout/QPS/cache
// budget — and the hits arrive as standard documents, so the fusion math
// needs zero special cases. Sources fan out concurrently; one slow or
// dead source costs nothing but its own budget (executor enforces that)
// and degrades this leg, never the search.
//
// Hit ordering across sources is rank-interleaved (source A rank-1, B
// rank-1, A rank-2, ...): with multiple sources sharing ONE route the
// interleave is the only fair seed for RRF's rank math — a straight
// concatenation would bury every source after the first.
func (h *APIHandler) rsaRoute(req *http.Request, query string, window int) (*elastic.SearchResponseWithMeta[core.Document], []rsa.SourceDiag, error) {
	settings := core.SearchSettings{}
	if cfg := appConfigFn(); cfg.SearchSettings != nil {
		settings = *cfg.SearchSettings
	}
	maxSources := settings.RsaMaxSourcesPerQuery
	if maxSources <= 0 {
		maxSources = 8
	}

	sources, err := rsa.ResolveEnabled(req.Context(), maxSources)
	if err != nil {
		return nil, nil, err
	}
	if len(sources) == 0 {
		return nil, nil, nil
	}

	type sourceOutput struct {
		res *rsa.Result
		diag *rsa.SourceDiag
	}
	outs := make([]sourceOutput, len(sources))
	var wg sync.WaitGroup
	for i := range sources {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			src := sources[i]
			q := rsa.Query{Text: query, From: 0, Size: window}
			outs[i].res, outs[i].diag = rsa.ExecuteQuery(req.Context(), &src, q)
		}(i)
	}
	wg.Wait()

	out := &elastic.SearchResponseWithMeta[core.Document]{}
	diags := make([]rsa.SourceDiag, 0, len(outs))
	var total int64

	// rank-interleave: round-robin across sources by their hit rank
	for rank := 0; ; rank++ {
		added := false
		for i := range outs {
			if outs[i].res != nil && rank < len(outs[i].res.Hits) {
				hit := outs[i].res.Hits[rank]
				score := float32(window - len(out.Hits.Hits))
				if score < 1 {
					score = 1
				}
				out.Hits.Hits = append(out.Hits.Hits, elastic.DocumentWithMeta[core.Document]{
					// the fusion keys identity on _id — every mapped RSA
					// document carries its deterministic pseudo id
					ID:     hit.Document.ID,
					Index:  "rsa_" + sources[i].ID,
					Source: hit.Document,
					Score:  score,
				})
				added = true
			}
		}
		if !added {
			break
		}
	}

	for i := range outs {
		if outs[i].diag != nil {
			diags = append(diags, *outs[i].diag)
			if outs[i].res != nil {
				total += outs[i].res.Total
			}
		}
	}

	// a plain map, not util.MapStr: GetTotal's type switch matches
	// map[string]interface{} only (same lesson as the wiki route)
	out.Hits.Total = map[string]interface{}{"value": total, "relation": "eq"}
	out.Took = int(totalTook(diags))
	log.Debugf("hybrid_rrf: rsa leg: %d sources, %d hits, sources=[%s]", len(sources), len(out.Hits.Hits), diagNames(diags))
	return out, diags, nil
}

func totalTook(diags []rsa.SourceDiag) int64 {
	var sum int64
	for _, d := range diags {
		sum += d.TookMS
	}
	return sum
}

func diagNames(diags []rsa.SourceDiag) string {
	names := ""
	for i, d := range diags {
		if i > 0 {
			names += ","
		}
		names += d.SourceName + ":" + d.Status
	}
	return names
}

