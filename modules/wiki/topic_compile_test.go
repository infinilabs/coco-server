/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"testing"

	"infini.sh/coco/core"
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

func TestMergeTopicSectionsUpdatesByHeading(t *testing.T) {
	existing := []core.WikiSection{
		{ID: "s1", Heading: "概述", Body: "old", DocIDs: []string{"d1"}},
		{ID: "s2", Heading: "时间线", Body: "keep"},
	}
	incoming := []core.WikiSection{
		{Heading: "概述", Body: "new", DocIDs: []string{"d1", "d2"}, UpdatedAt: 100},
		{Heading: "数据与事实", Body: "fresh", DocIDs: []string{"d3"}, UpdatedAt: 100},
		{Heading: "争议", Body: "", DocIDs: nil, UpdatedAt: 100}, // empty body: no new section
	}
	merged := mergeTopicSections(existing, incoming)
	if len(merged) != 3 {
		t.Fatalf("expected 3 sections (2 existing + 1 new), got %d", len(merged))
	}
	if merged[0].Body != "new" {
		t.Errorf("same-heading section must refresh body, got %q", merged[0].Body)
	}
	if len(merged[0].DocIDs) != 2 {
		t.Errorf("doc ids must union, got %v", merged[0].DocIDs)
	}
	if merged[1].Body != "keep" {
		t.Errorf("untouched section must keep its body, got %q", merged[1].Body)
	}
	// empty body never wipes existing prose
	merged2 := mergeTopicSections(merged, []core.WikiSection{{Heading: "时间线", Body: "", UpdatedAt: 200}})
	if merged2[1].Body != "keep" {
		t.Errorf("empty incoming body must not wipe prose, got %q", merged2[1].Body)
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
