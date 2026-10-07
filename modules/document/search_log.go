/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Search telemetry and the knowledge-gap loop (P2 + D5): every completed
// search is logged asynchronously; the operations overview aggregates the
// logs (strategy distribution, low-recall board, latency); queries that keep
// returning zero hits file a governance proposal so the knowledge base grows
// toward what people actually ask — the "system discovers what's missing"
// half of the iteration loop, the correction capture (D7) is the other half.

const (
	// searchLogScanLimit bounds the overview aggregation scan; the board is
	// about recent behavior, not all history.
	searchLogScanLimit = 10000
	// knowledgeGapThreshold is how many zero-hit searches one query needs
	// before a proposal is filed — one miss is a typo, three is a gap.
	knowledgeGapThreshold = 3
	// lowRecallBoardSize caps the low-recall board rows.
	lowRecallBoardSize = 20
)

// normalizeSearchQuery folds a query to its aggregatable form: trimmed,
// lowercased, inner whitespace collapsed.
func normalizeSearchQuery(query string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(query))), " ")
}

// searchQueryHash is the idempotency anchor for knowledge-gap proposals
// (proposal.ArticleID carries it, keeping the scanner-style
// ArticleID|Type key meaningful for query-level proposals).
func searchQueryHash(query string) string {
	return util.MD5digest("knowledge_gap:" + normalizeSearchQuery(query))
}

// recordSearchLog persists one completed search asynchronously; failures
// only log — telemetry must never take a search down.
func recordSearchLog(ctx context.Context, query, searchType, userID string, total int64, took time.Duration, rewritten bool) {
	normalized := normalizeSearchQuery(query)
	if normalized == "" || searchType == "" {
		return
	}
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	// refresh: the knowledge-gap check below reads the log it just wrote —
	// an un-refreshed read would under-count and miss the threshold
	octx.Refresh = orm.WaitForRefresh
	orm.WithModel(octx, &core.SearchLog{})
	entry := &core.SearchLog{
		Query:      normalized,
		SearchType: searchType,
		Total:      total,
		TookMS:     took.Milliseconds(),
		ZeroHit:    total == 0,
		Rewritten:  rewritten,
		UserID:     userID,
	}
	if err := orm.Create(octx, entry); err != nil {
		log.Warnf("search log create failed: %v", err)
		return
	}
	if entry.ZeroHit {
		maybeFileKnowledgeGap(octx, normalized)
	}
}

// maybeFileKnowledgeGap files (or refreshes) a knowledge-gap proposal once a
// query has crossed the zero-hit threshold. Best-effort: the proposal queue
// is the durable record, the log scan here only decides whether to write it.
func maybeFileKnowledgeGap(ctx *orm.Context, query string) {
	rctx := orm.NewContextWithParent(ctx)
	rctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(rctx, &core.SearchLog{})
	res, err := orm.SearchV2(rctx, orm.NewQuery().Size(1000).
		Filter(orm.TermQuery("query", query), orm.TermQuery("zero_hit", true)))
	if err != nil {
		return
	}
	hits, _, err := elastic.DecodeHits[core.SearchLog](res)
	if err != nil || len(hits) < knowledgeGapThreshold {
		return
	}

	gapID := searchQueryHash(query)
	wctx := orm.NewContextWithParent(ctx)
	wctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	wctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	// refresh: the governance queue must reflect the proposal immediately —
	// a reviewer looking right after the search that filed it should see it
	wctx.Refresh = orm.WaitForRefresh
	orm.WithModel(wctx, &core.WikiGovernanceProposal{})

	// anchored row id (article_id+type, generation-suffixed on reopen):
	// concurrent zero-hit searches racing through the check-then-create
	// window land on the same id, the loser overwrites identical content
	// instead of filing the same gap twice
	generation := 0
	pres, err := orm.SearchV2(wctx, orm.NewQuery().Size(100).
		Filter(orm.TermQuery("article_id", gapID), orm.TermQuery("type", core.WikiGovernanceKnowledgeGap)).
		SortBy(orm.Sort{Field: "created", SortType: orm.DESC}))
	if err == nil {
		existing, _, derr := elastic.DecodeHits[core.WikiGovernanceProposal](pres)
		if derr == nil && len(existing) > 0 {
			generation = len(existing)
			// refresh the counter on the row still open — found by scan, not
			// sort position: rows filed in the same tick order ambiguously,
			// and a resolved/dismissed gap must not be refiled by ordering luck
			for i := range existing {
				if existing[i].Status != core.WikiGovernanceOpen {
					continue
				}
				row := existing[i]
				row.Evidence = util.MapStr{"query": query, "zero_hit_count": len(hits)}
				_ = orm.Update(wctx, &row)
				break
			}
			return
		}
	}

	proposal := &core.WikiGovernanceProposal{
		ArticleID:    gapID,
		ArticleTitle: query,
		Type:         core.WikiGovernanceKnowledgeGap,
		Status:       core.WikiGovernanceOpen,
		Reason:       "query keeps returning zero hits — the knowledge base lacks a page for it",
		Evidence:     util.MapStr{"query": query, "zero_hit_count": len(hits)},
	}
	proposal.ID = core.AnchoredProposalID(gapID, core.WikiGovernanceKnowledgeGap, generation+1)
	if err := orm.Create(wctx, proposal); err != nil {
		log.Warnf("knowledge gap proposal create failed: %v", err)
	}
}

