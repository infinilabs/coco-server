/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"fmt"
	"net/http"

	"infini.sh/coco/core"
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

// entityToHit wraps an entity as a search hit pseudo-document. Entities
// have no dedicated console page: an entity with a curated article links
// there, one without stays a card-only hit (the entity card standard
// contract renders it in place).
func entityToHit(entity *core.WikiEntity, score float32) elastic.DocumentWithMeta[core.Document] {
	summary := ""
	if len(entity.Aliases) > 0 {
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
	doc.Metadata = util.MapStr{
		"entity":        true,
		"entity_id":     entity.ID,
		"entity_type":   entity.Type,
		"entity_status": entity.Status,
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
	builder.Query(query)
	builder.DefaultQueryField("name^20", "name.pinyin^8", "aliases^6", "subtype^2", "combined_fulltext")
	builder.Filter(
		orm.TermsQuery("status", []string{core.WikiEntityReviewed, core.WikiEntityPublished}),
	)
	builder.Exclude("relations", "sources", "properties")

	var entities []core.WikiEntity
	err, _ := elastic.SearchV2WithResultItemMapper(octx, &entities, builder, nil)
	if err != nil {
		return nil, err
	}

	out := &elastic.SearchResponseWithMeta[core.Document]{}
	for i := range entities {
		out.Hits.Hits = append(out.Hits.Hits, entityToHit(&entities[i], float32(window-i)))
	}
	// plain map, not util.MapStr — GetTotal's type switch (the D2 lesson)
	out.Hits.Total = map[string]interface{}{"value": int64(len(out.Hits.Hits)), "relation": "eq"}
	return out, nil
}
