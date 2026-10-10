/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

/* The governance queue's closed loop: until now every proposal was only a
 * notification — 标记处理 just retired the record and nothing changed. This
 * endpoint executes the fix each proposal type implies, then retires the
 * proposal as resolved, so one click lands the page in the tree, merges the
 * duplicate, rebuilds the stale page, adopts the entity type, or materializes
 * the compiled draft. */

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

	var applied string
	switch proposal.Type {
	case core.WikiGovernanceOrphan:
		applied, err = applyOrphanProposal(ctx, &proposal)
	case core.WikiGovernanceDuplicate:
		applied, err = applyDuplicateProposal(ctx, &proposal)
	case core.WikiGovernanceStale, core.WikiGovernanceArticleRefresh:
		applied, err = applyRebuildProposal(ctx, &proposal)
	case core.WikiGovernanceEntityConflict:
		applied, err = applyEntityConflictProposal(ctx, &proposal)
	case core.WikiGovernanceKnowledgeCompile:
		applied, err = applyCompileProposal(ctx, &proposal)
	case core.WikiGovernanceTreeOrganize:
		appliedTree, terr := applyTreeProposal(ctx, proposal.KbID, proposal.Evidence)
		if terr != nil {
			log.Warnf("wiki: governance apply [%s/%s] failed: %v", proposal.Type, proposal.ID, terr)
			h.WriteError(w, terr.Error(), http.StatusInternalServerError)
			return
		}
		if appliedTree {
			applied = "目录已应用"
		}
	default:
		h.Error400(w, fmt.Sprintf("proposal type %s has no automatic fix", proposal.Type))
		return
	}
	if err != nil {
		log.Warnf("wiki: governance apply [%s/%s] failed: %v", proposal.Type, proposal.ID, err)
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if applied == "" {
		h.Error400(w, "nothing to apply: the proposal's evidence is incomplete")
		return
	}

	proposal.Status = core.WikiGovernanceResolved
	if user, err := security.GetUserFromRequest(req); err == nil && user != nil && user.UserID != "" {
		proposal.ResolvedBy = user.UserID
	}
	if proposal.Evidence == nil {
		proposal.Evidence = util.MapStr{}
	}
	proposal.Evidence["applied"] = applied
	if err := orm.Update(ctx, &proposal); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, util.MapStr{"applied": applied, "result": "ok"})
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
		entity := core.WikiEntity{}
		entity.ID = article.EntityID
		if ok, err := orm.GetV2(ctx, &entity); err == nil && ok {
			article.Content = DraftEntityContent(&entity)
			article.Status = core.WikiArticleDraft
			if err := orm.Update(ctx, &article); err != nil {
				return "", err
			}
			return fmt.Sprintf("「%s」已按 %d 条来源主张重建(草稿)", article.Title, len(entity.Sources)), nil
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
// does not exist, append a supplement section if it does.
func applyCompileProposal(ctx *orm.Context, p *core.WikiGovernanceProposal) (string, error) {
	if p.ArticleTitle == "" {
		return "", nil
	}
	raw, ok := p.Evidence["statements"]
	if !ok {
		return "", nil
	}
	type stmt struct {
		DocID   string `json:"doc_id"`
		Excerpt string `json:"excerpt"`
		Claim   string `json:"claim"`
	}
	var statements []stmt
	if err := util.FromJSONBytes(util.MustToJSONBytes(raw), &statements); err != nil || len(statements) == 0 {
		return "", nil
	}

	kind, _ := p.Evidence["kind"].(string)
	pageType := core.WikiPageTypeConcept
	if kind == "entity" {
		pageType = core.WikiPageTypeEntity
	}

	var body strings.Builder
	body.WriteString(fmt.Sprintf("> 本页由知识编译依据 %d 条来源主张自动起草，等待人工审阅。\n\n", len(statements)))
	for i, s := range statements {
		claim := strings.TrimSpace(s.Claim)
		if claim == "" {
			claim = strings.TrimSpace(s.Excerpt)
		}
		if claim == "" {
			continue
		}
		fmt.Fprintf(&body, "%d. %s", i+1, claim)
		if s.DocID != "" {
			// the citation is a live link to the source document's preview
			fmt.Fprintf(&body, "（来源:[%s](#/preview/document/%s)）", s.DocID, s.DocID)
		}
		body.WriteString("\n")
	}
	// known entity names in the compiled text become wikilinks — the page
	// joins the graph and every mention is a clickable capsule
	content := LinkEntityMentions(ctx, body.String(), p.ArticleTitle)

	existingID, _ := p.Evidence["existing_id"].(string)
	if existingID != "" {
		article := core.WikiArticle{}
		article.ID = existingID
		if ok, err := orm.GetV2(ctx, &article); err == nil && ok {
			article.Content = strings.TrimSpace(article.Content) + "\n\n## 编译补充\n\n" + content
			if err := orm.Update(ctx, &article); err != nil {
				return "", err
			}
			return fmt.Sprintf("「%s」已追加编译补充(%d 条主张)", article.Title, len(statements)), nil
		}
	}

	aliases, _ := p.Evidence["aliases"].([]interface{})
	article := core.WikiArticle{
		KbID:        p.KbID,
		Title:       p.ArticleTitle,
		PageType:    pageType,
		Status:      core.WikiArticleDraft,
		AIGenerated: true,
		Content:     content,
	}
	for _, a := range aliases {
		if s, ok := a.(string); ok && s != "" && !util.AnyInArrayEquals(article.Aliases, s) {
			article.Aliases = append(article.Aliases, s)
		}
	}
	if err := orm.Create(ctx, &article); err != nil {
		return "", err
	}
	return fmt.Sprintf("已生成草稿页「%s」(%d 条主张)", article.Title, len(statements)), nil
}
