/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"strings"
	"testing"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
)

func graphEntity(id, name, status string, aliases []string, relations ...core.WikiEntityRelation) core.WikiEntity {
	entity := core.WikiEntity{}
	entity.ID = id
	entity.Name = name
	entity.Status = status
	entity.Aliases = aliases
	entity.Relations = relations
	return entity
}

func TestRecognizeGraphSeeds(t *testing.T) {
	entities := []core.WikiEntity{
		graphEntity("e-pay", "Payment Service", core.WikiEntityPublished, []string{"支付服务"}),
		graphEntity("e-incident", "Recurring Incident", core.WikiEntityProposed, nil), // draft: never a seed
		graphEntity("e-order", "Order Center", core.WikiEntityReviewed, nil),
		graphEntity("e-alias", "库存系统", core.WikiEntityPublished, []string{"WMS", "warehouse"}),
		graphEntity("e-short", "库", core.WikiEntityPublished, nil), // single rune: too short
	}

	// name (CJK) + alias + English name containment, longest match first:
	// "order center" (12 runes) > "支付服务" (4) > "WMS" (3)
	seeds := recognizeGraphSeeds("支付服务 order center 怎么对接 WMS", entities)
	if len(seeds) != 3 {
		t.Fatalf("expected 3 seeds, got %d: %+v", len(seeds), seeds)
	}
	got := []string{}
	for _, s := range seeds {
		got = append(got, s.Entity.ID)
	}
	if got[0] != "e-order" || got[1] != "e-pay" || got[2] != "e-alias" {
		t.Fatalf("seeds should sort longest-match-first, got %v", got)
	}

	// no recognition → empty, not an error
	if seeds := recognizeGraphSeeds("nothing matches here", entities); len(seeds) != 0 {
		t.Fatalf("expected no seeds, got %+v", seeds)
	}

	// single-rune names never fire (would match almost anything)
	if seeds := recognizeGraphSeeds("库存 库 存", entities); len(seeds) != 0 {
		t.Fatalf("single-rune name must not fire, got %+v", seeds)
	}

	// alias alone is enough
	seeds = recognizeGraphSeeds("warehouse dashboard", entities)
	if len(seeds) != 1 || seeds[0].Entity.ID != "e-alias" || seeds[0].MatchedTerm != "warehouse" {
		t.Fatalf("alias match failed: %+v", seeds)
	}
}

func TestRecognizeGraphSeedsCapAndDedup(t *testing.T) {
	entities := []core.WikiEntity{}
	for i := 0; i < graphRouteMaxSeeds+3; i++ {
		entities = append(entities, graphEntity(
			strings.Repeat("e", i+1), "term"+strings.Repeat("x", i+1), core.WikiEntityPublished, nil))
	}
	query := ""
	for _, e := range entities {
		query += e.Name + " "
	}
	seeds := recognizeGraphSeeds(query, entities)
	if len(seeds) != graphRouteMaxSeeds {
		t.Fatalf("expected cap at %d seeds, got %d", graphRouteMaxSeeds, len(seeds))
	}
	// longest match first: the cap keeps the most specific entities
	if seeds[0].Entity.Name != entities[len(entities)-1].Name {
		t.Fatalf("most specific seed should rank first, got %s", seeds[0].Entity.Name)
	}
}

