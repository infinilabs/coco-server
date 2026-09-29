/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Graph recall leg (D3): recognize entities in the query against the
// ontology vocabulary (names + aliases, zero LLM), traverse their typed
// relations one hop in both directions, and surface the curated pages of the
// seed entities and their neighbors. The graph answers queries the text
// routes structurally cannot — "what fails when the payment service is
// down" matches nothing by keywords but walks payment_service -> causes ->
// incident pages.

const (
	// graphRouteMaxEntities bounds the vocabulary scan: relations live inline
	// and unindexed, so recognition reads the entity store directly. Same
	// trade-off as the wiki module's graph canvas.
	graphRouteMaxEntities  = 1000
	graphRouteMaxSeeds     = 5
	graphRouteMaxNeighbors = 50
	// graphRouteMinTermLen keeps single-character names (which would match
	// almost any query) out of recognition.
	graphRouteMinTermLen = 2
	// graphRouteChunkSize is the ES terms-query safe batch, same as the wiki
	// graph module.
	graphRouteChunkSize = 200
)

// errGraphRouteSkipped is returned when the graph leg cannot run; callers
// treat it like any failing route — fusion continues with the rest.
var errGraphRouteSkipped = fmt.Errorf("graph route skipped: no wiki article search permission")

// graphSeed is an entity the query text mentions, plus the name/alias term
// that matched.
type graphSeed struct {
	Entity      core.WikiEntity
	MatchedTerm string
}

// graphNeighborRef is one entity pulled in by traversing a single relation
// edge from a seed; Paths counts the distinct edges that reached it.
type graphNeighborRef struct {
	EntityID string
	FromID   string
	FromName string
	Relation string
	Paths    int
}

// graphArticleRef is one candidate page on the graph leg: either a seed
// entity's own page, or a neighbor's page reached through a relation.
type graphArticleRef struct {
	articleID string
	via       *graphNeighborRef
}

// entityEligibleForGraph: proposed entities are drafts on the write side
// (the wiki workflow promotes them on review/publish); retrieval only sees
// the reviewed-and-published layer, mirroring passesWikiNoiseGate.
func entityEligibleForGraph(entity *core.WikiEntity) bool {
	return entity.Status == core.WikiEntityReviewed || entity.Status == core.WikiEntityPublished
}

// recognizeGraphSeeds finds the entities the query mentions: a name or alias
// appearing as a substring of the (lowercased) query. Substring containment,
// not tokenized matching, is the recognition semantic — the entity must be
// literally present in the query. When several terms of one entity match,
// the longest wins (more specific); seeds sort longest-match-first so the
// most specific entity anchors the traversal.
func recognizeGraphSeeds(query string, entities []core.WikiEntity) []graphSeed {
	lowerQuery := strings.ToLower(query)
	seeds := []graphSeed{}
	for i := range entities {
		entity := &entities[i]
		if !entityEligibleForGraph(entity) {
			continue
		}
		best := ""
		for _, term := range append([]string{entity.Name}, entity.Aliases...) {
			if utf8.RuneCountInString(term) < graphRouteMinTermLen {
				continue
			}
			if strings.Contains(lowerQuery, strings.ToLower(term)) && utf8.RuneCountInString(term) > utf8.RuneCountInString(best) {
				best = term
			}
		}
		if best != "" {
			seeds = append(seeds, graphSeed{Entity: *entity, MatchedTerm: best})
		}
	}
	sort.SliceStable(seeds, func(i, j int) bool {
		li, lj := utf8.RuneCountInString(seeds[i].MatchedTerm), utf8.RuneCountInString(seeds[j].MatchedTerm)
		if li != lj {
			return li > lj
		}
		if seeds[i].Entity.Name != seeds[j].Entity.Name {
			return seeds[i].Entity.Name < seeds[j].Entity.Name
		}
		return seeds[i].Entity.ID < seeds[j].Entity.ID
	})
	if len(seeds) > graphRouteMaxSeeds {
		seeds = seeds[:graphRouteMaxSeeds]
	}
	return seeds
}

