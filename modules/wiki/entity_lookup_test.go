/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

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

// Regression: extraction probing "DeepSeek" against a stored "Deepseek"
// (and any other mixed-case pair) minted a twin entity — the term lookups
// in FindEntityByNameOrAlias are byte-exact and the lowercase candidate
// only matches rows stored fully lowercase. The case-fold fallback must
// close that hole. Also pins the promotion id to the anchored scheme so
// promote shells and extraction rows collide on one row instead of
// coexisting (Deepseek/DeepSeek, Coco app/Coco App, Coco-Server/coco-server
// in the live pool on 2026-11-08).
var lookupStoreOnce sync.Once

func lookupSetup(t *testing.T) {
	t.Helper()
	lookupStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-entity-lookup.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		if err := handler.RegisterSchemaWithName(core.WikiEntity{}, "wiki-entity-lookup"); err != nil {
			panic(err)
		}
		orm.Register("sqlite-wiki-entity-lookup-test", handler)
	})
}

func TestFindEntityByNameOrAlias_CaseFold(t *testing.T) {
	lookupSetup(t)
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)

	seed := func(id, name string, aliases ...string) {
		e := &core.WikiEntity{}
		e.ID = id
		e.KbID = "kb-fold"
		e.Name = name
		e.Type = "product"
		e.Status = core.WikiEntityPublished
		e.Aliases = aliases
		err := orm.Create(ctx, e)
		require.NoError(t, err)
	}

	seed("ent-deepseek", "Deepseek")
	seed("ent-maxim", "美心", "Maxim's")

	// differently-cased name probe hits the stored row
	require.NotNil(t, FindEntityByNameOrAlias(ctx, "DeepSeek", "", "kb-fold"), "mixed-case name probe must find the row")
	require.NotNil(t, FindEntityByNameOrAlias(ctx, "DEEPSEEK", "", "kb-fold"))
	require.NotNil(t, FindEntityByNameOrAlias(ctx, "deepseek", "", "kb-fold"))

	// differently-cased alias probe hits the stored row
	require.NotNil(t, FindEntityByNameOrAlias(ctx, "MAXIM'S", "", "kb-fold"), "mixed-case alias probe must find the row")

	// exact matches still win and unrelated names still miss
	require.NotNil(t, FindEntityByNameOrAlias(ctx, "Deepseek", "", "kb-fold"))
	assert.Nil(t, FindEntityByNameOrAlias(ctx, "DeepSeek AI", "", "kb-fold"), "partial names must not fold-match")
	assert.Nil(t, FindEntityByNameOrAlias(ctx, "Deepseek", "", "other-kb"), "KB scope must hold")
}

func TestPromoteTopicEntities_AnchoredID(t *testing.T) {
	lookupSetup(t)
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)

	kbID := "kb-anchor"
	n := promoteTopicEntities(ctx, kbID, []string{"Kepler Checker"}, nil)
	require.Equal(t, 1, n)

	// the promoted row carries the same anchored id extraction would mint,
	// so a later extraction pass merges into it instead of forking a twin
	e := findKBEntityByNameOrAlias(ctx, "kepler checker", kbID)
	require.NotNil(t, e)
	assert.Equal(t, core.AnchoredEntityID(kbID, "Kepler Checker"), e.ID, "promoted entity must use the anchored id")
}
