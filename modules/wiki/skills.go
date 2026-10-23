/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/attachment"
	langchain "infini.sh/coco/modules/assistant/langchain"
	llmmodule "infini.sh/coco/modules/llm"
)

// The skills layer (LLM-wiki P2, the WikiSkill skills/ analog): the
// proposer turns distilled patterns into ONE atomic skill proposal per
// run, the reviewer LLM gates every apply, and skill pages export to the
// Agent Skills SKILL.md format. The wiki accumulates and never rolls back;
// only skills are gated.
//
// Where the paper's proposer is a ReAct loop over traces, P2 hands the
// proposer the full bounded context in one shot: pattern pages (it MUST
// read the wiki — the +15pts lesson), fresh experiences and the evolution
// log including rejected proposals (so the same idea is not re-proposed).

const (
	skillGateTimeout = 60 * time.Second
	skillProposeTimeout = 120 * time.Second
	evolutionLogReadSize = 30
)

/* ---------------- content rendering ---------------- */

func renderSkillContent(skill *core.WikiSkill, provenance []core.WikiSourceReference) string {
	var sb strings.Builder
	sb.WriteString("> Distilled procedural skill — how this knowledge base says to do this thing. Gated via the governance queue; provenance points at the failure patterns that motivated it.\n\n")
	if strings.TrimSpace(skill.Applicability) != "" {
		sb.WriteString("## When to use\n\n" + strings.TrimSpace(skill.Applicability) + "\n\n")
	}
	if len(skill.Steps) > 0 {
		sb.WriteString("## Steps\n\n")
		for i, step := range skill.Steps {
			fmt.Fprintf(&sb, "%d. %s\n", i+1, strings.TrimSpace(step))
		}
		sb.WriteString("\n")
	}
	if len(skill.AntiPatterns) > 0 {
		sb.WriteString("## Anti-patterns\n\n")
		for _, ap := range skill.AntiPatterns {
			fmt.Fprintf(&sb, "- %s\n", strings.TrimSpace(ap))
		}
		sb.WriteString("\n")
	}
	if len(skill.WhenNotToApply) > 0 {
		sb.WriteString("## When NOT to apply\n\n")
		for _, w := range skill.WhenNotToApply {
			fmt.Fprintf(&sb, "- %s\n", strings.TrimSpace(w))
		}
		sb.WriteString("\n")
	}
	if len(provenance) > 0 {
		sb.WriteString("## Provenance\n\n")
		for _, ref := range provenance {
			title := ref.Title
			if title == "" {
				title = ref.DocID
			}
			if ref.SourceType == "pattern" || ref.SourceType == "" {
				fmt.Fprintf(&sb, "- [[pattern:%s]]\n", title)
			} else {
				fmt.Fprintf(&sb, "- %s\n", title)
			}
		}
		sb.WriteString("\n")
	}
	if skill.Rationale != "" {
		sb.WriteString(fmt.Sprintf("> Proposer rationale: %s\n", strings.TrimSpace(skill.Rationale)))
	}
	return sb.String()
}

func skillFromEvidence(p *core.WikiGovernanceProposal) (*core.WikiSkill, []core.WikiSourceReference) {
	if p.Evidence == nil {
		return nil, nil
	}
	skill := &core.WikiSkill{}
	skill.Applicability, _ = p.Evidence["applicability"].(string)
	skill.Rationale, _ = p.Evidence["rationale"].(string)
	if raw, ok := p.Evidence["steps"]; ok {
		_ = util.FromJSONBytes(util.MustToJSONBytes(raw), &skill.Steps)
	}
	if raw, ok := p.Evidence["anti_patterns"]; ok {
		_ = util.FromJSONBytes(util.MustToJSONBytes(raw), &skill.AntiPatterns)
	}
	if raw, ok := p.Evidence["when_not_to_apply"]; ok {
		_ = util.FromJSONBytes(util.MustToJSONBytes(raw), &skill.WhenNotToApply)
	}
	if raw, ok := p.Evidence["reach_for"]; ok {
		reach := &core.SkillReachFor{}
		if err := util.FromJSONBytes(util.MustToJSONBytes(raw), reach); err == nil {
			skill.ReachFor = reach
		}
	}
	var provenance []core.WikiSourceReference
	if raw, ok := p.Evidence["sources"]; ok {
		_ = util.FromJSONBytes(util.MustToJSONBytes(raw), &provenance)
	}
	return skill, provenance
}

/* ---------------- apply ---------------- */

