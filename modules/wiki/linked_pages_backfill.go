/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"regexp"
	"strings"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
)

// One idempotent startup pass over existing wiki articles, upgrading
// content written by earlier formats:
//
//   - entity draft pages drafted before the linked 原文 citation existed
//     get their body regenerated from the entity record (claims now carry
//     a link that opens the source document with the excerpt highlighted)
//   - pages whose content carries [[...]] wikilinks but no LinkedPages get
//     them resolved (the graph reads LinkedPages)
//   - legacy plain-text citations （来源:DOC_ID） become live source links
func scheduleBackfillLinkedPages() {
	global.RegisterFuncAfterSetup(func() {
		backfillLinkedPages()
	})
}

// legacyCitationPattern matches the old plain-text citation form so the
// backfill can upgrade it to a live source link.
var legacyCitationPattern = regexp.MustCompile(`（来源:([0-9a-f]{16,40})）`)

func backfillLinkedPages() {
	ctx := orm.NewContextWithParent(context.Background())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})

	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000))
	if err != nil {
		log.Warnf("wiki: linked-pages backfill search failed: %v", err)
		return
	}
	articles, _, derr := elastic.DecodeHits[core.WikiArticle](res)
	if derr != nil {
		log.Warnf("wiki: linked-pages backfill decode failed: %v", derr)
		return
	}

	// entity records for page regeneration
	entityByID := map[string]core.WikiEntity{}
	eids := make([]string, 0, len(articles))
	for i := range articles {
		if articles[i].EntityID != "" && needsCitationUpgrade(&articles[i]) {
			eids = append(eids, articles[i].EntityID)
		}
	}
	if len(eids) > 0 {
		orm.WithModel(ctx, &core.WikiEntity{})
		if res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
			Filter(orm.TermsQuery("id", eids))); err == nil && res != nil {
			if entities, _, dd := elastic.DecodeHits[core.WikiEntity](res); dd == nil {
				for _, e := range entities {
					entityByID[e.ID] = e
				}
			}
		}
	}

	updated := 0
	for i := range articles {
		a := &articles[i]
		changed := false

		// entity page citation upgrade: regenerate the body so every claim
		// carries a link back to its source block
		if a.EntityID != "" && needsCitationUpgrade(a) {
			if entity, ok := entityByID[a.EntityID]; ok {
				a.Content = DraftEntityContent(&entity)
				changed = true
			}
		}

		// wikilink backfill: content carries [[...]] but no linked pages
		if a.Content != "" && len(a.LinkedPages) == 0 && strings.Contains(a.Content, "[[") {
			resolveLinkedPages(a)
			if len(a.LinkedPages) > 0 {
				changed = true
			}
		}

		// citation upgrade: legacy pages cite sources as bare （来源:DOC_ID）
		// text — rewrite to live links that open the source document
		if a.Content != "" && legacyCitationPattern.MatchString(a.Content) {
			a.Content = legacyCitationPattern.ReplaceAllString(a.Content, "（来源:[$1](#/preview/document/$1)）")
			changed = true
		}

		if !changed {
			continue
		}
		if err := orm.Update(ctx, a); err != nil {
			log.Debugf("wiki: linked-pages backfill update failed for [%s]: %v", a.Title, err)
			continue
		}
		if a.EntityID != "" {
			syncEntityMentions(a)
		}
		updated++
	}
	if updated > 0 {
		log.Infof("wiki: backfilled linked pages / upgraded citations for %d articles", updated)
	}
}

// needsCitationUpgrade reports whether the page body predates the linked
// 原文 citation (or has no body worth keeping).
func needsCitationUpgrade(a *core.WikiArticle) bool {
	if strings.Contains(a.Content, "[原文](") {
		return false
	}
	// regenerate anything entity-backed whose body is missing, thin, or
	// written by an older citation format
	if strings.TrimSpace(a.Content) == "" {
		return true
	}
	if legacyCitationPattern.MatchString(a.Content) {
		return true
	}
	// old DraftEntityContent bodies: 来源主张 claims without links
	return strings.Contains(a.Content, "## 来源主张") && !strings.Contains(a.Content, "[原文](")
}
