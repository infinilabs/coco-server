/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common/fingerprint"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

/* Topic-centric compilation (DESIGN-topic-pages.md).
 *
 * Pages are subjects, not single entities. After MAP/REDUCE:
 *
 *	CLUSTER  reduced keys merge into topics by document co-occurrence
 *	         (pure code) — the page granularity the left menu shows
 *	NAME     one LLM call titles/scopes every cluster
 *	OUTLINE  per topic the LLM plans 4-8 sections across dimensions
 *	         (概述/背景/核心机制/关键实体/时间线/数据/争议), update-aware:
 *	         it may claim an existing topic page to enrich
 *	FILL     per topic the statements route into the planned sections and
 *	         each section is drafted as finished prose with claim refs
 *
 * Every stage's LLM output is cached by content fingerprint — a recompile
 * of unchanged corpus costs zero LLM beyond cache reads. One proposal
 * carries the whole page; entities stay in the ontology pool (peer of
 * pages, never auto-promoted to articles here).
 */

const (
	topicMaxStatements  = 100 // per-topic claim pool cap for prompts/evidence
	topicOutlineSamples = 16  // claims the OUTLINE call may see
	topicMaxKeyEntities = 10
	topicLLMOutlineTTL  = 150 * time.Second
	topicLLMFillTTL     = 300 * time.Second

	// content-based update matching: the LLM names clusters slightly
	// differently every run, so title/alias equality alone mints a
	// parallel page per recompile. Source-document overlap is the
	// ground truth — a cluster built on the same docs enriches the page
	// those docs built, whatever it got named.
	topicMatchDocOverlap = 0.6 // min shared-doc ratio to claim a page
	topicTwinDocOverlap  = 0.7 // twin absorb: near-total doc sharing…
	topicTwinKeyOverlap  = 0.4 // …plus key-entity agreement
)

var topicCacheKVKey = "wiki-compile-topic-cache"

/* ---------------- types ---------------- */

// topicStatement is one pooled claim of a topic: the reduce statement plus
// the entity/concept name it was extracted under (the FILL prompt shows
// both so the model can route by subject, not just by text).
type topicStatement struct {
	Index   int    `json:"index"` // 1-based, stable across the prompts
	DocID   string `json:"doc_id"`
	Text    string `json:"text"`
	Excerpt string `json:"excerpt,omitempty"`
	KeyName string `json:"key_name,omitempty"`
}

// topicCluster is one subject: the reduced keys that cohere, their pooled
// claims, and the planning fields filled in by NAME/OUTLINE.
type topicCluster struct {
	Keys       []ReducedKey
	Statements []topicStatement
	Docs       map[string]bool

	Title   string
	Scope   string
	Aliases []string

	UpdateArticleID string   // existing topic page this run enriches
	AbsorbIDs       []string // thin AI pages folded into this topic on apply
}

// topicOutline is the OUTLINE call's JSON contract.
type topicOutline struct {
	UpdateTitle string   `json:"update_title"` // exact existing title when this topic IS that page
	Sections    []string `json:"sections"`
}

// topicFillSection is one FILL section: drafted prose plus the claim
// indexes (1-based, into the cluster's statement pool) it draws on.
type topicFillSection struct {
	Heading string `json:"heading"`
	Body    string `json:"body"`
	Refs    []int  `json:"refs"`
}

// topicFill is the FILL call's JSON contract.
type topicFill struct {
	Sections []topicFillSection `json:"sections"`
}

// topicDraft is a topic ready to file as a proposal.
type topicDraft struct {
	Cluster     *topicCluster
	Title       string
	Scope       string
	UpdateTitle string
	Sections    []core.WikiSection
	Claims      []compileStatement
	KeyEntities []string
	// entity name -> declared ontology type (person/organization/…), one
	// LLM call per page; absent names fall back to concept at apply time
	KeyEntityTypes map[string]string
	Planned        int
}

/* ---------------- CLUSTER (pure) ---------------- */

