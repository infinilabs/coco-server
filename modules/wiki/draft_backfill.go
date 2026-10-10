/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
)

// draft entity pages created before the pipeline wrote bodies open as bare
// titles — the user opens the page and sees a name with nothing under it.
// One idempotent pass at startup fills each still-empty draft with its
// entity's provenance (the same shape new drafts get): a provenance banner,
// the per-source claims, and aliases. Human-edited pages are never touched.
func scheduleBackfillDraftEntityContent() {
	global.RegisterFuncAfterSetup(func() {
		backfillDraftEntityContent()
	})
}

func backfillDraftEntityContent() {
	ctx := orm.NewContextWithParent(context.Background())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})

	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
		Filter(orm.TermQuery("page_type", core.WikiPageTypeEntity),
			orm.TermQuery("status", core.WikiArticleDraft)))
	if err != nil {
		log.Warnf("wiki: draft content backfill search failed: %v", err)
		return
	}
	articles, _, derr := elastic.DecodeHits[core.WikiArticle](res)
	if derr != nil {
		log.Warnf("wiki: draft content backfill decode failed: %v", derr)
		return
	}

	ids := make([]string, 0, len(articles))
	for _, a := range articles {
		if strings.TrimSpace(a.Content) == "" && a.EntityID != "" {
			ids = append(ids, a.EntityID)
		}
	}
	if len(ids) == 0 {
		return
	}
	entityByID := map[string]core.WikiEntity{}
	orm.WithModel(ctx, &core.WikiEntity{})
	if res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
		Filter(orm.TermsQuery("id", ids))); err == nil && res != nil {
		if entities, _, derr := elastic.DecodeHits[core.WikiEntity](res); derr == nil {
			for _, e := range entities {
				entityByID[e.ID] = e
			}
		}
	}

	updated := 0
	for i := range articles {
		a := &articles[i]
		if strings.TrimSpace(a.Content) != "" || a.EntityID == "" {
			continue
		}
		entity, ok := entityByID[a.EntityID]
		if !ok {
			continue
		}
		a.Content = DraftEntityContent(&entity)
		if err := orm.Update(ctx, a); err != nil {
			log.Debugf("wiki: draft content backfill update failed for [%s]: %v", a.Title, err)
			continue
		}
		updated++
	}
	if updated > 0 {
		log.Infof("wiki: backfilled %d draft entity pages with provenance content", updated)
	}
}

// DraftEntityContent builds a draft page body from the entity's own
// provenance: every recorded source contributes its evidence excerpt, so
// the page opens with quoted material a human can verify instead of a bare
// title. Shared by the extraction pipeline (new drafts) and the startup
// backfill (legacy stubs).
func DraftEntityContent(entity *core.WikiEntity) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "> 本页由知识管道依据 %d 条来源自动起草，内容以来源原文为准，等待人工审阅完善。\n\n", len(entity.Sources))
	if entity.Type != "" {
		fmt.Fprintf(&sb, "**%s** 是管道识别出的「%s」类实体。\n\n", entity.Name, entity.Type)
	}
	if len(entity.Sources) > 0 {
		sb.WriteString("## 来源主张\n\n")
		shown := 0
		for _, s := range entity.Sources {
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
			fmt.Fprintf(&sb, "%d. **%s** — %s\n", shown+1, title, line)
			shown++
		}
		if shown == 0 {
			sb.WriteString("（来源摘录为空，等待补充）\n")
		}
	}
	if len(entity.Aliases) > 0 {
		sb.WriteString("\n## 别名\n\n" + strings.Join(entity.Aliases, "、") + "\n")
	}
	return sb.String()
}
