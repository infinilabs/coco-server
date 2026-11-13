/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
	langchain "infini.sh/coco/modules/assistant/langchain"
	llmmodule "infini.sh/coco/modules/llm"
)

// The maintainer (LLM-wiki P1, the WikiSkill WikiMaintainer role): mine
// the raw experience layer for failure patterns and file pattern proposals.
// The loop is deliberately conservative — negative traces first, existing
// patterns shown for dedupe, empty output is a first-class result, and
// every output goes through the governance gate (pattern_create /
// pattern_update proposals). The wiki itself is never written directly,
// except the generated index page (machine-owned, like the company map).
//
// The proposer/skill half of the loop is P2.

const (
	distillTaskID           = "wiki-experience-distill"
	distillKVBucket         = "wiki-distill"
	distillDefaultInterval  = "6h"
	distillLookbackFallback = 24 * time.Hour
	distillMaxExperiences   = 12
	distillMaxPatterns      = 5
	distillMaxExisting      = 100
	// patternIndexTitle is the canonical title of the generated pattern
	// index page — one per KB, program-regenerated on every run; hand
	// edits are not the contract (same ritual as the company map, D6)
	patternIndexTitle = "Pattern Index"
)

func init() {
	switch strings.ToLower(distillScheduleInterval()) {
	case "off", "disabled", "0":
		return
	}
	task.RegisterScheduleTask(task.ScheduleTask{
		ID:          distillTaskID,
		Group:       "wiki",
		Description: "Wiki 经验蒸馏:维护者从经验记录里挖掘失败模式,产出 pattern 提案(LLM-wiki P1)",
		Interval:    distillScheduleInterval(),
		Singleton:   true,
		Task:        distillSweep,
	})
}

// distillScheduleInterval is the effective WIKI_DISTILL_INTERVAL (default 6h,
// never empty — "off"/"disabled"/"0" disable the schedule). Surfaced on the
// experience endpoint so the UI can say when the next automatic run is.
func distillScheduleInterval() string {
	interval := strings.TrimSpace(os.Getenv("WIKI_DISTILL_INTERVAL"))
	if interval == "" {
		return distillDefaultInterval
	}
	return interval
}

// distillLastRun reports the watermark of this KB's last distill pass; nil
// when it never ran (a fresh KB falls back to a lookback window internally,
// but "never ran" is the fact worth showing).
func distillLastRun(kbID string) *time.Time {
	value, err := kv.GetValue(distillKVBucket, []byte(kbID))
	if err != nil || len(value) == 0 {
		return nil
	}
	var watermark time.Time
	if err := util.FromJSONBytes(value, &watermark); err != nil {
		return nil
	}
	return &watermark
}

func distillSweep(ctx context.Context) {
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiKnowledgeBase{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(200).
		Filter(orm.TermQuery("experience_capture", true)))
	if err != nil || res == nil {
		return
	}
	kbs, _, derr := elastic.DecodeHits[core.WikiKnowledgeBase](res)
	if derr != nil {
		return
	}
	for i := range kbs {
		start := time.Now()
		run, err := distillKB(octx, kbs[i].ID)
		if err != nil {
			log.Warnf("wiki: distill failed for kb [%s]: %v", kbs[i].ID, err)
		} else if run != nil && run.Sampled > 0 {
			log.Infof("wiki: distilled kb [%s]: %d experiences sampled, %d proposals in %s",
				kbs[i].ID, run.Sampled, run.Proposals, time.Since(start))
		}
	}
}

// distillRun is one run's stats — also the manual endpoint's response.
type distillRun struct {
	Sampled      int      `json:"sampled"`
	Proposals    int      `json:"proposals"`
	Created      int      `json:"created"`
	Updated      int      `json:"updated"`
	Expired      int      `json:"expired"`
	NoAction     bool     `json:"no_action"`
	SkippedNotes []string `json:"skipped_notes,omitempty"`
}

// readDistillWatermark: experiences newer than this get sampled. A fresh
// KB looks back one window so enabling capture doesn't boil the ocean.
func readDistillWatermark(kbID string) time.Time {
	value, err := kv.GetValue(distillKVBucket, []byte(kbID))
	if err != nil || len(value) == 0 {
		return time.Now().Add(-distillLookbackFallback)
	}
	var watermark time.Time
	if err := util.FromJSONBytes(value, &watermark); err != nil {
		return time.Now().Add(-distillLookbackFallback)
	}
	return watermark
}