func applySkillCreateProposal(ctx *orm.Context, p *core.WikiGovernanceProposal) (string, string, error) {
	title := strings.TrimSpace(p.ArticleTitle)
	if title == "" || p.Evidence == nil {
		return "", "", nil
	}
	skill, provenance := skillFromEvidence(p)
	if skill == nil || len(skill.Steps) == 0 {
		return "", "", nil
	}
	// idempotency: page already exists → no-op
	orm.WithModel(ctx, &core.WikiArticle{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("kb_id", p.KbID), orm.TermQuery("title", title)))
	if err != nil {
		return "", "", err
	}
	if hits, _, derr := elastic.DecodeHits[core.WikiArticle](res); derr == nil && len(hits) > 0 {
		return "技能页已存在", hits[0].ID, nil
	}
	article := &core.WikiArticle{
		KbID:        p.KbID,
		Title:       title,
		Summary:     truncateRunes(skill.Applicability, 120),
		Content:     renderSkillContent(skill, provenance),
		PageType:    core.WikiPageTypeSkill,
		Status:      core.WikiArticlePublished,
		AIGenerated: true,
		Skill:       skill,
		Sources:     provenance,
	}
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	if err := orm.Create(ctx, article); err != nil {
		return "", "", err
	}
	persistLinkedPages(article)
	snapshotPipelineWrite(article, "skill proposed and gated")
	SyncArticleProjection(article)
	if err := addArticleToToc(p.KbID, article.ID, article.Title); err != nil {
		return "", "", err
	}
	if err := bumpKbArticleCount(p.KbID, 1); err != nil {
		return "", "", err
	}
	return "技能页已创建", article.ID, nil
}

func applySkillPatchProposal(ctx *orm.Context, p *core.WikiGovernanceProposal) (string, error) {
	if p.ArticleID == "" || p.Evidence == nil {
		return "", nil
	}
	article := core.WikiArticle{}
	article.ID = p.ArticleID
	ok, err := orm.GetV2(ctx, &article)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("article %s not found", p.ArticleID)
	}
	skill, provenance := skillFromEvidence(p)
	if skill == nil || len(skill.Steps) == 0 {
		return "", nil
	}
	article.Skill = skill
	article.Content = renderSkillContent(skill, provenance)
	if len(provenance) > 0 {
		article.Sources = provenance
	}
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	if err := orm.Update(ctx, &article); err != nil {
		return "", err
	}
	persistLinkedPages(&article)
	snapshotPipelineWrite(&article, "skill patched via gated proposal")
	SyncArticleProjection(&article)
	return "技能页已更新", nil
}

/* ---------------- reviewer gate (G4) ---------------- */

// errGateRejected marks a reviewer-gate rejection — the caller keeps the
// proposal open (the human queue is the next gate).
var errGateRejected = fmt.Errorf("rejected by reviewer gate")

// reviewSkillProposal is the second-LLM gate run before any skill apply:
// duplicate? evidence-backed? does it answer a logged failure? The verdict
// and rationale land in the proposal evidence and the evolution log either
// way. A missing reviewer model fails OPEN for manual applies (the human
// IS the gate) — the auto-approve sweep treats a missing model as a skip.
func reviewSkillProposal(ctx context.Context, kbID string, p *core.WikiGovernanceProposal) (allowed bool, rationale string) {
	model := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if model == nil {
		return true, "no reviewer model configured — human gate applies"
	}
	llm, err := langchain.SimplyGetLLM(model.ProviderID, model.ID, "")
	if err != nil {
		return true, "reviewer model unavailable — human gate applies"
	}

	// must-read context: existing skills + the KB's evolution log (what
	// was already tried, accepted AND rejected)
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	existing := findSkillPages(octx, kbID, distillMaxExisting)
	entries := findEvolutionEntries(octx, kbID, evolutionLogReadSize)

	skill, provenance := skillFromEvidence(p)
	var skillDesc strings.Builder
	skillDesc.WriteString(fmt.Sprintf("提案：%s（%s）\n", p.ArticleTitle, p.Type))
	if skill != nil {
		fmt.Fprintf(&skillDesc, "适用：%s\n", truncateRunes(skill.Applicability, 200))
		fmt.Fprintf(&skillDesc, "步骤：%s\n", truncateRunes(strings.Join(skill.Steps, " → "), 400))
		if skill.Rationale != "" {
			fmt.Fprintf(&skillDesc, "提议理由：%s\n", truncateRunes(skill.Rationale, 200))
		}
	}
	var known strings.Builder
	if len(existing) == 0 {
		known.WriteString("（暂无已有技能页）\n")
	}
	for i := range existing {
		fmt.Fprintf(&known, "- %s：%s\n", existing[i].Title, truncateRunes(existing[i].Summary, 80))
	}
	var logLines strings.Builder
	for i := range entries {
		fmt.Fprintf(&logLines, "- [%s] %s\n", entries[i].Kind, truncateRunes(entries[i].Summary, 100))
	}
	var provLines strings.Builder
	for _, ref := range provenance {
		fmt.Fprintf(&provLines, "- %s\n", ref.Title)
	}

	prompt := fmt.Sprintf(`你是知识库评审。下面有一个技能提案、已有技能页清单、以及该知识库的进化日志（含已被拒绝的提案）。

逐项判断：
1. 是否与已有技能页重复或高度重叠？
2. 是否有证据支撑（provenance 指向的模式页/经验是否成立）？
3. 它是否回应了进化日志里记录的真实失败？
4. 是否与已有知识矛盾？
任何一项不过就拒绝。输出纯 JSON：{"verdict":"accept"或"reject","rationale":"一句话理由"}

技能提案：
%s

已有技能页：
%s

进化日志（最近 %d 条）：
%s

提案引用的出处：
%s`, skillDesc.String(), known.String(), len(entries), logLines.String(), provLines.String())

	messages := []llms.MessageContent{
		langchain.SystemTextParts("你是严格的知识库评审：技能宁缺毋滥，重复与无证据的提案一律拒绝。输出必须是纯 JSON。"),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}
	cctx, cancel := context.WithTimeout(ctx, skillGateTimeout)
	defer cancel()
	resp, err := llm.GenerateContent(cctx, messages,
		llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error { return nil }))
	if err != nil {
		return true, "reviewer call failed — human gate applies"
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
		prose = strings.TrimSpace(prose[:i])
	}
	var out struct {
		Verdict   string `json:"verdict"`
		Rationale string `json:"rationale"`
	}
	if err := json.Unmarshal([]byte(prose), &out); err != nil {
		return true, "reviewer output not parseable — human gate applies"
	}
	rationale = strings.TrimSpace(out.Rationale)
	if strings.EqualFold(out.Verdict, "reject") {
		return false, rationale
	}
	return true, rationale
}

