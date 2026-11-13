/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"testing"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
)

func TestClusterTopicsMergesByDocCoOccurrence(t *testing.T) {
	reduced := []ReducedKey{
		{Kind: "entity", NormKey: "e|a", Name: "A", Statements: []ReducedStatement{
			{DocID: "d1", Text: "a1"}, {DocID: "d2", Text: "a2"}, {DocID: "d3", Text: "a3"},
		}},
		{Kind: "concept", NormKey: "c|b", Name: "B", Statements: []ReducedStatement{
			{DocID: "d1", Text: "b1"}, {DocID: "d2", Text: "b2"}, {DocID: "d4", Text: "b3"},
		}},
		// shares d3/d4 with both clusters above — joins the strongest sharer
		{Kind: "entity", NormKey: "e|c", Name: "C", Statements: []ReducedStatement{
			{DocID: "d3", Text: "c1"}, {DocID: "d4", Text: "c2"},
		}},
		// unrelated single-claim key: long tail, folds into a sharer if any
		{Kind: "concept", NormKey: "c|x", Name: "X", Statements: []ReducedStatement{
			{DocID: "d9", Text: "x1"},
		}},
	}
	clusters := clusterTopics(reduced, 24)
	if len(clusters) == 0 {
		t.Fatalf("expected clusters, got none")
	}
	total := 0
	for _, c := range clusters {
		if c.Title != "" {
			t.Errorf("title must be empty before naming, got %q", c.Title)
		}
		total += len(c.Statements)
	}
	// the noise floor drops the single-claim singleton (X alone on d9 with
	// no sharing neighbor): 3+3+2 statements survive, x1 does not
	if total != 8 {
		t.Errorf("expected 8 pooled statements, got %d", total)
	}
	// A and B share d1+d2: B must sit in the same cluster as A
	for _, c := range clusters {
		hasA, hasB := false, false
		for _, k := range c.Keys {
			if k.Name == "A" {
				hasA = true
			}
			if k.Name == "B" {
				hasB = true
			}
		}
		if hasA != hasB {
			t.Errorf("keys A and B share two docs but were split")
		}
	}
}

func TestMergeTopicSectionsWholeArticleUpdate(t *testing.T) {
	existing := []core.WikiSection{
		{ID: "s1", Heading: "概述", Body: "old", DocIDs: []string{"d1"}},
		{ID: "s2", Heading: "时间线", Body: "keep"},
		{ID: "s3", Heading: "陈年旧节", Body: "stale"},
	}
	incoming := []core.WikiSection{
		{Heading: "概述", Body: "new", DocIDs: []string{"d1", "d2", "d1"}, UpdatedAt: 100},
		{Heading: "数据与事实", Body: "fresh", DocIDs: []string{"d3"}, UpdatedAt: 100},
		{Heading: "时间线", Body: "", UpdatedAt: 100}, // fill failed: existing prose survives
	}
	merged := mergeTopicSections(existing, incoming)
	if len(merged) != 3 {
		t.Fatalf("incoming outline is the article, got %d sections", len(merged))
	}
	if merged[0].Body != "new" {
		t.Errorf("same-heading section must take the rewritten body, got %q", merged[0].Body)
	}
	if len(merged[0].DocIDs) != 2 {
		t.Errorf("doc ids must union and dedup, got %v", merged[0].DocIDs)
	}
	if merged[2].Body != "keep" {
		t.Errorf("empty fill must keep existing prose, got %q", merged[2].Body)
	}
	for _, s := range merged {
		if s.Heading == "陈年旧节" {
			t.Error("section dropped by the incoming outline must not linger")
		}
	}
	// a fill that produced nothing at all must not wipe the page
	if wiped := mergeTopicSections(merged, nil); len(wiped) != len(merged) {
		t.Errorf("nil incoming must keep the outline, got %d sections", len(wiped))
	}
}

func TestComputeTopicMaturityCoverage(t *testing.T) {
	sections := []core.WikiSection{
		{Heading: "a", Body: "x"}, {Heading: "b", Body: "x"}, {Heading: "c", Body: ""},
	}
	m := ComputeTopicMaturity(sections, 4, 5)
	if m.Sections != 2 || m.Planned != 4 {
		t.Errorf("filled/planned mismatch: %+v", m)
	}
	// 0.6*(2/4) + 0.4*min(1,5/5) = 0.7
	if m.Score < 0.69 || m.Score > 0.71 {
		t.Errorf("score mismatch: %v", m.Score)
	}
}

