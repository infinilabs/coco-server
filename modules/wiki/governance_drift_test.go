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
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
)

var driftStoreOnce sync.Once

func driftSetup(t *testing.T) {
	t.Helper()
	driftStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-drift.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiEntity{}, "wiki-entity-drift2"},
			{core.WikiArticle{}, "wiki-article-drift2"},
			{core.WikiKnowledgeBase{}, "wiki-kb-drift2"},
			{core.WikiGovernanceProposal{}, "wiki-proposal-drift2"},
			{core.WikiOntologySchema{}, "wiki-schema-drift2"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-drift-test", handler)
	})
	// seed the tenant vocabulary via the production seeder
	seedDefaultOntologySchema()
}

func driftWrite(t *testing.T, obj interface{}) {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	require.NoError(t, orm.Create(ctx, obj))
}

func TestSchemaDriftScanFilesUndeclaredTypes(t *testing.T) {
	driftSetup(t)
	// declared: organization, person, product, concept, event, document.
	// "vendor" is not → drift. Written in THIS test so the shared-store
	// execution order can't leak other fixtures in.
	e := &core.WikiEntity{}
	e.ID = "drift-vendor-x"
	e.Name = "ACME Vendor"
	e.Type = "vendor"
	e.Status = core.WikiEntityReviewed
	driftWrite(t, e)

	okE := &core.WikiEntity{}
	okE.ID = "drift-person-x"
	okE.Name = "Alice Person"
	okE.Type = "person"
	okE.Status = core.WikiEntityReviewed
	driftWrite(t, okE)

	// verify the entities are visible to the scan's read path first
	chk := orm.NewContext()
	chk.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(chk, &core.WikiEntity{})
	visRes, verr := orm.SearchV2(chk, orm.NewQuery().Size(10))
	require.NoError(t, verr)
	visHits, _, _ := elastic.DecodeHits[core.WikiEntity](visRes)
	require.GreaterOrEqual(t, len(visHits), 2, "both entities must be visible before scan")

	// debug: verify visibility inside the failing test
	chk2 := orm.NewContext()
	chk2.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(chk2, &core.WikiEntity{})
	visRes, verr2 := orm.SearchV2(chk2, orm.NewQuery().Size(10))
	t.Logf("visibility check: err=%v", verr2)
	if raw, ok := visRes.Payload.([]byte); ok {
		t.Logf("visible entities in raw: %s", string(raw))
	}

	filed := schemaDriftScan(context.Background())
	assert.GreaterOrEqual(t, filed, 1, "the undeclared-type entity files a drift proposal")

	// the declared-type entity must NOT appear in drift proposals
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(100).
		Filter(orm.TermQuery("type", core.WikiGovernanceDrift)))
	require.NoError(t, err)
	hits, _, derr := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	require.NoError(t, derr)
	for _, p := range hits {
		if name, _ := p.Evidence["entity_name"].(string); name == "Alice Person" {
			t.Fatalf("declared-type entity must not drift, got %+v", p)
		}
	}
}

func TestSchemaDriftSkipsDeclaredTypes(t *testing.T) {
	driftSetup(t)
	ok := &core.WikiEntity{}
	ok.ID = "drift-ok"
	ok.Name = "Alice"
	ok.Type = "person"
	ok.Status = core.WikiEntityReviewed
	driftWrite(t, ok)

	before := countDriftProposals(t)
	schemaDriftScan(context.Background())
	after := countDriftProposals(t)
	assert.Equal(t, before, after, "declared types file nothing")
}

func countDriftProposals(t *testing.T) int {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(100).
		Filter(orm.TermQuery("type", core.WikiGovernanceDrift)))
	require.NoError(t, err)
	hits, _, derr := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	require.NoError(t, derr)
	return len(hits)
}
