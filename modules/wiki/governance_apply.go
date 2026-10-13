/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
	langchain "infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common/fingerprint"
	llmmodule "infini.sh/coco/modules/llm"
)

/* The governance queue's closed loop: until now every proposal was only a
 * notification — 标记处理 just retired the record and nothing changed. This
 * endpoint executes the fix each proposal type implies, then retires the
 * proposal as resolved, so one click lands the page in the tree, merges the
 * duplicate, rebuilds the stale page, adopts the entity type, or materializes
 * the compiled draft. */

// compileStatement is one reduced claim a compile proposal carries: the
// normalized claim, the verbatim source excerpt backing it, and the source
// document id for the citation link.
type compileStatement struct {
	DocID   string `json:"doc_id"`
	Excerpt string `json:"excerpt"`
	Claim   string `json:"claim"`
}

// compiledProse is the compile REFINE writing pass: turns the reduced
// statements into actual page prose. Evidence-only drafts read as "a
// preamble and a quote list" — a page with no content of its own. The LLM
// writes ONLY from the provided material (nothing invented); the numbered
// cited claims stay appended below so every sentence stays verifiable.
// Empty when no model is configured or the call fails — the caller falls
// back to the claims draft.
func compiledProse(ctx context.Context, title, kind string, statements []compileStatement) string {
	model := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if model == nil {
		return ""
	}
	llm, err := langchain.SimplyGetLLM(model.ProviderID, model.ID, "")
	if err != nil {
		log.Debugf("wiki compiler: writing pass llm unavailable for [%s]: %v", title, err)
		return ""
	}
	var material strings.Builder
	for i, s := range statements {
		claim := strings.TrimSpace(s.Claim)
		if claim == "" {
			claim = strings.TrimSpace(s.Excerpt)
		}
		if claim == "" {
			continue
		}
		fmt.Fprintf(&material, "%d. %s\n", i+1, claim)
		if ex := strings.TrimSpace(s.Excerpt); ex != "" && ex != claim {
			fmt.Fprintf(&material, "   原文摘录：%s\n", ex)
		}
	}
	if material.Len() == 0 {
		return ""
	}
	kindLabel := "概念"
	if kind == "entity" {
		kindLabel = "实体"
	}
	prompt := fmt.Sprintf(`你是百科编辑。根据下面的来源主张与原文摘录，为%s词条「%s」撰写正文。

规则：
- 只使用给出的材料，绝不引入材料之外的事实、数字或评价
- 语言跟随材料（材料是繁体就用繁体，是英文就用英文）
- 第一段：2-4 句的定义性描述（它是什么、核心事实是什么）
- 材料充分时可再加一段展开；不足就只写第一段，宁短勿编
- 输出纯 Markdown 正文：不要一级标题，不要列表化主张，不要输出"来源引用"清单，不要任何客套说明

来源主张与摘录：
%s`, kindLabel, title, material.String())

	messages := []llms.MessageContent{
		langchain.SystemTextParts("你是严谨的百科编辑，只依据给定材料写作，宁可短也不编造。"),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}
	cctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var out strings.Builder
	resp, err := llm.GenerateContent(cctx, messages,
		llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
			out.Write(chunk)
			return nil
		}))
	if err != nil {
		log.Debugf("wiki compiler: writing pass failed for [%s]: %v", title, err)
		return ""
	}
	prose := ""
	if resp != nil && len(resp.Choices) > 0 {
		prose = resp.Choices[0].Content
	}
	if strings.TrimSpace(prose) == "" {
		prose = out.String()
	}
	// reasoning models may inline thinking before the answer — strip it,
	// then any markdown fence, then judge by substance
	if i := strings.Index(prose, "</think>"); i >= 0 {
		prose = prose[i+len("</think>"):]
	}
	prose = strings.TrimSpace(prose)
	if len([]rune(prose)) < 40 {
		return ""
	}
	return prose
}

// applyAction describes what one click does for a proposal type — the UI
// renders the label on its primary button; empty means no automatic fix
// exists and the proposal stays resolve/dismiss only.
func applyAction(p *core.WikiGovernanceProposal) string {
	switch p.Type {
	case core.WikiGovernanceOrphan:
		return "加入目录"
	case core.WikiGovernanceDuplicate:
		return "合并重复页"
	case core.WikiGovernanceStale, core.WikiGovernanceArticleRefresh:
		return "重建页面"
	case core.WikiGovernanceEntityConflict:
		return "采纳新类型"
	case core.WikiGovernanceKnowledgeCompile:
		return "生成页面"
	case core.WikiGovernanceTreeOrganize:
		return "应用目录"
	}
	return ""
}

