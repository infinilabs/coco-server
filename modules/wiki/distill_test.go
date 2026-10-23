/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"strings"
	"testing"
	"time"

	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

func TestRenderPatternContent(t *testing.T) {
	pat := &core.WikiPattern{
		Symptom:   "Answers cite the wrong quarter's revenue",
		RootCause: "Two datasource copies of the same deck with different fiscal calendars",
		Fix:       "Prefer the datasource with the higher source priority; check the deck date before citing",
		Evidence: []core.WikiSourceReference{
			{DocID: "exp-1", Title: "Q3 revenue is wrong", SourceType: "experience"},
		},
	}
	content := renderPatternContent(pat)
	for _, want := range []string{"## Symptom", "## Root cause", "## Fix", "## Evidence", "Q3 revenue is wrong"} {
		if !strings.Contains(content, want) {
			t.Fatalf("rendered content missing %q:\n%s", want, content)
		}
	}
}

func TestPatternFromEvidenceRoundtrip(t *testing.T) {
	pattern := &core.WikiPattern{
		Symptom:   "s",
		RootCause: "r",
		Fix:       "f",
		Evidence: []core.WikiSourceReference{
			{DocID: "e1", Title: "t1", SourceType: "experience"},
		},
	}
	proposal := &core.WikiGovernanceProposal{
		KbID:         "kb1",
		ArticleTitle: "pattern title",
		Type:         core.WikiGovernancePatternCreate,
		Evidence: util.MapStr{
			"symptom":    pattern.Symptom,
			"root_cause": pattern.RootCause,
			"fix":        pattern.Fix,
			"sources":    pattern.Evidence,
		},
	}
	// evidence survives the ES JSON roundtrip the apply path performs
	decoded := util.MapStr{}
	if err := util.FromJSONBytes(util.MustToJSONBytes(proposal.Evidence), &decoded); err != nil {
		t.Fatalf("evidence json roundtrip failed: %v", err)
	}
	proposal.Evidence = decoded
	got := patternFromEvidence(proposal)
	if got.Symptom != "s" || got.RootCause != "r" || got.Fix != "f" {
		t.Fatalf("pattern fields not decoded: %+v", got)
	}
	if len(got.Evidence) != 1 || got.Evidence[0].DocID != "e1" {
		t.Fatalf("evidence not decoded: %+v", got.Evidence)
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("hello", 3); got != "hel…" {
		t.Fatalf("truncateRunes = %q", got)
	}
	if got := truncateRunes("你好", 5); got != "你好" {
		t.Fatalf("truncateRunes cjk = %q", got)
	}
}

func TestRenderSkillContent(t *testing.T) {
	skill := &core.WikiSkill{
		Applicability:  "When quoting revenue figures from slide decks",
		Steps:          []string{"Check the deck date", "Prefer the higher-priority datasource"},
		AntiPatterns:   []string{"Citing the first hit without checking the fiscal calendar"},
		WhenNotToApply: []string{"Structured financial reports from the finance system"},
		ReachFor:       &core.SkillReachFor{Phrases: []string{"revenue", "季度收入"}},
		Rationale:      "two failures traced to the same calendar mismatch",
	}
	provenance := []core.WikiSourceReference{
		{DocID: "Fiscal calendar mismatch", SourceType: "pattern", Title: "Fiscal calendar mismatch"},
	}
	content := renderSkillContent(skill, provenance)
	for _, want := range []string{"## When to use", "## Steps", "1. Check the deck date", "## Anti-patterns", "## When NOT to apply", "[[pattern:Fiscal calendar mismatch]]", "Proposer rationale"} {
		if !strings.Contains(content, want) {
			t.Fatalf("skill content missing %q:\n%s", want, content)
		}
	}
}

func TestSkillFromEvidenceRoundtrip(t *testing.T) {
	proposal := &core.WikiGovernanceProposal{
		KbID:         "kb1",
		ArticleID:    "art-1",
		ArticleTitle: "Revenue quoting",
		Type:         core.WikiGovernanceSkillPatch,
		Evidence: util.MapStr{
			"applicability": "when quoting revenue",
			"steps":         []string{"a", "b"},
			"rationale":     "recurring failure",
			"reach_for":     core.SkillReachFor{Phrases: []string{"revenue"}},
			"sources": []core.WikiSourceReference{
				{DocID: "p1", SourceType: "pattern", Title: "Fiscal calendar mismatch"},
			},
		},
	}
	decoded := util.MapStr{}
	if err := util.FromJSONBytes(util.MustToJSONBytes(proposal.Evidence), &decoded); err != nil {
		t.Fatalf("evidence roundtrip failed: %v", err)
	}
	proposal.Evidence = decoded
	skill, provenance := skillFromEvidence(proposal)
	if skill == nil || len(skill.Steps) != 2 || skill.ReachFor == nil || len(skill.ReachFor.Phrases) != 1 {
		t.Fatalf("skill not decoded: %+v", skill)
	}
	if len(provenance) != 1 || provenance[0].SourceType != "pattern" {
		t.Fatalf("provenance not decoded: %+v", provenance)
	}
}

func TestImpactPolicy(t *testing.T) {
	now := time.Now()
	old := now.Add(-200 * 24 * time.Hour)
	created := now.Add(-100 * 24 * time.Hour)

	// old, idle, never cited → candidate
	a := &core.WikiArticle{Status: core.WikiArticlePublished}
	a.Created = &created
	a.Updated = &old
	if ok, why := impactArchiveCandidate(a, now); !ok || why == "" {
		t.Fatalf("expected archive candidate, got ok=%v why=%q", ok, why)
	}
	// recently cited enough → protected, not a candidate
	a.Impact = &core.WikiImpact{AnswerHits: 5}
	if ok, _ := impactArchiveCandidate(a, now); ok {
		t.Fatalf("heavily-used page must not be an archive candidate")
	}
	if !impactProtected(a) {
		t.Fatalf("heavily-used page must be impact-protected")
	}
	// young page → grace period
	a.Impact = nil
	young := now.Add(-10 * 24 * time.Hour)
	a.Created = &young
	a.Updated = &young
	if ok, _ := impactArchiveCandidate(a, now); ok {
		t.Fatalf("young page must not be an archive candidate")
	}
}
