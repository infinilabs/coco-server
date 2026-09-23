/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"fmt"
	"strings"

	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/api/crud"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
)

// permission keys shared between the generated CRUD routes and the
// hand-written endpoints in init.go
var (
	readKbPermission        = security.GetSimplePermission(Category, kbResource, string(security.Read))
	updateKbPermission      = security.GetSimplePermission(Category, kbResource, string(security.Update))
	readArticlePermission   = security.GetSimplePermission(Category, articleResource, string(security.Read))
	updateArticlePermission = security.GetSimplePermission(Category, articleResource, string(security.Update))
	readEntityPermission    = security.GetSimplePermission(Category, entityResource, string(security.Read))
	searchEntityPermission  = security.GetSimplePermission(Category, entityResource, string(security.Search))
)

func simplePermissionFn(resource string) func(action string) api.PermissionKey {
	return func(action string) api.PermissionKey {
		return security.GetSimplePermission(Category, resource, action)
	}
}

func registerWorkspaceCRUD() {
	crud.RegisterCRUD[core.WikiWorkspace](crud.Config[core.WikiWorkspace]{
		Prefix:             "/wiki/workspace",
		Resource:           workspaceResource,
		Permission:         simplePermissionFn(workspaceResource),
		DefaultQueryFields: []string{"name", "name.pinyin", "combined_fulltext"},
		MCP:                true,
		PrepareCreate: func(obj *core.WikiWorkspace) error {
			if obj.Name == "" {
				return fmt.Errorf("name is required")
			}
			return nil
		},
	})
}

func registerKbCRUD() {
	crud.RegisterCRUD(kbConfig())
}

func kbConfig() crud.Config[core.WikiKnowledgeBase] {
	return crud.Config[core.WikiKnowledgeBase]{
		Prefix:             "/wiki/kb",
		Resource:           kbResource,
		Permission:         simplePermissionFn(kbResource),
		DefaultQueryFields: []string{"name", "name.pinyin", "combined_fulltext"},
		SharingResource:    kbResource, // visibility=team goes through the share module (D3)
		MCP:                true,
		MCPDescs: map[string]string{
			crud.ActionSearch: "Search knowledge bases",
			crud.ActionRead:   "Get a knowledge base by id",
		},
		PrepareCreate: func(obj *core.WikiKnowledgeBase) error {
			if obj.Name == "" {
				return fmt.Errorf("name is required")
			}
			switch obj.Visibility {
			case core.WikiVisibilityPublic, core.WikiVisibilityPrivate, core.WikiVisibilityTeam:
			case "":
				obj.Visibility = core.WikiVisibilityTeam
			default:
				return fmt.Errorf("invalid visibility: %s", obj.Visibility)
			}
			return nil
		},
		ProtectedFields: []string{"article_count"},
		PostDelete: func(obj *core.WikiKnowledgeBase) error {
			return deleteKbChildren(obj.ID)
		},
	}
}

func registerArticleCRUD() {
	crud.RegisterCRUD(articleConfig())
}

func articleConfig() crud.Config[core.WikiArticle] {
	return crud.Config[core.WikiArticle]{
		Prefix:             "/wiki/article",
		Resource:           articleResource,
		Permission:         simplePermissionFn(articleResource),
		DefaultQueryFields: []string{"title", "title.pinyin", "summary", "tags", "combined_fulltext"},
		MCP:                true,
		MCPDescs: map[string]string{
			crud.ActionSearch: "Search wiki articles",
			crud.ActionRead:   "Get a wiki article by id, returns structured markdown content",
		},
		PrepareCreate: func(obj *core.WikiArticle) error {
			if obj.KbID == "" {
				return fmt.Errorf("kb_id is required")
			}
			if obj.Title == "" {
				return fmt.Errorf("title is required")
			}
			if obj.Status == "" {
				obj.Status = core.WikiArticleDraft
			}
			if err := validateArticleStatus(obj.Status); err != nil {
				return err
			}
			return nil
		},
		// status is protected on this route: transitions go through
		// PUT /wiki/article/:id/status only (silently stripped per the
		// crud ProtectedFields contract, like created); linked_pages is
		// server-computed from content wikilinks (B3)
		ProtectedFields: []string{"created", "status", "linked_pages"},
		PostCreate: func(obj *core.WikiArticle) error {
			if err := writeVersionSnapshot(obj, 1, changeTypeFor(obj), ""); err != nil {
				return err
			}
			persistLinkedPages(obj)
			if err := addArticleToToc(obj.KbID, obj.ID, obj.Title); err != nil {
				return err
			}
			return bumpKbArticleCount(obj.KbID, 1)
		},
		// versions are content snapshots: metadata-only updates don't
		// create history entries
		PostUpdate: func(obj *core.WikiArticle) error {
			if err := writeVersionIfChanged(obj); err != nil {
				return err
			}
			persistLinkedPages(obj)
			return syncArticleTitleInToc(obj.KbID, obj.ID, obj.Title)
		},
		PostDelete: func(obj *core.WikiArticle) error {
			ctx := orm.NewContext()
			ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
			ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
			if err := deleteArticleChildren(ctx, obj.ID); err != nil {
				return err
			}
			if err := removeArticleFromToc(obj.KbID, obj.ID); err != nil {
				return err
			}
			return bumpKbArticleCount(obj.KbID, -1)
		},
	}
}

