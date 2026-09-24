/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"

	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

/* Knowledge governance (lexiang-inspired, D1-bound): a scheduled scanner
 * inspects knowledge bases and files governance proposals — stale pages with
 * a pending auto-updated version, LLM-confirmed duplicates, contradictions
 * between live content and its regenerated draft, low-quality published
 * pages and articles missing from the TOC tree.
 *
 * The scanner only proposes. Every fix (apply a regenerated version, merge a
 * duplicate, archive, re-tree) stays a human decision on the queue — the
 * governance twin of the "AI drafts, humans publish" rule.
 */

const (
	governanceTaskID             = "wiki-governance-scan"
	governanceDefaultInterval    = "24h"
	governanceMaxKBs             = 1000
	governanceMaxArticlesPerKB   = 10000
	governanceMaxProposalsPerRun = 100
	governanceMaxDuplicatePairs  = 20
)

func init() {
	registerGovernanceTask()
}

func registerGovernanceTask() {
	interval := strings.TrimSpace(os.Getenv("WIKI_GOVERNANCE_INTERVAL"))
	switch strings.ToLower(interval) {
	case "":
		interval = governanceDefaultInterval
	case "off", "disabled", "0":
		return
	}
	task.RegisterScheduleTask(task.ScheduleTask{
		ID:          governanceTaskID,
		Group:       "wiki",
		Description: "Wiki governance scan: file stale/duplicate/conflict/low-quality/orphan proposals for human review",
		Interval:    interval,
		Singleton:   true,
		Task:        governanceCheck,
	})
}

// governanceLLM resolves the judge model; package var so tests can stub it.
// A nil-capable error is expected on unconfigured systems — the scan then
// degrades to the heuristic detectors only.
var governanceLLM = func() (llms.Model, error) {
	return resolveLanguageLLM("", "")
}

func governanceCheck(ctx context.Context) {
	llm, llmErr := governanceLLM()
	if llmErr != nil {
		// no model configured: heuristics still run, LLM confirmation waits
		log.Infof("wiki: governance scan running in heuristic-only mode, %v", llmErr)
	}
	stats := governanceSweep(ctx, llm)
	log.Infof("wiki: governance scan filed %d proposals (%d duplicates confirmed, %d conflicts detected)",
		stats.filed, stats.duplicates, stats.conflicts)
}

type governanceStats struct {
	filed      int
	duplicates int
	conflicts  int
}

// governanceSweep is the testable core: per KB it runs the detectors and
// files proposals that don't already have an open twin (same article + type).
func governanceSweep(ctx context.Context, llm llms.Model) governanceStats {
	stats := governanceStats{}

	open := openProposalKeys(ctx)
	kbs := governanceKBs(ctx)
	for i := range kbs {
		if stats.filed >= governanceMaxProposalsPerRun {
			break
		}
		kb := &kbs[i]
		articles := kbArticles(ctx, kb.ID)
		if len(articles) == 0 {
			continue
		}
		inTree := tocArticleSet(ctx, kb.ID)

		file := func(article *core.WikiArticle, pType, reason string, evidence util.MapStr) {
			if stats.filed >= governanceMaxProposalsPerRun {
				return
			}
			key := article.ID + "|" + pType
			if open[key] {
				return
			}
			if fileProposal(kb, article, pType, reason, evidence) {
				open[key] = true
				stats.filed++
			}
		}

		for j := range articles {
			article := &articles[j]

			// pending freshness output: a newer auto-updated version awaits review
			if pending := pendingAutoUpdate(ctx, article); pending != 0 {
				pType, reason := core.WikiGovernanceStale,
					fmt.Sprintf("cited sources changed; auto-updated version %d is awaiting review", pending)
				if llm != nil {
					if contradicts, why := conflictJudge(ctx, llm, article, pending); contradicts {
						pType, reason = core.WikiGovernanceConflict,
							fmt.Sprintf("regenerated version %d contradicts live content: %s", pending, why)
						stats.conflicts++
					}
				}
				file(article, pType, reason, util.MapStr{"pending_version": pending})
			}

			// published but thin: low confidence or no citation backlinks
			if article.Status == core.WikiArticlePublished &&
				(article.Confidence == "low" || len(article.Sources) == 0) {
				why := "no citation backlinks"
				if article.Confidence == "low" {
					why = "generation confidence is low"
				}
				file(article, core.WikiGovernanceLowQuality,
					fmt.Sprintf("published page is weak: %s", why),
					util.MapStr{"confidence": article.Confidence, "sources": len(article.Sources)})
			}

			// not reachable through the KB tree
			if !inTree[article.ID] {
				file(article, core.WikiGovernanceOrphan,
					"article is not referenced by any node of the knowledge-base tree",
					nil)
			}
		}

		// duplicates need the LLM judge; candidate pairs come from title overlap
		if llm != nil {
			for _, pair := range duplicateCandidates(articles) {
				if stats.filed >= governanceMaxProposalsPerRun {
					break
				}
				dup, reason, err := duplicateJudge(ctx, llm, pair)
				if err != nil {
					log.Debugf("wiki: governance duplicate judge failed: %v", err)
					continue
				}
				if !dup {
					continue
				}
				key := pair[0].ID + "|" + core.WikiGovernanceDuplicate
				if open[key] {
					continue
				}
				if fileProposal(kb, &pair[0], core.WikiGovernanceDuplicate, reason,
					util.MapStr{"duplicate_of": util.MapStr{"id": pair[1].ID, "title": pair[1].Title}}) {
					open[key] = true
					stats.filed++
					stats.duplicates++
				}
			}
		}
	}
	return stats
}