// clusterTopics merges reduced keys into topics by document co-occurrence:
// keys citing the same documents belong to the same subject. Greedy over
// the most-evidence keys first (ReduceExtracts sorts that way); a key
// joins the cluster it shares most documents with when the sharing is
// strong (≥2 docs, or any doc for the long tail), else seeds a new topic
// under the topic cap. Tiny leftover clusters fold into their nearest
// neighbor — a 24-doc KB lands at a handful of rich pages, not 300 stubs.
func clusterTopics(reduced []ReducedKey, docCount int) []*topicCluster {
	maxTopics := docCount / 3
	if maxTopics < 6 {
		maxTopics = 6
	}
	if maxTopics > 20 {
		maxTopics = 20
	}

	keyDocs := func(k ReducedKey) map[string]bool {
		docs := map[string]bool{}
		for _, s := range k.Statements {
			if s.DocID != "" {
				docs[s.DocID] = true
			}
		}
		return docs
	}

	emptyKeys := 0
	var clusters []*topicCluster
	for _, rk := range reduced {
		if len(rk.Statements) == 0 {
			emptyKeys++
			continue
		}
		docs := keyDocs(rk)
		best, bestShared := -1, 0
		for ci := range clusters {
			shared := 0
			for d := range docs {
				if clusters[ci].Docs[d] {
					shared++
				}
			}
			if shared > bestShared {
				best, bestShared = ci, shared
			}
		}
		join := best >= 0 && (bestShared >= 2 || (bestShared >= 1 && len(rk.Statements) <= 2))
		if !join && len(clusters) >= maxTopics {
			join = best >= 0 // cap reached: the strongest sharer absorbs it
		}
		if !join {
			clusters = append(clusters, &topicCluster{Keys: []ReducedKey{rk}, Docs: docs})
			continue
		}
		c := clusters[best]
		c.Keys = append(c.Keys, rk)
		for d := range docs {
			c.Docs[d] = true
		}
	}

	// fold tiny clusters (single key, ≤2 statements) into their best-sharing
	// peer, then pool statements per cluster with cross-key dedup
	var out []*topicCluster
	for ci := range clusters {
		c := clusters[ci]
		if len(c.Keys) == 1 && len(c.Keys[0].Statements) <= 2 {
			dropped := c
			best, bestShared := -1, 0
			for cj := range clusters {
				if cj == ci {
					continue
				}
				shared := 0
				for d := range dropped.Docs {
					if clusters[cj].Docs[d] {
						shared++
					}
				}
				if shared > bestShared {
					best, bestShared = cj, shared
				}
			}
			if best >= 0 && bestShared > 0 {
				clusters[best].Keys = append(clusters[best].Keys, dropped.Keys...)
				for d := range dropped.Docs {
					clusters[best].Docs[d] = true
				}
				continue
			}
		}
		out = append(out, c)
	}

	for _, c := range out {
		seen := map[string]bool{}
		idx := 0
		for _, k := range c.Keys {
			for _, s := range k.Statements {
				claim := strings.TrimSpace(s.Text)
				if claim == "" || s.DocID == "" {
					continue
				}
				h := fingerprint.Normalize(claim)
				if seen[h] {
					continue
				}
				seen[h] = true
				idx++
				c.Statements = append(c.Statements, topicStatement{
					Index: idx, DocID: s.DocID, Text: claim,
					Excerpt: s.Excerpt, KeyName: k.Name,
				})
				if len(c.Statements) >= topicMaxStatements {
					break
				}
			}
			if len(c.Statements) >= topicMaxStatements {
				break
			}
		}
	}

	// noise floor: a topic the corpus barely mentions is not a page
	kept := out[:0]
	for _, c := range out {
		if len(c.Statements) >= 3 {
			kept = append(kept, c)
		}
	}
	// most material first — the compose loop's capacity goes where the corpus is
	sort.SliceStable(kept, func(i, j int) bool { return len(kept[i].Statements) > len(kept[j].Statements) })
	log.Infof("wiki topic compiler: cluster debug — %d keys (%d without statements), %d pre-floor clusters, %d kept", len(reduced), emptyKeys, len(out), len(kept))
	for i, c := range kept {
		log.Infof("wiki topic compiler: cluster[%d] keys=%d stmts=%d docs=%d", i, len(c.Keys), len(c.Statements), len(c.Docs))
	}
	return kept
}

// keyEntityNames picks the cluster's headline entities (kind=entity, most
// evidence first) for the page's KeyEntities chips and the prompts.
func keyEntityNames(c *topicCluster) []string {
	names := make([]string, 0, topicMaxKeyEntities)
	for _, k := range c.Keys { // cluster keys are appended evidence-first by reduce order
		if k.Kind != "entity" || len(k.Statements) == 0 {
			continue
		}
		if util.AnyInArrayEquals(names, k.Name) {
			continue
		}
		names = append(names, k.Name)
		if len(names) >= topicMaxKeyEntities {
			break
		}
	}
	return names
}

// classifyKeyEntities types the page's key entities against the KB's
// ontology vocabulary with one small LLM call, so promoted entities land
// with a real type (person/organization/event…) instead of the concept
// fallback. Best-effort: any error returns nil and the apply side degrades
// every untyped entity to concept.
func classifyKeyEntities(ctx context.Context, model *core.ModelId, kbID, title string, names []string) map[string]string {
	if len(names) == 0 || kbID == "" {
		return nil
	}
	doc := ResolveOntologySchema(ctx, kbID)
	if doc == nil || len(doc.EntityTypes) == 0 {
		return nil
	}
	declared := make([]string, 0, len(doc.EntityTypes))
	for _, t := range doc.EntityTypes {
		declared = append(declared, fmt.Sprintf("%s(%s)", t.Name, orDash(t.Label)))
	}
	user := fmt.Sprintf(`主题页「%s」提取出以下关键实体，请为每个实体判断本体类型。

可选类型（只能从中选择）:
%s

实体列表:
%s

只输出一个 JSON 对象，键为实体名、值为类型名，不要输出任何其他内容。`,
		title, strings.Join(declared, "、"), strings.Join(names, "\n"))
	raw, err := topicLLMCall(ctx, model, "你是知识库编辑，负责为实体标注本体类型，只输出 JSON。", user, topicLLMOutlineTTL)
	if err != nil {
		log.Warnf("wiki topic compiler: entity typing LLM failed: %v", err)
		return nil
	}
	// fence-tolerant slice: keep only the outermost JSON object
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i > 0 {
		raw = raw[i:]
	}
	if j := strings.LastIndex(raw, "}"); j >= 0 && j < len(raw)-1 {
		raw = raw[:j+1]
	}
	out := map[string]string{}
	if err := util.FromJSONBytes([]byte(raw), &out); err != nil {
		log.Warnf("wiki topic compiler: entity typing unparsable: %v", err)
		return nil
	}
	typed := make(map[string]string, len(names))
	for _, name := range names {
		if t := strings.TrimSpace(out[name]); t != "" && doc.typeDef(t) != nil {
			typed[name] = t
		}
	}
	if len(typed) == 0 {
		return nil
	}
	return typed
}

/* ---------------- LLM plumbing ---------------- */

// topicLLMCall runs one structured topic call: stream-captured content,
// reasoning-strip, fence-tolerant. Empty content is an error (retry-able).
func topicLLMCall(ctx context.Context, model *core.ModelId, system, user string, timeout time.Duration) (string, error) {
	llm, err := langchain.SimplyGetLLM(model.ProviderID, model.ID, "")
	if err != nil {
		return "", err
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var out strings.Builder
	resp, err := llm.GenerateContent(cctx, []llms.MessageContent{
		langchain.SystemTextParts(system),
		llms.TextParts(llms.ChatMessageTypeHuman, user),
	}, llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
		out.Write(chunk)
		return nil
	}))
	if err != nil {
		return "", err
	}
	content := ""
	if resp != nil && len(resp.Choices) > 0 {
		content = resp.Choices[0].Content
	}
	if strings.TrimSpace(content) == "" {
		content = out.String()
	}
	if i := strings.Index(content, "</think>"); i >= 0 {
		content = content[i+len("</think>"):]
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("empty topic LLM response")
	}
	return content, nil
}