// gateSkillProposal runs the reviewer and records the verdict; rejections
// return errGateRejected so the proposal stays open for a human.
func gateSkillProposal(parent *orm.Context, kbID string, p *core.WikiGovernanceProposal) error {
	allowed, rationale := reviewSkillProposal(parent, kbID, p)
	if p.Evidence == nil {
		p.Evidence = util.MapStr{}
	}
	p.Evidence["gate_verdict"] = map[string]interface{}{
		"allowed":   allowed,
		"rationale": rationale,
		"at":        time.Now().Format(time.RFC3339),
	}
	AppendEvolutionEntry(parent, kbID, core.WikiEvolutionGateVerdict, "reviewer",
		"governance", p.ID,
		fmt.Sprintf("评审门控：%s — %s", map[bool]string{true: "通过", false: "拒绝"}[allowed], rationale),
		util.MapStr{"allowed": allowed, "rationale": rationale, "proposal_type": p.Type})
	if !allowed {
		return fmt.Errorf("%w: %s", errGateRejected, rationale)
	}
	return nil
}

/* ---------------- proposer ---------------- */

// findSkillPages lists a KB's skill pages.
func findSkillPages(ctx *orm.Context, kbID string, size int) []core.WikiArticle {
	orm.WithModel(ctx, &core.WikiArticle{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(size).
		Filter(orm.TermQuery("kb_id", kbID), orm.TermQuery("page_type", core.WikiPageTypeSkill)))
	if err != nil || res == nil {
		return nil
	}
	pages, _, derr := elastic.DecodeHits[core.WikiArticle](res)
	if derr != nil {
		return nil
	}
	return pages
}

// findEvolutionEntries lists a KB's recent evolution-log lines.
func findEvolutionEntries(ctx *orm.Context, kbID string, size int) []core.WikiEvolutionEntry {
	orm.WithModel(ctx, &core.WikiEvolutionEntry{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(size).
		Filter(orm.TermQuery("kb_id", kbID)).
		SortBy(orm.Sort{Field: "created", SortType: "desc"}))
	if err != nil || res == nil {
		return nil
	}
	entries, _, derr := elastic.DecodeHits[core.WikiEvolutionEntry](res)
	if derr != nil {
		return nil
	}
	return entries
}

type llmSkillOut struct {
	Action         string   `json:"action"`
	Target         string   `json:"target,omitempty"`
	Title          string   `json:"title"`
	Applicability  string   `json:"applicability"`
	Steps          []string `json:"steps"`
	AntiPatterns   []string `json:"anti_patterns,omitempty"`
	WhenNotToApply []string `json:"when_not_to_apply,omitempty"`
	ReachFor       struct {
		Phrases []string `json:"phrases"`
		Globs   []string `json:"globs"`
	} `json:"reach_for"`
	Provenance []string `json:"provenance,omitempty"`
	Rationale  string   `json:"rationale"`
}

// runProposer is one proposer pass: read patterns + experiences + evolution
// log, emit at most ONE atomic skill proposal (or no_action).
func runProposer(parent *orm.Context, kbID string) (*distillRun, error) {
	run := &distillRun{}

	readCtx := orm.NewContextWithParent(parent)
	readCtx.Set(orm.DirectReadWithoutPermissionCheck, true)

	// the wiki (proposer MUST read it): pattern pages
	patterns := findPatternPages(readCtx, kbID, distillMaxExisting)
	// fresh experiences (same window rules as the distiller)
	watermark := readDistillWatermark(kbID)
	orm.WithModel(readCtx, &core.WikiExperience{})
	res, err := orm.SearchV2(readCtx, orm.NewQuery().Size(distillMaxExperiences).
		Filter(orm.TermQuery("kb_id", kbID)).
		SortBy(orm.Sort{Field: "created", SortType: "desc"}))
	if err != nil {
		return run, err
	}
	experiences, _, derr := elastic.DecodeHits[core.WikiExperience](res)
	if derr != nil {
		return run, derr
	}
	fresh := make([]core.WikiExperience, 0, len(experiences))
	for i := range experiences {
		if experiences[i].Created != nil && experiences[i].Created.Before(watermark) {
			continue
		}
		fresh = append(fresh, experiences[i])
	}
	run.Sampled = len(fresh)
	// the evolution log: accepted AND rejected — the paper's core trick
	entries := findEvolutionEntries(readCtx, kbID, evolutionLogReadSize)

	// skills already exist? existing skills bound the dedupe surface
	existingSkills := findSkillPages(readCtx, kbID, distillMaxExisting)

	model := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if model == nil {
		return run, fmt.Errorf("no language model configured")
	}
	llm, err := langchain.SimplyGetLLM(model.ProviderID, model.ID, "")
	if err != nil {
		return run, err
	}

	var patternLines strings.Builder
	if len(patterns) == 0 {
		patternLines.WriteString("（暂无模式页——没有失败模式的沉淀，通常也没有提炼技能的依据）\n")
	}
	for i := range patterns {
		if patterns[i].Title == patternIndexTitle {
			continue
		}
		oneLiner := patterns[i].Title
		if patterns[i].Pattern != nil {
			oneLiner = fmt.Sprintf("%s — 症状：%s | 解法：%s", patterns[i].Title,
				truncateRunes(patterns[i].Pattern.Symptom, 80), truncateRunes(patterns[i].Pattern.Fix, 80))
		}
		fmt.Fprintf(&patternLines, "%d. %s\n", i+1, oneLiner)
	}
	var expLines strings.Builder
	for i := range fresh {
		fmt.Fprintf(&expLines, "- #%d（%s）%s %s\n", i+1, fresh[i].Feedback,
			truncateRunes(fresh[i].Question, 160), truncateRunes(fresh[i].Note, 160))
	}
	var logLines strings.Builder
	for i := range entries {
		fmt.Fprintf(&logLines, "- [%s] %s\n", entries[i].Kind, truncateRunes(entries[i].Summary, 120))
	}
	var skillLines strings.Builder
	for i := range existingSkills {
		fmt.Fprintf(&skillLines, "- %s：%s\n", existingSkills[i].Title, truncateRunes(existingSkills[i].Summary, 80))
	}

	prompt := fmt.Sprintf(`你是技能提议者。基于下面的模式页、经验记录和进化日志，提出至多一个"技能"：把知识库已经学到的做法固化成可执行的程序性知识（适用条件 + 步骤 + 反例）。

规则：
- 只输出 JSON，不要解释或 markdown 代码块
- 每次至多一个技能（原子提案）：新建（create）或修补已有技能（patch，需给 target）
- 必须有出处：provenance 填支持该技能的模式页标题（至少一个）；没有模式页支撑就不要提
- 与已有技能重复的不要提；进化日志里被拒绝过的类似提案不要原样重提
- 证据不足就输出 {"action":"none","rationale":"..."}——这是正当结果
- reach_for.phrases 是用户任务里出现哪些词时应加载此技能；globs 是文件名模式（可空）
- 步骤要具体可执行，每步一句话

已有技能页：
%s

模式页（wiki）：
%s

近期经验：
%s

进化日志（最近 %d 条，含被拒提案）：
%s

输出格式：
{"action":"create"或"patch"或"none","target":"仅 patch 需要","title":"技能名","applicability":"何时适用","steps":["1","2"],"anti_patterns":["..."],"when_not_to_apply":["..."],"reach_for":{"phrases":["..."],"globs":[]},"provenance":["模式页标题"],"rationale":"为何值得固化"}`,
		skillLines.String(), patternLines.String(), expLines.String(), len(entries), logLines.String())

	messages := []llms.MessageContent{
		langchain.SystemTextParts("你是严谨的技能提议者：技能必须有模式页出处、必须不同于已有技能，宁缺毋滥。输出必须是纯 JSON。"),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}
	cctx, cancel := context.WithTimeout(context.Background(), skillProposeTimeout)
	defer cancel()
	resp, err := llm.GenerateContent(cctx, messages,
		llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error { return nil }))
	if err != nil {
		return run, err
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
		prose = strings.TrimSpace(prose[:i])
	}
	var out llmSkillOut
	if err := json.Unmarshal([]byte(prose), &out); err != nil {
		return run, fmt.Errorf("proposer output not parseable: %w", err)
	}

	switch strings.ToLower(strings.TrimSpace(out.Action)) {
	case "none", "":
		run.NoAction = true
		AppendEvolutionEntry(parent, kbID, core.WikiEvolutionSkillProposed, "proposer",
			"kb", kbID,
			fmt.Sprintf("技能提议：无可提（%s）", truncateRunes(out.Rationale, 120)),
			util.MapStr{"no_action": true, "rationale": out.Rationale})
		return run, nil
	case "patch":
		target := strings.TrimSpace(out.Target)
		if target == "" {
			target = strings.TrimSpace(out.Title)
		}
		orm.WithModel(readCtx, &core.WikiArticle{})
		tres, terr := orm.SearchV2(readCtx, orm.NewQuery().Size(1).
			Filter(orm.TermQuery("kb_id", kbID),
				orm.TermQuery("page_type", core.WikiPageTypeSkill),
				orm.TermQuery("title", target)))
		if terr != nil || tres == nil {
			return run, fmt.Errorf("patch target lookup failed")
		}
		thits, _, tderr := elastic.DecodeHits[core.WikiArticle](tres)
		if tderr != nil || len(thits) == 0 {
			run.SkippedNotes = append(run.SkippedNotes, fmt.Sprintf("patch target not found: %s", target))
			return run, nil
		}
		proposal := buildSkillProposal(kbID, &out, &thits[0])
		if gateAndFileSkillProposal(parent, kbID, proposal) {
			run.Updated++
		} else {
			run.SkippedNotes = append(run.SkippedNotes, fmt.Sprintf("open twin already queued: %s", proposal.ArticleTitle))
		}
	default: // create
		proposal := buildSkillProposal(kbID, &out, nil)
		if gateAndFileSkillProposal(parent, kbID, proposal) {
			run.Created++
		} else {
			run.SkippedNotes = append(run.SkippedNotes, fmt.Sprintf("open twin already queued: %s", proposal.ArticleTitle))
		}
	}
	run.Proposals = run.Created + run.Updated

	AppendEvolutionEntry(parent, kbID, core.WikiEvolutionSkillProposed, "proposer",
		"kb", kbID,
		fmt.Sprintf("技能提议：%d 个提案", run.Proposals),
		util.MapStr{"proposals": run.Proposals, "created": run.Created, "updated": run.Updated})
	return run, nil
}

// buildSkillProposal assembles the governance proposal from the proposer
// output; target != nil marks a patch.
func buildSkillProposal(kbID string, out *llmSkillOut, target *core.WikiArticle) *core.WikiGovernanceProposal {
	skill := &core.WikiSkill{
		Applicability:  truncateRunes(out.Applicability, 600),
		Steps:          out.Steps,
		AntiPatterns:   out.AntiPatterns,
		WhenNotToApply: out.WhenNotToApply,
		Rationale:      truncateRunes(out.Rationale, 400),
	}
	reach := &core.SkillReachFor{Phrases: out.ReachFor.Phrases, Globs: out.ReachFor.Globs}
	if len(reach.Phrases) > 0 || len(reach.Globs) > 0 {
		skill.ReachFor = reach
	}
	var provenance []core.WikiSourceReference
	for _, title := range out.Provenance {
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}
		provenance = append(provenance, core.WikiSourceReference{
			DocID:      title,
			SourceType: "pattern",
			Title:      title,
		})
	}
	proposal := &core.WikiGovernanceProposal{
		KbID:      kbID,
		Type:      core.WikiGovernanceSkillCreate,
		Status:    core.WikiGovernanceOpen,
		Reason:    truncateRunes(out.Rationale, 200),
		Evidence:  skillEvidenceMap(skill, provenance),
	}
	if target != nil {
		proposal.Type = core.WikiGovernanceSkillPatch
		proposal.ArticleID = target.ID
		proposal.ArticleTitle = target.Title
	} else {
		proposal.ArticleTitle = truncateRunes(out.Title, 80)
	}
	return proposal
}