func (h *APIHandler) applyGovernanceProposal(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	proposalID := ps.ByName("id")

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})

	var proposal core.WikiGovernanceProposal
	proposal.SetID(proposalID)
	exists, err := orm.GetV2(ctx, &proposal)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, proposalID)
		return
	}
	if proposal.Status != core.WikiGovernanceOpen {
		h.Error400(w, fmt.Sprintf("proposal %s is %s, only open proposals can be applied", proposalID, proposal.Status))
		return
	}

	applied, articleID, applyErr := applyOneProposal(ctx, &proposal, requestUserID(req))
	if applyErr != nil {
		log.Warnf("wiki: governance apply [%s/%s] failed: %v", proposal.Type, proposal.ID, applyErr)
		h.WriteError(w, applyErr.Error(), http.StatusInternalServerError)
		return
	}
	if applied == "" {
		h.Error400(w, "nothing to apply: the proposal's evidence is incomplete")
		return
	}
	resp := util.MapStr{"applied": applied, "result": "ok"}
	if articleID != "" {
		resp["article_id"] = articleID
	}
	h.WriteOKJSON(w, resp)
}

// requestUserID extracts the acting reviewer for audit stamps; empty when
// the call carries no user (batch worker context).
func requestUserID(req *http.Request) string {
	if user, err := security.GetUserFromRequest(req); err == nil && user != nil {
		return user.UserID
	}
	return ""
}

// applyOneProposal executes the fix one proposal type implies and retires
// it as resolved — the shared core of the single-item endpoint and the
// batch apply worker. Returns the human-readable outcome and, when the fix
// generated a page, its id.
func applyOneProposal(ctx *orm.Context, proposal *core.WikiGovernanceProposal, userID string) (string, string, error) {
	if proposal.Status != core.WikiGovernanceOpen {
		return "", "", fmt.Errorf("proposal %s is %s, only open proposals can be applied", proposal.ID, proposal.Status)
	}

	var applied string
	var articleID string
	var err error
	switch proposal.Type {
	case core.WikiGovernanceOrphan:
		applied, err = applyOrphanProposal(ctx, proposal)
	case core.WikiGovernanceDuplicate:
		applied, err = applyDuplicateProposal(ctx, proposal)
	case core.WikiGovernanceStale, core.WikiGovernanceArticleRefresh:
		applied, err = applyRebuildProposal(ctx, proposal)
	case core.WikiGovernanceEntityConflict:
		applied, err = applyEntityConflictProposal(ctx, proposal)
	case core.WikiGovernanceKnowledgeCompile:
		applied, articleID, err = applyCompileProposal(ctx, proposal)
		if err == nil && articleID != "" {
			proposal.Evidence["generated_article_id"] = articleID
		}
	case core.WikiGovernanceTreeOrganize:
		appliedTree, terr := applyTreeProposal(ctx, proposal.KbID, proposal.Evidence)
		if terr != nil {
			return "", "", terr
		}
		if appliedTree {
			applied = "目录已应用"
		}
	default:
		return "", "", fmt.Errorf("proposal type %s has no automatic fix", proposal.Type)
	}
	if err != nil {
		return "", "", err
	}
	if applied == "" {
		return "", "", nil
	}

	proposal.Status = core.WikiGovernanceResolved
	proposal.ResolvedBy = userID
	if proposal.Evidence == nil {
		proposal.Evidence = util.MapStr{}
	}
	proposal.Evidence["applied"] = applied
	if err := orm.Update(ctx, proposal); err != nil {
		return "", "", err
	}
	return applied, articleID, nil
}

/* ---------------- per-type fixes ---------------- */

// applyOrphanProposal hangs the article off the tree root (non-destructive —
// the tree_organize proposal can regroup it later).
func applyOrphanProposal(ctx *orm.Context, p *core.WikiGovernanceProposal) (string, error) {
	if p.ArticleID == "" {
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
	orm.WithModel(ctx, &core.WikiToc{})
	toc, found, err := findToc(ctx, p.KbID)
	if err != nil {
		return "", err
	}
	if !found {
		toc = &core.WikiToc{KbID: p.KbID}
	}
	// already reachable? nothing to do beyond retiring the proposal
	if tocContains(toc.Nodes, p.ArticleID) {
		return "已在目录中", nil
	}
	toc.Nodes = append(toc.Nodes, core.WikiTocNode{
		ID: "toc-" + article.ID, Title: article.Title, Type: "article", ArticleID: article.ID,
	})
	if toc.ID != "" {
		err = orm.Update(ctx, toc)
	} else {
		err = orm.Create(ctx, toc)
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("「%s」已加入目录根", article.Title), nil
}

func tocContains(nodes []core.WikiTocNode, articleID string) bool {
	for _, n := range nodes {
		if n.ArticleID == articleID || tocContains(n.Children, articleID) {
			return true
		}
	}
	return false
}

// applyDuplicateProposal merges the flagged page into the page it
// duplicates: provenance moves over, the twin is archived and dropped from
// the tree. The primary stays the canonical page.
func applyDuplicateProposal(ctx *orm.Context, p *core.WikiGovernanceProposal) (string, error) {
	dupOf, ok := p.Evidence["duplicate_of"].(map[string]interface{})
	if !ok {
		return "", nil
	}
	twinID, _ := dupOf["id"].(string)
	if twinID == "" || p.ArticleID == "" {
		return "", nil
	}
	primary := core.WikiArticle{}
	primary.ID = p.ArticleID
	ok, err := orm.GetV2(ctx, &primary)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("primary article %s not found", p.ArticleID)
	}
	twin := core.WikiArticle{}
	twin.ID = twinID
	ok, err = orm.GetV2(ctx, &twin)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("duplicate article %s not found", twinID)
	}

	// provenance and aliases survive the merge
	for _, s := range twin.Sources {
		known := false
		for _, ps := range primary.Sources {
			if ps.DocID == s.DocID {
				known = true
				break
			}
		}
		if !known {
			primary.Sources = append(primary.Sources, s)
		}
	}
	for _, a := range twin.Aliases {
		if !util.AnyInArrayEquals(primary.Aliases, a) {
			primary.Aliases = append(primary.Aliases, a)
		}
	}
	if strings.TrimSpace(primary.Content) == "" && strings.TrimSpace(twin.Content) != "" {
		primary.Content = twin.Content
	}
	if err := orm.Update(ctx, &primary); err != nil {
		return "", err
	}

	twin.Status = core.WikiArticleArchived
	if err := orm.Update(ctx, &twin); err != nil {
		return "", err
	}
	// drop the twin from the tree so it stops rendering in the navigation
	orm.WithModel(ctx, &core.WikiToc{})
	if toc, found, err := findToc(ctx, p.KbID); err == nil && found {
		kept := tocRemove(toc.Nodes, twin.ID)
		if len(kept) != len(toc.Nodes) {
			toc.Nodes = kept
			if err := orm.Update(ctx, toc); err != nil {
				log.Warnf("wiki: duplicate merge toc prune failed: %v", err)
			}
		}
	}
	return fmt.Sprintf("「%s」已并入「%s」并归档", twin.Title, primary.Title), nil
}

