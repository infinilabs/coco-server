/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/log"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
)

// Knowledge-graph query face (ontology phase O3): one request returns the
// whole KB subgraph — article nodes, the entities their wikilinks resolve
// to, one-hop relation expansion, and schema-resolved edge labels including
// the declared inverse relation names. Read-only; feeds the KbGraph canvas
// and its entity cards.

const (
	graphMaxArticles     = 500
	graphMaxEntities     = 800
	graphMaxEdges        = 2000
	graphEntityChunkSize = 200 // ES terms-query safe batch
	// graphMaxQueryEntities caps the ?q= focus entity pull: the canvas
	// surfaces the query's own entities first, relations stay one hop.
	graphMaxQueryEntities = 8
	// graphQueryCitedArticles bounds the article pull when the scope comes
	// from the query alone (search-page transient states keep the panel
	// alive with article_ids empty — entity-only hits). Without the cap a
	// hub entity cited by hundreds of pages would drag the whole KB back
	// onto the canvas. Title-sorted first N keeps output stable.
	graphQueryCitedArticles = 12
	// graphQueryNeighborCap bounds the one-hop relation expansion of the
	// query-matched seeds: the transient canvas answers "what did I hit and
	// what directly touches it" — a hub's whole neighborhood would swell it
	// right back. Sorted-id truncation keeps output stable.
	graphQueryNeighborCap = 24
)