// extractJSONValue finds the first JSON value of the target shape in a
// response that may wrap it in prose or a code fence.
func extractJSONValue[T any](s string) (*T, error) {
	if m := compileFencePattern.FindStringSubmatch(s); len(m) > 1 {
		s = m[1]
	}
	start, end := strings.Index(s, "{"), strings.LastIndex(s, "}")
	arrStart, arrEnd := strings.Index(s, "["), strings.LastIndex(s, "]")
	if arrStart >= 0 && (start < 0 || arrStart < start) {
		start, end = arrStart, arrEnd
	}
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON in topic LLM output")
	}
	var v T
	if err := json.Unmarshal([]byte(s[start:end+1]), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// topicCacheGet/topicCachePut persist outline/fill outputs by content
// fingerprint — a recompile of the same corpus replays them with zero LLM.
func topicCacheGet(fp string, out any) bool {
	buf, err := kv.GetCompressedValue(topicCacheKVKey, []byte(fp))
	if err != nil || buf == nil {
		return false
	}
	if err := util.FromJSONBytes(buf, out); err != nil {
		return false
	}
	return true
}

func topicCachePut(fp string, v any) {
	if err := kv.AddValueCompress(topicCacheKVKey, []byte(fp), util.MustToJSONBytes(v)); err != nil {
		log.Debugf("wiki topic compiler: persist topic cache [%s] failed: %v", fp, err)
	}
}

// topicFingerprint stabilizes the cache key of one stage: model, stage,
// topic title and the exact material the prompt carries.
func topicFingerprint(model *core.ModelId, stage, title string, material []string) string {
	fp, _, ok := fingerprint.Compute(strings.Join(append([]string{model.ProviderID + "/" + model.ID, stage, title}, material...), "\x1f"))
	if !ok {
		return ""
	}
	return fp
}

/* ---------------- NAME (one call for all clusters) ---------------- */

type topicNaming struct {
	Index   int      `json:"index"`
	Title   string   `json:"title"`
	Scope   string   `json:"scope"`
	Aliases []string `json:"aliases"`
}

// nameTopics titles and scopes every cluster in ONE call: each cluster is
// summarized by its key names and a few sample claims; the model returns
// index-addressed metadata. Fallback title on any trouble: the cluster's
// strongest key name — the pipeline never dies on the naming step.
func nameTopics(ctx context.Context, model *core.ModelId, clusters []*topicCluster, existingTitles []string) {
	if len(clusters) == 0 {
		return
	}
	type clusterBrief struct {
		Index   int      `json:"index"`
		Keys    []string `json:"keys"`
		Samples []string `json:"samples"`
	}
	briefs := make([]clusterBrief, 0, len(clusters))
	for i, c := range clusters {
		keys := make([]string, 0, 12)
		for _, k := range c.Keys {
			if len(k.Statements) == 0 {
				continue
			}
			keys = append(keys, k.Name)
			if len(keys) >= 12 {
				break
			}
		}
		samples := make([]string, 0, 4)
		for _, s := range c.Statements {
			samples = append(samples, truncateCompile(s.Text, 100))
			if len(samples) >= 4 {
				break
			}
		}
		briefs = append(briefs, clusterBrief{Index: i, Keys: keys, Samples: samples})
	}
	material, _ := json.Marshal(briefs)
	existing := "（无）"
	if len(existingTitles) > 0 {
		existing = strings.Join(existingTitles, "、")
	}
	user := fmt.Sprintf(`以下是知识库语料聚类出的 %d 个主题。为每个主题起一个主题页标题、一句话范围说明，并给出最多 3 个别名。

规则：
- 标题是名词短语，概括该主题的主体（如「支付与清结算」「客户服务体系」），不要用单个实体名
- 标题语言跟随材料语言
- 标题之间要互相区分，且不与已有页面重复（已有页面：%s）
- scope 一句话说清这个主题页覆盖什么

聚类摘要：
%s

只返回 JSON 数组：[{"index":0,"title":"...","scope":"...","aliases":["..."]}]`, len(clusters), existing, material)

	fp := topicFingerprint(model, "name", fmt.Sprintf("kb:%d", len(clusters)), []string{string(material), existing})
	var named []topicNaming
	if fp != "" && topicCacheGet(fp, &named) {
		applyNaming(clusters, named)
		return
	}
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := topicLLMCall(ctx, model, "你是知识库主编，负责把零散的实体与主张组织成少数几个主题页。", user, topicLLMOutlineTTL)
		if err != nil {
			continue
		}
		if arr, err := extractJSONValue[[]topicNaming](raw); err == nil && arr != nil {
			named = *arr
			break
		}
	}
	if fp != "" && len(named) > 0 {
		topicCachePut(fp, named)
	}
	applyNaming(clusters, named)
}

func applyNaming(clusters []*topicCluster, named []topicNaming) {
	for _, n := range named {
		if n.Index < 0 || n.Index >= len(clusters) {
			continue
		}
		if t := strings.TrimSpace(n.Title); t != "" {
			clusters[n.Index].Title = t
			clusters[n.Index].Scope = strings.TrimSpace(n.Scope)
			clusters[n.Index].Aliases = n.Aliases
		}
	}
	// fallback: strongest key name — every cluster must carry a title
	for i, c := range clusters {
		if c.Title == "" && len(c.Keys) > 0 {
			c.Title = c.Keys[0].Name
			clusters[i] = c
		}
	}
}

/* ---------------- update-first matching (pure) ---------------- */