func skillEvidenceMap(skill *core.WikiSkill, provenance []core.WikiSourceReference) util.MapStr {
	evidence := util.MapStr{
		"applicability": skill.Applicability,
		"steps":         skill.Steps,
		"rationale":     skill.Rationale,
	}
	if len(skill.AntiPatterns) > 0 {
		evidence["anti_patterns"] = skill.AntiPatterns
	}
	if len(skill.WhenNotToApply) > 0 {
		evidence["when_not_to_apply"] = skill.WhenNotToApply
	}
	if skill.ReachFor != nil {
		evidence["reach_for"] = skill.ReachFor
	}
	if len(provenance) > 0 {
		evidence["sources"] = provenance
	}
	return evidence
}

// gateAndFileSkillProposal: reviewer gate first, then file — a rejection
// is still recorded (evidence + evolution log) but no proposal lands, so
// the human queue never sees pre-rejected noise. Returns false when the
// twin check blocked the file.
func gateAndFileSkillProposal(parent *orm.Context, kbID string, proposal *core.WikiGovernanceProposal) bool {
	pType := proposal.Type
	title := proposal.ArticleTitle
	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	if openPatternTwin(ctx, kbID, pType, title) {
		return false
	}
	if err := gateSkillProposal(parent, kbID, proposal); err != nil {
		if !errors.Is(err, errGateRejected) {
			log.Warnf("wiki: skill gate errored: %v", err)
		}
		return true // processed; rejection recorded, nothing filed
	}
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})
	if err := orm.Create(ctx, proposal); err != nil {
		log.Warnf("wiki: skill proposal create failed: %v", err)
	}
	return true
}