func registerBookmarkCRUD() {
	crud.RegisterCRUD[core.WikiBookmark](crud.Config[core.WikiBookmark]{
		Prefix:             "/wiki/bookmark",
		Resource:           bookmarkResource,
		Permission:         simplePermissionFn(bookmarkResource),
		DefaultQueryFields: []string{"article_id"},
		MCP:                false,
		PrepareCreate: func(obj *core.WikiBookmark) error {
			if obj.ArticleID == "" {
				return fmt.Errorf("article_id is required")
			}
			return nil
		},
	})
}

func registerCommentCRUD() {
	crud.RegisterCRUD(commentConfig())
}

func commentConfig() crud.Config[core.WikiComment] {
	return crud.Config[core.WikiComment]{
		Prefix:             "/wiki/comment",
		Resource:           commentResource,
		Permission:         simplePermissionFn(commentResource),
		DefaultQueryFields: []string{"content", "combined_fulltext"},
		MCP:                false,
		PrepareCreate: func(obj *core.WikiComment) error {
			if obj.ArticleID == "" {
				return fmt.Errorf("article_id is required")
			}
			if strings.TrimSpace(obj.Content) == "" {
				return fmt.Errorf("content is required")
			}
			if len([]rune(obj.Content)) > 4000 {
				return fmt.Errorf("content is too long")
			}
			return nil
		},
		// identity comes from the payload on create and locked afterwards;
		// author-only edit/delete is the UI contract
		ProtectedFields: []string{"user_id", "user_name"},
	}
}

func registerNotificationCRUD() {
	crud.RegisterCRUD[core.WikiNotification](crud.Config[core.WikiNotification]{
		Prefix:             "/wiki/notification",
		Resource:           notificationRes,
		Permission:         simplePermissionFn(notificationRes),
		DefaultQueryFields: []string{"message"},
		MCP:                false,
		// update is only used to flip read=true
		ProtectedFields: []string{"user_id", "target_id", "target_type", "action", "message"},
	})
}

func registerEntityCRUD() {
	crud.RegisterCRUD[core.WikiEntity](crud.Config[core.WikiEntity]{
		Prefix:             "/wiki/entity",
		Resource:           entityResource,
		Permission:         simplePermissionFn(entityResource),
		DefaultQueryFields: []string{"name", "name.pinyin", "aliases", "combined_fulltext"},
		SharingResource:    entityResource,
		MCP:                true,
		MCPDescs: map[string]string{
			crud.ActionSearch: "Search ontology entities by name, alias or type",
			crud.ActionRead:   "Get an entity by id, includes relations",
		},
		// entities are retired via status, not deletion (design doc §4.1)
		SkipActions: []string{crud.ActionDelete},
		PrepareCreate: func(obj *core.WikiEntity) error {
			if obj.Name == "" {
				return fmt.Errorf("name is required")
			}
			if obj.Status == "" {
				obj.Status = core.WikiEntityProposed
			}
			switch obj.Status {
			case core.WikiEntityProposed, core.WikiEntityReviewed, core.WikiEntityPublished:
			default:
				return fmt.Errorf("invalid entity status: %s", obj.Status)
			}
			return nil
		},
		ProtectedFields: []string{"created", "sources"}, // sources are pipeline provenance (B2/B3)
	})
}

// deleteKbChildren removes the articles, their versions and the toc
// belonging to a deleted KB (bookmark/notification records are left for
// their owners).
func deleteKbChildren(kbID string) error {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)

	articles, err := findIDs(ctx, &core.WikiArticle{}, orm.TermQuery("kb_id", kbID))
	if err != nil {
		return err
	}
	for _, id := range articles {
		if err := deleteByID(ctx, &core.WikiArticle{}, id); err != nil {
			return err
		}
		if err := deleteArticleChildren(ctx, id); err != nil {
			return err
		}
	}

	tocs, err := findIDs(ctx, &core.WikiToc{}, orm.TermQuery("kb_id", kbID))
	if err != nil {
		return err
	}
	for _, id := range tocs {
		if err := deleteByID(ctx, &core.WikiToc{}, id); err != nil {
			return err
		}
	}
	return nil
}

// deleteArticleChildren removes the versions and comments of an article.
func deleteArticleChildren(ctx *orm.Context, articleID string) error {
	versions, err := findIDs(ctx, &core.WikiVersion{}, orm.TermQuery("article_id", articleID))
	if err != nil {
		return err
	}
	for _, vid := range versions {
		if err := deleteByID(ctx, &core.WikiVersion{}, vid); err != nil {
			return err
		}
	}
	comments, err := findIDs(ctx, &core.WikiComment{}, orm.TermQuery("article_id", articleID))
	if err != nil {
		return err
	}
	for _, cid := range comments {
		if err := deleteByID(ctx, &core.WikiComment{}, cid); err != nil {
			return err
		}
	}
	return nil
}

func bumpKbArticleCount(kbID string, delta int) error {
	if kbID == "" {
		return nil
	}
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiKnowledgeBase{})

	var kb core.WikiKnowledgeBase
	kb.SetID(kbID)
	exists, err := orm.GetV2(ctx, &kb)
	if err != nil || !exists {
		return err
	}
	kb.ArticleCount += delta
	if kb.ArticleCount < 0 {
		kb.ArticleCount = 0
	}
	return orm.Update(ctx, &kb)
}