// matchTopicTargets resolves each cluster against the KB's existing pages:
// a topic page whose title/alias matches becomes the enrichment target
// (update-first — new material enriches what exists); failing that, a page
// built on largely the same source documents claims the cluster, because
// cluster naming drifts between runs and title equality alone would mint a
// parallel page per recompile. Topic pages that shadow the same documents
// become absorb twins (folded into the target when the proposal applies);
// thin AI entity pages whose title matches a cluster key are absorb
// candidates too. Hand-written pages are never absorb candidates.
func matchTopicTargets(clusters []*topicCluster, existing []core.WikiArticle) {
	topicPages := map[string]string{} // normalized title/alias -> article id
	// live topic pages pre-summarized for content matching
	type pageSig struct {
		id    string
		ai    bool
		docs  map[string]bool
		keys  map[string]bool
		title string
	}
	var sigs []pageSig
	var thinAIPages []core.WikiArticle // absorb candidates by title
	for _, a := range existing {
		if a.Status == core.WikiArticleArchived {
			continue
		}
		norm := fingerprint.Normalize(a.Title)
		if a.PageType == core.WikiPageTypeTopic {
			topicPages[norm] = a.ID
			for _, al := range a.Aliases {
				topicPages[fingerprint.Normalize(al)] = a.ID
			}
			sig := pageSig{id: a.ID, ai: a.AIGenerated, title: a.Title, docs: map[string]bool{}, keys: map[string]bool{}}
			for _, s := range a.Sources {
				if s.DocID != "" {
					sig.docs[s.DocID] = true
				}
			}
			for _, sec := range a.Outline {
				for _, d := range sec.DocIDs {
					sig.docs[d] = true
				}
			}
			for _, k := range a.KeyEntities {
				if n := fingerprint.Normalize(k); n != "" {
					sig.keys[n] = true
				}
			}
			sigs = append(sigs, sig)
			continue
		}
		if a.AIGenerated && (a.PageType == core.WikiPageTypeEntity || a.PageType == core.WikiPageTypeConcept) {
			thin := len(a.Sources) < 3 || len([]rune(a.Content)) < 500
			if thin {
				thinAIPages = append(thinAIPages, a)
			}
		}
	}

	claimed := map[string]bool{} // one cluster per page: the next-strongest match must pick another page
	for _, c := range clusters {
		if id, ok := topicPages[fingerprint.Normalize(c.Title)]; ok {
			c.UpdateArticleID = id
			claimed[id] = true
		}
		for _, al := range c.Aliases {
			if id, ok := topicPages[fingerprint.Normalize(al)]; ok {
				c.UpdateArticleID = id
				claimed[id] = true
			}
		}
		keyNorms := map[string]bool{}
		for _, k := range c.Keys {
			keyNorms[fingerprint.Normalize(k.Name)] = true
		}

		// content pass: only when naming drifted past the title match
		if c.UpdateArticleID == "" && len(c.Docs) > 0 {
			best, bestDoc, bestKey := -1, 0.0, 0.0
			for i := range sigs {
				if claimed[sigs[i].id] {
					continue
				}
				dr := overlapRatio(c.Docs, sigs[i].docs)
				if dr < topicMatchDocOverlap {
					continue
				}
				kr := overlapRatio(keyNorms, sigs[i].keys)
				if dr > bestDoc || (dr == bestDoc && kr > bestKey) {
					best, bestDoc, bestKey = i, dr, kr
				}
			}
			if best >= 0 {
				c.UpdateArticleID = sigs[best].id
				claimed[sigs[best].id] = true
			}
		}

		// twin absorb: AI topic pages shadowing this cluster's documents
		// (the parallel pages past drift-prone runs left behind); human
		// writing is never folded away
		for i := range sigs {
			s := &sigs[i]
			if s.id == "" || !s.ai || s.id == c.UpdateArticleID || claimed[s.id] {
				continue
			}
			if overlapRatio(c.Docs, s.docs) >= topicTwinDocOverlap && overlapRatio(keyNorms, s.keys) >= topicTwinKeyOverlap {
				c.AbsorbIDs = append(c.AbsorbIDs, s.id)
				claimed[s.id] = true
			}
		}

		for _, a := range thinAIPages {
			if keyNorms[fingerprint.Normalize(a.Title)] {
				c.AbsorbIDs = append(c.AbsorbIDs, a.ID)
			}
		}
	}
}

// overlapRatio is |a∩b| over the smaller set: a 1-doc cluster citing a page
// that also cites that doc matches at 1.0, while unrelated docs sit at 0.
func overlapRatio(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if b[k] {
			inter++
		}
	}
	return float64(inter) / float64(min(len(a), len(b)))
}

/* ---------------- OUTLINE + FILL (per topic, cached) ---------------- */

// planTopicOutline asks the LLM for the page's section plan across
// dimensions, update-aware: given the KB's existing topic pages it may
// claim one as this topic's enrichment target.
func planTopicOutline(ctx context.Context, model *core.ModelId, c *topicCluster, existingPages []core.WikiArticle) (*topicOutline, error) {
	keyEntities := keyEntityNames(c)
	claims := make([]string, 0, topicOutlineSamples)
	for _, s := range c.Statements {
		claims = append(claims, fmt.Sprintf("%d. %s", s.Index, truncateCompile(s.Text, 120)))
		if len(claims) >= topicOutlineSamples {
			break
		}
	}
	var pageBriefs []string
	for _, a := range existingPages {
		if a.PageType != core.WikiPageTypeTopic || a.Status == core.WikiArticleArchived {
			continue
		}
		pageBriefs = append(pageBriefs, "「"+a.Title+"」")
		if len(pageBriefs) >= 20 {
			break
		}
	}
	existing := "（无）"
	if len(pageBriefs) > 0 {
		existing = strings.Join(pageBriefs, "、")
	}
	// the update target's own headings anchor the outline: reuse their
	// wording verbatim so re-compiles refresh sections instead of growing
	// paraphrased twins
	anchorRule := ""
	for _, a := range existingPages {
		if c.UpdateArticleID != "" && a.ID == c.UpdateArticleID && len(a.Outline) > 0 {
			hs := make([]string, 0, len(a.Outline))
			for _, s := range a.Outline {
				hs = append(hs, s.Heading)
			}
			anchorRule = fmt.Sprintf("\n- 该主题将更新既有页面「%s」，其现有章节标题为：%s。大纲必须沿用这些标题的原措辞（可增删章节，不要改写、不要翻译）", a.Title, strings.Join(hs, "、"))
			break
		}
	}
	user := fmt.Sprintf(`为知识库主题页「%s」规划文章大纲。范围：%s。

参考维度（按材料取舍，不要求全部）：概述、背景与上下文、核心机制或关键维度（可拆多节）、关键实体与关系、时间线（仅当材料含时间脉络）、数据与事实、争议与开放问题、来源与延伸。
规则：
- 4-8 个章节标题，标题语言跟随材料语言，必须覆盖材料实际支持的内容
- 核心维度优先：材料最丰富的侧面拆成独立章节
- 若该主题实质上是已有主题页的一部分，返回 update_title 为那个页面的完整标题；否则 update_title 留空
- 已有主题页：%s%s

主题核心实体：%s
编号主张样例：
%s

只返回 JSON：{"update_title":"","sections":["...","..."]}`, c.Title, orDash(c.Scope), existing, anchorRule,
		strings.Join(keyEntities, "、"), strings.Join(claims, "\n"))

	fp := topicFingerprint(model, "outline", c.Title, claims)
	if fp != "" {
		var cached topicOutline
		if topicCacheGet(fp, &cached) && len(cached.Sections) > 0 {
			return &cached, nil
		}
	}
	var outline *topicOutline
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := topicLLMCall(ctx, model,
			"你是百科主编，只依据给定材料规划大纲，不引入材料之外的章节。", user, topicLLMOutlineTTL)
		if err != nil {
			continue
		}
		if o, err := extractJSONValue[topicOutline](raw); err == nil && len(o.Sections) > 0 {
			outline = o
			break
		}
	}
	if outline == nil {
		return nil, fmt.Errorf("outline planning failed for [%s]", c.Title)
	}
	if len(outline.Sections) > 8 {
		outline.Sections = outline.Sections[:8]
	}
	if fp != "" {
		topicCachePut(fp, outline)
	}
	return outline, nil
}