func writeDistillWatermark(kbID string, t time.Time) {
	value, err := util.ToJSONBytes(t)
	if err == nil {
		_ = kv.AddValue(distillKVBucket, []byte(kbID), value)
	}
}

// distillKB runs the maintainer once for one KB: sample → root-cause via
// LLM → file pattern proposals → regenerate the index → advance the
// watermark. Proposals only — article content is written by the apply
// path, behind the same gate as every other proposal.
func distillKB(parent *orm.Context, kbID string) (*distillRun, error) {
	run := &distillRun{}
	watermark := readDistillWatermark(kbID)

	sampleCtx := orm.NewContextWithParent(parent)
	sampleCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(sampleCtx, &core.WikiExperience{})
	res, err := orm.SearchV2(sampleCtx, orm.NewQuery().Size(distillMaxExperiences*4).
		Filter(orm.TermQuery("kb_id", kbID)).
		SortBy(orm.Sort{Field: "created", SortType: "desc"}))
	if err != nil {
		return run, err
	}
	experiences, _, derr := elastic.DecodeHits[core.WikiExperience](res)
	if derr != nil {
		return run, derr
	}
	// window + stratification: samples since the watermark, negative
	// feedback first (failure is the signal), then newest
	sample := make([]core.WikiExperience, 0, distillMaxExperiences)
	rest := make([]core.WikiExperience, 0, len(experiences))
	for i := range experiences {
		if experiences[i].Created != nil && experiences[i].Created.Before(watermark) {
			continue
		}
		if experiences[i].Feedback == core.ExperienceFeedbackNegative {
			if len(sample) < distillMaxExperiences {
				sample = append(sample, experiences[i])
			}
		} else {
			rest = append(rest, experiences[i])
		}
	}
	for i := range rest {
		if len(sample) >= distillMaxExperiences {
			break
		}
		sample = append(sample, rest[i])
	}
	run.Sampled = len(sample)

	// housekeeping runs regardless of whether the LLM gets involved
	run.Expired = expireExperiences(sampleCtx, kbID)

	if len(sample) == 0 {
		writeDistillWatermark(kbID, time.Now())
		return run, nil
	}

	// existing patterns are the dedupe surface — the maintainer must see
	// the wiki (the paper's +15pts for wiki access)
	existing := findPatternPages(sampleCtx, kbID, distillMaxExisting)

	proposals, perr := proposePatterns(context.Background(), sample, existing)
	if perr != nil {
		return run, perr
	}
	for i := range proposals {
		switch proposals[i].action {
		case "update":
			if filePatternUpdateProposal(parent, kbID, sample, &proposals[i]) {
				run.Updated++
			} else {
				run.SkippedNotes = append(run.SkippedNotes, fmt.Sprintf("update target not found or already queued: %s", proposals[i].title))
			}
		default:
			if filePatternCreateProposal(parent, kbID, sample, &proposals[i]) {
				run.Created++
			} else {
				run.SkippedNotes = append(run.SkippedNotes, fmt.Sprintf("open twin already queued: %s", proposals[i].title))
			}
		}
	}
	run.Proposals = run.Created + run.Updated
	if run.Proposals == 0 {
		run.NoAction = true
	}

	if rerr := regeneratePatternPages(parent, kbID); rerr != nil {
		log.Warnf("wiki: pattern index regeneration failed for kb [%s]: %v", kbID, rerr)
	}

	writeDistillWatermark(kbID, time.Now())
	AppendEvolutionEntry(parent, kbID, core.WikiEvolutionPatternDistilled, "distiller",
		"kb", kbID,
		fmt.Sprintf("经验蒸馏：%d 条经验采样，%d 个模式提案", run.Sampled, run.Proposals),
		util.MapStr{
			"sampled":   run.Sampled,
			"proposals": run.Proposals,
			"created":   run.Created,
			"updated":   run.Updated,
			"no_action": run.NoAction,
		})
	return run, nil
}

