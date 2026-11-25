/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"fmt"
	"net/http"
	"sort"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/log"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

var entityStatusRank = map[string]int{
	core.WikiEntityProposed:  1,
	core.WikiEntityReviewed:  2,
	core.WikiEntityPublished: 3,
	"retired":                0,
}

// POST /wiki/entity/_merge {ids: [...]} — fold N same-concept entities into
// one canonical row (the one with the most sources): names become aliases,
// sources and relations union, references retarget, the rest are deleted.
func (h *APIHandler) mergeEntities(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.Error400(w, err.Error())
		return
	}
	if len(body.IDs) < 2 {
		h.Error400(w, "at least two entities are required to merge")
		return
	}
	if len(body.IDs) > 100 {
		h.Error400(w, "too many ids (cap 100 per merge)")
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.DirectAccess()
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.WikiEntity{})

	entities := make([]*core.WikiEntity, 0, len(body.IDs))
	seen := map[string]bool{}
	for _, id := range body.IDs {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		var e core.WikiEntity
		e.SetID(id)
		exists, err := orm.GetV2(ctx, &e)
		if err != nil || !exists {
			h.Error400(w, fmt.Sprintf("entity %s not found", id))
			return
		}
		entities = append(entities, &e)
	}
	if len(entities) < 2 {
		h.Error400(w, "fewer than two entities resolved")
		return
	}
	// one pool: every entity must belong to the same KB
	kbID := entities[0].KbID
	for _, e := range entities {
		if e.KbID != kbID {
			h.Error400(w, "cannot merge entities from different knowledge bases")
			return
		}
	}

	// canonical: the row with the most provenance; stable tiebreak by id
	sort.Slice(entities, func(i, j int) bool {
		if len(entities[i].Sources) != len(entities[j].Sources) {
			return len(entities[i].Sources) > len(entities[j].Sources)
		}
		return entities[i].ID < entities[j].ID
	})
	canonical := entities[0]
	absorbed := entities[1:]

	if err := mergeEntityRows(ctx, canonical, absorbed); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, util.MapStr{
		"merged":    len(absorbed),
		"canonical": canonical.ID,
		"name":      canonical.Name,
	})
}