func tocRemove(nodes []core.WikiTocNode, articleID string) []core.WikiTocNode {
	out := nodes[:0]
	for _, n := range nodes {
		if n.ArticleID == articleID {
			continue
		}
		n.Children = tocRemove(n.Children, articleID)
		out = append(out, n)
	}
	return out
}

// applyRebuildProposal rebuilds a stale page from its own provenance —
// entity pages from their entity record, everything else from the article's
// source claims. The rebuild is deterministic (quotes, no LLM), lands as a
// draft for human review, and clears the pending-version signal.
func applyRebuildProposal(ctx *orm.Context, p *core.WikiGovernanceProposal) (string, error) {
	if p.ArticleID == "" {
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

	if article.EntityID != "" {
		ent := core.WikiEntity{}
		ent.ID = article.EntityID
		if ok, err := orm.GetV2(ctx, &ent); err == nil && ok {
			article.Content = DraftEntityContentFor(ctx, article.KbID, &ent)
			article.Status = core.WikiArticleDraft
			if err := orm.Update(ctx, &article); err != nil {
				return "", err
			}
			snapshotPipelineWrite(&article, fmt.Sprintf("治理重建：按 %d 条来源主张重排页面（提案 %s）", len(ent.Sources), p.ID))
			return fmt.Sprintf("「%s」已按 %d 条来源主张重建(草稿)", article.Title, len(ent.Sources)), nil
		}
	}

	// non-entity page: rebuild from the article's own source references
	var sb strings.Builder
	fmt.Fprintf(&sb, "> 本页由治理队列依据 %d 条来源引用自动重建，内容以来源原文为准，等待人工审阅。\n\n", len(article.Sources))
	if len(article.Sources) > 0 {
		sb.WriteString("## 来源主张\n\n")
		for i, s := range article.Sources {
			line := strings.TrimSpace(s.Excerpt)
			if line == "" {
				line = strings.TrimSpace(s.Title)
			}
			if line == "" {
				continue
			}
			title := strings.TrimSpace(s.Title)
			if title == "" {
				title = s.DocID
			}
			fmt.Fprintf(&sb, "%d. **%s** — %s\n", i+1, title, line)
		}
	}
	article.Content = sb.String()
	article.Status = core.WikiArticleDraft
	if err := orm.Update(ctx, &article); err != nil {
		return "", err
	}
	snapshotPipelineWrite(&article, fmt.Sprintf("治理重建：按 %d 条来源引用重排页面（提案 %s）", len(article.Sources), p.ID))
	return fmt.Sprintf("「%s」已按来源引用重建(草稿)", article.Title), nil
}

// applyEntityConflictProposal adopts the newly extracted type — the human
// clicking apply IS the type decision; dismissing keeps the existing type.
func applyEntityConflictProposal(ctx *orm.Context, p *core.WikiGovernanceProposal) (string, error) {
	newType, _ := p.Evidence["extracted_type"].(string)
	if newType == "" || p.ArticleID == "" {
		return "", nil
	}
	entity := core.WikiEntity{}
	entity.ID = p.ArticleID // entity proposals anchor on the entity id
	ok, err := orm.GetV2(ctx, &entity)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("entity %s not found", p.ArticleID)
	}
	old := entity.Type
	entity.Type = newType
	if err := orm.Update(ctx, &entity); err != nil {
		return "", err
	}
	return fmt.Sprintf("「%s」类型已从 %s 改为 %s", entity.Name, old, newType), nil
}