// findPatternPages lists a KB's pattern pages — the dedupe/update surface
// handed to the maintainer LLM.
func findPatternPages(ctx *orm.Context, kbID string, size int) []core.WikiArticle {
	orm.WithModel(ctx, &core.WikiArticle{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(size).
		Filter(orm.TermQuery("kb_id", kbID), orm.TermQuery("page_type", core.WikiPageTypePattern)))
	if err != nil || res == nil {
		return nil
	}
	pages, _, derr := elastic.DecodeHits[core.WikiArticle](res)
	if derr != nil {
		return nil
	}
	return pages
}

// distilledPattern is one maintainer output: a pattern to create or a
// patch for an existing one, with pointers back at the supporting traces.
type distilledPattern struct {
	action    string
	title     string
	symptom   string
	rootCause string
	fix       string
	evidence  []int // 1-based indexes into the sampled experiences
}

type llmPatternOut struct {
	Action    string `json:"action"`
	Target    string `json:"target,omitempty"`
	Title     string `json:"title"`
	Symptom   string `json:"symptom"`
	RootCause string `json:"root_cause"`
	Fix       string `json:"fix"`
	Evidence  []int  `json:"evidence"`
}

// proposePatterns runs the maintainer LLM. Returns only well-formed
// proposals; zero proposals is a valid, expected outcome (no_action).
func proposePatterns(ctx context.Context, sample []core.WikiExperience, existing []core.WikiArticle) ([]distilledPattern, error) {
	model := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if model == nil {
		return nil, fmt.Errorf("no language model configured")
	}
	llm, err := langchain.SimplyGetLLM(model.ProviderID, model.ID, "")
	if err != nil {
		return nil, err
	}

	var traces strings.Builder
	for i := range sample {
		fmt.Fprintf(&traces, "### 经验 #%d（反馈：%s）\n", i+1, sample[i].Feedback)
		fmt.Fprintf(&traces, "问题：%s\n", truncateRunes(sample[i].Question, 300))
		if note := strings.TrimSpace(sample[i].Note); note != "" {
			fmt.Fprintf(&traces, "用户反馈：%s\n", truncateRunes(note, 300))
		}
		if sample[i].RouteHint != "" {
			fmt.Fprintf(&traces, "归因：%s\n", sample[i].RouteHint)
		}
		if answer := strings.TrimSpace(sample[i].Answer); answer != "" {
			fmt.Fprintf(&traces, "回答摘录：%s\n", truncateRunes(answer, 400))
		}
	}

	var known strings.Builder
	if len(existing) == 0 {
		known.WriteString("（暂无已有模式页）\n")
	}
	for i := range existing {
		if existing[i].Title == patternIndexTitle {
			continue
		}
		oneLiner := existing[i].Title
		if existing[i].Pattern != nil && strings.TrimSpace(existing[i].Pattern.Symptom) != "" {
			oneLiner = fmt.Sprintf("%s — %s", existing[i].Title, truncateRunes(existing[i].Pattern.Symptom, 80))
		}
		fmt.Fprintf(&known, "%d. %s\n", i+1, oneLiner)
	}

	prompt := fmt.Sprintf(`你是知识库维护者。下面是用户与 AI 问答的真实经验记录（多为负面反馈），以及本知识库已有的模式页清单。

规则：
- 只输出 JSON，不要任何解释或 markdown 代码块
- 从失败经验中提炼可复用的"模式"：反复出现的问题形态（症状）+ 根因 + 下次遇到该怎么办（解法）
- 与已有模式重复的不要新建；若已有模式需要补充/修正，用 update 并给出 target（已有页标题）
- 证据薄弱（仅一条弱信号）的模式宁可不提；输出空数组也是正确答案
- 最多 %d 个模式；每个字段 1-3 句话，克制、具体、可执行
- evidence 填支持该模式的经验编号（1 起始）

已有模式页：
%s

经验记录：
%s

输出格式：
{"patterns": [{"action": "create"或"update", "target": "仅 update 需要", "title": "模式名", "symptom": "症状", "root_cause": "根因", "fix": "解法", "evidence": [1,2]}]}`,
		distillMaxPatterns, known.String(), traces.String())

	messages := []llms.MessageContent{
		langchain.SystemTextParts("你是严谨的知识库维护者：只提炼证据支撑的模式，宁缺毋滥。输出必须是纯 JSON。"),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}
	cctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	resp, err := llm.GenerateContent(cctx, messages,
		llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error { return nil }))
	if err != nil {
		return nil, err
	}
	prose := ""
	if resp != nil && len(resp.Choices) > 0 {
		prose = resp.Choices[0].Content
	}
	if i := strings.Index(prose, "</think>"); i >= 0 {
		prose = prose[i+len("</think>"):]
	}
	prose = strings.TrimSpace(prose)
	prose = strings.TrimPrefix(prose, "```json")
	prose = strings.TrimPrefix(prose, "```")
	prose = strings.TrimSpace(prose)
	if i := strings.LastIndex(prose, "```"); i >= 0 {
		prose = prose[:i]
		prose = strings.TrimSpace(prose)
	}

	var out struct {
		Patterns []llmPatternOut `json:"patterns"`
	}
	if err := json.Unmarshal([]byte(prose), &out); err != nil {
		return nil, fmt.Errorf("distiller output not parseable: %w", err)
	}
	proposals := make([]distilledPattern, 0, len(out.Patterns))
	for i := range out.Patterns {
		p := &out.Patterns[i]
		title := strings.TrimSpace(p.Title)
		if title == "" || strings.TrimSpace(p.Symptom) == "" {
			continue
		}
		action := "create"
		if strings.EqualFold(strings.TrimSpace(p.Action), "update") {
			action = "update"
		}
		ev := make([]int, 0, len(p.Evidence))
		for _, n := range p.Evidence {
			if n >= 1 && n <= len(sample) {
				ev = append(ev, n)
			}
		}
		proposals = append(proposals, distilledPattern{
			action:    action,
			title:     truncateRunes(title, 80),
			symptom:   truncateRunes(p.Symptom, 600),
			rootCause: truncateRunes(p.RootCause, 600),
			fix:       truncateRunes(p.Fix, 600),
			evidence:  ev,
		})
	}
	return proposals, nil
}

func truncateRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "…"
}

// experienceEvidence converts the LLM's 1-based indexes back into source
// references so the proposal carries its own provenance.
func experienceEvidence(sample []core.WikiExperience, indexes []int) []core.WikiSourceReference {
	refs := make([]core.WikiSourceReference, 0, len(indexes))
	for _, n := range indexes {
		if n < 1 || n > len(sample) {
			continue
		}
		exp := &sample[n-1]
		refs = append(refs, core.WikiSourceReference{
			DocID:      exp.ID,
			SourceType: "experience",
			Title:      truncateRunes(exp.Question, 80),
			Excerpt:    truncateRunes(exp.Note, 200),
		})
	}
	return refs
}

// openPatternTwin short-circuits duplicates: one open proposal per pattern
// title per KB.
func openPatternTwin(ctx *orm.Context, kbID, pType, title string) bool {
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("kb_id", kbID),
			orm.TermQuery("type", pType),
			orm.TermQuery("article_title", title),
			orm.TermQuery("status", core.WikiGovernanceOpen)))
	if err != nil || res == nil {
		return false
	}
	hits, _, err2 := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	if err2 != nil {
		return false
	}
	return len(hits) > 0
}

func filePatternCreateProposal(parent *orm.Context, kbID string, sample []core.WikiExperience, p *distilledPattern) bool {
	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	if openPatternTwin(ctx, kbID, core.WikiGovernancePatternCreate, p.title) {
		return false
	}
	proposal := &core.WikiGovernanceProposal{
		KbID:         kbID,
		ArticleTitle: p.title,
		Type:         core.WikiGovernancePatternCreate,
		Status:       core.WikiGovernanceOpen,
		Reason:       truncateRunes(p.symptom, 200),
		Evidence: util.MapStr{
			"symptom":    p.symptom,
			"root_cause": p.rootCause,
			"fix":        p.fix,
		},
	}
	if refs := experienceEvidence(sample, p.evidence); len(refs) > 0 {
		proposal.Evidence["sources"] = refs
	}
	if err := orm.Create(ctx, proposal); err != nil {
		log.Warnf("wiki: pattern_create proposal failed: %v", err)
		return false
	}
	return true
}

