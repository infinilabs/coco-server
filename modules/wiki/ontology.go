/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"regexp"
	"strings"

	log "github.com/cihub/seelog"

	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
)

// wikilink syntax: [[type:name]] or [[name]] (design doc §3.3); the type
// prefix routes rendering, resolution is by name/alias regardless of type.
var wikilinkPattern = regexp.MustCompile(`\[\[([^\[\]]+)\]\]`)

func parseWikilinks(content string) []core.WikiLinkedPage {
	seen := map[string]bool{}
	pages := make([]core.WikiLinkedPage, 0, 8)
	for _, match := range wikilinkPattern.FindAllStringSubmatch(content, -1) {
		inner := strings.TrimSpace(match[1])
		if inner == "" {
			continue
		}
		if strings.HasPrefix(inner, ":") {
			continue // no type before the colon
		}
		link := core.WikiLinkedPage{Name: inner}
		if idx := strings.Index(inner, ":"); idx > 0 {
			link.Type = strings.TrimSpace(inner[:idx])
			link.Name = strings.TrimSpace(inner[idx+1:])
		}
		if link.Name == "" || seen[link.Name] {
			continue
		}
		seen[link.Name] = true
		pages = append(pages, link)
	}
	return pages
}

// persistLinkedPages re-resolves the article's wikilinks after a save and
// stores them on the article (B3), then maintains the entity mentions
// edges. Runs from PostCreate/PostUpdate where the persisted object (full
// content) is available — the partial-update Prepare hooks only see the id.
func persistLinkedPages(article *core.WikiArticle) {
	resolveLinkedPages(article)
	ctx := orm.NewContext()
	// Update internally re-reads the object (GetPrevObject) through the
	// OpGet hook, so both direct flags are required in a userless context.
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})
	if err := orm.Update(ctx, article); err != nil {
		log.Warnf("wiki: failed to persist linked_pages for article %s: %v", article.ID, err)
	}
	syncEntityMentions(article)
}

// resolveLinkedPages parses the article content and resolves each wikilink
// against existing entities by name then alias (B3 redundancy).
func resolveLinkedPages(article *core.WikiArticle) {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)

	pages := parseWikilinks(article.Content)
	for i := range pages {
		if entity := findEntityByNameOrAlias(ctx, pages[i].Name); entity != nil {
			pages[i].EntityID = entity.ID
		}
	}
	article.LinkedPages = pages
}

// syncEntityMentions maintains the mentions edges of an entity page: on
// every save the edges sourced from this article are rebuilt from the
// resolved wikilinks (replace-by-provenance, no duplicates).
func syncEntityMentions(article *core.WikiArticle) {
	if article.EntityID == "" {
		return
	}
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)

	var entity core.WikiEntity
	entity.SetID(article.EntityID)
	orm.WithModel(ctx, &core.WikiEntity{})
	exists, err := orm.GetV2(ctx, &entity)
	if err != nil || !exists {
		return
	}

	kept := entity.Relations[:0]
	for _, r := range entity.Relations {
		if r.Provenance != article.ID || r.Relation != core.WikiRelationMentions {
			kept = append(kept, r)
		}
	}
	for _, page := range article.LinkedPages {
		if page.EntityID == "" || page.EntityID == entity.ID {
			continue
		}
		var target core.WikiEntity
		target.SetID(page.EntityID)
		exists, err := orm.GetV2(ctx, &target)
		if err != nil || !exists {
			continue
		}
		kept = append(kept, core.WikiEntityRelation{
			TargetID:   target.ID,
			TargetType: target.Type,
			Relation:   core.WikiRelationMentions,
			Provenance: article.ID,
		})
	}
	entity.Relations = kept
	if err := orm.Update(ctx, &entity); err != nil {
		log.Warnf("wiki: failed to sync mentions edges for entity %s: %v", entity.ID, err)
	}
}

// syncEntityStatus mirrors the article workflow onto the linked entity
// (design doc D1/B4): review/publish of an entity page promotes the entity;
// draft/archive leave the entity untouched.
func syncEntityStatus(article *core.WikiArticle, status string) {
	if article.EntityID == "" || article.PageType != core.WikiPageTypeEntity {
		return
	}
	var entityStatus string
	switch status {
	case core.WikiArticleReviewed:
		entityStatus = core.WikiEntityReviewed
	case core.WikiArticlePublished:
		entityStatus = core.WikiEntityPublished
	default:
		return
	}

	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)

	var entity core.WikiEntity
	entity.SetID(article.EntityID)
	orm.WithModel(ctx, &core.WikiEntity{})
	exists, err := orm.GetV2(ctx, &entity)
	if err != nil || !exists {
		return
	}
	entity.Status = entityStatus
	entity.ArticleID = article.ID
	if err := orm.Update(ctx, &entity); err != nil {
		log.Warnf("wiki: failed to sync status for entity %s: %v", entity.ID, err)
	}
}

// findEntityByNameOrAlias resolves a wikilink target by exact name, then
// case-insensitive name, then alias.
func findEntityByNameOrAlias(ctx *orm.Context, name string) *core.WikiEntity {
	return FindEntityByNameOrAlias(ctx, name, "")
}

// FindEntityByNameOrAlias resolves a name to an entity across name and
// aliases (exact, then case-insensitive). Alias matching runs a term query
// first (element-level on ES keyword arrays) and falls back to a bounded
// client-side scan: backends like sqlite store arrays as JSON text, where
// equality can't see individual elements. entityType optionally narrows
// the resolution.
func FindEntityByNameOrAlias(ctx *orm.Context, name, entityType string) *core.WikiEntity {
	orm.WithModel(ctx, &core.WikiEntity{})
	for _, candidate := range []string{name, strings.ToLower(name)} {
		if candidate == "" {
			continue
		}
		builder := orm.NewQuery().Size(1).Filter(orm.TermQuery("name", candidate))
		if entityType != "" {
			builder.Filter(orm.TermQuery("type", strings.ToLower(entityType)))
		}
		if entity := searchOneEntity(ctx, builder); entity != nil {
			return entity
		}
	}
	for _, candidate := range []string{name, strings.ToLower(name)} {
		if candidate == "" {
			continue
		}
		builder := orm.NewQuery().Size(1).Filter(orm.TermQuery("aliases", candidate))
		if entityType != "" {
			builder.Filter(orm.TermQuery("type", strings.ToLower(entityType)))
		}
		if entity := searchOneEntity(ctx, builder); entity != nil {
			return entity
		}
	}

	// portable fallback for array-impaired backends
	const scanLimit = 1000
	builder := orm.NewQuery().Size(scanLimit)
	if entityType != "" {
		builder.Filter(orm.TermQuery("type", strings.ToLower(entityType)))
	}
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return nil
	}
	entities, _, err := elastic.DecodeHits[core.WikiEntity](res)
	if err != nil {
		return nil
	}
	for i := range entities {
		if util.AnyInArrayEquals(entities[i].Aliases, name) {
			return &entities[i]
		}
	}
	return nil
}

func searchOneEntity(ctx *orm.Context, builder *orm.QueryBuilder) *core.WikiEntity {
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return nil
	}
	entities, _, err := elastic.DecodeHits[core.WikiEntity](res)
	if err != nil || len(entities) == 0 {
		return nil
	}
	return &entities[0]
}