// applyCompileProposal materializes the compiled draft: every statement in
// the evidence becomes a cited section of a draft page — create if the page
// does not exist, append a supplement section if it does. Topic proposals
// (evidence.sections, DESIGN-topic-pages.md) take the structured path:
// outline-driven page create, or section-level merge into the existing page.
func applyCompileProposal(ctx *orm.Context, p *core.WikiGovernanceProposal) (string, string, error) {
	if p.ArticleTitle == "" {
		return "", "", nil
	}
	// legacy stub cleanup: archive the pre-topic-era thin pages in one click
	if p.Evidence["legacy_cleanup"] == true {
		pageIDs := evidenceStrings(p.Evidence["page_ids"])
		if len(pageIDs) == 0 {
			return "", "", nil
		}
		archived := 0
		for _, id := range pageIDs {
			stub := core.WikiArticle{}
			stub.ID = id
			ok, err := orm.GetV2(ctx, &stub)
			if err != nil || !ok || stub.Status == core.WikiArticleArchived {
				continue
			}
			stub.Status = core.WikiArticleArchived
			if err := orm.Update(ctx, &stub); err != nil {
				log.Warnf("wiki: cleanup archive [%s] failed: %v", stub.Title, err)
				continue
			}
			archived++
			orm.WithModel(ctx, &core.WikiToc{})
			if toc, found, terr := findToc(ctx, p.KbID); terr == nil && found {
				kept := tocRemove(toc.Nodes, stub.ID)
				if len(kept) != len(toc.Nodes) {
					toc.Nodes = kept
					if err := orm.Update(ctx, toc); err != nil {
						log.Warnf("wiki: cleanup toc prune failed: %v", err)
					}
				}
			}
			orm.WithModel(ctx, &core.WikiEntity{})
			if res, err := orm.SearchV2(ctx, orm.NewQuery().Size(10).
				Filter(orm.TermQuery("article_id", stub.ID)).Include("id")); err == nil && res != nil {
				if ents, _, derr := elastic.DecodeHits[core.WikiEntity](res); derr == nil {
					for i := range ents {
						ents[i].ArticleID = ""
						if err := orm.Update(ctx, &ents[i]); err != nil {
							log.Warnf("wiki: cleanup entity unlink failed: %v", err)
						}
					}
				}
			}
		}
		return fmt.Sprintf("已归档 %d 个编译残页", archived), "", nil
	}
	if _, ok := p.Evidence["sections"]; ok {
		return applyTopicProposal(ctx, p)
	}
	raw, ok := p.Evidence["statements"]
	if !ok {
		return "", "", nil
	}
	type stmt struct {
		DocID   string `json:"doc_id"`
		Excerpt string `json:"excerpt"`
		Claim   string `json:"claim"`
	}
	var statements []compileStatement
	if err := util.FromJSONBytes(util.MustToJSONBytes(raw), &statements); err != nil || len(statements) == 0 {
		return "", "", nil
	}

	kind, _ := p.Evidence["kind"].(string)
	pageType := core.WikiPageTypeConcept
	if kind == "entity" {
		pageType = core.WikiPageTypeEntity
	}

	// layout-aware draft (WeKnora templates): the page's ontology type
	// (existing entity's type, else the compile kind mapped through the
	// vocabulary) declares the page_sections that organize the draft —
	// statements render under the citation section, typed properties and
	// relation groups fill their declared slots
	var layout []OntologyPageSection
	var targetNames map[string]string
	synthetic := core.WikiEntity{Name: p.ArticleTitle, Type: "concept"}
	if e := wikiEntityByName(ctx, p.ArticleTitle); e != nil {
		synthetic.Type = e.Type
		synthetic.Properties = e.Properties
		synthetic.Relations = e.Relations
	}
	if schema := ResolveOntologySchema(ctx.Context, p.KbID); schema != nil {
		for i := range schema.EntityTypes {
			if strings.EqualFold(schema.EntityTypes[i].Name, synthetic.Type) {
				layout = schema.EntityTypes[i].PageSections
				break
			}
		}
		if len(layout) > 0 {
			targetNames = relationTargetNames(ctx, &synthetic)
		}
	}
	for _, st := range statements {
		if strings.TrimSpace(st.Claim) == "" && strings.TrimSpace(st.Excerpt) == "" {
			continue
		}
		synthetic.Sources = append(synthetic.Sources, core.WikiSourceReference{
			DocID:   st.DocID,
			Title:   p.ArticleTitle,
			Excerpt: strings.TrimSpace(st.Claim),
		})
	}

	var body strings.Builder
	if len(layout) > 0 {
		// REFINE writing pass first: prose turns the section skeleton (which
		// on single-claim pages is just a quote list) into an actual article
		// body; the layout's own 来源引用/别名 sections follow as the
		// verification floor
		sections := composeEntityContentWithIntro(&synthetic, layout, targetNames, "")
		if prose := compiledProse(ctx, p.ArticleTitle, kind, statements); prose != "" {
			body.WriteString(fmt.Sprintf("> 本页由知识编译起草：正文为 AI 依据来源主张撰写，按「%s」类型的页面结构组织，等待人工审阅。\n\n%s\n\n",
				synthetic.Type, prose))
			body.WriteString(sections)
		} else {
			intro := fmt.Sprintf("> 本页由知识编译依据 %d 条来源主张自动起草，按「%s」类型的页面结构组织，等待人工审阅。\n\n",
				len(statements), synthetic.Type)
			body.WriteString(composeEntityContentWithIntro(&synthetic, layout, targetNames, intro))
		}
	} else {
		// numbered cited claims: the verification floor every draft keeps,
		// whatever the writing pass above it produces
		claims := renderClaimList(statements)
		// REFINE writing pass: turn the statements into actual prose —
		// evidence-only drafts read as "a preamble and a quote list",
		// which is exactly the 正文为空 complaint on single-claim pages
		if prose := compiledProse(ctx, p.ArticleTitle, kind, statements); prose != "" {
			body.WriteString(fmt.Sprintf("> 本页由知识编译起草：正文为 AI 依据下方来源主张撰写，等待人工审阅。\n\n%s\n\n## 来源主张\n\n", prose))
			body.WriteString(claims)
		} else {
			body.WriteString(fmt.Sprintf("> 本页由知识编译依据 %d 条来源主张自动起草，等待人工审阅。\n\n", len(statements)))
			body.WriteString(claims)
		}
	}
	// known entity names in the compiled text become wikilinks — the page
	// joins the graph and every mention is a clickable capsule
	content := LinkEntityMentions(ctx, body.String(), p.ArticleTitle)
	aliases, _ := p.Evidence["aliases"].([]interface{})

	existingID, _ := p.Evidence["existing_id"].(string)
	if existingID == "" {
		// create-dedup: a recurring compile re-files the same key every
		// run, and each application of a create proposal would stack a
		// twin page (observed: every concept page twice under auto-approve).
		// A same-title page in this KB absorbs the supplement instead.
		if dup := findArticleByTitle(ctx, p.KbID, p.ArticleTitle); dup != nil {
			existingID = dup.ID
		}
	}
	if existingID != "" {
		article := core.WikiArticle{}
		article.ID = existingID
		if ok, err := orm.GetV2(ctx, &article); err == nil && ok {
			article.Content = strings.TrimSpace(article.Content) + "\n\n## 编译补充\n\n" + content
			// the supplement cites sources too — merge them so the 来源文献
			// panel and freshness tracking see the full picture
			seen := map[string]bool{}
			for _, s := range article.Sources {
				seen[s.DocID] = true
			}
			for _, s := range statements {
				if s.DocID == "" || seen[s.DocID] {
					continue
				}
				seen[s.DocID] = true
				article.Sources = append(article.Sources, core.WikiSourceReference{
					DocID:   s.DocID,
					Title:   s.DocID,
					Excerpt: s.Excerpt,
				})
			}
			for _, a := range aliases {
				if s, ok := a.(string); ok && s != "" && !util.AnyInArrayEquals(article.Aliases, s) {
					article.Aliases = append(article.Aliases, s)
				}
			}
			persistLinkedPages(&article)
			if err := orm.Update(ctx, &article); err != nil {
				return "", "", err
			}
			snapshotPipelineWrite(&article, fmt.Sprintf("知识编译追加 %d 条来源主张（治理提案 %s）", len(statements), p.ID))
			return fmt.Sprintf("「%s」已追加编译补充(%d 条主张)", article.Title, len(statements)), article.ID, nil
		}
	}

	// auto-approve KBs take the generated page straight to published —
	// the toggle's whole point is a KB that updates without a review gate
	createStatus := core.WikiArticleDraft
	if KBAutoApprove(ctx, p.KbID) {
		createStatus = core.WikiArticlePublished
	}
	article := core.WikiArticle{
		KbID:        p.KbID,
		Title:       p.ArticleTitle,
		PageType:    pageType,
		Status:      createStatus,
		AIGenerated: true,
		Content:     content,
	}
	// provenance: every cited source doc becomes a structured source entry
	seenSrc := map[string]bool{}
	for _, s := range statements {
		if s.DocID == "" || seenSrc[s.DocID] {
			continue
		}
		seenSrc[s.DocID] = true
		article.Sources = append(article.Sources, core.WikiSourceReference{
			DocID:   s.DocID,
			Title:   s.DocID,
			Excerpt: s.Excerpt,
		})
	}
	// the wikilinks inside the compiled text resolve here — without this
	// the page ships with no LinkedPages and the graph shows it orphaned
	// (after Create: the id must exist for the links to persist)
	for _, a := range aliases {
		if s, ok := a.(string); ok && s != "" && !util.AnyInArrayEquals(article.Aliases, s) {
			article.Aliases = append(article.Aliases, s)
		}
	}
	if err := orm.Create(ctx, &article); err != nil {
		return "", "", err
	}
	persistLinkedPages(&article)
	if err := orm.Update(ctx, &article); err != nil {
		return "", "", err
	}
	// placement step: file the new page into the folder path the compiler
	// planned (evidence.target_folder) — without this the page ships
	// orphaned from the navigation
	if target, _ := p.Evidence["target_folder"].(string); target != "" {
		if err := placeInToc(ctx, p.KbID, target, article.ID, article.Title); err != nil {
			log.Warnf("wiki: failed to place [%s] into [%s]: %v", article.Title, target, err)
		}
	}
	snapshotPipelineWrite(&article, fmt.Sprintf("知识编译起草，%d 条来源主张（治理提案 %s）", len(statements), p.ID))
	return fmt.Sprintf("已生成草稿页「%s」(%d 条主张)", article.Title, len(statements)), article.ID, nil
}