// fillTopicSections routes the pooled claims into the planned sections and
// drafts each as finished prose citing claim indexes — one call per topic.
func fillTopicSections(ctx context.Context, model *core.ModelId, c *topicCluster, headings []string) (*topicFill, error) {
	claims := make([]string, 0, len(c.Statements))
	for _, s := range c.Statements {
		line := fmt.Sprintf("%d. [%s] %s", s.Index, orDash(s.KeyName), s.Text)
		if ex := strings.TrimSpace(s.Excerpt); ex != "" && ex != s.Text {
			line += "（摘录：" + truncateCompile(ex, 100) + "）"
		}
		claims = append(claims, line)
	}
	user := fmt.Sprintf(`为知识库主题页「%s」撰写各章节正文。大纲章节：%s。

规则：
- 只使用下方编号主张及其摘录，绝不引入材料之外的事实、数字或评价
- 每节 1-3 段成文散文（不要列表化罗列主张），句末用 [编号] 标注所依据的主张
- 每节至少引用 1 条主张；某章节确实无材料支撑时，body 留空、refs 为空数组
- 语言跟随材料语言（材料是繁体就用繁体，是英文就用英文）
- 正文里提及主题核心实体时直接写实体名，不要加引号或前缀

编号主张：
%s

只返回 JSON：{"sections":[{"heading":"与大纲一致的章节标题","body":"...","refs":[1,3]}]}`,
		c.Title, strings.Join(headings, "、"), strings.Join(claims, "\n"))

	fp := topicFingerprint(model, "fill", c.Title, append([]string{strings.Join(headings, "|")}, claims...))
	if fp != "" {
		var cached topicFill
		if topicCacheGet(fp, &cached) && len(cached.Sections) > 0 {
			return &cached, nil
		}
	}
	var fill *topicFill
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := topicLLMCall(ctx, model,
			"你是严谨的百科编辑，只依据给定材料写作，宁可留空也不编造。", user, topicLLMFillTTL)
		if err != nil {
			continue
		}
		if f, err := extractJSONValue[topicFill](raw); err == nil && len(f.Sections) > 0 {
			fill = f
			break
		}
	}
	if fill == nil {
		return nil, fmt.Errorf("section drafting failed for [%s]", c.Title)
	}
	if fp != "" {
		topicCachePut(fp, fill)
	}
	return fill, nil
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return strings.TrimSpace(s)
}

// composeTopicDraft runs OUTLINE → FILL for one cluster and assembles the
// structured draft: WikiSections with per-section provenance, the pooled
// claims for the evidence chain, and the page's key entities.
func composeTopicDraft(ctx context.Context, model *core.ModelId, kbID string, c *topicCluster, existing []core.WikiArticle) (*topicDraft, error) {
	outline, err := planTopicOutline(ctx, model, c, existing)
	if err != nil {
		return nil, err
	}
	// the outline may claim an existing page by title — honor it only when
	// the title really exists in this KB
	updateTitle := ""
	if t := strings.TrimSpace(outline.UpdateTitle); t != "" {
		for _, a := range existing {
			if a.ID != "" && a.PageType == core.WikiPageTypeTopic && fingerprint.Normalize(a.Title) == fingerprint.Normalize(t) {
				updateTitle = a.Title
				break
			}
		}
	}
	// an existing structured page anchors the shape: its headings lead the
	// FILL list verbatim (stable wording across re-compiles), and outline
	// headings that merely paraphrase them drop out — without this an update
	// appends renamed twins (8 sections becoming 16) instead of refreshing
	var anchorHeadings []string
	for _, a := range existing {
		if c.UpdateArticleID != "" && a.ID == c.UpdateArticleID && len(a.Outline) > 0 {
			for _, s := range a.Outline {
				anchorHeadings = append(anchorHeadings, s.Heading)
			}
		}
	}
	headings := outline.Sections
	if len(anchorHeadings) > 0 {
		fresh := make([]string, 0, len(outline.Sections))
		for _, h := range outline.Sections {
			dup := false
			for _, ah := range anchorHeadings {
				if headingSimilar(h, ah) {
					dup = true
					break
				}
			}
			if !dup {
				fresh = append(fresh, h)
			}
		}
		headings = append(append([]string{}, anchorHeadings...), fresh...)
	}
	fill, err := fillTopicSections(ctx, model, c, headings)
	if err != nil {
		return nil, err
	}

	byIndex := map[int]topicStatement{}
	for _, s := range c.Statements {
		byIndex[s.Index] = s
	}
	sections := make([]core.WikiSection, 0, len(fill.Sections))
	for _, fs := range fill.Sections {
		heading := strings.TrimSpace(fs.Heading)
		if heading == "" {
			continue
		}
		docs := map[string]bool{}
		for _, ref := range fs.Refs {
			if s, ok := byIndex[ref]; ok && s.DocID != "" {
				docs[s.DocID] = true
			}
		}
		docIDs := make([]string, 0, len(docs))
		for d := range docs {
			docIDs = append(docIDs, d)
		}
		sort.Strings(docIDs)
		sections = append(sections, core.WikiSection{
			ID:        sectionID(heading),
			Heading:   heading,
			Body:      strings.TrimSpace(fs.Body),
			DocIDs:    docIDs,
			UpdatedAt: time.Now().Unix(),
		})
	}
	if len(sections) == 0 {
		return nil, fmt.Errorf("topic [%s] produced no sections", c.Title)
	}

	claims := make([]compileStatement, 0, len(c.Statements))
	for _, s := range c.Statements {
		claims = append(claims, compileStatement{DocID: s.DocID, Excerpt: s.Excerpt, Claim: s.Text})
	}
	d := &topicDraft{
		Cluster: c, Title: c.Title, Scope: c.Scope, UpdateTitle: updateTitle,
		Sections: sections, Claims: claims, KeyEntities: keyEntityNames(c),
		Planned: len(sections),
	}
	// type the key entities against the KB vocabulary now (model is in
	// scope, the call is cheap) — the apply side just upserts with the
	// stored types instead of re-running an LLM at approval time
	d.KeyEntityTypes = classifyKeyEntities(ctx, model, kbID, d.Title, d.KeyEntities)
	return d, nil
}

