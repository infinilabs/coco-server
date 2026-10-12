/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
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
		a.Content = DraftEntityContentFor(ctx, a.KbID, &entity)
		if err := orm.Update(ctx, a); err != nil {
			log.Debugf("wiki: draft content backfill update failed for [%s]: %v", a.Title, err)
			continue
		}
		// the backfill is the page's first real content — snapshot it so the
		// history starts at v1 instead of an empty state
		snapshotPipelineWrite(a, fmt.Sprintf("草稿内容回填，依据 %d 条来源主张", len(entity.Sources)))
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
	return composeEntityContent(entity, nil, nil)
}

// DraftEntityContentFor honors the entity type's declared page layout
// (ontology schema: kb override → tenant default) — a 客户 type can declare
// 客户档案/关键联系人/相关文档 sections and every generated page follows.
// Types without a declared layout fall back to the canonical structure.
func DraftEntityContentFor(ctx *orm.Context, kbID string, entity *core.WikiEntity) string {
	var layout []OntologyPageSection
	if schema := ResolveOntologySchema(ctx.Context, kbID); schema != nil {
		for i := range schema.EntityTypes {
			if strings.EqualFold(schema.EntityTypes[i].Name, entity.Type) {
				layout = schema.EntityTypes[i].PageSections
				break
			}
		}
	}
	if len(layout) == 0 {
		return DraftEntityContent(entity)
	}
	// relation sections need target display names — batch-resolve once
	return composeEntityContent(entity, layout, relationTargetNames(ctx, entity))
}

// relationTargetNames resolves the display names of every relation target
// referenced by the entity's layout sections.
func relationTargetNames(ctx *orm.Context, entity *core.WikiEntity) map[string]string {
	ids := make([]string, 0, len(entity.Relations))
	seen := map[string]bool{}
	for _, r := range entity.Relations {
		if r.TargetID != "" && !seen[r.TargetID] {
			seen[r.TargetID] = true
			ids = append(ids, r.TargetID)
		}
	}
	names := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return names
	}
	orm.WithModel(ctx, &core.WikiEntity{})
	if res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
		Filter(orm.TermsQuery("id", ids))); err == nil && res != nil {
		if targets, _, derr := elastic.DecodeHits[core.WikiEntity](res); derr == nil {
			for _, t := range targets {
				names[t.ID] = t.Name
			}
		}
	}
	return names
}

// composeEntityContent renders the page: the provenance intro, then the
// declared sections in order (or the canonical 来源引用/别名 pair).
func composeEntityContent(entity *core.WikiEntity, layout []OntologyPageSection, targetNames map[string]string) string {
	typeLabel := entity.Type
	if typeLabel == "" {
		typeLabel = "实体"
	}
	intro := fmt.Sprintf("**%s** 是知识管道从 %d 份来源资料中识别出的「%s」类实体，以下主张均附来源出处，等待人工审阅完善。\n\n",
		entity.Name, len(entity.Sources), typeLabel)
	return composeEntityContentWithIntro(entity, layout, targetNames, intro)
}

// composeEntityContentWithIntro renders the declared sections (or the
// canonical 来源引用/别名 pair) after a caller-supplied intro line — the
// compile pipeline reuses the same layout with its own preamble.
func composeEntityContentWithIntro(entity *core.WikiEntity, layout []OntologyPageSection, targetNames map[string]string, intro string) string {
	var sb strings.Builder
	sb.WriteString(intro)

	writeSources := func() {
		if len(entity.Sources) == 0 {
			return
		}
		sb.WriteString("## 来源引用\n\n")
		shown := 0
		for _, s2 := range entity.Sources {
			claim := strings.TrimSpace(s2.Excerpt)
			if claim == "" {
				claim = strings.TrimSpace(s2.Title)
			}
			if claim == "" {
				continue
			}
			title := strings.TrimSpace(s2.Title)
			if title == "" {
				title = s2.DocID
			}
			cite := title
			if s2.DocID != "" {
				excerpt := strings.TrimSpace(s2.Excerpt)
				if len([]rune(excerpt)) > 120 {
					excerpt = string([]rune(excerpt)[:120]) + "…"
				}
				cite = fmt.Sprintf("[%s](#/preview/document/%s?q=%s \"%s\")", title, s2.DocID, url.QueryEscape(excerpt), excerpt)
			}
			fmt.Fprintf(&sb, "- %s（来源：%s）\n", claim, cite)
			shown++
		}
		if shown == 0 {
			sb.WriteString("-（来源摘录为空，等待补充）\n")
		}
	}
	writeAliases := func() {
		if len(entity.Aliases) > 0 {
			sb.WriteString("\n## 别名\n\n" + strings.Join(entity.Aliases, "、") + "\n")
		}
	}
	writeProperties := func(title string) {
		sb.WriteString("\n## " + title + "\n\n")
		written := 0
		for k, v := range entity.Properties {
			vs := strings.TrimSpace(util.ToString(v))
			if vs == "" || vs == "<nil>" {
				continue
			}
			fmt.Fprintf(&sb, "- **%s**: %s\n", k, vs)
			written++
		}
		if written == 0 {
			sb.WriteString("（暂无属性，等待补充）\n")
		}
	}
	writeRelations := func(title, relName string) {
		sb.WriteString("\n## " + title + "\n\n")
		written := 0
		for _, r := range entity.Relations {
			if r.Relation != relName {
				continue
			}
			targetName := targetNames[r.TargetID]
			if targetName == "" {
				targetName = r.TargetID
			}
			targetType := r.TargetType
			if targetType == "" {
				targetType = "实体"
			}
			fmt.Fprintf(&sb, "- [[%s:%s]]（%s）\n", targetType, targetName, r.Relation)
			written++
		}
		if written == 0 {
			sb.WriteString("（暂无关联，等待补充）\n")
		}
	}

	if len(layout) == 0 {
		writeSources()
		writeAliases()
		return sb.String()
	}

	for _, sec := range layout {
		title := sec.Title
		switch {
		case sec.Key == "sources":
			writeSources()
		case sec.Key == "aliases":
			writeAliases()
		case sec.Key == "properties":
			writeProperties(title)
		case strings.HasPrefix(sec.Key, "relations:"):
			writeRelations(title, strings.TrimPrefix(sec.Key, "relations:"))
		default:
			// declared-but-unknown section: an explicit slot for human editing
			sb.WriteString("\n## " + title + "\n\n（待补充）\n")
		}
	}
	return sb.String()
}

// wikiEntityByName resolves an entity by its exact name (ontology type for
// the layout resolution lives on it).
func wikiEntityByName(ctx *orm.Context, name string) *core.WikiEntity {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	ctx2 := orm.NewContextWithParent(ctx)
	ctx2.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx2, &core.WikiEntity{})
	res, err := orm.SearchV2(ctx2, orm.NewQuery().Size(1).Filter(orm.TermQuery("name", name)))
	if err != nil {
		return nil
	}
	hits, _, err := elastic.DecodeHits[core.WikiEntity](res)
	if err != nil || len(hits) == 0 {
		return nil
	}
	return &hits[0]
}