// placeInToc walks (creating as needed) the "/"-separated folder path in
// the KB's TOC and appends the article node at the leaf.
func placeInToc(ctx *orm.Context, kbID, folderPath, articleID, title string) error {
	orm.WithModel(ctx, &core.WikiToc{})
	toc, found, err := findToc(ctx, kbID)
	if err != nil {
		return err
	}
	if !found {
		toc = &core.WikiToc{KbID: kbID}
	}
	nodes, leaf := ensureFolderChain(toc.Nodes, strings.Split(folderPath, "/"))
	// dedupe: a re-applied proposal must not stack twin nodes
	already := false
	for _, c := range leaf.Children {
		if c.ArticleID == articleID {
			already = true
			break
		}
	}
	if !already {
		leaf.Children = append(leaf.Children, core.WikiTocNode{
			ID: "toc-" + articleID, Title: title, Type: "article", ArticleID: articleID,
		})
	}
	toc.Nodes = nodes
	return orm.Update(ctx, toc)
}

// ensureFolderChain walks/creates the folder chain, returning the (possibly
// rebuilt) node slice plus the leaf folder — index-based reassignment keeps
// appends visible to the caller through reallocation.
func ensureFolderChain(nodes []core.WikiTocNode, segments []string) ([]core.WikiTocNode, *core.WikiTocNode) {
	seg := segments[0]
	for i := range nodes {
		if nodes[i].Type == "folder" && nodes[i].Title == seg {
			if len(segments) == 1 {
				return nodes, &nodes[i]
			}
			child, leaf := ensureFolderChain(nodes[i].Children, segments[1:])
			nodes[i].Children = child
			return nodes, leaf
		}
	}
	nodes = append(nodes, core.WikiTocNode{ID: "folder-" + seg, Title: seg, Type: "folder"})
	leaf := &nodes[len(nodes)-1]
	if len(segments) > 1 {
		child, l := ensureFolderChain(leaf.Children, segments[1:])
		leaf.Children = child
		leaf = l
	}
	return nodes, leaf
}

