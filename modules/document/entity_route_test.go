/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
)

func swapAppConfigForTest(t *testing.T) {
	t.Helper()
	restore := appConfigFn
	t.Cleanup(func() { appConfigFn = restore })
	appConfigFn = func() core.Config {
		return core.Config{ServerInfo: &core.ServerInfo{Endpoint: "http://coco.test"}}
	}
}

func TestEntityToHitWrapsEntity(t *testing.T) {
	swapAppConfigForTest(t)
	e := core.WikiEntity{}
	e.ID = "ent-1"
	e.Name = "支付网关"
	e.Type = "product"
	e.Status = core.WikiEntityReviewed
	e.Aliases = []string{"payment gateway", "PGW"}
	e.ArticleID = "art-9"
	e.Sources = []core.WikiSourceReference{{DocID: "doc-7", Title: "架构说明", Locator: "page/3"}}

	hit := entityToHit(&e, 42)
	assert.Equal(t, "ent-1", hit.ID)
	assert.Equal(t, "wiki_entity", hit.Index)
	assert.Equal(t, "支付网关", hit.Source.Title)
	assert.Equal(t, "entity", hit.Source.Type)
	assert.Equal(t, "wiki", hit.Source.Source.ID)
	assert.Contains(t, hit.Source.URL, "/wiki/article/art-9", "an entity with a curated article links there")
	assert.Equal(t, true, hit.Source.Metadata["entity"])
	assert.Equal(t, "product", hit.Source.Metadata["entity_type"])

	refs, ok := hit.Source.Metadata["entity_sources"].([]interface{})
	require.True(t, ok)
	require.Len(t, refs, 1)
	ref := refs[0].(util.MapStr)
	assert.Equal(t, "doc-7", ref["doc_id"])
	assert.Equal(t, "page/3", ref["locator"])
}

func TestEntityToHitWithoutArticle(t *testing.T) {
	swapAppConfigForTest(t)
	e := core.WikiEntity{}
	e.ID = "ent-2"
	e.Name = "孤立实体"
	e.Status = core.WikiEntityPublished
	hit := entityToHit(&e, 1)
	assert.Empty(t, hit.Source.URL, "card-only hit when no article backs the entity")
	assert.Empty(t, hit.Source.Metadata["entity_sources"])
}

func TestEntityRouteFusionMath(t *testing.T) {
	// entity route participates in the fused math like any other leg:
	// doc "pay" is text#2 and entity#1 → 1/62 + w/(k+1)
	text := []elastic.DocumentWithMeta[core.Document]{rrfHit("other", 9), rrfHit("pay", 8)}
	entity := []elastic.DocumentWithMeta[core.Document]{rrfHit("pay", 5)}

	hits, breakdowns := rrfFuseMulti(rrfRoutes(
		rrfRouteHits{Name: rrfRouteText, Hits: text},
		rrfRouteHits{Name: rrfRouteEntity, Hits: entity},
	), RRFConfig{K: 60, Weights: map[string]float64{
		rrfRouteText:   1,
		rrfRouteEntity: 2,
	}})

	byID := map[string]rrfBreakdown{}
	for _, b := range breakdowns {
		byID[b.ID] = b
	}
	want := 1.0/62.0 + 2.0/61.0
	if diff := byID["pay"].Score - want; diff > 1e-12 || diff < -1e-12 {
		t.Fatalf("pay score = %v, want %v", byID["pay"].Score, want)
	}
	assert.Equal(t, "pay", hits[0].ID, "the entity-backed doc wins under weight 2")
}

func TestEntityRouteMuteKeepsDocuments(t *testing.T) {
	// entity_weight=0 mutes the leg's contribution but its documents still
	// surface (a muted route never drops recall) — same semantics as the
	// other legs
	text := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 9)}
	entity := []elastic.DocumentWithMeta[core.Document]{rrfHit("ent-only", 5)}

	hits, _ := rrfFuseMulti(rrfRoutes(
		rrfRouteHits{Name: rrfRouteText, Hits: text},
		rrfRouteHits{Name: rrfRouteEntity, Hits: entity},
	), RRFConfig{K: 60, Weights: map[string]float64{
		rrfRouteText:   1,
		rrfRouteEntity: 0,
	}})

	ids := map[string]bool{}
	for _, h := range hits {
		ids[h.ID] = true
	}
	assert.True(t, ids["ent-only"], "muted entity route still returns its documents")
}