// graphExpansion walks one hop from the seeds over typed relations, both
// directions: forward along each seed's Relations, and inverse from pool
// entities whose relations point at a seed (relations are stored inline and
// unindexed, so the inverse direction is a filter over the same bounded
// scan). Distinct edges are counted per target; neighbors with more paths
// to the seeds rank higher.
func graphExpansion(seeds []graphSeed, pool []core.WikiEntity) []graphNeighborRef {
	seedEntities := map[string]*core.WikiEntity{}
	for i := range seeds {
		seedEntities[seeds[i].Entity.ID] = &seeds[i].Entity
	}

	byID := map[string]*graphNeighborRef{}
	order := []string{}
	add := func(targetID, fromID, fromName, relation string) {
		if targetID == "" || seedEntities[targetID] != nil || targetID == fromID {
			return
		}
		ref, ok := byID[targetID]
		if !ok {
			ref = &graphNeighborRef{EntityID: targetID}
			byID[targetID] = ref
			order = append(order, targetID)
		}
		ref.Paths++
		if ref.FromID == "" {
			ref.FromID, ref.FromName, ref.Relation = fromID, fromName, relation
		}
	}

	// forward: along each seed's own relations
	for i := range seeds {
		entity := &seeds[i].Entity
		for _, rel := range entity.Relations {
			add(rel.TargetID, entity.ID, entity.Name, rel.Relation)
		}
	}
	// inverse: a pool entity pointing at a seed is a neighbor too, reached
	// from that seed over the edge's declared relation
	for i := range pool {
		entity := &pool[i]
		if seedEntities[entity.ID] != nil {
			continue
		}
		for _, rel := range entity.Relations {
			if seed := seedEntities[rel.TargetID]; seed != nil {
				add(entity.ID, seed.ID, seed.Name, rel.Relation)
			}
		}
	}

	out := make([]graphNeighborRef, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Paths != out[j].Paths {
			return out[i].Paths > out[j].Paths
		}
		return out[i].EntityID < out[j].EntityID
	})
	if len(out) > graphRouteMaxNeighbors {
		out = out[:graphRouteMaxNeighbors]
	}
	return out
}

// graphRoute runs the graph recall leg: recognize seed entities, expand one
// hop over relations, and return the curated pages of seeds and neighbors as
// pseudo-document hits (wrapped like the wiki leg, plus graph provenance in
// metadata). The note summarizes what was recognized — the studio surfaces
// it so users see why the graph leg fired (or didn't).
func (h *APIHandler) graphRoute(req *http.Request, query string, window int) (*elastic.SearchResponseWithMeta[core.Document], string, error) {
	if !wikiSearchPermission(req) {
		return nil, "", errGraphRouteSkipped
	}

	octx := orm.NewContextWithParent(req.Context())
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.WikiEntity{})

	res, err := orm.SearchV2(octx, orm.NewQuery().Size(graphRouteMaxEntities).
		Include("id", "name", "aliases", "type", "status", "article_id", "relations"))
	if err != nil {
		return nil, "", err
	}
	entities, _, err := elastic.DecodeHits[core.WikiEntity](res)
	if err != nil {
		return nil, "", err
	}

	seeds := recognizeGraphSeeds(query, entities)
	if len(seeds) == 0 {
		out := &elastic.SearchResponseWithMeta[core.Document]{}
		out.Hits.Total = elastic.NewGeneralTotal(0)
		return out, "no entity recognized in query", nil
	}

	neighbors := graphExpansion(seeds, entities)

	// neighbor entities outside the scanned pool still need their article
	// link: fetch the missing ones by id in terms-safe chunks.
	poolByID := map[string]*core.WikiEntity{}
	for i := range entities {
		poolByID[entities[i].ID] = &entities[i]
	}
	missing := []string{}
	for _, n := range neighbors {
		if poolByID[n.EntityID] == nil {
			missing = append(missing, n.EntityID)
		}
	}
	if len(missing) > 0 {
		fetched, err := loadGraphNeighbors(octx, missing)
		if err != nil {
			log.Warnf("hybrid_rrf: graph neighbor fetch failed: %v", err)
		}
		for i := range fetched {
			poolByID[fetched[i].ID] = &fetched[i]
		}
	}

	// ranked article candidates: seed pages first (most specific match
	// first), then neighbor pages by path count.
	candidates := []graphArticleRef{}
	for i := range seeds {
		if seeds[i].Entity.ArticleID != "" {
			candidates = append(candidates, graphArticleRef{articleID: seeds[i].Entity.ArticleID})
		}
	}
	for i := range neighbors {
		entity := poolByID[neighbors[i].EntityID]
		if entity == nil || entity.ArticleID == "" {
			continue
		}
		candidates = append(candidates, graphArticleRef{articleID: entity.ArticleID, via: &neighbors[i]})
	}
	seenArticle := map[string]bool{}
	refs := []graphArticleRef{}
	for _, c := range candidates {
		if seenArticle[c.articleID] || len(refs) >= window {
			continue
		}
		seenArticle[c.articleID] = true
		refs = append(refs, c)
	}

	out := &elastic.SearchResponseWithMeta[core.Document]{}
	if len(refs) == 0 {
		out.Hits.Total = elastic.NewGeneralTotal(0)
		return out, graphRouteNote(seeds), nil
	}

	articles, err := loadGraphArticles(octx, refs)
	if err != nil {
		return nil, "", err
	}

	seedMeta := make([]interface{}, 0, len(seeds))
	for i := range seeds {
		seedMeta = append(seedMeta, util.MapStr{
			"entity_id":    seeds[i].Entity.ID,
			"entity_name":  seeds[i].Entity.Name,
			"entity_type":  seeds[i].Entity.Type,
			"matched_term": seeds[i].MatchedTerm,
		})
	}

	rules := loadCompileRules(req.Context())
	rank := 0
	for _, ref := range refs {
		article := articles[ref.articleID]
		if article == nil || !passesWikiNoiseGate(article, rules.ConceptMinSources) {
			continue
		}
		hit := wikiArticleToHit(article, float32(window-rank))
		neighborName := ""
		if ref.via != nil {
			if neighbor := poolByID[ref.via.EntityID]; neighbor != nil {
				neighborName = neighbor.Name
			}
		}
		decorateGraphHit(&hit, seedMeta, ref.via, neighborName)
		out.Hits.Hits = append(out.Hits.Hits, hit)
		rank++
	}
	out.Hits.Total = elastic.NewGeneralTotal(int64(len(out.Hits.Hits)))
	return out, graphRouteNote(seeds), nil
}