// errors_Is avoided


/* ---------------- SKILL.md export ---------------- */

// exportSkillMd renders a skill page in the Agent Skills SKILL.md format
// (frontmatter name/description + body) — the cross-agent interchange
// format the community implementations converge on.
func (h *APIHandler) exportSkillMd(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	articleID := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})
	article := core.WikiArticle{}
	article.SetID(articleID)
	exists, err := orm.GetV2(ctx, &article)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, articleID)
		return
	}
	if article.PageType != core.WikiPageTypeSkill || article.Skill == nil {
		h.Error400(w, "not a skill page")
		return
	}
	var sb strings.Builder
	name := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(article.Title), " ", "-"))
	description := truncateRunes(article.Skill.Applicability, 120)
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "name: %s\n", name)
	fmt.Fprintf(&sb, "description: %s\n", description)
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n\n", article.Title))
	// structured skill metadata renders as canonical sections; pages whose
	// steps live only in the body (imported SKILL.md documents, hand-written
	// skills) would otherwise export as a bare header — keep their body
	if len(article.Skill.Steps) == 0 && strings.TrimSpace(article.Content) != "" {
		sb.WriteString(strings.TrimSpace(article.Content))
		sb.WriteString("\n\n")
	}
	if len(article.Skill.Steps) > 0 {
		sb.WriteString("## Steps\n\n")
		for i, step := range article.Skill.Steps {
			fmt.Fprintf(&sb, "%d. %s\n", i+1, strings.TrimSpace(step))
		}
		sb.WriteString("\n")
	}
	if len(article.Skill.AntiPatterns) > 0 {
		sb.WriteString("## Anti-patterns\n\n")
		for _, ap := range article.Skill.AntiPatterns {
			fmt.Fprintf(&sb, "- %s\n", ap)
		}
		sb.WriteString("\n")
	}
	if len(article.Skill.WhenNotToApply) > 0 {
		sb.WriteString("## When NOT to apply\n\n")
		for _, w := range article.Skill.WhenNotToApply {
			fmt.Fprintf(&sb, "- %s\n", w)
		}
		sb.WriteString("\n")
	}
	if len(article.Skill.Loads) > 0 {
		sb.WriteString("## Bundled references\n\n")
		for _, load := range article.Skill.Loads {
			fmt.Fprintf(&sb, "- %s — %s\n", load.Path, load.When)
		}
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=SKILL-%s.md", name))
	_, _ = w.Write([]byte(sb.String()))
}

