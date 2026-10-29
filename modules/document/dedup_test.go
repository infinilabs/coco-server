/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"fmt"
	"testing"
	"time"
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

func resetDedupCache(t *testing.T, r *dedupReport) {
	t.Helper()
	dedupMu.Lock()
	oldCache, oldAt := dedupCache, dedupCacheAt
	dedupCache, dedupCacheAt = r, time.UnixMilli(1000)
	dedupMu.Unlock()
	t.Cleanup(func() {
		dedupMu.Lock()
		dedupCache, dedupCacheAt = oldCache, oldAt
		dedupMu.Unlock()
	})
}

func TestPatchDedupCacheOnAction(t *testing.T) {
	report := buildDedupReport([]dedupDocMeta{
		dedupDoc("a", "title long enough", "identical body text lorem ipsum dolor sit amet consectetur adipiscing elit", 1000),
		dedupDoc("b", "title long enough", "identical body text lorem ipsum dolor sit amet consectetur adipiscing elit", 2000),
		dedupDoc("c", "title long enough", "identical body text lorem ipsum dolor sit amet consectetur adipiscing elit", 3000),
	}, nil)
	if report.GroupCount != 1 || report.Groups[0].KeepID != "c" {
		t.Fatal("precondition: one group, newest doc is keeper")
	}
	resetDedupCache(t, report)

	// delete the keeper: member leaves, a new keeper is elected
	patchDedupCacheOnAction([]string{"c"}, "delete")
	g := report.Groups[0]
	if len(g.Members) != 2 || g.KeepID != "b" {
		t.Fatalf("delete must drop the member and re-elect the keeper: %+v", g)
	}

	// exclude: member stays, flagged disabled
	patchDedupCacheOnAction([]string{"a"}, "exclude")
	g = report.Groups[0]
	if len(g.Members) != 2 {
		t.Fatalf("exclude must keep the member: %+v", g)
	}
	disabled := false
	for _, m := range g.Members {
		if m.ID == "a" {
			disabled = m.Disabled
		}
	}
	if !disabled {
		t.Fatalf("excluded member must be flagged disabled: %+v", g.Members)
	}

	// delete down to a single member: the group dissolves
	patchDedupCacheOnAction([]string{"a"}, "delete")
	if report.GroupCount != 0 || len(report.Groups) != 0 {
		t.Fatalf("a one-member group must dissolve: %+v", report.Groups)
	}
}

func TestPatchDedupCacheOnDismiss(t *testing.T) {
	report := buildDedupReport([]dedupDocMeta{
		dedupDoc("a", "title long enough", "identical body text lorem ipsum dolor sit amet consectetur adipiscing elit", 1000),
		dedupDoc("b", "title long enough", "identical body text lorem ipsum dolor sit amet consectetur adipiscing elit", 2000),
	}, nil)
	resetDedupCache(t, report)

	// dismissing all member ids drops the group and bumps the counter
	patchDedupCacheOnDismiss([]string{"a", "b"}, 1)
	if report.GroupCount != 0 || report.Dismissed != 1 {
		t.Fatalf("dismiss must drop the group: count=%d dismissed=%d", report.GroupCount, report.Dismissed)
	}
}