type GraphNode struct {
	ID          string                 `json:"id"`   // article:<aid> | entity:<eid> | lp:<type>:<name>
	Kind        string                 `json:"kind"` // article | entity | unresolved
	Label       string                 `json:"label"`
	Type        string                 `json:"type,omitempty"` // entity type, or the link type of an unresolved wikilink
	TypeLabel   string                 `json:"type_label,omitempty"`
	Status      string                 `json:"status,omitempty"`
	PageType    string                 `json:"page_type,omitempty"`  // articles only
	ArticleID   string                 `json:"article_id,omitempty"` // entity curated page
	Aliases     []string               `json:"aliases,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Confidence  float64                `json:"confidence,omitempty"`
	ViaRelation bool                   `json:"via_relation,omitempty"` // pulled in by one-hop expansion, not cited by any article
	QueryMatch  bool                   `json:"query_match,omitempty"`  // name/aliases matched the search query (?q=)
}

type GraphEdge struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	Kind         string `json:"kind"`               // wikilink | relation
	Relation     string `json:"relation,omitempty"` // raw relation name
	Label        string `json:"label,omitempty"`    // schema label, falls back to relation/type
	Inverse      string `json:"inverse,omitempty"`  // declared inverse relation name
	InverseLabel string `json:"inverse_label,omitempty"`
}

// kbGraph serves GET /wiki/kb/:id/_graph. Optional ?article_ids=a,b scopes
// the subgraph to those articles (the search results page passes the wiki
// hits of the current query, so the canvas reflects WHAT was searched, not
// just WHICH knowledge base) — each id still has to belong to this KB.
// Optional ?q= additionally pulls in the entities whose name/aliases match
// the query (wikilink-driven focus alone misses "search the thing itself":
// the voucher page may never wikilink the voucher entity). When the query is
// present but no hit articles are, the canvas is strictly one hop: the
// query-matched seeds, their direct typed-relation neighbors, and the
// articles that directly cite them — pulling every wikilink target of every
// citing article re-rendered the whole KB behind the query's back. The whole
// KB is served only when the caller passes no scope at all (the wiki app's
// graph page). Status is not filtered here — proposed drafts already enter
// the canvas via wikilinks.
func (h *APIHandler) kbGraph(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	var focusIDs []string
	for _, id := range strings.Split(req.URL.Query().Get("article_ids"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			focusIDs = append(focusIDs, id)
		}
	}
	query := strings.TrimSpace(req.URL.Query().Get("q"))
	// request audit: the graph endpoint is cheap but callers fetch it on every
	// search/query change — a one-line trace makes client-side fetch storms
	// and scoping mistakes visible in the server log
	log.Infof("wiki-graph: kb=%s articles=%v q=%q", kbID, focusIDs, query)
	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.WikiKnowledgeBase{})

	var kb core.WikiKnowledgeBase
	kb.SetID(kbID)
	exists, err := orm.GetV2(ctx, &kb)
	if err != nil && strings.Contains(err.Error(), "record not found") {
		// sqlite reports missing rows as an error (ES returns exists=false)
		err, exists = nil, false
	}
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, kbID)
		return
	}

	var focusEntityIDs []string
	if query != "" {
		if focusEntityIDs, err = kbEntityIDsByQuery(ctx, kbID, query); err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// query with neither entity matches nor hit articles: the query has no
	// wiki footprint in this KB — an empty graph hides the panel on the
	// search page instead of dumping an unscoped whole-KB canvas
	if query != "" && len(focusEntityIDs) == 0 && len(focusIDs) == 0 {
		h.WriteOKJSON(w, &knowledgeGraph{Nodes: []GraphNode{}, Edges: []GraphEdge{}})
		return
	}

	graph, err := buildKnowledgeGraph(req.Context(), kbID, focusIDs, focusEntityIDs)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, graph)
}

type knowledgeGraph struct {
	Nodes       []GraphNode            `json:"nodes"`
	Edges       []GraphEdge            `json:"edges"`
	EntityTypes map[string]util.MapStr `json:"entity_types,omitempty"`
}

// graphSearchConfig reads the tenant config for the graph query focus —
// indirect through a recover wrapper because common.AppConfig touches the
// kv store, which panics in unit tests without a kv handler (same reason
// projection.go swaps projectionConfigFn in tests).
var graphSearchConfig = func() (cfg core.Config) {
	defer func() { _ = recover() }()
	return common.AppConfig()
}

// kbEntityIDsByQuery resolves the entities of one KB whose name or aliases
// match the query text (the search page's ?q=). Match semantics follow the
// configured search match_operator — the focus answers "the searched thing
// itself", junk OR-matches would only crowd the 8-slot canvas. name.text /
// name.pinyin are queried before combined_fulltext so kb-less rows (which
// never got the fulltext sink backfilled) still resolve.
//
// Dynamic scope (plan D): legacy rows pin kb_id at write time; kb-less rows
// belong through their source documents (DocIDsForKB). The two memberships
// are unioned in the application — the framework bool renders should clauses
// as score-only boosts when must/filter coexist, so a DSL-level OR is not
// expressible without touching the framework.
func kbEntityIDsByQuery(ctx *orm.Context, kbID, query string) ([]string, error) {
	readCtx := orm.NewContextWithParent(ctx)
	readCtx.DirectReadAccess()
	orm.WithModel(readCtx, &core.WikiEntity{})

	// name.text/name.pinyin first: kb-less entities written by
	// extract_entities carry no combined_fulltext (that sink is only
	// backfilled by the index rebuild), so name-only matches must not
	// depend on it. combined_fulltext covers alias/subtype text matches.
	clauses, err := orm.BuildFuzzinessQueryClauses(query, 0, []string{"name.text", "name.pinyin", "combined_fulltext"})
	if err != nil {
		return nil, err
	}
	if cfg := graphSearchConfig(); cfg.SearchSettings != nil && cfg.SearchSettings.MatchOperatorVal() == "AND" {
		for _, c := range clauses {
			if c.IsLeaf() && c.Operator == orm.QueryMatch {
				c.Parameter("operator", "AND")
			}
		}
	}

	scope := core.KBDocIDsFor(ctx, kbID)

	// One score-ranked scan, membership filtered in the app: the entity
	// index stores sources with enabled:false — sources.doc_id is NOT
	// queryable — so dynamic attribution cannot be an ES filter. Explicit
	// kb_id rows and kb-less rows compete on the same match score (same
	// index, same clauses), so a strong name match like a searched kb-less
	// entity outranks weak combined_fulltext mentions instead of being
	// starved by an ES-level kb_id leg that always fills the slots first.
	entities, err := queryGraphEntities(readCtx, clauses, graphQueryEntityScan)
	if err != nil {
		return nil, err
	}
	scopeSet := make(map[string]bool, len(scope))
	for _, id := range scope {
		scopeSet[id] = true
	}
	seen := map[string]bool{}
	ids := make([]string, 0, graphMaxQueryEntities)
	for i := range entities {
		e := &entities[i]
		if seen[e.ID] {
			continue
		}
		owns := e.KbID == kbID
		if !owns && e.KbID == "" {
			// kb-less rows belong through their source documents; rows
			// scoped to a DIFFERENT KB never enter this canvas
			for _, src := range e.Sources {
				if scopeSet[src.DocID] {
					owns = true
					break
				}
			}
		}
		if !owns {
			continue
		}
		seen[e.ID] = true
		ids = append(ids, e.ID)
		if len(ids) >= graphMaxQueryEntities {
			break
		}
	}
	return ids, nil
}

// graphQueryEntityScan bounds the app-side membership scan: enough
// top-scored rows to cover hot-word matches before the filter, cheap at
// this corpus size.
const graphQueryEntityScan = 60

func queryGraphEntities(ctx *orm.Context, clauses []*orm.Clause, size int) ([]core.WikiEntity, error) {
	builder := orm.NewQuery().From(0).Size(size)
	builder.SkipFuzziness()
	if len(clauses) == 1 {
		builder.Must(clauses[0])
	} else {
		// multiple fields/fuzziness tiers: per BuildFuzzinessQueryClauses
		// contract these are alternatives, must-match would demand every
		// field to hit
		builder.Must(orm.ShouldQuery(clauses...))
	}

	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return nil, err
	}
	entities, _, derr := elastic.DecodeHits[core.WikiEntity](res)
	if derr != nil {
		return nil, derr
	}
	return entities, nil
}

// buildKnowledgeGraph assembles the KB subgraph: articles and their
// wikilink-resolved entities form the core; typed relations expand the
// canvas by one hop. focusIDs narrows the article set (search-results
// mode: the whole graph then hangs off the actually-hit articles); nil
// pulls every article in the KB. focusEntityIDs pins query-matched
// entities onto the same canvas (they carry no wikilink from any article).
// The KB's ontology schema resolves display labels and inverse relation
// names; without a schema the raw vocabulary is used.
func buildKnowledgeGraph(ctx context.Context, kbID string, focusIDs []string, focusEntityIDs []string) (*knowledgeGraph, error) {
	readCtx := orm.NewContextWithParent(ctx)
	readCtx.DirectReadAccess()

	var articles []core.WikiArticle
	var err error
	if len(focusIDs) > 0 {
		articles, err = kbArticlesForGraphIDs(readCtx, kbID, focusIDs)
	} else if len(focusEntityIDs) > 0 {
		// query-only scope (search page, entity-only hits): the canvas
		// hangs off the articles that cite the query-matched entities —
		// never the whole KB, or the panel would show everything and
		// nothing about what was searched
		articles, err = kbArticlesCitingEntities(readCtx, kbID, focusEntityIDs)
	} else {
		articles, err = kbArticlesForGraph(readCtx, kbID)
	}
	if err != nil {
		return nil, err
	}

	schema := loadOntologySchemaForKB(ctx, kbID)
	typeDefs := map[string]*OntologyEntityTypeDef{}
	if schema != nil {
		for i := range schema.EntityTypes {
			typeDefs[schema.EntityTypes[i].Name] = &schema.EntityTypes[i]
		}
	}
	typeLabel := func(t string) string {
		if def := typeDefs[t]; def != nil && def.Label != "" {
			return def.Label
		}
		return ""
	}

	// entities the KB's wikilinks resolve to; in query-only mode the canvas
	// is the seeds plus their direct relation neighbors instead
	citedIDs := map[string]bool{}
	for i := range articles {
		for _, lp := range articles[i].LinkedPages {
			if lp.EntityID != "" {
				citedIDs[lp.EntityID] = true
			}
		}
	}
	queryOnly := len(focusIDs) == 0 && len(focusEntityIDs) > 0
	queryMatch := map[string]bool{}
	var entities []core.WikiEntity
	if queryOnly {
		// Search-page transient state (entity-only hits): the canvas is the
		// matched entities plus ONE hop — their typed-relation targets and
		// the articles that directly cite them. The old scope (every
		// wikilink target of every citing article) dragged ~30 articles'
		// whole entity sets back: q=coco rendered the KB's full graph,
		// which reads as "the panel ignores the query".
		seedSet := make(map[string]bool, len(focusEntityIDs))
		for _, id := range focusEntityIDs {
			seedSet[id] = true
		}
		seeds, serr := loadEntitiesCapped(readCtx, seedSet, len(seedSet))
		if serr != nil {
			return nil, serr
		}
		neighborIDs := map[string]bool{}
		for i := range seeds {
			for _, rel := range seeds[i].Relations {
				if rel.TargetID != "" && !seedSet[rel.TargetID] {
					neighborIDs[rel.TargetID] = true
				}
			}
		}
		if len(neighborIDs) > graphQueryNeighborCap {
			keys := make([]string, 0, len(neighborIDs))
			for id := range neighborIDs {
				keys = append(keys, id)
			}
			sort.Strings(keys)
			for _, id := range keys[graphQueryNeighborCap:] {
				delete(neighborIDs, id)
			}
		}
		neighbors, nerr := loadEntitiesCapped(readCtx, neighborIDs, len(neighborIDs))
		if nerr != nil {
			return nil, nerr
		}
		entities = append(entities, seeds...)
		entities = append(entities, neighbors...)
		// badge seeds no pulled article wikilinks (the "search the thing
		// itself" face); citedIDs stays wikilink-only so sort priority and
		// ViaRelation keep their article-grounded meaning
		for _, id := range focusEntityIDs {
			if !citedIDs[id] {
				queryMatch[id] = true
			}
		}
	} else {
		// query-matched entities join the canvas at the same priority as
		// wikilink-cited ones; the flag tells the frontend to badge them
		for _, id := range focusEntityIDs {
			if !citedIDs[id] {
				queryMatch[id] = true
			}
			citedIDs[id] = true
		}
		var lerr error
		entities, lerr = loadEntitiesCapped(readCtx, citedIDs, graphMaxEntities)
		if lerr != nil {
			return nil, lerr
		}
	}
	entityByID := map[string]*core.WikiEntity{}
	for i := range entities {
		entityByID[entities[i].ID] = &entities[i]
	}

	// one-hop relation expansion: query-only canvases are already exactly
	// one hop (seeds + direct neighbors), expanding again would grow the
	// transient canvas to two hops behind the user's back
	if !queryOnly {
		expandedIDs := map[string]bool{}
		for _, entity := range entities {
			for _, rel := range entity.Relations {
				if rel.TargetID != "" && entityByID[rel.TargetID] == nil {
					expandedIDs[rel.TargetID] = true
				}
			}
		}
		if remaining := graphMaxEntities - len(entities); remaining > 0 && len(expandedIDs) > 0 {
			expanded, err := loadEntitiesCapped(readCtx, expandedIDs, remaining)
			if err != nil {
				return nil, err
			}
			for i := range expanded {
				if entityByID[expanded[i].ID] == nil {
					entityByID[expanded[i].ID] = &expanded[i]
					entities = append(entities, expanded[i])
				}
			}
		}
	}

	graph := &knowledgeGraph{Nodes: []GraphNode{}, Edges: []GraphEdge{}}

	// article nodes
	articleNodeIDs := map[string]bool{}
	for i := range articles {
		article := &articles[i]
		id := "article:" + article.ID
		articleNodeIDs[id] = true
		graph.Nodes = append(graph.Nodes, GraphNode{
			ID:       id,
			Kind:     "article",
			Label:    article.Title,
			Status:   article.Status,
			PageType: article.PageType,
		})
	}

	// entity nodes (stable order: cited first, then expansion, name-sorted)
	sort.SliceStable(entities, func(i, j int) bool {
		ci, cj := citedIDs[entities[i].ID], citedIDs[entities[j].ID]
		if ci != cj {
			return ci
		}
		if entities[i].Name != entities[j].Name {
			return entities[i].Name < entities[j].Name
		}
		return entities[i].ID < entities[j].ID
	})
	for i := range entities {
		entity := &entities[i]
		node := GraphNode{
			ID:          "entity:" + entity.ID,
			Kind:        "entity",
			Label:       entity.Name,
			Type:        entity.Type,
			TypeLabel:   typeLabel(entity.Type),
			Status:      entity.Status,
			Aliases:     entity.Aliases,
			Properties:  entity.Properties,
			Confidence:  entity.Confidence,
			ArticleID:   entity.ArticleID,
			ViaRelation: !citedIDs[entity.ID],
			QueryMatch:  queryMatch[entity.ID],
		}
		graph.Nodes = append(graph.Nodes, node)
	}

	// entity type metadata for the frontend (labels + icons)
	if len(typeDefs) > 0 {
		graph.EntityTypes = map[string]util.MapStr{}
		for name, def := range typeDefs {
			meta := util.MapStr{}
			if def.Label != "" {
				meta["label"] = def.Label
			}
			if def.Icon != "" {
				meta["icon"] = def.Icon
			}
			graph.EntityTypes[name] = meta
		}
	}

	edgeSet := map[string]bool{}
	addEdge := func(edge GraphEdge) {
		if len(graph.Edges) >= graphMaxEdges {
			return
		}
		key := edge.Source + "\x00" + edge.Target + "\x00" + edge.Relation
		if edgeSet[key] {
			return
		}
		edgeSet[key] = true
		graph.Edges = append(graph.Edges, edge)
	}

	// wikilink edges: article -> resolved entity or unresolved placeholder
	unresolvedSet := map[string]bool{}
	for i := range articles {
		article := &articles[i]
		source := "article:" + article.ID
		for _, lp := range article.LinkedPages {
			if lp.EntityID != "" && entityByID[lp.EntityID] != nil {
				addEdge(GraphEdge{
					Source:   source,
					Target:   "entity:" + lp.EntityID,
					Kind:     "wikilink",
					Label:    typeLabel(lp.Type),
					Relation: lp.Type,
				})
				continue
			}
			if queryOnly {
				// strict one-hop canvas: placeholders for wikilinks to
				// entities off the canvas are noise on the search panel
				continue
			}
			target := "lp:" + lp.Type + ":" + lp.Name
			if !unresolvedSet[target] {
				unresolvedSet[target] = true
				graph.Nodes = append(graph.Nodes, GraphNode{
					ID:        target,
					Kind:      "unresolved",
					Label:     lp.Name,
					Type:      lp.Type,
					TypeLabel: typeLabel(lp.Type),
				})
			}
			addEdge(GraphEdge{
				Source:   source,
				Target:   target,
				Kind:     "wikilink",
				Label:    typeLabel(lp.Type),
				Relation: lp.Type,
			})
		}
	}

	// typed relation edges between entities on the canvas
	for i := range entities {
		entity := &entities[i]
		for _, rel := range entity.Relations {
			if rel.TargetID == "" || entityByID[rel.TargetID] == nil || rel.TargetID == entity.ID {
				continue
			}
			edge := GraphEdge{
				Source:   "entity:" + entity.ID,
				Target:   "entity:" + rel.TargetID,
				Kind:     "relation",
				Relation: rel.Relation,
			}
			decorateRelationEdge(&edge, typeDefs, entity.Type, rel.Relation)
			addEdge(edge)
		}
	}

	return graph, nil
}

// decorateRelationEdge resolves display labels and the declared inverse
// relation from the source type's relation definition.
func decorateRelationEdge(edge *GraphEdge, typeDefs map[string]*OntologyEntityTypeDef, sourceType, relation string) {
	edge.Label = relation
	def := typeDefs[sourceType]
	if def == nil {
		return
	}
	for i := range def.Relations {
		rel := &def.Relations[i]
		if rel.Name != relation {
			continue
		}
		if rel.Label != "" {
			edge.Label = rel.Label
		}
		if rel.Inverse != "" {
			edge.Inverse = rel.Inverse
			edge.InverseLabel = rel.Inverse
			if targetDef := typeDefs[rel.TargetType]; targetDef != nil {
				for j := range targetDef.Relations {
					if targetDef.Relations[j].Name == rel.Inverse && targetDef.Relations[j].Label != "" {
						edge.InverseLabel = targetDef.Relations[j].Label
					}
				}
			}
		}
		return
	}
}

func kbArticlesForGraph(ctx *orm.Context, kbID string) ([]core.WikiArticle, error) {
	orm.WithModel(ctx, &core.WikiArticle{})
	builder := orm.NewQuery().Size(graphMaxArticles).
		Filter(orm.TermQuery("kb_id", kbID)).
		SortBy(orm.Sort{Field: "title", SortType: orm.ASC})
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return nil, err
	}
	articles, _, err := elastic.DecodeHits[core.WikiArticle](res)
	return articles, err
}

// kbArticlesForGraphIDs loads only the named articles of one KB — the
// search-results scope. Ids that don't exist or belong to another KB are
// silently dropped (an empty result yields an empty graph, never an error).
func kbArticlesForGraphIDs(ctx *orm.Context, kbID string, ids []string) ([]core.WikiArticle, error) {
	orm.WithModel(ctx, &core.WikiArticle{})
	builder := orm.NewQuery().Size(graphMaxArticles).
		Filter(orm.TermQuery("kb_id", kbID), orm.TermsQuery("id", ids))
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return nil, err
	}
	articles, _, err := elastic.DecodeHits[core.WikiArticle](res)
	return articles, err
}

// kbArticlesCitingEntities keeps only the KB articles whose wikilinks
// resolve to one of the given entities — the query-only scope for the
// search page, where the canvas must hang off WHAT was searched. LinkedPages
// is stored inline and unindexed, so the filter runs in the application over
// the (bounded) article list; graphQueryCitedArticles caps the worst case of
// a hub entity cited by most of the KB.
func kbArticlesCitingEntities(ctx *orm.Context, kbID string, entityIDs []string) ([]core.WikiArticle, error) {
	articles, err := kbArticlesForGraph(ctx, kbID)
	if err != nil {
		return nil, err
	}
	want := make(map[string]bool, len(entityIDs))
	for _, id := range entityIDs {
		want[id] = true
	}
	out := make([]core.WikiArticle, 0, len(articles))
	for i := range articles {
		for _, lp := range articles[i].LinkedPages {
			if lp.EntityID != "" && want[lp.EntityID] {
				out = append(out, articles[i])
				break
			}
		}
	}
	if len(out) > graphQueryCitedArticles {
		out = out[:graphQueryCitedArticles]
	}
	return out, nil
}

// loadEntitiesCapped fetches entities by id in terms-query-safe chunks.
func loadEntitiesCapped(ctx *orm.Context, ids map[string]bool, limit int) ([]core.WikiEntity, error) {
	if len(ids) == 0 || limit <= 0 {
		return []core.WikiEntity{}, nil
	}
	orm.WithModel(ctx, &core.WikiEntity{})

	out := make([]core.WikiEntity, 0, len(ids))
	idList := make([]string, 0, len(ids))
	for id := range ids {
		idList = append(idList, id)
	}
	sort.Strings(idList)
	if len(idList) > limit {
		idList = idList[:limit]
	}
	for start := 0; start < len(idList); start += graphEntityChunkSize {
		end := start + graphEntityChunkSize
		if end > len(idList) {
			end = len(idList)
		}
		builder := orm.NewQuery().Size(end - start).
			Filter(orm.TermsQuery("id", idList[start:end]))
		res, err := orm.SearchV2(ctx, builder)
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

// entityInverseEdges scans the entity store for relations pointing at one
// entity — relations live inline and unindexed, so the reverse direction is
// a client-side filter over a bounded scan. Bounded by graphMaxEntities;
// called by the neighbors endpoint so entity cards and assistants see who
// points at an entity, labeled with the schema's inverse relation names.
func entityInverseEdges(ctx context.Context, entity *core.WikiEntity) []util.MapStr {
	readCtx := orm.NewContextWithParent(ctx)
	readCtx.DirectReadAccess()
	orm.WithModel(readCtx, &core.WikiEntity{})

	res, err := orm.SearchV2(readCtx, orm.NewQuery().Size(graphMaxEntities))
	if err != nil {
		seelog.Warnf("wiki: inverse relation scan failed: %v", err)
		return nil
	}
	others, _, err := elastic.DecodeHits[core.WikiEntity](res)
	if err != nil {
		return nil
	}

	schema := loadOntologySchemaForKB(ctx, "")
	typeDefs := map[string]*OntologyEntityTypeDef{}
	if schema != nil {
		for i := range schema.EntityTypes {
			typeDefs[schema.EntityTypes[i].Name] = &schema.EntityTypes[i]
		}
	}

	incoming := []util.MapStr{}
	for i := range others {
		other := &others[i]
		if other.ID == entity.ID {
			continue
		}
		for _, rel := range other.Relations {
			if rel.TargetID != entity.ID {
				continue
			}
			entry := util.MapStr{
				"source_id":   other.ID,
				"source_name": other.Name,
				"source_type": other.Type,
				"relation":    rel.Relation,
				"label":       rel.Relation,
			}
			edge := GraphEdge{Label: rel.Relation}
			decorateRelationEdge(&edge, typeDefs, other.Type, rel.Relation)
			entry["label"] = edge.Label
			if edge.Inverse != "" {
				entry["inverse"] = edge.Inverse
				entry["inverse_label"] = edge.InverseLabel
			}
			incoming = append(incoming, entry)
			break // one edge per source entity keeps cards readable
		}
	}
	return incoming
}
