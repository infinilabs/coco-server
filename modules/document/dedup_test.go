/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"fmt"
	"testing"
)

func dedupDoc(id, title, content string, updated int64) dedupDocMeta {
	return dedupDocMeta{ID: id, Title: title, Content: content, Updated: updated, Size: 100}
}

func findGroup(t *testing.T, report *dedupReport, id string) *dedupGroup {
	t.Helper()
	for i := range report.Groups {
		for _, m := range report.Groups[i].Members {
			if m.ID == id {
				return &report.Groups[i]
			}
		}
	}
	return nil
}

func TestBuildDedupReportTiers(t *testing.T) {
	exactA := dedupDoc("a", "Report 2025", "annual report with revenue numbers and growth figures for fiscal year 2025, audited", 1000)
	exactB := dedupDoc("b", "Report 2025 (copy)", "ANNUAL  report with revenue numbers and growth figures for FISCAL year 2025,\taudited", 2000)
	exactC := dedupDoc("c", "report-2025.pdf", "ａｎｎｕａｌ report with revenue numbers and growth figures for fiscal year 2025, audited", 3000) // full-width variant

	nearA := dedupDoc("d", "Meeting notes", "weekly sync notes: roadmap discussion, hiring plan, and the q3 launch checklist was reviewed in detail today", 4000)
	nearB := dedupDoc("e", "Meeting notes v2", "weekly sync notes: roadmap discussion, hiring plan, and the q3 launch checklist was reviewed in detail today.", 5000)
	similarA := dedupDoc("i", "Meeting notes draft", "weekly sync notes: roadmap discussion, hiring plan, and the q3 launch checklist was reviewed in detail yesterday", 4500)

	versionA := dedupDoc("f", "Budget Plan", "the marketing budget allocates forty percent to brand campaigns across all regions next year", 6000)
	versionB := dedupDoc("g", "Budget Plan", "completely different content about database capacity planning and shard rebalancing schedules", 7000)

	unique := dedupDoc("h", "One-off", "a genuinely unique document about ostrich farming techniques in southern climates", 8000)

	report := buildDedupReport([]dedupDocMeta{exactA, exactB, exactC, nearA, nearB, similarA, versionA, versionB, unique}, nil)

	if report.Scanned != 9 || report.Fingerprinted != 9 {
		t.Fatalf("scanned=%d fingerprinted=%d", report.Scanned, report.Fingerprinted)
	}
	// d/e/i all land in ONE cluster: d-e is a near-identical pair and i is a
	// bigger edit of the same text — union-find merges the variants
	if report.GroupCount != 3 {
		t.Fatalf("expected 3 groups, got %d: %+v", report.GroupCount, report.Groups)
	}

	g := findGroup(t, report, "a")
	if g == nil || g.Tier != dedupTierExact || len(g.Members) != 3 {
		t.Fatalf("exact group wrong: %+v", g)
	}
	// keeper is the newest update
	if g.KeepID != "c" {
		t.Fatalf("keeper = %s, want newest (c)", g.KeepID)
	}

	g = findGroup(t, report, "d")
	if g == nil || g.Tier != dedupTierNear || len(g.Members) != 3 {
		t.Fatalf("near group wrong: %+v", g)
	}
	if findGroup(t, report, "i") != g {
		t.Fatal("the bigger edit (i) belongs to the same cluster")
	}

	g = findGroup(t, report, "f")
	if g == nil || g.Tier != dedupTierVersion {
		t.Fatalf("version group wrong: %+v", g)
	}

	if findGroup(t, report, "h") != nil {
		t.Fatal("unique doc must not be grouped")
	}
}

func TestBuildDedupReportDismissal(t *testing.T) {
	a := dedupDoc("a", "X", "duplicate content that appears twice in the corpus with identical wording everywhere", 1000)
	b := dedupDoc("b", "Y", "duplicate content that appears twice in the corpus with identical wording everywhere", 2000)

	if r := buildDedupReport([]dedupDocMeta{a, b}, nil); r.GroupCount != 1 {
		t.Fatalf("without dismissal: %d groups", r.GroupCount)
	}
	// dismissing the pair removes the group
	if r := buildDedupReport([]dedupDocMeta{a, b}, map[string]bool{dedupPairKey("a", "b"): true}); r.GroupCount != 0 {
		t.Fatalf("dismissed pair still reported: %+v", r.Groups)
	}
}

func TestBuildDedupReportTooShort(t *testing.T) {
	a := dedupDoc("a", "tiny", "short", 1)
	b := dedupDoc("b", "tiny", "short", 2)
	if r := buildDedupReport([]dedupDocMeta{a, b}, nil); r.GroupCount != 0 {
		t.Fatalf("too-short docs must not group: %+v", r.Groups)
	}
}

func TestDedupGroupKeyStable(t *testing.T) {
	docs := []dedupDocMeta{}
	for i := 0; i < 3; i++ {
		docs = append(docs, dedupDoc(fmt.Sprintf("doc%d", i), "same title long enough", "identical body text lorem ipsum dolor sit amet consectetur adipiscing elit", int64(1000+i)))
	}
	r1 := buildDedupReport(append([]dedupDocMeta{}, docs...), nil)
	r2 := buildDedupReport([]dedupDocMeta{docs[2], docs[0], docs[1]}, nil) // shuffled input
	if r1.Groups[0].Key != r2.Groups[0].Key {
		t.Fatal("group key must be member-order independent")
	}
}