/* ---------------- detectors ---------------- */

// duplicateCandidates returns same-KB title pairs that share enough tokens
// to be worth an LLM look (cheap pre-filter, cap per run).
func duplicateCandidates(articles []core.WikiArticle) [][2]core.WikiArticle {
	tokensOf := func(title string) map[string]bool {
		out := map[string]bool{}
		for _, field := range strings.FieldsFunc(strings.ToLower(title), func(r rune) bool {
			return r != '_' && r != '-' && !('a' <= r && r <= 'z') && !('0' <= r && r <= '9')
		}) {
			if len(field) > 1 {
				out[field] = true
			}
		}
		return out
	}

	var pairs [][2]core.WikiArticle
	for i := range articles {
		if len(pairs) >= governanceMaxDuplicatePairs {
			break
		}
		ti := tokensOf(articles[i].Title)
		if len(ti) == 0 {
			continue
		}
		for j := i + 1; j < len(articles); j++ {
			tj := tokensOf(articles[j].Title)
			shared := 0
			for token := range tj {
				if ti[token] {
					shared++
				}
			}
			// two shared tokens or one side fully contained
			if shared >= 2 || (shared > 0 && (shared == len(ti) || shared == len(tj))) {
				pairs = append(pairs, [2]core.WikiArticle{articles[i], articles[j]})
				break
			}
		}
	}
	return pairs
}

// duplicateJudge asks the model whether two pages cover the same subject.
func duplicateJudge(ctx context.Context, llm llms.Model, pair [2]core.WikiArticle) (bool, string, error) {
	snippet := func(a core.WikiArticle) string {
		s := a.Summary
		if s == "" {
			s = a.Content
		}
		if len(s) > 500 {
			s = s[:500]
		}
		return s
	}
	prompt := fmt.Sprintf(
		"Two wiki pages from the same knowledge base may cover the same subject.\n\n"+
			"Page A title: %s\nPage A summary: %s\n\n"+
			"Page B title: %s\nPage B summary: %s\n\n"+
			"Decide whether they are near-duplicates that should be merged (same subject), "+
			"or complementary pages about related but distinct subjects.\n"+
			`Return ONLY JSON: {"duplicate":true/false,"reason":"short justification"}`,
		pair[0].Title, snippet(pair[0]), pair[1].Title, snippet(pair[1]))
	raw, err := callLLM(ctx, llm, "You are a knowledge-governance judge. Answer with JSON only.", prompt, &llmUsage{})
	if err != nil {
		return false, "", err
	}
	var verdict struct {
		Duplicate bool   `json:"duplicate"`
		Reason    string `json:"reason"`
	}
	if err := decodeLLMJSON(raw, &verdict); err != nil {
		return false, "", err
	}
	if !verdict.Duplicate {
		return false, "", nil
	}
	reason := verdict.Reason
	if reason == "" {
		reason = fmt.Sprintf("covers the same subject as %q", pair[1].Title)
	}
	return true, reason, nil
}