/* ---------------- SKILL.md import ---------------- */

// parseSkillFrontmatter splits an Agent Skills SKILL.md document into its
// `name`/`description` frontmatter and the markdown body. Minimal line
// parsing — the format's frontmatter is flat key: value pairs.
func parseSkillFrontmatter(doc string) (name, description, body string, ok bool) {
	doc = strings.TrimSpace(doc)
	if !strings.HasPrefix(doc, "---") {
		return "", "", doc, false
	}
	end := strings.Index(doc[3:], "\n---")
	if end < 0 {
		return "", "", doc, false
	}
	front := doc[3 : end+3]
	body = strings.TrimSpace(doc[end+3+4:])
	for _, line := range strings.Split(front, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			switch key {
			case "name":
				name = strings.Trim(val, `"'`)
			case "description":
				description = strings.Trim(val, `"'`)
			}
		}
	}
	return name, description, body, name != ""
}

// importSkillMd creates a draft skill page from an external Agent Skills
// document — the import half of the SKILL.md interoperation (external
// packs land as drafts; the review gate stays human).
func (h *APIHandler) importSkillMd(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	body := struct {
		Content string `json:"content"`
	}{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	name, description, mdBody, ok := parseSkillFrontmatter(body.Content)
	if !ok {
		h.Error400(w, "content is not a SKILL.md document (frontmatter with name required)")
		return
	}
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	// same title in the KB → report instead of stacking a twin
	orm.WithModel(ctx, &core.WikiArticle{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("kb_id", kbID), orm.TermQuery("title", name)))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if hits, _, derr := elastic.DecodeHits[core.WikiArticle](res); derr == nil && len(hits) > 0 {
		h.WriteOKJSON(w, util.MapStr{"id": hits[0].ID, "result": "exists"})
		return
	}
	article := &core.WikiArticle{
		KbID:     kbID,
		Title:    truncateRunes(name, 80),
		Summary:  truncateRunes(description, 200),
		Content:  mdBody,
		PageType: core.WikiPageTypeSkill,
		Status:   core.WikiArticleDraft,
		Skill:    &core.WikiSkill{Applicability: truncateRunes(description, 400)},
	}
	if err := orm.Create(ctx, article); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	persistLinkedPages(article)
	if err := addArticleToToc(kbID, article.ID, article.Title); err != nil {
		log.Warnf("wiki: skill import toc failed: %v", err)
	}
	if err := bumpKbArticleCount(kbID, 1); err != nil {
		log.Warnf("wiki: skill import count failed: %v", err)
	}
	h.WriteCreatedOKJSON(w, article.ID)
}

/* ---------------- skill pack import (Agent Skills package, P3) ---------------- */

const (
	skillPackMaxZipBytes = 20 << 20
	skillPackMaxRefs     = 20
	skillPackMaxRefBytes = 2 << 20
)

// bytesFile makes an in-memory buffer satisfy multipart.File so reference
// files from the zip can ride the standard blob-upload path.
type bytesFile struct{ *bytes.Reader }

func (bytesFile) Close() error { return nil }

// skillPackRef is one non-SKILL.md file worth keeping from a package.
type skillPackRef struct {
	relPath string // path inside the package, e.g. references/api-notes.md
	data    []byte
}

// importSkillPack: POST /wiki/kb/:id/_import_skill_pack (multipart "file" =
// an Agent Skills package zip — SKILL.md at the root or one folder deep,
// optional references/ and scripts/ files). The SKILL.md lands as a draft
// skill page; package files become page attachments wired into
// Skill.Loads (tier 3, loaded on demand). Same-title pages report instead
// of stacking a twin; the review gate stays human.
func (h *APIHandler) importSkillPack(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	req.Body = http.MaxBytesReader(w, req.Body, skillPackMaxZipBytes)
	if err := req.ParseMultipartForm(skillPackMaxZipBytes); err != nil {
		h.WriteError(w, "failed to parse upload (20MB cap)", http.StatusBadRequest)
		return
	}
	file, _, err := req.FormFile("file")
	if err != nil {
		h.WriteError(w, "no file uploaded", http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(file)
	if err != nil || len(data) == 0 {
		h.WriteError(w, "failed to read the package", http.StatusBadRequest)
		return
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		h.WriteError(w, "not a zip package", http.StatusBadRequest)
		return
	}

	skillDoc, prefix, refs, perr := parseSkillPack(zr)
	if perr != nil {
		h.WriteError(w, perr.Error(), http.StatusBadRequest)
		return
	}
	name, description, mdBody, ok := parseSkillFrontmatter(skillDoc)
	if !ok {
		h.WriteError(w, "SKILL.md is missing its frontmatter (name required)", http.StatusBadRequest)
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	// same title in the KB → report instead of stacking a twin
	orm.WithModel(ctx, &core.WikiArticle{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("kb_id", kbID), orm.TermQuery("title", truncateRunes(name, 80))))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if hits, _, derr := elastic.DecodeHits[core.WikiArticle](res); derr == nil && len(hits) > 0 {
		h.WriteOKJSON(w, util.MapStr{"id": hits[0].ID, "result": "exists", "references": len(refs)})
		return
	}

	article := &core.WikiArticle{
		KbID:     kbID,
		Title:    truncateRunes(name, 80),
		Summary:  truncateRunes(description, 200),
		Content:  mdBody,
		PageType: core.WikiPageTypeSkill,
		Status:   core.WikiArticleDraft,
		Skill:    &core.WikiSkill{Applicability: truncateRunes(description, 400)},
	}
	if err := orm.Create(ctx, article); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// package files → page attachments; deterministic IDs keep a re-import
	// of the same pack from stacking blob duplicates
	loaded := 0
	for _, ref := range refs {
		fid := util.MD5digest("skillref|" + article.ID + "|" + ref.relPath)
		attID, uerr := attachment.UploadToBlobStore(ctx, fid, bytesFile{bytes.NewReader(ref.data)}, nil,
			path.Base(ref.relPath), "", util.MapStr{"skill_pack": name, "skill_ref_path": ref.relPath}, "", true)
		if uerr != nil {
			log.Warnf("wiki: skill pack reference [%s] failed: %v", ref.relPath, uerr)
			continue
		}
		article.Attachments = append(article.Attachments, attID)
		article.Skill.Loads = append(article.Skill.Loads, core.SkillLoad{
			Path: ref.relPath,
			When: "imported from package — load on demand",
		})
		loaded++
	}
	if loaded > 0 {
		if err := orm.Update(ctx, article); err != nil {
			log.Warnf("wiki: skill pack attachment wiring failed for [%s]: %v", article.Title, err)
		}
	}

	persistLinkedPages(article)
	if err := addArticleToToc(kbID, article.ID, article.Title); err != nil {
		log.Warnf("wiki: skill pack import toc failed: %v", err)
	}
	if err := bumpKbArticleCount(kbID, 1); err != nil {
		log.Warnf("wiki: skill pack import count failed: %v", err)
	}
	log.Infof("wiki: imported skill pack [%s] into kb [%s] — %d reference files (prefix %q)", name, kbID, loaded, prefix)
	h.WriteOKJSON(w, util.MapStr{"_id": article.ID, "result": "created", "references": loaded})
}

// parseSkillPack walks a package zip: SKILL.md must sit at the root or one
// folder deep (the skill-name/ convention); files under references/ and
// scripts/ come along as tier-3 loads. Everything else is ignored.
func parseSkillPack(zr *zip.Reader) (skillDoc, prefix string, refs []skillPackRef, err error) {
	skillPath := ""
	for _, zf := range zr.File {
		name := strings.TrimSuffix(zf.Name, "/")
		if name == "" || strings.HasPrefix(path.Base(name), ".") || strings.Contains(name, "__MACOSX") {
			continue
		}
		if zf.FileInfo().IsDir() {
			continue
		}
		if path.Base(name) == "SKILL.md" && strings.Count(name, "/") <= 1 {
			if skillPath != "" {
				return "", "", nil, fmt.Errorf("multiple SKILL.md entries (%s, %s)", skillPath, name)
			}
			skillPath = name
		}
	}
	if skillPath == "" {
		return "", "", nil, fmt.Errorf("no SKILL.md at the package root (or one folder deep)")
	}
	prefix = ""
	if idx := strings.LastIndex(skillPath, "/"); idx > 0 {
		prefix = skillPath[:idx+1]
	}

	for _, zf := range zr.File {
		name := strings.TrimSuffix(zf.Name, "/")
		if zf.FileInfo().IsDir() || strings.HasPrefix(path.Base(name), ".") || strings.Contains(name, "__MACOSX") {
			continue
		}
		if name == skillPath {
			continue
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rel := strings.TrimPrefix(name, prefix)
		if !(strings.HasPrefix(rel, "references/") || strings.HasPrefix(rel, "scripts/")) {
			continue
		}
		if len(refs) >= skillPackMaxRefs {
			break
		}
		if zf.UncompressedSize64 > skillPackMaxRefBytes {
			continue
		}
		rc, oerr := zf.Open()
		if oerr != nil {
			continue
		}
		buf, rerr := io.ReadAll(io.LimitReader(rc, skillPackMaxRefBytes+1))
		_ = rc.Close()
		if rerr != nil || len(buf) > skillPackMaxRefBytes {
			continue
		}
		refs = append(refs, skillPackRef{relPath: rel, data: buf})
	}

	rc, oerr := func() (io.ReadCloser, error) {
		for _, zf := range zr.File {
			if zf.Name == skillPath {
				return zf.Open()
			}
		}
		return nil, fmt.Errorf("SKILL.md entry vanished")
	}()
	if oerr != nil {
		return "", "", nil, fmt.Errorf("failed to read SKILL.md from the package: %v", oerr)
	}
	defer func() { _ = rc.Close() }()
	doc, _ := io.ReadAll(io.LimitReader(rc, 1<<20))
	return string(doc), prefix, refs, nil
}

/* ---------------- tier-1 injection ---------------- */

// SkillIndexForAssistant renders the tier-1 skill index for an assistant:
// one line per published skill of every KB bound to it — title,
// applicability one-liner and the reach-for phrases. Injected into the
// answer prompt so the model knows the skills exist without loading their
// bodies (progressive disclosure, tier 1); the full pages surface through
// retrieval (tier 2) and their Loads attachments stay tier 3. Empty when
// the assistant has no KB with published skills — callers omit the slot.
func SkillIndexForAssistant(parent *orm.Context, assistantID string) string {
	if assistantID == "" {
		return ""
	}
	defer func() {
		if r := recover(); r != nil {
			log.Warnf("wiki: skill index render recovered: %v", r)
		}
	}()
	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiKnowledgeBase{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(20).
		Filter(orm.TermQuery("assistant_id", assistantID)))
	if err != nil || res == nil {
		return ""
	}
	kbs, _, derr := elastic.DecodeHits[core.WikiKnowledgeBase](res)
	if derr != nil {
		return ""
	}
	var sb strings.Builder
	count := 0
	boundKb := map[string]bool{}
	kbName := map[string]string{}
	for i := range kbs {
		boundKb[kbs[i].ID] = true
		kbName[kbs[i].ID] = kbs[i].Name
	}
	for i := range kbs {
		for _, page := range findSkillPages(ctx, kbs[i].ID, 20) {
			if page.Status != core.WikiArticlePublished || page.Skill == nil {
				continue
			}
			if count == 0 {
				sb.WriteString("\nSkills (procedural knowledge maintained by the knowledge base — follow these steps when the task matches):\n")
			}
			line := fmt.Sprintf("- %s: %s", page.Title, truncateRunes(page.Skill.Applicability, 100))
			if page.Skill.ReachFor != nil && len(page.Skill.ReachFor.Phrases) > 0 {
				line += fmt.Sprintf(" [触发词: %s]", strings.Join(page.Skill.ReachFor.Phrases, ", "))
			}
			sb.WriteString(line + "\n")
			count++
			if count >= 20 {
				return sb.String()
			}
		}
	}

	// tenant-shared skills (LLM-wiki P3 transfer): published skill pages
	// flagged scope=tenant from KBs NOT bound to this assistant join the
	// index read-only — one KB's evolved skill benefits every assistant.
	if count >= 20 {
		return sb.String()
	}
	orm.WithModel(ctx, &core.WikiArticle{})
	res2, err2 := orm.SearchV2(ctx, orm.NewQuery().Size(200).
		Filter(orm.TermQuery("page_type", core.WikiPageTypeSkill), orm.TermQuery("status", core.WikiArticlePublished)))
	if err2 == nil && res2 != nil {
		if shared, _, derr2 := elastic.DecodeHits[core.WikiArticle](res2); derr2 == nil {
			seen := map[string]bool{}
			for i := range shared {
				page := shared[i]
				if page.Skill == nil || page.Skill.Scope != "tenant" || boundKb[page.KbID] || seen[page.ID] {
					continue
				}
				seen[page.ID] = true
				if count == 0 {
					sb.WriteString("\nSkills (procedural knowledge maintained by the knowledge base — follow these steps when the task matches):\n")
				}
				line := fmt.Sprintf("- %s: %s", page.Title, truncateRunes(page.Skill.Applicability, 100))
				if page.Skill.ReachFor != nil && len(page.Skill.ReachFor.Phrases) > 0 {
					line += fmt.Sprintf(" [触发词: %s]", strings.Join(page.Skill.ReachFor.Phrases, ", "))
				}
				if name := sharedKbName(ctx, kbName, page.KbID); name != "" {
					line += fmt.Sprintf(" [共享自 %s]", name)
				}
				sb.WriteString(line + "\n")
				count++
				if count >= 20 {
					break
				}
			}
		}
	}
	return sb.String()
}

// sharedKbName resolves a KB's display name, memoized in the caller's map.
func sharedKbName(ctx *orm.Context, memo map[string]string, kbID string) string {
	if name, ok := memo[kbID]; ok {
		return name
	}
	kb := core.WikiKnowledgeBase{}
	kb.ID = kbID
	orm.WithModel(ctx, &core.WikiKnowledgeBase{})
	if ok, err := orm.GetV2(ctx, &kb); err != nil || !ok {
		memo[kbID] = ""
		return ""
	}
	memo[kbID] = kb.Name
	return kb.Name
}

/* ---------------- manual trigger ---------------- */

func (h *APIHandler) proposeSkillHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	run, err := runProposer(ctx, kbID)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, run)
}
