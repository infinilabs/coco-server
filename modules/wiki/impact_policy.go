/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"fmt"
	"time"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
)

// Impact-driven retention policy (LLM-wiki P3, the skill-impact.md loop):
// usage counts earn pages protection from cleanup passes, and long-idle,
// never-cited pages become archive candidates — the counterweight to the
// "wiki never rolls back" growth problem the paper itself lists as a
// limitation.

// impactProtectHits: cited in at least this many answer contexts, a page
// is doing its job — consolidate must not absorb it.
const impactProtectHits = 3

// impactArchive bounds: a published page older than MinAge that has been
// cited fewer than MaxHits times and untouched for Idle becomes an archive
// candidate. New pages get their grace period before the policy judges.
const (
	impactArchiveMinAge  = 90 * 24 * time.Hour
	impactArchiveIdle    = 120 * 24 * time.Hour
	impactArchiveMaxHits = 2
)

// impactProtected reports whether real usage shields the page from
// cleanup passes (consolidate absorption).
func impactProtected(a *core.WikiArticle) bool {
	return a != nil && a.Impact != nil && a.Impact.AnswerHits >= impactProtectHits
}

// impactArchiveCandidate reports a published page the impact policy would
// retire: old enough, barely (never) cited, long untouched. Returns a
// human-readable reason for the proposal.
func impactArchiveCandidate(a *core.WikiArticle, now time.Time) (bool, string) {
	if a == nil || a.Status != core.WikiArticlePublished {
		return false, ""
	}
	if impactProtected(a) {
		return false, ""
	}
	hits := 0
	if a.Impact != nil {
		hits = a.Impact.AnswerHits
	}
	if hits >= impactArchiveMaxHits {
		return false, ""
	}
	if a.Created == nil || now.Sub(*a.Created) < impactArchiveMinAge {
		return false, ""
	}
	idleSince := a.Updated
	if a.Updated == nil {
		idleSince = a.Created
	}
	if idleSince == nil || now.Sub(*idleSince) < impactArchiveIdle {
		return false, ""
	}
	usage := "从未被回答引用"
	if hits > 0 {
		usage = fmt.Sprintf("仅被回答引用 %d 次", hits)
	}
	return true, fmt.Sprintf("%s，且 %d 天没有更新——可归档给活跃知识让位", usage,
		int(now.Sub(*idleSince).Hours()/24))
}

// applyImpactArchiveProposal retires the page: status → archived, the
// retrieval projection dies with it. Content and versions stay — archive,
// not delete.
func applyImpactArchiveProposal(ctx *orm.Context, p *core.WikiGovernanceProposal) (string, error) {
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
	if article.Status == core.WikiArticleArchived {
		return "已处于归档状态", nil
	}
	article.Status = core.WikiArticleArchived
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	if err := orm.Update(ctx, &article); err != nil {
		return "", err
	}
	SyncArticleProjection(&article)
	return "已归档", nil
}