// searchOpsOverview serves GET /search/ops/overview: what people searched,
// which strategies ran, where recall came up empty and what it cost in
// latency — the operator's half of the iteration loop.
func (h *APIHandler) searchOpsOverview(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	octx := orm.NewContextWithParent(req.Context())
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.SearchLog{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(searchLogScanLimit).
		SortBy(orm.Sort{Field: "created", SortType: orm.DESC}))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logs, _, err := elastic.DecodeHits[core.SearchLog](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stats := aggregateSearchLogs(logs)
	stats.SignalsCaveat = zeroHitSignalsCaveat
	markFiledGaps(req.Context(), &stats)
	h.WriteJSON(w, stats, http.StatusOK)
}

// markFiledGaps flags board rows whose query already has an open
// knowledge-gap proposal, so reviewers see what's already in the queue.
func markFiledGaps(ctx context.Context, stats *searchOpsStats) {
	if len(stats.LowRecall) == 0 {
		return
	}
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiGovernanceProposal{})
	for i := range stats.LowRecall {
		res, err := orm.SearchV2(octx, orm.NewQuery().Size(1).
			Filter(orm.TermQuery("article_id", searchQueryHash(stats.LowRecall[i].Query)),
				orm.TermQuery("type", core.WikiGovernanceKnowledgeGap)))
		if err != nil {
			continue
		}
		if hits, _, derr := elastic.DecodeHits[core.WikiGovernanceProposal](res); derr == nil && len(hits) > 0 {
			stats.LowRecall[i].Proposal = true
		}
	}
}

// searchOpsStats is the overview response body; aggregateSearchLogs is pure
// so the numbers are unit-testable without a store.
type searchOpsStats struct {
	TotalSearches int64   `json:"total_searches"`
	ZeroHits      int64   `json:"zero_hit_searches"`
	ZeroHitRate   float64 `json:"zero_hit_rate"`
	AvgTookMS     int64   `json:"avg_took_ms"`
	MaxTookMS     int64   `json:"max_took_ms"`
	// Rewrite counts searches where the D8 rewrite leg fired and how many
	// of those still came back empty — the zero-hit rate among rewritten
	// searches vs the overall one is the leg's observable lift.
	Rewritten     int64              `json:"rewritten_searches"`
	RewrittenMiss int64              `json:"rewritten_zero_hit_searches"`
	Strategies    []searchOpsRow     `json:"strategies"`
	LowRecall     []searchOpsLowMiss `json:"low_recall"`
	// SignalsCaveat qualifies the zero-hit family: the pinyin analyzer emits
	// single-letter tokens under an OR match, so the text leg matches almost
	// any non-empty query and these signals undercount by construction.
	// Demo-grade until the analyzer is tightened — set by the handler, not
	// by the pure aggregation.
	SignalsCaveat string `json:"signals_caveat"`
}

