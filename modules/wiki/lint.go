/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"fmt"
	"strings"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
)

// Save-time lint (wiki-skills doctrine: lint per page, advisory only).
// persistLinkedPages already runs on every article write path (CRUD hooks,
// AI generate/edit, governance applies), so the warnings ride the same
// re-save instead of adding another write.

// minReadableContent is the thin-content floor in runes: under this the
// page reads as a stub, not knowledge.
const minReadableContent = 200

// computeLintWarnings inspects one article. Two resolutions hit the store:
// dead-link candidates are checked against article titles in the same KB
// (entity resolution already ran in resolveLinkedPages).
func computeLintWarnings(article *core.WikiArticle) []core.WikiLintWarning {
	warnings := make([]core.WikiLintWarning, 0, 4)

	// dead links: wikilinks that resolved to no entity AND match no
	// article title in this KB. LinkedPages already dedupes by name.
	unresolved := make([]core.WikiLinkedPage, 0, 2)
	for _, link := range article.LinkedPages {
		if link.EntityID != "" {
			continue
		}
		if articleTitleExists(article.KbID, link.Name) {
			continue
		}
		unresolved = append(unresolved, link)
	}
	if len(unresolved) > 0 {
		names := make([]string, 0, len(unresolved))
		for _, link := range unresolved {
			names = append(names, link.Name)
		}
		warnings = append(warnings, core.WikiLintWarning{
			Code:    core.WikiLintDeadLink,
			Field:   "content",
			Message: fmt.Sprintf("以下引用无法解析: %s", strings.Join(names, ", ")),
			Detail:  strings.Join(names, ","),
		})
	}

	if len([]rune(strings.TrimSpace(article.Content))) < minReadableContent {
		warnings = append(warnings, core.WikiLintWarning{
			Code:    core.WikiLintThinContent,
			Field:   "content",
			Message: fmt.Sprintf("正文不足 %d 字，页面还接近草稿", minReadableContent),
		})
	}

	// concept/topic pages are compiled knowledge — zero citations means
	// nothing backs them (entity/map pages legitimately have none)
	if (article.PageType == core.WikiPageTypeConcept || article.PageType == core.WikiPageTypeTopic) && len(article.Sources) == 0 {
		warnings = append(warnings, core.WikiLintWarning{
			Code:    core.WikiLintNoSources,
			Field:   "sources",
			Message: "概念/专题页没有任何引用来源，进入检索前应补充证据",
		})
	}

	// a skill without provenance is an opinion — skills must trace back to
	// the pattern pages (or at least wikilinks) that motivated them
	if article.PageType == core.WikiPageTypeSkill && len(article.Sources) == 0 && len(article.LinkedPages) == 0 {
		warnings = append(warnings, core.WikiLintWarning{
			Code:    core.WikiLintMissingProvenance,
			Field:   "sources",
			Message: "技能页没有出处（pattern 页引用或 wiki 链接）——无出处的技能只是观点",
		})
	}

	return warnings
}

// articleTitleExists reports whether an article with this exact title
// lives in the KB (case-sensitive keyword match, size-capped).
func articleTitleExists(kbID, title string) bool {
	if kbID == "" || title == "" {
		return false
	}
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})
	builder := orm.NewQuery()
	builder.Size(1)
	builder.Filter(orm.TermQuery("kb_id", kbID), orm.TermQuery("title", title))
	res, err := orm.SearchV2(ctx, builder)
	if err != nil || res == nil {
		return false
	}
	_, total, err := elastic.DecodeHits[core.WikiArticle](res)
	if err != nil {
		return false
	}
	return total > 0
}
