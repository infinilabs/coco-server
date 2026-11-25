/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"testing"

	"infini.sh/coco/core"
)

func TestCompileRulesNormalized(t *testing.T) {
	// nil rules become the defaults
	rules := (*core.WikiCompileRules)(nil).Normalized()
	if rules.ConceptMinSources != core.DefaultConceptMinSources || rules.ConflictStrategy != core.WikiConflictStrategyMark {
		t.Fatalf("nil rules must normalize to defaults, got %+v", rules)
	}

	// out-of-range floor and unknown strategy clamp back
	rules = (&core.WikiCompileRules{ConceptMinSources: 0, ConflictStrategy: "nonsense"}).Normalized()
	if rules.ConceptMinSources != core.DefaultConceptMinSources {
		t.Fatalf("floor must clamp to default, got %d", rules.ConceptMinSources)
	}
	if rules.ConflictStrategy != core.WikiConflictStrategyMark {
		t.Fatalf("unknown strategy must clamp to mark, got %s", rules.ConflictStrategy)
	}

	// valid values stay
	rules = (&core.WikiCompileRules{ConceptMinSources: 4, ConflictStrategy: core.WikiConflictStrategyIsolate,
		SourcePriority: []string{"ds-a", "ds-b"}}).Normalized()
	if rules.ConceptMinSources != 4 || rules.ConflictStrategy != core.WikiConflictStrategyIsolate || len(rules.SourcePriority) != 2 {
		t.Fatalf("valid rules must survive normalization, got %+v", rules)
	}
}

func TestSourcePriorityRank(t *testing.T) {
	rules := &core.WikiCompileRules{SourcePriority: []string{"ds-official", "ds-mirror"}}
	if rules.SourcePriorityRank("ds-official") >= rules.SourcePriorityRank("ds-mirror") {
		t.Fatal("earlier entry must rank higher (lower value)")
	}
	if rules.SourcePriorityRank("ds-unknown") != rules.SourcePriorityRank("ds-also-unknown") {
		t.Fatal("unknown sources must share the lowest rank")
	}
	var nilRules *core.WikiCompileRules
	if nilRules.SourcePriorityRank("x") == 0 {
		t.Fatal("nil rules must rank everything lowest, not first")
	}
}

func TestNoiseGateRespectsConfiguredFloor(t *testing.T) {
	singleSource := &core.WikiArticle{
		Status:   core.WikiArticlePublished,
		PageType: core.WikiPageTypeConcept,
		Sources:  []core.WikiSourceReference{{DocID: "d1"}, {DocID: "d2"}},
	}

	// floor 3: two sources are not enough
	if passesWikiNoiseGate(singleSource, 3) {
		t.Fatal("two sources must fail a floor of 3")
	}
	// floor 1: passes
	if !passesWikiNoiseGate(singleSource, 1) {
		t.Fatal("two sources must pass a floor of 1")
	}

	// map pages are hand-maintained: no source floor at all
	mapPage := &core.WikiArticle{Status: core.WikiArticlePublished, PageType: core.WikiPageTypeMap}
	if !passesWikiNoiseGate(mapPage, 5) {
		t.Fatal("map pages must never hit the concept source floor")
	}
}