// conflictJudge asks whether a pending regenerated version contradicts the
// live page (a fact change) or merely extends it.
func conflictJudge(ctx context.Context, llm llms.Model, article *core.WikiArticle, version int) (bool, string) {
	live := article.Content
	if len(live) > 1500 {
		live = live[:1500]
	}
	updated := versionContent(ctx, article.ID, version)
	if updated == "" {
		return false, ""
	}
	if len(updated) > 1500 {
		updated = updated[:1500]
	}
	prompt := fmt.Sprintf(
		"A wiki page has a regenerated draft version awaiting review.\n\n"+
			"=== CURRENT LIVE CONTENT ===\n%s\n\n"+
			"=== REGENERATED DRAFT ===\n%s\n\n"+
			"Does the draft CONTRADICT the live content (states different facts about the same thing), "+
			"or does it merely extend/update it consistently?\n"+
			`Return ONLY JSON: {"conflict":true/false,"reason":"short justification"}`,
		live, updated)
	raw, err := callLLM(ctx, llm, "You are a knowledge-governance judge. Answer with JSON only.", prompt, &llmUsage{})
	if err != nil {
		return false, ""
	}
	var verdict struct {
		Conflict bool   `json:"conflict"`
		Reason   string `json:"reason"`
	}
	if err := decodeLLMJSON(raw, &verdict); err != nil {
		return false, ""
	}
	return verdict.Conflict, verdict.Reason
}

// pendingAutoUpdate returns the newest auto-updated version number that is
// newer than the live article's last update, or 0 when nothing is pending.
func pendingAutoUpdate(ctx context.Context, article *core.WikiArticle) int {
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ormCtx, &core.WikiVersion{})

	builder := orm.NewQuery().Size(1).
		Filter(orm.TermQuery("article_id", article.ID), orm.TermQuery("change_type", core.WikiChangeAutoUpdated)).
		SortBy(orm.Sort{Field: "version", SortType: orm.DESC})
	res, err := orm.SearchV2(ormCtx, builder)
	if err != nil {
		return 0
	}
	versions, _, err := elastic.DecodeHits[core.WikiVersion](res)
	if err != nil || len(versions) == 0 {
		return 0
	}
	v := versions[0]
	if article.Updated != nil && v.Created != nil && v.Created.Before(*article.Updated) {
		return 0 // already folded into the live content
	}
	return v.Version
}

// versionContent returns one version's snapshot body (empty when missing).
func versionContent(ctx context.Context, articleID string, version int) string {
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ormCtx, &core.WikiVersion{})

	builder := orm.NewQuery().Size(1).
		Filter(orm.TermQuery("article_id", articleID), orm.TermQuery("version", version))
	res, err := orm.SearchV2(ormCtx, builder)
	if err != nil {
		return ""
	}
	versions, _, err := elastic.DecodeHits[core.WikiVersion](res)
	if err != nil || len(versions) == 0 {
		return ""
	}
	return versions[0].Content
}

/* ---------------- data access ---------------- */

func governanceKBs(ctx context.Context) []core.WikiKnowledgeBase {
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ormCtx, &core.WikiKnowledgeBase{})
	res, err := orm.SearchV2(ormCtx, orm.NewQuery().Size(governanceMaxKBs))
	if err != nil {
		log.Warnf("wiki: governance kb scan failed: %v", err)
		return nil
	}
	kbs, _, err := elastic.DecodeHits[core.WikiKnowledgeBase](res)
	if err != nil {
		return nil
	}
	return kbs
}

func kbArticles(ctx context.Context, kbID string) []core.WikiArticle {
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ormCtx, &core.WikiArticle{})
	res, err := orm.SearchV2(ormCtx, orm.NewQuery().Size(governanceMaxArticlesPerKB).
		Filter(orm.TermQuery("kb_id", kbID)))
	if err != nil {
		log.Warnf("wiki: governance article scan failed: %v", err)
		return nil
	}
	articles, _, err := elastic.DecodeHits[core.WikiArticle](res)
	if err != nil {
		return nil
	}
	return articles
}

// tocArticleSet walks the KB tree and collects every article id reachable
// from it (orphan = present as article, absent from this set).
func tocArticleSet(ctx context.Context, kbID string) map[string]bool {
	out := map[string]bool{}
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ormCtx, &core.WikiToc{})
	res, err := orm.SearchV2(ormCtx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("kb_id", kbID)))
	if err != nil {
		return out
	}
	tocs, _, err := elastic.DecodeHits[core.WikiToc](res)
	if err != nil || len(tocs) == 0 {
		return out
	}
	var walk func(nodes []core.WikiTocNode)
	walk = func(nodes []core.WikiTocNode) {
		for _, node := range nodes {
			if node.ArticleID != "" {
				out[node.ArticleID] = true
			}
			walk(node.Children)
		}
	}
	walk(tocs[0].Nodes)
	return out
}

