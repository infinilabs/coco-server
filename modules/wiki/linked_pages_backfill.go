/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"strings"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
)

// pages created through paths that bypass the crud save (batch proposal
// applies, early pipeline writes) carry content with wikilinks but no
// LinkedPages — the graph reads LinkedPages, so those pages render as
// orphans. One idempotent startup pass re-resolves every article whose
// content contains wikilinks but carries no linked pages (and refreshes
// the mentions edges of entity pages while at it).
func scheduleBackfillLinkedPages() {
	global.RegisterFuncAfterSetup(func() {
		backfillLinkedPages()
	})
}

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

	updated := 0
	for i := range articles {
		a := &articles[i]
		if a.Content == "" || len(a.LinkedPages) > 0 || !strings.Contains(a.Content, "[[") {
			continue
		}
		resolveLinkedPages(a)
		if len(a.LinkedPages) == 0 {
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
		log.Infof("wiki: backfilled linked pages for %d articles", updated)
	}
}
