/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"net/http"
	"strings"

	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/log"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

/* Entity twin reconcile — closes the identity gap between the two entity
 * writers. extract_entities resolves identity by name within the KB scope,
 * but rows written before a datasource was bound to a KB carry no kb_id
 * (resolveKB finds none); promote/compile then shells out same-name
 * kb-scoped rows that articles cite, and the relations/sources provenance
 * stay stranded on the kb-less twin — the graph loses its typed relation
 * edges and the cited row looks empty. This reconcile folds each kb-less
 * twin into the kb-scoped row of the same name (the cited one survives),
 * inheriting provenance and the review status (rank-up). */

const reconcileScanSize = 1000

// ReconcileEntityTwins merges kb-less same-name entity rows into the
// kb-scoped rows of the given KB. The kb-scoped row always survives
// (articles' wikilinks cite it); the twin contributes aliases, sources,
// relations, properties and its review status (proposed < reviewed <
// published). Re-runs are no-ops: after a successful pass no matching
// kb-less twins remain.
func ReconcileEntityTwins(ctx *orm.Context, kbID string) (int, error) {
	if kbID == "" {
		return 0, nil
	}
	orm.WithModel(ctx, &core.WikiEntity{})

	// the KB's own pool — these rows are the citation targets
	kbRows, err := scanEntities(ctx, orm.NewQuery().Size(reconcileScanSize).Filter(orm.TermQuery("kb_id", kbID)))
	if err != nil {
		return 0, err
	}
	if len(kbRows) == 0 {
		return 0, nil
	}

	// candidate twins: kb-less rows matching any kb row by name or alias
	// (case-insensitive). A bounded global scan of kb-less rows keeps this
	// simple — the pool is small in practice and the match is in-memory.
	kbless, err := scanEntities(ctx, orm.NewQuery().Size(reconcileScanSize).
		Filter(orm.MustNotQuery(orm.ExistsQuery("kb_id"))))
	if err != nil {
		return 0, err
	}
	if len(kbless) == 0 {
		return 0, nil
	}

	byName := map[string][]*core.WikiEntity{}
	for i := range kbless {
		e := &kbless[i]
		byName[strings.ToLower(e.Name)] = append(byName[strings.ToLower(e.Name)], e)
		for _, a := range e.Aliases {
			a = strings.ToLower(strings.TrimSpace(a))
			if a != "" {
				byName[a] = append(byName[a], e)
			}
		}
	}

	merged := 0
	absorbedIDs := map[string]string{} // twin id -> surviving kb row id
	for i := range kbRows {
		canonical := &kbRows[i]
		twins := byName[strings.ToLower(canonical.Name)]
		for _, a := range canonical.Aliases {
			twins = append(twins, byName[strings.ToLower(strings.TrimSpace(a))]...)
		}
		twins = dedupeTwins(twins, canonical.ID)
		if len(twins) == 0 {
			continue
		}

		// description: a proposed shell's description is the topic-claim
		// fallback (entityDescriptionsFromStatements), which can be
		// misattributed prose about a DIFFERENT entity (live data: the
		// "Apache Tika" shell carried Coco AI's summary). The twin's
		// extraction-grounded description is the better body copy — take
		// it when the canonical row has not been human-reviewed yet.
		if canonical.Status == core.WikiEntityProposed {
			for _, t := range twins {
				if strings.TrimSpace(t.Description) != "" {
					canonical.Description = strings.TrimSpace(t.Description)
					break
				}
			}
		}

		// pass a copy: mergeEntityRows persists and we track absorbed ids
		for _, t := range twins {
			absorbedIDs[t.ID] = canonical.ID
		}
		// re-fetch canonical/absorbed rows fresh so mergeEntityRows works
		// on their persisted state (earlier iterations may have mutated
		// in-memory slices shared with byName)
		cFresh, tFresh, err := fetchFreshPair(ctx, canonical.ID, twins)
		if err != nil || cFresh == nil {
			if err != nil {
				log.Warnf("[entity_reconcile] fetch canonical %s: %v", canonical.ID, err)
			}
			continue
		}
		if err := mergeEntityRows(ctx, cFresh, tFresh); err != nil {
			log.Warnf("[entity_reconcile] merge %s <- %d twin(s): %v", canonical.ID, len(tFresh), err)
			continue
		}
		merged += len(tFresh)
		log.Infof("[entity_reconcile] kb %s: folded %d twin(s) into %q (%s)", kbID, len(tFresh), canonical.Name, canonical.ID)
	}

	if merged == 0 {
		return 0, nil
	}

	// retarget typed relation edges in memory: absorbed twins' relations
	// now live on kb rows but still point at OTHER absorbed twin ids (the
	// ES relations.target_id terms scan inside mergeEntityRows is a no-op
	// — the relations mapping is enabled:false). Rewrite everything that
	// points at an absorbed id.
	if len(absorbedIDs) > 0 {
		rows, err := scanEntities(ctx, orm.NewQuery().Size(reconcileScanSize).Filter(orm.TermQuery("kb_id", kbID)))
		if err == nil {
			for i := range rows {
				e := &rows[i]
				changed := false
				for j := range e.Relations {
					if to, ok := absorbedIDs[e.Relations[j].TargetID]; ok && to != e.ID {
						e.Relations[j].TargetID = to
						changed = true
					}
				}
				if changed {
					orm.WithModel(ctx, &core.WikiEntity{})
					if err := orm.Update(ctx, e); err != nil {
						log.Warnf("[entity_reconcile] retarget edges of %s: %v", e.ID, err)
					}
				}
			}
		} else {
			log.Warnf("[entity_reconcile] edge retarget scan: %v", err)
		}
	}

	log.Infof("[entity_reconcile] kb %s: %d kb-less twin(s) reconciled", kbID, merged)
	return merged, nil
}

