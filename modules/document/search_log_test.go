/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"testing"
	"time"

	"infini.sh/coco/core"
)

func searchLogEntry(query, searchType string, total int64, tookMS int64, created *time.Time) core.SearchLog {
	entry := core.SearchLog{Query: query, SearchType: searchType, Total: total, TookMS: tookMS, ZeroHit: total == 0}
	entry.Created = created
	return entry
}

func TestNormalizeSearchQuery(t *testing.T) {
	got := normalizeSearchQuery("  How   to Deploy   EasySearch? ")
	if got != "how to deploy easysearch?" {
		t.Fatalf("normalization wrong: %q", got)
	}
	if normalizeSearchQuery("   ") != "" {
		t.Fatal("whitespace-only query must normalize to empty")
	}
	// the same query written with different casing/spacing aggregates together
	if normalizeSearchQuery("PAYMENT  down") != normalizeSearchQuery("payment down ") {
		t.Fatal("case/space variants must collide")
	}
}

func TestSearchQueryHashStable(t *testing.T) {
	a := searchQueryHash("Payment  Down")
	b := searchQueryHash("payment down")
	if a == "" || a != b {
		t.Fatalf("query hash must be stable across normalization: %q vs %q", a, b)
	}
	if a == searchQueryHash("payment up") {
		t.Fatal("different queries must hash differently")
	}
}

func TestAggregateSearchLogsEmpty(t *testing.T) {
	stats := aggregateSearchLogs(nil)
	if stats.TotalSearches != 0 || len(stats.Strategies) != 0 || len(stats.LowRecall) != 0 {
		t.Fatalf("empty input must give empty stats, got %+v", stats)
	}
}

func TestAggregateSearchLogs(t *testing.T) {
	now := time.Now()
	older := now.Add(-time.Hour)
	logs := []core.SearchLog{
		searchLogEntry("payment down", "hybrid_rrf", 5, 120, &now),
		searchLogEntry("payment down", "hybrid_rrf", 0, 90, &now), // zero hit
		searchLogEntry("payment down", "keyword", 0, 40, &older),  // zero hit
		searchLogEntry("payment down", "keyword", 0, 50, &now),    // zero hit — crosses the gap threshold
		searchLogEntry("k8s ingress", "semantic", 3, 300, &now),
		searchLogEntry("", "keyword", 0, 20, &now), // no query: logs but never boards
	}

	stats := aggregateSearchLogs(logs)

	if stats.TotalSearches != 6 {
		t.Fatalf("total searches: %d", stats.TotalSearches)
	}
	if stats.ZeroHits != 4 {
		t.Fatalf("zero hits: %d", stats.ZeroHits)
	}
	if stats.ZeroHitRate < 0.66 || stats.ZeroHitRate > 0.67 {
		t.Fatalf("zero rate: %v", stats.ZeroHitRate)
	}
	// avg took = (120+90+40+50+300+20)/6 = 103.33 → 103
	if stats.AvgTookMS != 103 {
		t.Fatalf("avg took: %d", stats.AvgTookMS)
	}
	if stats.MaxTookMS != 300 {
		t.Fatalf("max took: %d", stats.MaxTookMS)
	}

	byType := map[string]searchOpsRow{}
	for _, row := range stats.Strategies {
		byType[row.Type] = row
	}
	if len(byType) != 3 {
		t.Fatalf("strategy rows: %+v", stats.Strategies)
	}
	if byType["hybrid_rrf"].Count != 2 || byType["hybrid_rrf"].ZeroHits != 1 {
		t.Fatalf("hybrid_rrf row: %+v", byType["hybrid_rrf"])
	}
	// keyword ran 3x (two payment misses + the empty query), all zero-hit;
	// the empty query counts in strategy rows but never boards
	if byType["keyword"].Count != 3 || byType["keyword"].ZeroHits != 3 {
		t.Fatalf("keyword row: %+v", byType["keyword"])
	}

	// low-recall board: "payment down" (3 misses), empty query never boards
	if len(stats.LowRecall) != 1 || stats.LowRecall[0].Query != "payment down" || stats.LowRecall[0].Count != 3 {
		t.Fatalf("low recall board: %+v", stats.LowRecall)
	}
	if stats.LowRecall[0].LastSeen == "" {
		t.Fatal("last_seen must be set from the newest log")
	}
}

func TestAggregateSearchLogsBoardSortedAndCapped(t *testing.T) {
	logs := []core.SearchLog{}
	for i := 0; i < lowRecallBoardSize+5; i++ {
		// distinct queries with decreasing miss counts: q0 misses 25, q1 24, ...
		misses := lowRecallBoardSize + 5 - i
		for j := 0; j < misses; j++ {
			logs = append(logs, searchLogEntry("q"+string(rune('a'+i%26))+string(rune('a'+i/26)), "keyword", 0, 10, nil))
		}
	}
	stats := aggregateSearchLogs(logs)
	if len(stats.LowRecall) != lowRecallBoardSize {
		t.Fatalf("board must cap at %d, got %d", lowRecallBoardSize, len(stats.LowRecall))
	}
	// most-missed query leads
	if stats.LowRecall[0].Count < stats.LowRecall[lowRecallBoardSize-1].Count {
		t.Fatalf("board must sort by miss count desc: %+v", stats.LowRecall)
	}
}
