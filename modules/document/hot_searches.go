/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// 热搜 (搜索设置 · 搜索建议): the top queries people actually ran, folded
// from the SearchLog telemetry that every completed search already writes —
// the query strings are stored normalized (trimmed, lowercased, inner
// whitespace collapsed), so counting is a pure in-memory fold. Zero-hit
// searches are excluded: a hot query that returns nothing is the low-recall
// board's job, not a suggestion.

const (
	// hotSearchScanLimit bounds the log scan; the board describes recent
	// behavior, not all history.
	hotSearchScanLimit = 5000
	// hotSearchDefaultDays is the trailing window when the request carries
	// no explicit ?days=.
	hotSearchDefaultDays = 7
)

type hotSearchRow struct {
	Query    string `json:"query"`
	Count    int    `json:"count"`
	LastSeen string `json:"last_seen"`
}

// hotSearchesHandler serves GET /query/_hot — the search box's hot-queries
// hint source. Honors 搜索设置 · 搜索建议: disabled → an empty but
// well-shaped response; the operator's default count fills in when the
// request carries no explicit ?size=.
func (h *APIHandler) hotSearchesHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	cfg := searchSettingsSnapshot()
	if !cfg.HotSearchesEnabledVal() {
		h.WriteJSON(w, util.MapStr{"items": []hotSearchRow{}, "days": 0}, http.StatusOK)
		return
	}

	size := cfg.EffectiveHotSearchesSize()
	if v, err := strconv.Atoi(strings.TrimSpace(req.URL.Query().Get("size"))); err == nil && v > 0 && v <= 20 {
		size = v
	}
	days := hotSearchDefaultDays
	if v, err := strconv.Atoi(strings.TrimSpace(req.URL.Query().Get("days"))); err == nil && v > 0 && v <= 90 {
		days = v
	}

	octx := orm.NewContextWithParent(req.Context())
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.SearchLog{})
	query := orm.NewQuery().Size(hotSearchScanLimit).
		Filter(orm.TermQuery("zero_hit", false), orm.Range("created").Gte("now-"+strconv.Itoa(days)+"d")).
		SortBy(orm.Sort{Field: "created", SortType: orm.DESC})
	res, err := orm.SearchV2(octx, query)
	if err != nil {
		log.Warnf("hot searches: log scan failed: %v", err)
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logs, _, err := elastic.DecodeHits[core.SearchLog](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	counts := map[string]*hotSearchRow{}
	order := []string{}
	for i := range logs {
		q := strings.TrimSpace(logs[i].Query)
		if q == "" {
			continue
		}
		row, ok := counts[q]
		if !ok {
			row = &hotSearchRow{Query: q}
			counts[q] = row
			order = append(order, q)
		}
		row.Count++
		// logs arrive newest-first, so the first sighting is the latest
		if row.LastSeen == "" && logs[i].Created != nil {
			row.LastSeen = logs[i].Created.Format("2006-01-02 15:04")
		}
	}

	items := make([]hotSearchRow, 0, len(order))
	for _, q := range order {
		items = append(items, *counts[q])
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Query < items[j].Query
	})
	if len(items) > size {
		items = items[:size]
	}

	h.WriteJSON(w, util.MapStr{"items": items, "days": days}, http.StatusOK)
}