// scanEntities runs the query and decodes WikiEntity hits.
func scanEntities(ctx *orm.Context, builder *orm.QueryBuilder) ([]core.WikiEntity, error) {
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	rows, _, err := elastic.DecodeHits[core.WikiEntity](res)
	return rows, err
}

// dedupeTwins removes duplicates and any entry colliding with the
// canonical id itself.
func dedupeTwins(twins []*core.WikiEntity, canonicalID string) []*core.WikiEntity {
	seen := map[string]bool{}
	out := make([]*core.WikiEntity, 0, len(twins))
	for _, t := range twins {
		if t == nil || t.ID == canonicalID || seen[t.ID] {
			continue
		}
		seen[t.ID] = true
		out = append(out, t)
	}
	return out
}

// fetchFreshPair re-reads the canonical row and its twins by id so the
// merge works on persisted state, not on possibly stale scan copies.
func fetchFreshPair(ctx *orm.Context, canonicalID string, twins []*core.WikiEntity) (*core.WikiEntity, []*core.WikiEntity, error) {
	c := &core.WikiEntity{}
	c.SetID(canonicalID)
	orm.WithModel(ctx, &core.WikiEntity{})
	exists, err := orm.GetV2(ctx, c)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, nil
	}
	fresh := make([]*core.WikiEntity, 0, len(twins))
	for _, t := range twins {
		row := &core.WikiEntity{}
		row.SetID(t.ID)
		exists, err := orm.GetV2(ctx, row)
		if err != nil || !exists {
			continue // deleted by a concurrent reconcile pass
		}
		fresh = append(fresh, row)
	}
	return c, fresh, nil
}

// POST /wiki/kb/:id/_reconcile_entities — manual trigger for the twin
// reconcile (ops/verification; the promote path also reconciles
// automatically after each governance apply).
func (h *APIHandler) reconcileKbEntities(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.MustGetParameter("id")
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh

	merged, err := ReconcileEntityTwins(ctx, kbID)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, util.MapStr{
		"kb_id":  kbID,
		"merged": merged,
	})
}