func TestGraphExpansion(t *testing.T) {
	payment := graphEntity("e-pay", "Payment Service", core.WikiEntityPublished, nil,
		core.WikiEntityRelation{TargetID: "e-gateway", Relation: "depends_on"},
		core.WikiEntityRelation{TargetID: "e-order", Relation: "causes"},
	)
	order := graphEntity("e-order", "Order Center", core.WikiEntityReviewed, nil,
		core.WikiEntityRelation{TargetID: "e-pay", Relation: "escalates"},
	)
	// pure forward target: no relations of its own
	gateway := graphEntity("e-gateway", "API Gateway", core.WikiEntityPublished, nil)
	// pure inverse neighbor: only points at the seed
	observability := graphEntity("e-obs", "Observability", core.WikiEntityPublished, nil,
		core.WikiEntityRelation{TargetID: "e-pay", Relation: "monitors"},
	)
	selfLoop := graphEntity("e-loop", "Looper", core.WikiEntityPublished, nil,
		core.WikiEntityRelation{TargetID: "e-loop", Relation: "mentions"},
	)
	pool := []core.WikiEntity{payment, order, gateway, observability, selfLoop}

	seeds := []graphSeed{{Entity: payment, MatchedTerm: "Payment Service"}}
	neighbors := graphExpansion(seeds, pool)

	if len(neighbors) != 3 {
		t.Fatalf("expected 3 neighbors (order mutual, gateway forward, obs inverse), got %+v", neighbors)
	}
	byID := map[string]graphNeighborRef{}
	for _, n := range neighbors {
		byID[n.EntityID] = n
	}
	// order is reached forward (payment causes order) and inverse (order escalates payment)
	if ref, ok := byID["e-order"]; !ok || ref.Paths != 2 || ref.FromID != "e-pay" || ref.Relation != "causes" {
		t.Fatalf("order should have 2 paths from the seed, got %+v", ref)
	}
	if ref, ok := byID["e-gateway"]; !ok || ref.Paths != 1 || ref.FromID != "e-pay" || ref.Relation != "depends_on" {
		t.Fatalf("gateway should come from the seed's forward edge, got %+v", ref)
	}
	// inverse-only neighbor is reached from the seed over its declared relation
	if ref, ok := byID["e-obs"]; !ok || ref.Paths != 1 || ref.FromID != "e-pay" || ref.FromName != "Payment Service" || ref.Relation != "monitors" {
		t.Fatalf("inverse neighbor wrong, got %+v", ref)
	}
	// paths-first ordering, then id: order (2) before gateway (1) before obs (1)
	if neighbors[0].EntityID != "e-order" || neighbors[1].EntityID != "e-gateway" || neighbors[2].EntityID != "e-obs" {
		t.Fatalf("neighbor order wrong: %+v", neighbors)
	}
}

func TestGraphExpansionExcludesSeeds(t *testing.T) {
	a := graphEntity("e-a", "A", core.WikiEntityPublished, nil,
		core.WikiEntityRelation{TargetID: "e-b", Relation: "mentions"})
	b := graphEntity("e-b", "B", core.WikiEntityPublished, nil,
		core.WikiEntityRelation{TargetID: "e-a", Relation: "mentions"})
	seeds := []graphSeed{{Entity: a, MatchedTerm: "A"}, {Entity: b, MatchedTerm: "B"}}
	if neighbors := graphExpansion(seeds, []core.WikiEntity{a, b}); len(neighbors) != 0 {
		t.Fatalf("seeds must not be their own neighbors, got %+v", neighbors)
	}
}

func TestDecorateGraphHit(t *testing.T) {
	restore := appConfigFn
	defer func() { appConfigFn = restore }()
	appConfigFn = func() core.Config {
		return core.Config{ServerInfo: &core.ServerInfo{Endpoint: "http://coco.test"}}
	}

	article := &core.WikiArticle{KbID: "kb1", Title: "API Gateway"}
	hit := wikiArticleToHit(article, 7)
	seedMeta := []interface{}{util.MapStr{"entity_id": "e-pay", "entity_name": "Payment Service", "matched_term": "Payment Service"}}

	// seed hit: no via
	decorateGraphHit(&hit, seedMeta, nil, "")
	if hit.Source.Metadata["graph_route"] != true {
		t.Fatalf("graph_route marker missing: %+v", hit.Source.Metadata)
	}
	if seeds, ok := hit.Source.Metadata["graph_seeds"].([]interface{}); !ok || len(seeds) != 1 {
		t.Fatalf("graph_seeds missing: %+v", hit.Source.Metadata)
	}
	if _, ok := hit.Source.Metadata["graph_via"]; ok {
		t.Fatalf("seed hit must not carry graph_via")
	}

	// neighbor hit: via edge, neighbor name filled when known
	via := &graphNeighborRef{EntityID: "e-gateway", FromID: "e-pay", FromName: "Payment Service", Relation: "depends_on"}
	decorateGraphHit(&hit, seedMeta, via, "API Gateway")
	viaMeta, ok := hit.Source.Metadata["graph_via"].(util.MapStr)
	if !ok || viaMeta["relation"] != "depends_on" || viaMeta["from_entity"] != "Payment Service" || viaMeta["entity_name"] != "API Gateway" {
		t.Fatalf("graph_via malformed: %+v", viaMeta)
	}

	// missing neighbor entity: no name, no panic
	decorateGraphHit(&hit, seedMeta, via, "")
	viaMeta, ok = hit.Source.Metadata["graph_via"].(util.MapStr)
	if !ok {
		t.Fatalf("graph_via malformed: %+v", hit.Source.Metadata)
	}
	if _, has := viaMeta["entity_name"]; has {
		t.Fatalf("unknown neighbor must omit entity_name")
	}
}