// decorateGraphHit stamps graph provenance onto a wrapped wiki hit: the
// recognized seeds always travel along, and neighbor hits carry the edge
// they were reached through.
func decorateGraphHit(hit *elastic.DocumentWithMeta[core.Document], seedMeta []interface{}, via *graphNeighborRef, neighborName string) {
	if hit.Source.Metadata == nil {
		hit.Source.Metadata = util.MapStr{}
	}
	hit.Source.Metadata["graph_route"] = true
	hit.Source.Metadata["graph_seeds"] = seedMeta
	if via != nil {
		meta := util.MapStr{
			"entity_id":   via.EntityID,
			"relation":    via.Relation,
			"from_entity": via.FromName,
		}
		if neighborName != "" {
			meta["entity_name"] = neighborName
		}
		hit.Source.Metadata["graph_via"] = meta
	}
}

func graphRouteNote(seeds []graphSeed) string {
	names := make([]string, 0, len(seeds))
	for i := range seeds {
		names = append(names, seeds[i].Entity.Name)
	}
	return fmt.Sprintf("seeds: %s", strings.Join(names, ", "))
}

// loadGraphNeighbors fetches neighbor entities (id/name/type/article_id) in
// terms-query-safe chunks.
func loadGraphNeighbors(octx *orm.Context, ids []string) ([]core.WikiEntity, error) {
	orm.WithModel(octx, &core.WikiEntity{})
	out := make([]core.WikiEntity, 0, len(ids))
	for start := 0; start < len(ids); start += graphRouteChunkSize {
		end := start + graphRouteChunkSize
		if end > len(ids) {
			end = len(ids)
		}
		res, err := orm.SearchV2(octx, orm.NewQuery().Size(end-start).
			Filter(orm.TermsQuery("id", ids[start:end])).
			Include("id", "name", "type", "article_id"))
		if err != nil {
			return nil, err
		}
		hits, _, err := elastic.DecodeHits[core.WikiEntity](res)
		if err != nil {
			return nil, err
		}
		out = append(out, hits...)
	}
	return out, nil
}

// loadGraphArticles fetches the candidate pages (only the fields the noise
// gate and the hit wrapper read) in terms-query-safe chunks, keyed by id.
func loadGraphArticles(octx *orm.Context, refs []graphArticleRef) (map[string]*core.WikiArticle, error) {
	orm.WithModel(octx, &core.WikiArticle{})
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.articleID)
	}
	out := map[string]*core.WikiArticle{}
	for start := 0; start < len(ids); start += graphRouteChunkSize {
		end := start + graphRouteChunkSize
		if end > len(ids) {
			end = len(ids)
		}
		res, err := orm.SearchV2(octx, orm.NewQuery().Size(end-start).
			Filter(orm.TermsQuery("id", ids[start:end])).
			Include("id", "title", "summary", "kb_id", "page_type", "status", "confidence", "sources"))
		if err != nil {
			return nil, err
		}
		hits, _, err := elastic.DecodeHits[core.WikiArticle](res)
		if err != nil {
			return nil, err
		}
		for i := range hits {
			out[hits[i].ID] = &hits[i]
		}
	}
	return out, nil
}