// mergeEntityRows folds absorbed rows into canonical and persists the
// result: names become aliases, sources/relations/properties union,
// review status only moves up, references retarget, absorbed rows are
// deleted. Shared by the manual merge endpoint and the automatic
// kb-less-twin reconcile — the handler only adds HTTP concerns and the
// canonical-choice policy on top of this.
func mergeEntityRows(ctx *orm.Context, canonical *core.WikiEntity, absorbed []*core.WikiEntity) error {
	// names of the absorbed rows become aliases
	aliasSet := map[string]bool{}
	for _, a := range canonical.Aliases {
		if a != "" {
			aliasSet[a] = true
		}
	}
	for _, e := range absorbed {
		for _, name := range []string{e.Name, e.Subtype} {
			_ = name
		}
		if e.Name != "" && e.Name != canonical.Name {
			aliasSet[e.Name] = true
		}
		for _, a := range e.Aliases {
			if a != "" {
				aliasSet[a] = true
			}
		}
	}
	delete(aliasSet, canonical.Name)
	canonical.Aliases = aliasListSorted(aliasSet)

	// sources: union dedup by doc+excerpt
	srcSeen := map[string]bool{}
	sources := make([]core.WikiSourceReference, 0, len(canonical.Sources))
	for _, sr := range canonical.Sources {
		key := sr.DocID + "|" + sr.Excerpt
		if srcSeen[key] {
			continue
		}
		srcSeen[key] = true
		sources = append(sources, sr)
	}
	for _, e := range absorbed {
		for _, sr := range e.Sources {
			key := sr.DocID + "|" + sr.Excerpt
			if srcSeen[key] {
				continue
			}
			srcSeen[key] = true
			sources = append(sources, sr)
		}
	}
	canonical.Sources = sources

	// relations: union dedup by relation+target, no self-loops
	relSeen := map[string]bool{}
	rels := canonical.Relations[:0]
	for _, r := range canonical.Relations {
		key := r.Relation + "|" + r.TargetID
		if r.TargetID == canonical.ID || relSeen[key] {
			continue
		}
		relSeen[key] = true
		rels = append(rels, r)
	}
	for _, e := range absorbed {
		for _, r := range e.Relations {
			if r.TargetID == canonical.ID {
				continue
			}
			key := r.Relation + "|" + r.TargetID
			if relSeen[key] {
				continue
			}
			relSeen[key] = true
			rels = append(rels, r)
		}
	}
	canonical.Relations = rels

	// properties: shallow merge — canonical wins on conflicts
	for _, e := range absorbed {
		for k, v := range e.Properties {
			if _, ok := canonical.Properties[k]; !ok {
				if canonical.Properties == nil {
					canonical.Properties = util.MapStr{}
				}
				canonical.Properties[k] = v
			}
		}
	}

	// status: the review state only ever moves up
	for _, e := range absorbed {
		if entityStatusRank[e.Status] > entityStatusRank[canonical.Status] {
			canonical.Status = e.Status
		}
	}

	// retarget typed relations across the whole pool: entities pointing at
	// an absorbed id now point at the canonical row. The relations mapping
	// is enabled:false, so a relations.target_id terms scan here is a
	// silent no-op — edges held by OTHER rows pointing at an absorbed id
	// would dangle after the delete below. Scan the KB pool and the
	// kb-less pool instead and rewrite edges in memory (full-row Update,
	// the only write path that survives the enabled:false mapping).
	absorbedIDs := make([]string, 0, len(absorbed))
	absorbedSet := map[string]bool{}
	for _, e := range absorbed {
		absorbedIDs = append(absorbedIDs, e.ID)
		absorbedSet[e.ID] = true
	}
	scans := []*orm.QueryBuilder{}
	if canonical.KbID != "" {
		scans = append(scans, orm.NewQuery().Size(reconcileScanSize).
			Filter(orm.TermQuery("kb_id", canonical.KbID)))
	}
	// kb-less holders: edges written before a datasource was bound to a KB
	scans = append(scans, orm.NewQuery().Size(reconcileScanSize).
		Filter(orm.MustNotQuery(orm.ExistsQuery("kb_id"))))
	for _, builder := range scans {
		rows, err := scanEntities(ctx, builder)
		if err != nil {
			log.Warnf("[entity_merge] retarget scan: %v", err)
			continue
		}
		for i := range rows {
			other := &rows[i]
			if absorbedSet[other.ID] || other.ID == canonical.ID {
				continue
			}
			next, changed := retargetEdgeList(other.Relations, absorbedSet, canonical.ID, canonical.Type)
			if !changed {
				continue
			}
			other.Relations = next
			orm.WithModel(ctx, &core.WikiEntity{})
			if err := orm.Update(ctx, other); err != nil {
				log.Warnf("[entity_merge] retarget edges of %s: %v", other.ID, err)
			}
		}
	}

	// documents citing an absorbed entity retarget to the canonical id
	orm.WithModel(ctx, &core.Document{})
	if res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
		Filter(orm.TermsQuery("entity_ids", absorbedIDs)).Include("id", "entity_ids")); err == nil && res != nil {
		if docs, _, derr := elastic.DecodeHits[core.Document](res); derr == nil {
			for i := range docs {
				ids := map[string]bool{}
				changed := false
				for _, eid := range docs[i].EntityIDs {
					if absorbedSet[eid] {
						if !ids[canonical.ID] {
							ids[canonical.ID] = true
						}
						changed = true
					} else {
						ids[eid] = true
					}
				}
				if !changed {
					continue
				}
				next := make([]string, 0, len(ids))
				for eid := range ids {
					next = append(next, eid)
				}
				docs[i].EntityIDs = next
				_ = orm.Update(ctx, &docs[i])
			}
		}
	}

	// absorbed wikilink references in articles: linked_pages entity_id
	// retargets (in-memory scan over the KB's articles, bounded). Skipped
	// for kb-less rows — articles only cite kb-scoped rows.
	orm.WithModel(ctx, &core.WikiArticle{})
	if canonical.KbID != "" {
		if res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
			Filter(orm.TermQuery("kb_id", canonical.KbID)).Include("id", "linked_pages")); err == nil && res != nil {
			if arts, _, derr := elastic.DecodeHits[core.WikiArticle](res); derr == nil {
				for i := range arts {
					a := &arts[i]
					changed := false
					for j := range a.LinkedPages {
						if absorbedSet[a.LinkedPages[j].EntityID] {
							a.LinkedPages[j].EntityID = canonical.ID
							changed = true
						}
					}
					if changed {
						_ = orm.Update(ctx, a)
					}
				}
			}
		}
	}

	// absorbed rows are subsumed — their content lives on in the canonical
	for _, e := range absorbed {
		if err := orm.Delete(ctx, e); err != nil {
			return err
		}
	}
	if err := orm.Update(ctx, canonical); err != nil {
		return err
	}
	return nil
}

// retargetEdgeList rewrites edges pointing at absorbed ids so they point at
// the canonical row (target_type follows the canonical row's type), then
// drops the ones that would duplicate an edge the holder already has
// towards the canonical row. Returns the kept slice (non-destructive) and
// whether anything changed.
func retargetEdgeList(rels []core.WikiEntityRelation, absorbedSet map[string]bool, canonicalID, canonicalType string) ([]core.WikiEntityRelation, bool) {
	if len(rels) == 0 {
		return rels, false
	}
	// pre-existing direct edges to canonical occupy the dedup space
	seen := map[string]bool{}
	for _, r := range rels {
		if !absorbedSet[r.TargetID] && r.TargetID == canonicalID {
			seen[r.Relation+"|"+r.TargetID] = true
		}
	}
	out := make([]core.WikiEntityRelation, 0, len(rels))
	changed := false
	for _, r := range rels {
		if !absorbedSet[r.TargetID] {
			out = append(out, r)
			continue
		}
		changed = true
		r.TargetID = canonicalID
		if canonicalType != "" {
			r.TargetType = canonicalType
		}
		key := r.Relation + "|" + r.TargetID
		if seen[key] {
			continue // the holder already cites canonical via this relation
		}
		seen[key] = true
		out = append(out, r)
	}
	return out, changed
}

// aliasListSorted returns the sorted keys of an alias set.
func aliasListSorted(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for a := range set {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}