// headingSimilar judges two section headings as the same section across
// compiles: exact after normalize, containment (a renamed prefix/suffix),
// or rune-bigram overlap — LLM re-runs paraphrase headings, and without a
// tolerant match an update appends a renamed twin instead of replacing
// (observed: 8 sections becoming 16, one Chinese and one English each).
func headingSimilar(a, b string) bool {
	na, nb := fingerprint.Normalize(a), fingerprint.Normalize(b)
	if na == nb {
		return true
	}
	if len(na) == 0 || len(nb) == 0 {
		return false
	}
	shorter := na
	if len(nb) < len(na) {
		shorter = nb
	}
	if len([]rune(shorter)) >= 4 && (strings.Contains(na, nb) || strings.Contains(nb, na)) {
		return true
	}
	ba, bb := headingBigrams(na), headingBigrams(nb)
	inter := 0
	for g := range ba {
		if bb[g] {
			inter++
		}
	}
	union := len(ba) + len(bb) - inter
	return union > 0 && float64(inter)/float64(union) >= 0.5
}

func headingBigrams(s string) map[string]bool {
	r := []rune(s)
	out := make(map[string]bool, len(r))
	for i := 0; i+1 < len(r); i++ {
		out[string(r[i:i+2])] = true
	}
	if len(out) == 0 && len(r) > 0 {
		out[string(r)] = true
	}
	return out
}

func sectionID(heading string) string {
	fp, _, ok := fingerprint.Compute("section:" + fingerprint.Normalize(heading))
	if !ok {
		return "sec-" + fingerprint.Normalize(heading)
	}
	return "sec-" + fp[:10]
}

/* ---------------- render + maturity ---------------- */

// RenderTopicContent renders the structured sections as the page markdown:
// cited prose only — the [n] markers carry provenance interactively (hover
// chip in the reader), so no per-section source lines waste layout; the
// full claim list trails as the verification floor.
func RenderTopicContent(sections []core.WikiSection, claims []compileStatement, docTitles map[string]string) string {
	var sb strings.Builder
	// no lead banner: the header's AI badge already marks the page as
	// compiled-and-awaiting-review, and a boilerplate quote above every
	// article was noise users deleted by hand
	for _, s := range sections {
		if strings.TrimSpace(s.Body) == "" {
			continue
		}
		fmt.Fprintf(&sb, "## %s\n\n%s\n\n", s.Heading, s.Body)
	}
	if len(claims) > 0 {
		sb.WriteString("## 来源主张\n\n")
		sb.WriteString(renderClaimList(claims))
	}
	return strings.TrimSpace(sb.String())
}

// collapseInline flattens a text blob onto a single line and strips double
// quotes: claim lines are single-line markdown, but raw excerpts (vision
// descriptions especially) carried real newlines that split the trailing
// [源](…) link mid-attribute — the frontend parser then lost the excerpt and
// leaked the excerpt body into the claim list as ghost numbered rows.
func collapseInline(s string) string {
	return strings.ReplaceAll(strings.Join(strings.Fields(s), " "), `"`, "'")
}

// shortPhrase picks a compact highlight query from an excerpt: the document
// preview matches it verbatim, so the full 120-rune excerpt (never an exact
// substring) was dead weight — and URL-encoding costs ~9 bytes per CJK char,
// which is what bloated compiled pages past 100KB.
func shortPhrase(s string) string {
	r := []rune(s)
	if len(r) > 24 {
		return string(r[:24])
	}
	return s
}

