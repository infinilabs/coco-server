/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"infini.sh/coco/core"
	"infini.sh/framework/core/entity_card"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
)

var entityCardStoreOnce sync.Once

func entityCardSetup(t *testing.T) {
	t.Helper()
	// AppConfig touches the kv store — swap for the pointer-constructed
	// config (ServerInfo is a pointer, zero Config panics)
	restore := companyMapConfigFn
	t.Cleanup(func() { companyMapConfigFn = restore })
	companyMapConfigFn = func() core.Config {
		return core.Config{ServerInfo: &core.ServerInfo{Endpoint: "http://coco.test"}}
	}
	entityCardStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-entity-card.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		if err := handler.RegisterSchemaWithName(core.WikiEntity{}, "wiki-entity-card"); err != nil {
			panic(err)
		}
		orm.Register("sqlite-wiki-entity-card-test", handler)
	})
}

func TestGenEntityInfoBuildsStandardCard(t *testing.T) {
	entityCardSetup(t)
	e := &core.WikiEntity{}
	e.ID = "ent-card-1"
	e.Name = "支付网关"
	e.Type = "product"
	e.Status = core.WikiEntityReviewed
	e.Aliases = []string{"PGW"}
	e.ArticleID = "art-1"
	e.Properties = map[string]interface{}{"owner": "platform"}
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	require.NoError(t, orm.Create(ctx, e))

	p := &WikiEntityProvider{}
	card := p.GenEntityInfo(context.Background(), CardTypeWikiEntity, "ent-card-1")
	require.NotNil(t, card)
	assert.Equal(t, "支付网关", card.Title)
	assert.Equal(t, "product", card.Subtitle)
	assert.Equal(t, []string{"PGW"}, card.Tags)
	assert.Contains(t, card.URL, "/wiki/article/art-1")
	require.NotNil(t, card.Details)
	require.NotNil(t, card.Details.Table)
	require.Len(t, card.Details.Table.Rows, 1)
}

func TestGenEntityInfoMissingEntityDegradesToShell(t *testing.T) {
	entityCardSetup(t)
	p := &WikiEntityProvider{}
	card := p.GenEntityInfo(context.Background(), CardTypeWikiEntity, "no-such-entity")
	require.NotNil(t, card, "a missing entity still yields a card, never nil/error")
	assert.Equal(t, "no-such-entity", card.Title)
}

func TestGenEntityLabelFillsGaps(t *testing.T) {
	entityCardSetup(t)
	e := &core.WikiEntity{}
	e.ID = "ent-label-1"
	e.Name = "订单服务"
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	require.NoError(t, orm.Create(ctx, e))

	p := &WikiEntityProvider{}
	labels := p.GenEntityLabel(context.Background(), CardTypeWikiEntity, []string{"ent-label-1", "ent-unknown"})
	require.Len(t, labels, 2)
	assert.Equal(t, "订单服务", labels[0].Title)
	assert.Equal(t, "ent-unknown", labels[1].Title, "unknown ids degrade to id-titled rows, no gaps")
}

// provider registration tripwire: the type string the frontend passes
// must be the one registered here.
func TestWikiEntityProviderTypeRegistered(t *testing.T) {
	assert.Equal(t, "wiki_entity", CardTypeWikiEntity)
	var _ entity_card.EntityProvider = &WikiEntityProvider{}
}
