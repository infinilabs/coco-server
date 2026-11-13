/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://www.infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
)

// Own store, same discipline as contributions_test: the global orm handler
// is last-registration-wins and the flow store's lazy registration point
// must stay inside flow_test.go.
var shareStoreOnce sync.Once

func setupShareStore(t *testing.T) {
	t.Helper()
	shareStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-skill-share.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiKnowledgeBase{}, "wiki-kb-share"},
			{core.WikiArticle{}, "wiki-article-share"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-share", handler)
	})
}

func TestSkillIndexIncludesTenantShared(t *testing.T) {
	setupShareStore(t)

	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)

	// KB A is bound to the assistant and owns a private published skill
	kbA := core.WikiKnowledgeBase{Name: "KB A", AssistantID: "asst-1"}
	require.NoError(t, orm.Create(ctx, &kbA))
	own := core.WikiArticle{
		KbID: kbA.ID, Title: "own-skill", PageType: core.WikiPageTypeSkill,
		Status: core.WikiArticlePublished,
		Skill:  &core.WikiSkill{Applicability: "A's own private skill"},
	}
	require.NoError(t, orm.Create(ctx, &own))

	// KB B is NOT bound; it holds one tenant-shared published skill, one
	// private published skill and one shared-but-draft skill — only the
	// first may join the index
	kbB := core.WikiKnowledgeBase{Name: "KB B"}
	require.NoError(t, orm.Create(ctx, &kbB))
	mk := func(title string, status string, scope string) core.WikiArticle {
		return core.WikiArticle{
			KbID: kbB.ID, Title: title, PageType: core.WikiPageTypeSkill,
			Status: status,
			Skill:  &core.WikiSkill{Applicability: title + " applicability", Scope: scope},
		}
	}
	for _, a := range []core.WikiArticle{
		mk("shared-skill", core.WikiArticlePublished, "tenant"),
		mk("private-skill", core.WikiArticlePublished, ""),
		mk("draft-shared", core.WikiArticleDraft, "tenant"),
	} {
		require.NoError(t, orm.Create(ctx, &a))
	}

	idx := SkillIndexForAssistant(orm.NewContext(), "asst-1")
	assert.Contains(t, idx, "own-skill", "the bound KB's skill stays in the index")
	assert.Contains(t, idx, "shared-skill", "tenant-shared skill from the unbound KB joins")
	assert.Contains(t, idx, "共享自 KB B", "shared entries carry their origin KB")
	assert.NotContains(t, idx, "private-skill", "unbound KB's private skill stays out")
	assert.NotContains(t, idx, "draft-shared", "unpublished skills never join")

	// an assistant with no bound KB still sees tenant-shared skills only
	idxOnly := SkillIndexForAssistant(orm.NewContext(), "asst-2")
	assert.NotContains(t, idxOnly, "own-skill")
	assert.Contains(t, idxOnly, "shared-skill")
}