// openProposalKeys loads "articleID|type" for every open proposal so the
// sweep stays idempotent across runs.
func openProposalKeys(ctx context.Context) map[string]bool {
	out := map[string]bool{}
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ormCtx, &core.WikiGovernanceProposal{})
	res, err := orm.SearchV2(ormCtx, orm.NewQuery().Size(10000).
		Filter(orm.TermQuery("status", core.WikiGovernanceOpen)))
	if err != nil {
		return out
	}
	proposals, _, err := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	if err != nil {
		return out
	}
	for _, p := range proposals {
		out[p.ArticleID+"|"+p.Type] = true
	}
	return out
}

// fileProposal persists one proposal and notifies the KB owner; best-effort
// beyond the create itself.
func fileProposal(kb *core.WikiKnowledgeBase, article *core.WikiArticle, pType, reason string, evidence util.MapStr) bool {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})

	proposal := &core.WikiGovernanceProposal{
		KbID:         kb.ID,
		ArticleID:    article.ID,
		ArticleTitle: article.Title,
		Type:         pType,
		Status:       core.WikiGovernanceOpen,
		Reason:       reason,
		Evidence:     evidence,
	}
	if err := orm.Create(ctx, proposal); err != nil {
		log.Warnf("wiki: governance proposal create failed: %v", err)
		return false
	}
	notifyOwner(kb.GetOwnerID(), "article", article.ID, "governance",
		fmt.Sprintf("Governance: %s — %q (%s)", governanceTypeLabel(pType), article.Title, reason))
	return true
}

func governanceTypeLabel(pType string) string {
	switch pType {
	case core.WikiGovernanceStale:
		return "stale page"
	case core.WikiGovernanceDuplicate:
		return "possible duplicate"
	case core.WikiGovernanceConflict:
		return "conflicting update"
	case core.WikiGovernanceLowQuality:
		return "low quality"
	case core.WikiGovernanceOrphan:
		return "orphaned page"
	}
	return pType
}

/* ---------------- PUT /wiki/governance/:id/status ---------------- */

// governanceStatusTransitions is deliberately tiny: a proposal starts open
// and ends resolved or dismissed. Re-opening would fight the scanner's own
// idempotency key (same article+type stays open), so it is not offered.
var governanceStatusTransitions = map[string][]string{
	core.WikiGovernanceOpen: {core.WikiGovernanceResolved, core.WikiGovernanceDismissed},
}

func validateGovernanceStatus(status string) error {
	switch status {
	case core.WikiGovernanceOpen, core.WikiGovernanceResolved, core.WikiGovernanceDismissed:
		return nil
	}
	return fmt.Errorf("invalid governance status: %s", status)
}

// updateGovernanceStatus is the human gate of the queue: marking resolved or
// dismissed records who decided. Applying the actual fix (accepting a
// regenerated version, merging a duplicate, re-treeing an orphan) happens on
// the article face — this endpoint only retires the proposal (D1).
func (h *APIHandler) updateGovernanceStatus(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	proposalID := ps.ByName("id")

	var body struct {
		Status string `json:"status"`
	}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.Error400(w, err.Error())
		return
	}
	if err := validateGovernanceStatus(body.Status); err != nil {
		h.Error400(w, err.Error())
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
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
	if proposal.Status == body.Status {
		h.WriteUpdatedOKJSON(w, proposal.ID)
		return
	}
	if !util.AnyInArrayEquals(governanceStatusTransitions[proposal.Status], body.Status) {
		h.Error400(w, fmt.Sprintf("cannot transition governance status from %s to %s", proposal.Status, body.Status))
		return
	}

	proposal.Status = body.Status
	if user, err := security.GetUserFromRequest(req); err == nil && user != nil && user.UserID != "" {
		proposal.ResolvedBy = user.UserID
	}
	now := time.Now()
	proposal.ResolvedAt = &now
	if err := orm.Update(ctx, &proposal); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteUpdatedOKJSON(w, proposal.ID)
}