func filePatternUpdateProposal(parent *orm.Context, kbID string, sample []core.WikiExperience, p *distilledPattern) bool {
	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	// resolve the target page first — no page, no update proposal
	var target *core.WikiArticle
	orm.WithModel(ctx, &core.WikiArticle{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("kb_id", kbID),
			orm.TermQuery("page_type", core.WikiPageTypePattern),
			orm.TermQuery("title", p.title)))
	if err != nil || res == nil {
		return false
	}
	hits, _, derr := elastic.DecodeHits[core.WikiArticle](res)
	if derr != nil || len(hits) == 0 {
		return false
	}
	target = &hits[0]
	if openPatternTwin(ctx, kbID, core.WikiGovernancePatternUpdate, p.title) {
		return false
	}
	proposal := &core.WikiGovernanceProposal{
		KbID:         kbID,
		ArticleID:    target.ID,
		ArticleTitle: target.Title,
		Type:         core.WikiGovernancePatternUpdate,
		Status:       core.WikiGovernanceOpen,
		Reason:       truncateRunes(p.symptom, 200),
		Evidence: util.MapStr{
			"symptom":    p.symptom,
			"root_cause": p.rootCause,
			"fix":        p.fix,
		},
	}
	if refs := experienceEvidence(sample, p.evidence); len(refs) > 0 {
		proposal.Evidence["sources"] = refs
	}
	if err := orm.Create(ctx, proposal); err != nil {
		log.Warnf("wiki: pattern_update proposal failed: %v", err)
		return false
	}
	return true
}

// regeneratePatternPages rewrites the KB's generated pattern index page —
// every line one pattern: [[pattern:Title]] — symptom one-liner.
func regeneratePatternPages(parent *orm.Context, kbID string) error {
	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	pages := findPatternPages(ctx, kbID, distillMaxExisting)
	var sb strings.Builder
	sb.WriteString("> Generated by the experience distiller — read this first, then follow the pattern links. Do not hand-edit; it is regenerated on every distill run.\n\n")
	count := 0
	for i := range pages {
		if pages[i].Title == patternIndexTitle {
			continue
		}
		oneLiner := ""
		if pages[i].Pattern != nil {
			oneLiner = truncateRunes(pages[i].Pattern.Symptom, 100)
		}
		fmt.Fprintf(&sb, "- [[pattern:%s]] — %s\n", pages[i].Title, oneLiner)
		count++
	}
	if count == 0 {
		sb.WriteString("No patterns distilled yet.\n")
	}
	return upsertPatternIndexPage(ctx, kbID, sb.String())
}

func upsertPatternIndexPage(ctx *orm.Context, kbID, content string) error {
	orm.WithModel(ctx, &core.WikiArticle{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("kb_id", kbID),
			orm.TermQuery("title", patternIndexTitle)))
	if err != nil {
		return err
	}
	hits, _, derr := elastic.DecodeHits[core.WikiArticle](res)
	if derr != nil {
		return derr
	}
	if len(hits) > 0 {
		page := hits[0]
		if page.Content == content {
			return nil
		}
		page.Content = content
		page.AIGenerated = true
		orm.WithModel(ctx, &core.WikiArticle{})
		if err := orm.Update(ctx, &page); err != nil {
			return err
		}
		persistLinkedPages(&page)
		SyncArticleProjection(&page)
		return nil
	}
	page := &core.WikiArticle{
		KbID:     kbID,
		Title:    patternIndexTitle,
		Summary:  "Generated pattern index for this knowledge base",
		Content:  content,
		PageType: core.WikiPageTypePattern,
		Status:   core.WikiArticlePublished,
	}
	page.AIGenerated = true
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	if err := orm.Create(ctx, page); err != nil {
		return err
	}
	persistLinkedPages(page)
	SyncArticleProjection(page)
	if err := addArticleToToc(kbID, page.ID, page.Title); err != nil {
		return err
	}
	return bumpKbArticleCount(kbID, 1)
}

/* ---------------- manual trigger ---------------- */

func (h *APIHandler) distillKBHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	run, err := distillKB(ctx, kbID)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, run)
}
