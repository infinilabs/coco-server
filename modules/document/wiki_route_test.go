/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"testing"

	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

func TestPassesWikiNoiseGate(t *testing.T) {
	published := &core.WikiArticle{Status: core.WikiArticlePublished}
	if !passesWikiNoiseGate(published, core.DefaultConceptMinSources) {
		t.Fatal("published page should pass")
	}

	draft := &core.WikiArticle{Status: core.WikiArticleDraft}
	if passesWikiNoiseGate(draft, core.DefaultConceptMinSources) {
		t.Fatal("draft must not enter retrieval")
	}

	lowConfidence := &core.WikiArticle{Status: core.WikiArticlePublished, Confidence: "low"}
	if passesWikiNoiseGate(lowConfidence, core.DefaultConceptMinSources) {
		t.Fatal("low-confidence page must not enter retrieval")
	}

	// concept pages need two independent sources
	oneSource := &core.WikiArticle{
		Status:   core.WikiArticlePublished,
		PageType: core.WikiPageTypeConcept,
		Sources:  []core.WikiSourceReference{{DocID: "d1"}},
	}
	if passesWikiNoiseGate(oneSource, core.DefaultConceptMinSources) {
		t.Fatal("single-source concept page must not enter retrieval")
	}
	twoSources := &core.WikiArticle{
		Status:   core.WikiArticlePublished,
		PageType: core.WikiPageTypeConcept,
		Sources:  []core.WikiSourceReference{{DocID: "d1"}, {DocID: "d2"}},
	}
	if !passesWikiNoiseGate(twoSources, core.DefaultConceptMinSources) {
		t.Fatal("two-source concept page should pass")
	}

	// entity/source pages carry their own provenance, no extra source floor
	entity := &core.WikiArticle{Status: core.WikiArticlePublished, PageType: core.WikiPageTypeEntity}
	if !passesWikiNoiseGate(entity, core.DefaultConceptMinSources) {
		t.Fatal("entity page should pass without the concept source floor")
	}
}

func TestWikiArticleToHit(t *testing.T) {
	restore := appConfigFn
	defer func() { appConfigFn = restore }()
	appConfigFn = func() core.Config {
		return core.Config{ServerInfo: &core.ServerInfo{Endpoint: "http://coco.test"}}
	}

	article := &core.WikiArticle{
		Status:  core.WikiArticlePublished,
		Title:   "Latency Budget",
		Summary: "How we account for latency across tiers",
		Sources: []core.WikiSourceReference{
			{DocID: "d1", Title: "Perf doc", Locator: "section:3", Excerpt: "p99 budget"},
		},
	}
	hit := wikiArticleToHit(article, 12.5)

	if hit.ID != article.ID || hit.Score != 12.5 {
		t.Fatalf("hit meta wrong: %+v", hit)
	}
	if hit.Source.Title != "Latency Budget" || hit.Source.Source.ID != "wiki" {
		t.Fatalf("hit source wrong: %+v", hit.Source)
	}
	if hit.Source.URL == "" {
		t.Fatal("wiki hits must carry an article URL")
	}
	meta := hit.Source.Metadata
	if meta["wiki_article"] != true || meta["page_type"] != "" {
		t.Fatalf("metadata flags wrong: %+v", meta)
	}
	sources, ok := meta["wiki_sources"].([]interface{})
	if !ok || len(sources) != 1 {
		t.Fatalf("wiki sources must ride along: %+v", meta["wiki_sources"])
	}
	first := sources[0].(util.MapStr)
	if first["doc_id"] != "d1" || first["locator"] != "section:3" {
		t.Fatalf("source provenance wrong: %+v", first)
	}
}
