/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"net/http"
	"regexp"
	"strings"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// 搜索设置 · 结果干预 / 安全 helpers: the operator knobs that shape the
// result list around the recall. Blocked docs/datasources and the semantic
// threshold ride inside QueryDocuments; the pieces here run in the
// /query/_search handler around it.

// isBlockedQuery reports whether the cleaned query matches an
// operator-configured blocked pattern (搜索设置 · 安全). Patterns are RE2,
// matched case-insensitively; broken patterns are skipped (validation
// rejects them at save time anyway).
func isBlockedQuery(query string) bool {
	cfg := searchSettingsSnapshot()
	if cfg == nil || len(cfg.BlockedQueryPatterns) == 0 {
		return false
	}
	for _, p := range cfg.BlockedQueryPatterns {
		if strings.TrimSpace(p) == "" {
			continue
		}
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			continue
		}
		if re.MatchString(query) {
			return true
		}
	}
	return false
}

// pinnedHits fetches the documents of the first pinned rule whose query
// exactly matches (case-insensitive) the cleaned search query (搜索设置 ·
// 结果干预). The fetch rides QueryDocuments with an EMPTY query — pure
// filter mode — so curated docs surface regardless of whether their text
// matches the rule query; the normal permission / datasource / blocked
// filters still apply, and a deleted doc simply drops out of the terms
// clause. Returned hits follow the rule's document order.
func (h *APIHandler) pinnedHits(req *http.Request, query, integrationID string) []elastic.DocumentWithMeta[core.Document] {
	cfg := searchSettingsSnapshot()
	if cfg == nil || len(cfg.PinnedRules) == 0 || strings.TrimSpace(query) == "" {
		return nil
	}
	for _, rule := range cfg.PinnedRules {
		if !strings.EqualFold(strings.TrimSpace(rule.Query), query) || len(rule.DocumentIDs) == 0 {
			continue
		}
		builder := orm.NewQuery()
		builder.Must(orm.TermsQuery("id", rule.DocumentIDs))
		resp, err := QueryDocuments(req.Context(), builder, "", "", integrationID, "", "", "", "keyword", 0, nil)
		if err != nil {
			log.Warnf("search: pinned rule %q fetch failed: %v", rule.Query, err)
			return nil
		}
		out := &elastic.SearchResponseWithMeta[core.Document]{}
		if len(resp.Raw) > 0 {
			util.MustFromJSONBytes(resp.Raw, out)
		}
		log.Infof("search: pinned rule %q matched, fetched %d hits (raw total %d)", rule.Query, len(out.Hits.Hits), out.GetTotal())
		if len(out.Hits.Hits) == 0 {
			// operator signal: the rule matched but its docs did not come
			// back (permission/datasource scope, text-match miss, settings
			// interplay) — dump the exact query that ran to make this
			// diagnosable from the log alone
			log.Infof("search: pinned fetch empty, builder: %s", builder.ToString())
			raw := string(resp.Raw)
			if len(raw) > 800 {
				raw = raw[:800]
			}
			log.Infof("search: pinned fetch raw: %s", raw)
		}
		byID := map[string]elastic.DocumentWithMeta[core.Document]{}
		for _, hit := range out.Hits.Hits {
			byID[hit.ID] = hit
		}
		ordered := make([]elastic.DocumentWithMeta[core.Document], 0, len(rule.DocumentIDs))
		for _, id := range rule.DocumentIDs {
			if hit, ok := byID[id]; ok {
				ordered = append(ordered, hit)
			}
		}
		return ordered
	}
	return nil
}
