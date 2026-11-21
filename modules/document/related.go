/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// ErrRelatedNoEngine marks "engine unreachable / IK unavailable" so the
// tokenizer can degrade to whitespace splitting without surfacing an error.
var ErrRelatedNoEngine = errors.New("related: engine analyze unavailable")

// 相关搜索 (方案 B · 词面关联): queries other users actually ran that are
// lexically related to the current one, folded from the same SearchLog
// telemetry the hot-searches board uses. Scoring favors, in order:
//  1. extensions — queries that contain the current one ("easysearch" →
//     "easysearch 集群部署"): the most useful related search
//  2. shortenings — queries the current one contains ("easysearch 集群
//     部署 超时" → "easysearch 集群部署")
//  3. token overlap — tokens of the query appearing in the candidate.
//     Tokenization goes through the engine's _analyze API (ik_smart when the
//     analysis-ik plugin is installed, so CJK queries get real word
//     segmentation); any engine failure degrades to whitespace splitting.
//
// Zero-hit searches are excluded (same rationale as /query/_hot), the
// current query itself is never suggested.

const (
	relatedScanLimit     = 5000
	relatedDefaultDays   = 30
	relatedDefaultSize   = 8
	relatedMaxSize       = 20
	relatedMinTokenRunes = 2

	// engine-side tokenization: tiny payload, fail fast to the fallback
	relatedAnalyzeTimeout   = 2 * time.Second
	relatedAnalyzeAnalyzer  = "ik_smart"
	relatedAnalyzeMaxTokens = 16
)

// shared client; InsecureSkipVerify matches the framework's elastic
// client convention (self-signed dev engine certificates)
var relatedAnalyzeClient = &http.Client{
	Timeout: relatedAnalyzeTimeout,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

type relatedRow struct {
	Query string `json:"query"`
	Count int    `json:"count"`
}

// esAnalyzeTokens asks the configured system engine to segment q via
// _analyze. Returned tokens are deduplicated, order-preserving and filtered
// by the same minimum length as the fallback. Errors are expected (engine
// down, plugin absent) — callers degrade to whitespace splitting.
func esAnalyzeTokens(q string) ([]string, error) {
	cfg := elastic.GetConfigNoPanic(global.MustLookupString(elastic.GlobalSystemElasticsearchID))
	if cfg == nil {
		return nil, ErrRelatedNoEngine
	}
	endpoint := cfg.Endpoint
	if endpoint == "" && len(cfg.Endpoints) > 0 {
		endpoint = cfg.Endpoints[0]
	}
	if endpoint == "" {
		return nil, ErrRelatedNoEngine
	}
	body := util.MustToJSONBytes(util.MapStr{"analyzer": relatedAnalyzeAnalyzer, "text": q})
	req, err := http.NewRequest(http.MethodPost, strings.TrimSuffix(endpoint, "/")+"/_analyze", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if cfg.BasicAuth != nil {
		req.SetBasicAuth(cfg.BasicAuth.Username, string(cfg.BasicAuth.Password))
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := relatedAnalyzeClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrRelatedNoEngine
	}
	var parsed struct {
		Tokens []struct {
			Token string `json:"token"`
		} `json:"tokens"`
	}
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if err := util.FromJSONBytes(out, &parsed); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	tokens := make([]string, 0, len(parsed.Tokens))
	for _, t := range parsed.Tokens {
		tok := strings.ToLower(strings.TrimSpace(t.Token))
		if len([]rune(tok)) < relatedMinTokenRunes || seen[tok] {
			continue
		}
		seen[tok] = true
		tokens = append(tokens, tok)
		if len(tokens) >= relatedAnalyzeMaxTokens {
			break
		}
	}
	return tokens, nil
}

func relatedTokens(q string) []string {
	tokens, err := esAnalyzeTokens(q)
	if err != nil {
		log.Debugf("related searches: engine analyze fallback (whitespace) for %q: %v", q, err)
		return whitespaceTokens(q)
	}
	return tokens
}

func whitespaceTokens(q string) []string {
	// fallback: whitespace splitting (no engine / no IK plugin)
	parts := strings.Fields(q)
	tokens := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len([]rune(p)) < relatedMinTokenRunes || seen[p] {
			continue
		}
		seen[p] = true
		tokens = append(tokens, p)
	}
	return tokens
}

func relatedScore(q string, tokens []string, candidate string, count int) int {
	if candidate == q {
		return 0
	}
	// 1. extension of the current query
	if strings.Contains(candidate, q) {
		return 1000 + count
	}
	// 2. the current query covers the candidate (user typed a longer form);
	// tiny candidates ("c" inside "easysearch") are noise, not related
	if len([]rune(candidate)) >= 3 && strings.Contains(q, candidate) {
		return 500 + count
	}
	// 3. token overlap
	overlap := 0
	for _, t := range tokens {
		if strings.Contains(candidate, t) {
			overlap++
		}
	}
	if overlap > 0 {
		return overlap*100 + count
	}
	return 0
}

// relatedSearchesHandler serves GET /query/_related?q= — the result page
// sidebar's genuinely query-conditioned "related searches" source. Honors
// 搜索设置 · 搜索建议: disabled → an empty but well-shaped response.
func (h *APIHandler) relatedSearchesHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	q := strings.ToLower(util.CleanUserQuery(req.URL.Query().Get("q")))
	response := util.MapStr{"query": q, "items": []relatedRow{}}
	if q == "" {
		h.WriteJSON(w, response, http.StatusOK)
		return
	}
	if !searchSettingsSnapshot().RelatedSearchesEnabledVal() {
		h.WriteJSON(w, response, http.StatusOK)
		return
	}

	size := relatedDefaultSize
	if v, err := strconv.Atoi(strings.TrimSpace(req.URL.Query().Get("size"))); err == nil && v > 0 && v <= relatedMaxSize {
		size = v
	}
	days := relatedDefaultDays
	if v, err := strconv.Atoi(strings.TrimSpace(req.URL.Query().Get("days"))); err == nil && v > 0 && v <= 90 {
		days = v
	}

	octx := orm.NewContextWithParent(req.Context())
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.SearchLog{})
	query := orm.NewQuery().Size(relatedScanLimit).
		Filter(orm.TermQuery("zero_hit", false), orm.Range("created").Gte("now-"+strconv.Itoa(days)+"d")).
		SortBy(orm.Sort{Field: "created", SortType: orm.DESC})
	res, err := orm.SearchV2(octx, query)
	if err != nil {
		log.Warnf("related searches: log scan failed: %v", err)
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logs, _, err := elastic.DecodeHits[core.SearchLog](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	counts := map[string]int{}
	order := []string{}
	for i := range logs {
		c := strings.TrimSpace(logs[i].Query)
		if c == "" {
			continue
		}
		if _, ok := counts[c]; !ok {
			order = append(order, c)
		}
		counts[c]++
	}

	tokens := relatedTokens(q)
	type scoredRow struct {
		row   relatedRow
		score int
	}
	scored := make([]scoredRow, 0, len(order))
	for _, c := range order {
		if s := relatedScore(q, tokens, c, counts[c]); s > 0 {
			scored = append(scored, scoredRow{row: relatedRow{Query: c, Count: counts[c]}, score: s})
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].row.Query < scored[j].row.Query
	})
	if len(scored) > size {
		scored = scored[:size]
	}
	items := make([]relatedRow, 0, len(scored))
	for _, s := range scored {
		items = append(items, s.row)
	}

	response["items"] = items
	h.WriteJSON(w, response, http.StatusOK)
}
