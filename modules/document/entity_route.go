/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"fmt"
	"net/http"
	"strings"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/wiki"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Entity route (W10, RRF sixth leg): BM25 over the ontology entities
// themselves — name, aliases, type. Complements the graph leg (relation
// traversal from recognized seeds): the graph leg answers "what is
// connected to X", this leg answers "X itself" without requiring any
// relation to exist. Only reviewed/published entities enter retrieval,
// mirroring the wiki noise gate: a proposed entity is a draft, not
// knowledge.

var errEntityRouteSkipped = fmt.Errorf("entity route skipped: no wiki article search permission")

// entityRecallWindow bounds how many entities the fusion leg may carry:
// entities answer "X itself" — a handful of genuine name/alias matches —
// and a wide window would promote weak combined_fulltext mentions of
// barely-related entities into the fused list. Precision over recall; the
// graph leg covers the "related to X" territory.
const entityRecallWindow = 3

// entityToHit wraps an entity as a search hit pseudo-document. Entities
// have no dedicated console page: an entity with a curated article links
// there, one without stays a card-only hit (the entity card standard
// contract renders it in place). typeLabels carries the ontology display
// labels resolved once per request — the row's entity_type_label drives the
// type badge on the search result list (absent → the frontend shows the raw
// type name).
func entityToHit(entity *core.WikiEntity, score float32, parent *orm.Context, typeLabels map[string]string) elastic.DocumentWithMeta[core.Document] {
	summary := ""
	if desc := strings.TrimSpace(entity.Description); desc != "" {
		summary = desc // the entity's own definition beats the alias list
	} else if len(entity.Aliases) > 0 {
		summary = fmt.Sprintf("aliases: %v", entity.Aliases)
	}
	doc := core.Document{
		Title:   entity.Name,
		Type:    "entity",
		Summary: summary,
		Source: core.DataSourceReference{
			ID:   "wiki",
			Name: "Wiki",
			Icon: "font_book",
		},
	}
	if entity.ArticleID != "" {
		doc.URL = fmt.Sprintf("%s/#/wiki/article/%s", appConfigFn().ServerInfo.Endpoint, entity.ArticleID)
	}
	// KB grouping (plan D dynamic scope): an entity may belong to a KB
	// explicitly (legacy kb_id rows, promote, manual writes) or through its
	// source documents being visible to the KB (pre-bind sync rows, which
	// carry no kb_id). The union feeds metadata.kb_ids; the frontend accepts
	// both the legacy single kb_id and the new list.
	kbIDs := []string{}
	if entity.KbID != "" {
		kbIDs = append(kbIDs, entity.KbID)
	}
	sourceDocIDs := make([]string, 0, len(entity.Sources))
	for _, s := range entity.Sources {
		if s.DocID != "" {
			sourceDocIDs = append(sourceDocIDs, s.DocID)
		}
	}
	for _, kbID := range core.KBsForDocs(parent, sourceDocIDs) {
		if kbID != entity.KbID {
			kbIDs = append(kbIDs, kbID)
		}
	}
	doc.Metadata = util.MapStr{
		"entity":        true,
		"entity_id":     entity.ID,
		"entity_type":   entity.Type,
		"entity_status": entity.Status,
		// lets the search page group entity hits under their KBs — the graph
		// panel scopes each KB's canvas to the KBs actually present in the
		// results, entity-only hits included
		"kb_id":  entity.KbID,
		"kb_ids": kbIDs,
	}
	// display label for the search list's type badge: the ontology schema's
	// localized label when declared, the raw type name otherwise
	if entity.Type != "" {
		if lbl := typeLabels[entity.Type]; lbl != "" {
			doc.Metadata["entity_type_label"] = lbl
		} else {
			doc.Metadata["entity_type_label"] = entity.Type
		}
	}
	if len(entity.Sources) > 0 {
		refs := make([]interface{}, 0, len(entity.Sources))
		for _, s := range entity.Sources {
			refs = append(refs, util.MapStr{
				"doc_id":  s.DocID,
				"title":   s.Title,
				"url":     s.URL,
				"excerpt": s.Excerpt,
				"locator": s.Locator,
			})
		}
		doc.Metadata["entity_sources"] = refs
	}
	return elastic.DocumentWithMeta[core.Document]{
		ID:     entity.ID,
		Index:  "wiki_entity",
		Source: doc,
		Score:  score,
	}
}

// entityRoute runs the entity recall leg under the wiki permission gate
// (entities are wiki-module assets; a caller allowed to search articles is
// allowed to meet entities).
func (h *APIHandler) entityRoute(req *http.Request, query string, window int) (*elastic.SearchResponseWithMeta[core.Document], error) {
	if !wikiSearchPermission(req) {
		return nil, errEntityRouteSkipped
	}

	octx := orm.NewContextWithParent(req.Context())
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.WikiEntity{})

	builder := orm.NewQuery().From(0).Size(window)
	builder.DefaultQueryField("name^20", "name.pinyin^8", "aliases^6", "subtype^2", "combined_fulltext")
	builder.SkipFuzziness()
	if err := applyCuratedTextQuery(builder, searchSettingsSnapshot(), query); err != nil {
		return nil, err
	}
	builder.Filter(
		orm.TermsQuery("status", []string{core.WikiEntityReviewed, core.WikiEntityPublished}),
	)
	// sources stays in the projection: the plan-D KB grouping resolves a
	// kb-less entity through its source document ids, and entity_sources
	// metadata exposes them — excluding it here silently emptied both.
	builder.Exclude("relations", "properties")

	var entities []core.WikiEntity
	err, _ := elastic.SearchV2WithResultItemMapper(octx, &entities, builder, nil)
	if err != nil {
		return nil, err
	}

	// entity type display labels for the result rows: tenant schema is the
	// baseline, each KB schema overrides — resolved once per request, not
	// per row (the window is tiny but the schema lookup is an ES query)
	typeLabels := map[string]string{}
	for t, lbl := range wiki.EntityTypeLabelsFor(req.Context(), "") {
		typeLabels[t] = lbl
	}
	seenKB := map[string]bool{}
	for i := range entities {
		if kb := entities[i].KbID; kb != "" && !seenKB[kb] {
			seenKB[kb] = true
			for t, lbl := range wiki.EntityTypeLabelsFor(req.Context(), kb) {
				typeLabels[t] = lbl
			}
		}
	}

	out := &elastic.SearchResponseWithMeta[core.Document]{}
	for i := range entities {
		e := &entities[i]
		// quality gate on top of the status gate: an entity with no
		// description is an unenriched shell (promoted name or extraction
		// that never merged) — recall noise, not knowledge. ES "exists"
		// can't express this: an empty string still counts as present,
		// so filter after decode.
		if strings.TrimSpace(e.Description) == "" {
			continue
		}
		out.Hits.Hits = append(out.Hits.Hits, entityToHit(e, float32(window-i), octx, typeLabels))
	}
	// plain map, not util.MapStr — GetTotal's type switch (the D2 lesson)
	out.Hits.Total = map[string]interface{}{"value": int64(len(out.Hits.Hits)), "relation": "eq"}
	return out, nil
}