// findArticleByTitle resolves an existing page by exact title within one
// KB (title is keyword-mapped — exact match). nil when absent.
func findArticleByTitle(ctx *orm.Context, kbID, title string) *core.WikiArticle {
	if strings.TrimSpace(title) == "" {
		return nil
	}
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("kb_id", kbID), orm.TermQuery("title", title)).
		Include("id", "title"))
	if err != nil || res == nil {
		return nil
	}
	arts, _, derr := elastic.DecodeHits[core.WikiArticle](res)
	if derr != nil || len(arts) == 0 {
		return nil
	}
	return &arts[0]
}

/* ---------------- topic proposals (DESIGN-topic-pages.md) ---------------- */

// evidenceStrings decodes an evidence field that was filed as []string but
// round-trips through storage as []interface{}.
func evidenceStrings(v interface{}) []string {
	items, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		if s, ok := it.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

// docTitlesFor resolves document titles for the citation links the topic
// renderer emits (the claim pool only carries doc ids).
func docTitlesFor(ctx *orm.Context, sections []core.WikiSection, statements []compileStatement) map[string]string {
	ids := map[string]bool{}
	for _, s := range sections {
		for _, d := range s.DocIDs {
			ids[d] = true
		}
	}
	for _, s := range statements {
		if s.DocID != "" {
			ids[s.DocID] = true
		}
	}
	if len(ids) == 0 {
		return nil
	}
	idList := make([]string, 0, len(ids))
	for id := range ids {
		idList = append(idList, id)
	}
	titles := map[string]string{}
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.Document{})
	if res, err := orm.SearchV2(octx, orm.NewQuery().Size(100).
		Filter(orm.TermsQuery("id", idList)).Include("id", "title")); err == nil && res != nil {
		if docs, _, derr := elastic.DecodeHits[core.Document](res); derr == nil {
			for i := range docs {
				if docs[i].Title != "" {
					titles[docs[i].ID] = docs[i].Title
				}
			}
		}
	}
	return titles
}