// renderClaimList is the numbered, source-linked claim list every compiled
// draft keeps — the verification floor under the prose.
func renderClaimList(statements []compileStatement) string {
	var sb strings.Builder
	for i, s := range statements {
		claim := collapseInline(strings.TrimSpace(s.Claim))
		if claim == "" {
			claim = collapseInline(strings.TrimSpace(s.Excerpt))
		}
		if claim == "" {
			continue
		}
		fmt.Fprintf(&sb, "%d. %s", i+1, claim)
		if s.DocID != "" {
			excerpt := collapseInline(strings.TrimSpace(s.Excerpt))
			if len([]rune(excerpt)) > 120 {
				excerpt = string([]rune(excerpt)[:120]) + "…"
			}
			fmt.Fprintf(&sb, " [源](#/preview/document/%s?q=%s \"%s\")", s.DocID, url.QueryEscape(shortPhrase(excerpt)), excerpt)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// ComputeTopicMaturity scores a topic page: distinct sources cited, filled
// vs planned sections, coverage 0-1. The TOC badge and the consolidate
// rule read it.
func ComputeTopicMaturity(sections []core.WikiSection, planned, sources int) *core.WikiPageMaturity {
	filled := 0
	for _, s := range sections {
		if strings.TrimSpace(s.Body) != "" {
			filled++
		}
	}
	if planned < len(sections) {
		planned = len(sections)
	}
	score := 0.0
	if planned > 0 {
		score = 0.6 * float64(filled) / float64(planned)
	}
	if sources > 0 {
		score += 0.4 * math.Min(1, float64(sources)/5)
	}
	return &core.WikiPageMaturity{
		Sources: sources, Sections: filled, Planned: planned,
		Score: math.Round(score*100) / 100,
	}
}

/* ---------------- proposal filing ---------------- */

// runTopicPipeline drives CLUSTER → NAME → match → compose per cluster and
// files one proposal per topic. Returns the count filed.
func runTopicPipeline(ctx context.Context, kb *core.WikiKnowledgeBase, model *core.ModelId,
	job *CompileJob, reduced []ReducedKey, existing []core.WikiArticle, docCount int, pctx *orm.Context) int {

	if job != nil {
		job.Stage = "cluster"
	}
	clusters := clusterTopics(reduced, docCount)
	if job != nil {
		job.Topics = len(clusters)
		job.log("clustered %d keys into %d topics", len(reduced), len(clusters))
	}
	if len(clusters) == 0 {
		return 0
	}

	titles := make([]string, 0, len(existing))
	for _, a := range existing {
		if a.PageType == core.WikiPageTypeTopic && a.Status != core.WikiArticleArchived {
			titles = append(titles, a.Title)
		}
	}
	nameTopics(ctx, model, clusters, titles)
	matchTopicTargets(clusters, existing)

	if job != nil {
		job.Stage = "compose"
	}
	filed := 0
	for _, c := range clusters {
		if job != nil && job.cancelled() {
			return filed
		}
		if job != nil {
			job.Current = c.Title
		}
		draft, err := composeTopicDraft(ctx, model, kb.ID, c, existing)
		if err != nil {
			if job != nil {
				job.log("compose failed [%s]: %v", c.Title, err)
			}
			log.Warnf("wiki topic compiler: compose [%s] failed: %v", c.Title, err)
			continue
		}
		if fileTopicProposal(pctx, kb, draft, model) {
			filed++
			if job != nil {
				job.Proposals = filed
			}
		}
		if job != nil {
			job.TopicsDone++
		}
	}
	fileLegacyCleanupProposal(pctx, kb, existing, clusters)
	return filed
}

// fileLegacyCleanupProposal sweeps the thin AI stub pages the per-entity
// compiler era left behind (page per key, a claim list and no body): the
// ones this run's topics did not already absorb get ONE cleanup proposal —
// applying archives them and drops them from the tree. Hand-written pages
// are never candidates.
func fileLegacyCleanupProposal(ctx *orm.Context, kb *core.WikiKnowledgeBase, existing []core.WikiArticle, clusters []*topicCluster) {
	var stale []string
	entityish := 0
	for _, a := range existing {
		if a.AIGenerated && (a.PageType == core.WikiPageTypeEntity || a.PageType == core.WikiPageTypeConcept) && a.Status != core.WikiArticleArchived {
			entityish++
		}
	}
	log.Infof("wiki topic compiler: cleanup scan — existing=%d live-entity/concept=%d", len(existing), entityish)
	for _, a := range existing {
		// pages a cluster promised to absorb are NOT skipped here: the
		// absorb only fires when its proposal files and applies, and a
		// dropped proposal would strand the page forever — re-archiving
		// an already-absorbed page is a no-op
		if a.Status == core.WikiArticleArchived {
			continue
		}
		if !a.AIGenerated || (a.PageType != core.WikiPageTypeEntity && a.PageType != core.WikiPageTypeConcept) {
			continue
		}
		if len(a.Outline) > 0 { // structured page, not a legacy stub
			continue
		}
		// full sweep: every AI entity/concept page is pre-topic-era
		// residue — content lives in the topic pages now and the entities
		// stay in the ontology pool; history is cleaned wholesale, not
		// graded for substantiality
		stale = append(stale, a.ID)
	}
	if len(stale) == 0 {
		return
	}

	proposal := &core.WikiGovernanceProposal{
		KbID:         kb.ID,
		Type:         compileProposalType,
		Status:       core.WikiGovernanceOpen,
		ArticleTitle: "编译残页清理",
		Reason:       fmt.Sprintf("归档 %d 个旧编译产生的实体/概念页(全部历史存档,内容已由主题页重新组织),清出左侧目录", len(stale)),
	}
	// day-anchored: one living cleanup proposal per KB, re-runs refresh it
	proposal.ID = core.AnchoredProposalID("legacy-cleanup:"+kb.ID, compileProposalType, int(time.Now().Unix()/86400))
	proposal.Evidence = util.MapStr{
		"cascade":        "topic-compile",
		"legacy_cleanup": true,
		"page_ids":       stale,
	}
	if err := orm.Create(ctx, proposal); err != nil {
		// same-day twin occupies the anchor: an open one is refreshed in
		// place; a resolved one already archived its list — this run files
		// a fresh proposal so a widened sweep still lands today
		twin := core.WikiGovernanceProposal{}
		twin.SetID(proposal.ID)
		if found, gerr := orm.GetV2(ctx, &twin); gerr == nil && found && twin.Status == core.WikiGovernanceOpen {
			twin.Reason = proposal.Reason
			twin.Evidence = proposal.Evidence
			if uerr := orm.Update(ctx, &twin); uerr != nil {
				log.Debugf("wiki topic compiler: cleanup twin refresh failed: %v", uerr)
			}
			return
		}
		proposal.SetID(util.GetUUID())
		if err := orm.Create(ctx, proposal); err != nil {
			log.Warnf("wiki topic compiler: cleanup proposal refile failed: %v", err)
			return
		}
	}
	if kb.GetOwnerID() != "" {
		notifyOwner(kb.GetOwnerID(), "kb", kb.ID, "governance",
			fmt.Sprintf("知识编译:%d 个旧实体/概念页待清理(内容已由主题页重新组织),在治理队列确认后归档", len(stale)))
	}
}

// fileTopicProposal files one topic proposal carrying the structured page:
// sections, claims, key entities, the update target and the thin pages it
// absorbs. Anchored per topic key + generation like the legacy compiler so
// re-runs overwrite instead of stacking twins.
func fileTopicProposal(ctx *orm.Context, kb *core.WikiKnowledgeBase, d *topicDraft, model *core.ModelId) bool {
	action := "create"
	reason := fmt.Sprintf("create topic page from %d docs / %d claims (%d sections)", len(d.Cluster.Docs), len(d.Claims), len(d.Sections))
	if d.Cluster.UpdateArticleID != "" || d.UpdateTitle != "" {
		action = "update"
		reason = fmt.Sprintf("enrich topic page from %d claims (%d sections)", len(d.Claims), len(d.Sections))
	}
	if len(d.Cluster.AbsorbIDs) > 0 {
		reason += fmt.Sprintf(", absorb %d thin page(s)", len(d.Cluster.AbsorbIDs))
	}

	claims := make([]interface{}, 0, len(d.Claims))
	for _, s := range d.Claims {
		claims = append(claims, util.MapStr{"doc_id": s.DocID, "excerpt": s.Excerpt, "claim": s.Claim})
	}
	sectionsRaw := json.RawMessage(util.MustToJSONBytes(d.Sections))

	updateID := d.Cluster.UpdateArticleID
	normKey := "topic|" + fingerprint.Normalize(d.Title)
	proposal := &core.WikiGovernanceProposal{
		KbID:         kb.ID,
		Type:         compileProposalType,
		Status:       core.WikiGovernanceOpen,
		ArticleTitle: d.Title,
		Reason:       reason,
	}
	proposal.ID = core.AnchoredProposalID("compile:"+kb.ID+":"+normKey, compileProposalType,
		compileGeneration(ctx, kb.ID, normKey))
	proposal.Evidence = util.MapStr{
		"cascade":           "topic-compile",
		"topic":             true,
		"norm_key":          normKey,
		"scope":             d.Scope,
		"sections":          sectionsRaw,
		"statements":        claims,
		"key_entities":      d.KeyEntities,
		"aliases":           d.Cluster.Aliases,
		"update_article_id": updateID,
		"absorb_ids":        d.Cluster.AbsorbIDs,
		"planned":           d.Planned,
		"model":             model.ProviderID + "/" + model.ID,
	}
	if len(d.KeyEntityTypes) > 0 {
		proposal.Evidence["key_entity_types"] = d.KeyEntityTypes
	}
	if d.UpdateTitle != "" && updateID == "" {
		proposal.Evidence["update_title"] = d.UpdateTitle
	}
	if action == "create" {
		docIDs := make([]string, 0, len(d.Cluster.Docs))
		for id := range d.Cluster.Docs {
			docIDs = append(docIDs, id)
		}
		if target := topicTargetFolder(ctx, kb, docIDs); target != "" {
			proposal.Evidence["target_folder"] = target
			proposal.Reason += " → " + target
		}
	}
	// an open twin under any anchor (this run or a generation back) is the
	// living proposal — refresh it in place instead of stacking a second
	// open proposal for the same subject
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})
	if res, serr := orm.SearchV2(ctx, orm.NewQuery().Size(20).
		Filter(orm.TermQuery("kb_id", kb.ID),
			orm.TermQuery("type", compileProposalType),
			orm.TermQuery("status", core.WikiGovernanceOpen))); serr == nil && res != nil {
		if open, _, derr := elastic.DecodeHits[core.WikiGovernanceProposal](res); derr == nil {
			for i := range open {
				if open[i].Evidence["norm_key"] == normKey {
					twin := open[i]
					twin.Reason = proposal.Reason
					twin.Evidence = proposal.Evidence
					if uerr := orm.Update(ctx, &twin); uerr != nil {
						log.Debugf("wiki topic compiler: twin refresh failed for [%s]: %v", d.Title, uerr)
						return false
					}
					return true
				}
			}
		}
	}
	if err := orm.Create(ctx, proposal); err != nil {
		// anchor collision with a resolved twin (a miscounted generation,
		// a same-name cluster across runs) must not drop the draft — file
		// under a fresh id instead
		log.Warnf("wiki topic compiler: proposal create failed for [%s]: %v (refiling)", d.Title, err)
		proposal.SetID(util.GetUUID())
		if err := orm.Create(ctx, proposal); err != nil {
			log.Warnf("wiki topic compiler: proposal refile failed for [%s]: %v", d.Title, err)
			return false
		}
	}
	if kb.GetOwnerID() != "" {
		notifyOwner(kb.GetOwnerID(), "kb", kb.ID, "governance",
			fmt.Sprintf("知识编译:主题页「%s」提议(%s,%d 节,%d 条主张)", d.Title, action, len(d.Sections), len(d.Claims)))
	}
	return true
}

// topicTargetFolder resolves a topic page's TOC folder: department from the
// cited documents' folder path, dimension slot 2 (专题与概念) with the
// department's own rename honored.
func topicTargetFolder(ctx *orm.Context, kb *core.WikiKnowledgeBase, docIDs []string) string {
	if len(kb.TocLayout) == 0 {
		return ""
	}
	dept, idx := matchDeptByDocIDs(ctx, kb, docIDs)
	var deptPtr *core.WikiTocLayoutFolder
	if idx >= 0 {
		deptPtr = &kb.TocLayout[idx]
	}
	dim := dimensionDisplayName(deptPtr, 2)
	if dept == "" {
		return dim
	}
	return dept + "/" + dim
}