// zeroHitSignalsCaveat is the standing annotation for the zero-hit signals
// (zero_hit_rate, low-recall board, knowledge-gap backflow). Analyzer-level
// tightening is the root fix and a separate project; until it lands, the
// operator-facing numbers must say what they are.
const zeroHitSignalsCaveat = "zero-hit signals are demo-grade: the pinyin analyzer matches nearly any non-empty query on the text leg, so zero_hit_rate undercounts and the low-recall board and gap backflow rarely fire on organic traffic"

type searchOpsRow struct {
	Type      string `json:"type"`
	Count     int64  `json:"count"`
	ZeroHits  int64  `json:"zero_hits"`
	Rewritten int64  `json:"rewritten"`
	AvgTookMS int64  `json:"avg_took_ms"`
}

type searchOpsLowMiss struct {
	Query    string `json:"query"`
	Count    int    `json:"count"`
	LastSeen string `json:"last_seen"`
	Proposal bool   `json:"proposal"` // a knowledge_gap proposal exists for this query
}

// aggregateSearchLogs folds the scan into the overview numbers.
func aggregateSearchLogs(logs []core.SearchLog) searchOpsStats {
	stats := searchOpsStats{Strategies: []searchOpsRow{}, LowRecall: []searchOpsLowMiss{}}
	if len(logs) == 0 {
		return stats
	}

	byType := map[string]*searchOpsRow{}
	typeOrder := []string{}
	zeroByQuery := map[string]*searchOpsLowMiss{}
	zeroOrder := []string{}
	var tookTotal int64

	for i := range logs {
		entry := &logs[i]
		stats.TotalSearches++
		stats.ZeroHits += btoi(entry.ZeroHit)
		stats.Rewritten += btoi(entry.Rewritten)
		if entry.Rewritten && entry.ZeroHit {
			stats.RewrittenMiss++
		}
		tookTotal += entry.TookMS
		if entry.TookMS > stats.MaxTookMS {
			stats.MaxTookMS = entry.TookMS
		}

		row, ok := byType[entry.SearchType]
		if !ok {
			row = &searchOpsRow{Type: entry.SearchType}
			byType[entry.SearchType] = row
			typeOrder = append(typeOrder, entry.SearchType)
		}
		row.Count++
		row.ZeroHits += btoi(entry.ZeroHit)
		row.Rewritten += btoi(entry.Rewritten)
		row.AvgTookMS += entry.TookMS

		if entry.ZeroHit && entry.Query != "" {
			miss, ok := zeroByQuery[entry.Query]
			if !ok {
				miss = &searchOpsLowMiss{Query: entry.Query}
				zeroByQuery[entry.Query] = miss
				zeroOrder = append(zeroOrder, entry.Query)
			}
			miss.Count++
			// logs arrive newest-first, so the first sighting is the latest
			if miss.LastSeen == "" && entry.Created != nil {
				miss.LastSeen = entry.Created.Format(time.RFC3339)
			}
		}
	}

	stats.ZeroHitRate = float64(stats.ZeroHits) / float64(stats.TotalSearches)
	stats.AvgTookMS = tookTotal / int64(len(logs))
	for _, name := range typeOrder {
		row := byType[name]
		row.AvgTookMS /= int64(row.Count)
		stats.Strategies = append(stats.Strategies, *row)
	}

	for _, query := range zeroOrder {
		stats.LowRecall = append(stats.LowRecall, *zeroByQuery[query])
	}
	sort.SliceStable(stats.LowRecall, func(i, j int) bool {
		if stats.LowRecall[i].Count != stats.LowRecall[j].Count {
			return stats.LowRecall[i].Count > stats.LowRecall[j].Count
		}
		return stats.LowRecall[i].Query < stats.LowRecall[j].Query
	})
	if len(stats.LowRecall) > lowRecallBoardSize {
		stats.LowRecall = stats.LowRecall[:lowRecallBoardSize]
	}
	return stats
}

func btoi(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