// mergeTopicSections merges incoming compiled sections into an existing
// outline at section granularity — update-first enrichment: same-heading
// sections get their body refreshed and provenance unioned, new headings
// append. An incoming empty body never wipes existing prose.
func mergeTopicSections(outline []core.WikiSection, incoming []core.WikiSection) []core.WikiSection {
	for _, inc := range incoming {
		matched := false
		for i := range outline {
			if fingerprint.Normalize(outline[i].Heading) == fingerprint.Normalize(inc.Heading) {
				matched = true
				if strings.TrimSpace(inc.Body) != "" {
					outline[i].Body = strings.TrimSpace(inc.Body)
					outline[i].UpdatedAt = inc.UpdatedAt
				}
				seen := map[string]bool{}
				for _, d := range outline[i].DocIDs {
					seen[d] = true
				}
				for _, d := range inc.DocIDs {
					if !seen[d] {
						seen[d] = true
						outline[i].DocIDs = append(outline[i].DocIDs, d)
					}
				}
				break
			}
		}
		if !matched && strings.TrimSpace(inc.Body) != "" {
			outline = append(outline, inc)
		}
	}
	return outline
}

// absorbThinPages folds the compiled topic's thin AI pages into the topic
// page: their provenance merges in, the pages archive and drop out of the
// tree, and their entities let go of the page link (the entity stays in
// the ontology — pages and entities are peers, absorb never deletes one).
func absorbThinPages(ctx *orm.Context, kbID string, article *core.WikiArticle, absorbIDs []string) int {
	absorbed := 0
	for _, id := range absorbIDs {
		if id == "" || id == article.ID {
			continue
		}
		twin := core.WikiArticle{}
		twin.ID = id
		ok, err := orm.GetV2(ctx, &twin)
		if err != nil || !ok {
			continue
		}
		for _, s := range twin.Sources {
			known := false
			for _, ps := range article.Sources {
				if ps.DocID == s.DocID {
					known = true
					break
				}
			}
			if !known {
				article.Sources = append(article.Sources, s)
			}
		}
		twin.Status = core.WikiArticleArchived
		if err := orm.Update(ctx, &twin); err != nil {
			log.Warnf("wiki: absorb archive [%s] failed: %v", twin.Title, err)
			continue
		}
		absorbed++
		// out of the tree so the menu stops showing the thin twin
		orm.WithModel(ctx, &core.WikiToc{})
		if toc, found, err := findToc(ctx, kbID); err == nil && found {
			kept := tocRemove(toc.Nodes, twin.ID)
			if len(kept) != len(toc.Nodes) {
				toc.Nodes = kept
				if err := orm.Update(ctx, toc); err != nil {
					log.Warnf("wiki: absorb toc prune failed: %v", err)
				}
			}
		}
		orm.WithModel(ctx, &core.WikiEntity{})
		if res, err := orm.SearchV2(ctx, orm.NewQuery().Size(50).
			Filter(orm.TermQuery("article_id", twin.ID)).Include("id")); err == nil && res != nil {
			if ents, _, derr := elastic.DecodeHits[core.WikiEntity](res); derr == nil {
				for i := range ents {
					ents[i].ArticleID = ""
					if err := orm.Update(ctx, &ents[i]); err != nil {
						log.Warnf("wiki: absorb entity unlink [%s] failed: %v", ents[i].Name, err)
					}
				}
			}
		}
	}
	return absorbed
}

