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
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
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

// kbGraph serves GET /wiki/kb/:id/_graph.
func (h *APIHandler) kbGraph(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
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

	graph, err := buildKnowledgeGraph(req.Context(), kbID)
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

// buildKnowledgeGraph assembles the KB subgraph: articles and their
// wikilink-resolved entities form the core; typed relations expand the
// canvas by one hop. The KB's ontology schema resolves display labels and
// inverse relation names; without a schema the raw vocabulary is used.
func buildKnowledgeGraph(ctx context.Context, kbID string) (*knowledgeGraph, error) {
	readCtx := orm.NewContextWithParent(ctx)
	readCtx.Set(orm.DirectReadWithoutPermissionCheck, true)

	articles, err := kbArticlesForGraph(readCtx, kbID)
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

	// entities the KB's wikilinks resolve to, then one-hop relation targets
	citedIDs := map[string]bool{}
	for i := range articles {
		for _, lp := range articles[i].LinkedPages {
			if lp.EntityID != "" {
				citedIDs[lp.EntityID] = true
			}
		}
	}
	entities, err := loadEntitiesCapped(readCtx, citedIDs, graphMaxEntities)
	if err != nil {
		return nil, err
	}
	entityByID := map[string]*core.WikiEntity{}
	for i := range entities {
		entityByID[entities[i].ID] = &entities[i]
	}

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
	readCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
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