func TestHeadingSimilarAnchorsRenamedSections(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"概述", "概述", true},
		{"Product Data & Pricing Facts", "Product Data and Pricing Facts", true}, // paraphrase
		{"产品数据与定价事实", "产品数据与定价", true},                                           // containment
		{"Overview: Maxim's Cakes Online Product Catalogue", "Overview: Maxim's Cakes Online Catalogue", true},
		{"时间线", "数据与事实", false}, // unrelated
		{"争议与开放问题", "来源与延伸", false},
	}
	for _, c := range cases {
		if got := headingSimilar(c.a, c.b); got != c.want {
			t.Errorf("headingSimilar(%q,%q)=%v want %v", c.a, c.b, got, c.want)
		}
	}
	// merge must replace a paraphrased twin, not append it
	outline := []core.WikiSection{{ID: "s1", Heading: "Product Data & Pricing Facts", Body: "old"}}
	incoming := []core.WikiSection{{ID: "s2", Heading: "Product Data and Pricing Facts", Body: "new", UpdatedAt: 1}}
	merged := mergeTopicSections(outline, incoming)
	if len(merged) != 1 {
		t.Fatalf("paraphrased heading must merge in place, got %d sections", len(merged))
	}
	if merged[0].Body != "new" {
		t.Errorf("body must refresh, got %q", merged[0].Body)
	}
}

func TestMatchTopicTargetsByDocOverlap(t *testing.T) {
	// last run's page: same documents, drifted title
	page := core.WikiArticle{ORMObjectBase: orm.ORMObjectBase{ID: "p1"}, Title: "美心食品與餐飲禮券", PageType: core.WikiPageTypeTopic,
		AIGenerated: true, KeyEntities: []string{"禮券", "美心食品"},
		Outline: []core.WikiSection{{Heading: "h", DocIDs: []string{"d1", "d2"}}}}
	// parallel twin from another run: same docs, third title
	twin := core.WikiArticle{ORMObjectBase: orm.ORMObjectBase{ID: "p2"}, Title: "禮券業務", PageType: core.WikiPageTypeTopic,
		AIGenerated: true, KeyEntities: []string{"禮券"},
		Outline: []core.WikiSection{{Heading: "h", DocIDs: []string{"d1", "d2"}}}}
	// human page on the same docs: enrichable, never absorbable
	human := core.WikiArticle{ORMObjectBase: orm.ORMObjectBase{ID: "p3"}, Title: "禮券手冊", PageType: core.WikiPageTypeTopic,
		Outline: []core.WikiSection{{Heading: "h", DocIDs: []string{"d1", "d2"}}}}
	// unrelated topic page: must not be claimed
	other := core.WikiArticle{ORMObjectBase: orm.ORMObjectBase{ID: "p4"}, Title: "門市網絡", PageType: core.WikiPageTypeTopic,
		AIGenerated: true, KeyEntities: []string{"分店"},
		Outline: []core.WikiSection{{Heading: "h", DocIDs: []string{"d9"}}}}

	cluster := &topicCluster{Title: "美心食品與禮券業務", Docs: map[string]bool{"d1": true, "d2": true},
		Keys: []ReducedKey{{Name: "禮券"}, {Name: "美心食品"}}}
	matchTopicTargets([]*topicCluster{cluster}, []core.WikiArticle{page, twin, human, other})

	if cluster.UpdateArticleID == "" {
		t.Fatal("drifted-title cluster must match an existing page by doc overlap")
	}
	if cluster.UpdateArticleID == "p3" {
		t.Error("argmax must prefer the page with key-entity agreement (p1), not the human page")
	}
	for _, id := range cluster.AbsorbIDs {
		if id == "p3" {
			t.Error("hand-written page must never be an absorb twin")
		}
	}
	if !containsID(cluster.AbsorbIDs, "p2") {
		t.Error("AI twin shadowing the same docs must be absorbed")
	}
	if containsID(cluster.AbsorbIDs, "p4") {
		t.Error("unrelated page must not be absorbed")
	}
}

func TestMatchTopicTargetsClaimsOnce(t *testing.T) {
	page := core.WikiArticle{ORMObjectBase: orm.ORMObjectBase{ID: "p1"}, Title: "月餅", PageType: core.WikiPageTypeTopic, AIGenerated: true,
		Outline: []core.WikiSection{{Heading: "h", DocIDs: []string{"d1", "d2"}}}}
	first := &topicCluster{Title: "中秋月餅", Docs: map[string]bool{"d1": true, "d2": true}}
	second := &topicCluster{Title: "節日禮品", Docs: map[string]bool{"d1": true, "d2": true}}
	matchTopicTargets([]*topicCluster{first, second}, []core.WikiArticle{page})
	if first.UpdateArticleID != "p1" {
		t.Fatalf("first cluster must claim the page, got %q", first.UpdateArticleID)
	}
	if second.UpdateArticleID != "" {
		t.Errorf("a page is enriched once per run; second cluster got %q", second.UpdateArticleID)
	}
}

func containsID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