func TestRRFFuseMultiFourRoutes(t *testing.T) {
	text := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 9), rrfHit("b", 8)}
	semantic := []elastic.DocumentWithMeta[core.Document]{rrfHit("b", 0.9), rrfHit("c", 0.8)}
	wiki := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 19), rrfHit("c", 18)}
	graph := []elastic.DocumentWithMeta[core.Document]{rrfHit("d", 9), rrfHit("a", 8)}

	cfg := RRFConfig{K: 60, Weights: map[string]float64{
		rrfRouteText: 1, rrfRouteSemantic: 1, rrfRouteWiki: 1, rrfRouteGraph: 2,
	}}
	hits, breakdowns := rrfFuseMulti(rrfRoutes(
		rrfRouteHits{Name: rrfRouteText, Hits: text},
		rrfRouteHits{Name: rrfRouteSemantic, Hits: semantic},
		rrfRouteHits{Name: rrfRouteWiki, Hits: wiki},
		rrfRouteHits{Name: rrfRouteGraph, Hits: graph},
	), cfg)

	if len(hits) != 4 {
		t.Fatalf("expected 4 unique docs, got %d", len(hits))
	}
	byID := map[string]rrfBreakdown{}
	for _, b := range breakdowns {
		byID[b.ID] = b
	}
	// a: text rank1 (1/61) + wiki rank1 (1/61) + graph rank2 (2/62)
	wantA := 1.0/61 + 1.0/61 + 2.0/62
	if diff := byID["a"].Score - wantA; diff > 1e-12 || diff < -1e-12 {
		t.Fatalf("doc a fused score %v, want %v", byID["a"].Score, wantA)
	}
	if byID["a"].Ranks[rrfRouteGraph] != 2 || byID["a"].Contributions[rrfRouteGraph] != 2.0/62 {
		t.Fatalf("graph contribution wrong: %+v", byID["a"])
	}
	// d only comes from the graph leg with weight 2
	if diff := byID["d"].Score - 2.0/61; diff > 1e-12 || diff < -1e-12 {
		t.Fatalf("doc d fused score %v, want %v", byID["d"].Score, 2.0/61)
	}
	if hits[0].ID != "a" {
		t.Fatalf("doc a should fuse to the top, got %s", hits[0].ID)
	}
}

func TestRRFFuseMultiGraphMuted(t *testing.T) {
	graph := []elastic.DocumentWithMeta[core.Document]{rrfHit("d", 9)}
	text := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 5)}
	cfg := RRFConfig{K: 60, Weights: map[string]float64{rrfRouteText: 1, rrfRouteGraph: 0}}
	hits, breakdowns := rrfFuseMulti(rrfRoutes(
		rrfRouteHits{Name: rrfRouteText, Hits: text},
		rrfRouteHits{Name: rrfRouteGraph, Hits: graph},
	), cfg)

	byID := map[string]rrfBreakdown{}
	for _, b := range breakdowns {
		byID[b.ID] = b
	}
	if byID["d"].Score != 0 {
		t.Fatalf("muted graph route must contribute nothing, got %v", byID["d"].Score)
	}
	// muted docs still surface (zero contribution), ranked below contributing docs
	if hits[0].ID != "a" {
		t.Fatalf("contributing doc should outrank muted-route doc, got %s", hits[0].ID)
	}
}