// applyTopicProposal materializes a topic proposal: create the
// outline-driven page, or merge its sections into the existing topic page
// (update-first), then absorb the thin pages it consolidates.
func applyTopicProposal(ctx *orm.Context, p *core.WikiGovernanceProposal) (string, string, error) {
	var sections []core.WikiSection
	if err := util.FromJSONBytes(util.MustToJSONBytes(p.Evidence["sections"]), &sections); err != nil || len(sections) == 0 {
		return "", "", nil
	}
	var statements []compileStatement
	if raw, ok := p.Evidence["statements"]; ok {
		util.FromJSONBytes(util.MustToJSONBytes(raw), &statements)
	}
	keyEntities := evidenceStrings(p.Evidence["key_entities"])
	aliases := evidenceStrings(p.Evidence["aliases"])
	absorbIDs := evidenceStrings(p.Evidence["absorb_ids"])
	planned := len(sections)
	switch v := p.Evidence["planned"].(type) {
	case float64:
		if int(v) > planned {
			planned = int(v)
		}
	case int:
		if v > planned {
			planned = v
		}
	}
	docTitles := docTitlesFor(ctx, sections, statements)

	// update-first: the planner's target, the outline's claimed title, then
	// a same-title page — in that order
	updateID, _ := p.Evidence["update_article_id"].(string)
	if updateID == "" {
		if t, _ := p.Evidence["update_title"].(string); t != "" {
			if a := findArticleByTitle(ctx, p.KbID, t); a != nil {
				updateID = a.ID
			}
		}
	}
	if updateID == "" {
		if dup := findArticleByTitle(ctx, p.KbID, p.ArticleTitle); dup != nil {
			updateID = dup.ID
		}
	}

	if updateID != "" {
		article := core.WikiArticle{}
		article.ID = updateID
		if ok, err := orm.GetV2(ctx, &article); err == nil && ok {
			// a hand-written page never gets clobbered by the merge — its
			// content survives as the leading section
			if !article.AIGenerated && article.PageType != core.WikiPageTypeTopic && strings.TrimSpace(article.Content) != "" {
				sections = append([]core.WikiSection{{
					ID: sectionID("原页面内容"), Heading: "原页面内容",
					Body: strings.TrimSpace(article.Content), UpdatedAt: time.Now().Unix(),
				}}, sections...)
			}
			article.Outline = mergeTopicSections(article.Outline, sections)
			if article.PageType == "" || article.PageType == core.WikiPageTypeEntity || article.PageType == core.WikiPageTypeConcept {
				article.PageType = core.WikiPageTypeTopic
			}
			for _, ke := range keyEntities {
				if !util.AnyInArrayEquals(article.KeyEntities, ke) {
					article.KeyEntities = append(article.KeyEntities, ke)
				}
			}
			seenSrc := map[string]bool{}
			for _, s := range article.Sources {
				seenSrc[s.DocID] = true
			}
			for _, s := range statements {
				if s.DocID == "" || seenSrc[s.DocID] {
					continue
				}
				seenSrc[s.DocID] = true
				article.Sources = append(article.Sources, core.WikiSourceReference{DocID: s.DocID, Title: docTitles[s.DocID], Excerpt: s.Excerpt})
			}
			for _, a := range aliases {
				if !util.AnyInArrayEquals(article.Aliases, a) {
					article.Aliases = append(article.Aliases, a)
				}
			}
			absorbed := absorbThinPages(ctx, p.KbID, &article, absorbIDs)
			article.Maturity = ComputeTopicMaturity(article.Outline, planned, len(article.Sources))
			article.Content = LinkEntityMentions(ctx, RenderTopicContent(article.Outline, statements, docTitles), article.Title)
			persistLinkedPages(&article)
			if err := orm.Update(ctx, &article); err != nil {
				return "", "", err
			}
			outcome := fmt.Sprintf("「%s」已按 %d 节合并更新(%d 条主张)", article.Title, len(article.Outline), len(statements))
			if absorbed > 0 {
				outcome += fmt.Sprintf("，收编 %d 个薄页", absorbed)
			}
			snapshotPipelineWrite(&article, fmt.Sprintf("知识编译更新主题页 %d 节（治理提案 %s）", len(sections), p.ID))
			return outcome, article.ID, nil
		}
	}

	// create: the outline IS the page
	content := LinkEntityMentions(ctx, RenderTopicContent(sections, statements, docTitles), p.ArticleTitle)
	createStatus := core.WikiArticleDraft
	if KBAutoApprove(ctx, p.KbID) {
		createStatus = core.WikiArticlePublished
	}
	article := core.WikiArticle{
		KbID:        p.KbID,
		Title:       p.ArticleTitle,
		PageType:    core.WikiPageTypeTopic,
		Status:      createStatus,
		AIGenerated: true,
		Content:     content,
		Outline:     sections,
		KeyEntities: keyEntities,
	}
	seenSrc := map[string]bool{}
	for _, s := range statements {
		if s.DocID == "" || seenSrc[s.DocID] {
			continue
		}
		seenSrc[s.DocID] = true
		article.Sources = append(article.Sources, core.WikiSourceReference{DocID: s.DocID, Title: docTitles[s.DocID], Excerpt: s.Excerpt})
	}
	for _, a := range aliases {
		if !util.AnyInArrayEquals(article.Aliases, a) {
			article.Aliases = append(article.Aliases, a)
		}
	}
	if err := orm.Create(ctx, &article); err != nil {
		return "", "", err
	}
	// linked pages need the generated id — persisting before Create lost
	// them on every compiled page (graph showed topic pages orphaned)
	persistLinkedPages(&article)
	absorbed := absorbThinPages(ctx, p.KbID, &article, absorbIDs)
	article.Maturity = ComputeTopicMaturity(article.Outline, planned, len(article.Sources))
	if err := orm.Update(ctx, &article); err != nil {
		log.Warnf("wiki: topic maturity stamp failed for [%s]: %v", article.Title, err)
	}
	if target, _ := p.Evidence["target_folder"].(string); target != "" {
		if err := placeInToc(ctx, p.KbID, target, article.ID, article.Title); err != nil {
			log.Warnf("wiki: failed to place [%s] into [%s]: %v", article.Title, target, err)
		}
	}
	snapshotPipelineWrite(&article, fmt.Sprintf("知识编译起草主题页 %d 节（治理提案 %s）", len(sections), p.ID))
	outcome := fmt.Sprintf("已生成主题页「%s」(%d 节,%d 条主张)", article.Title, len(sections), len(statements))
	if absorbed > 0 {
		outcome += fmt.Sprintf("，收编 %d 个薄页", absorbed)
	}
	return outcome, article.ID, nil
}
