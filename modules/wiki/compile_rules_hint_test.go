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
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"

	"infini.sh/coco/core"
)

// rulesStoreOnce gives the compile-rules tests their own store: model
// routing across the shared test stores binds by latest registration, so a
// test file that fires another file's sync.Once early changes which store
// later files write to. Self-contained setup keeps the ordering intact.
var rulesStoreOnce sync.Once

func rulesSetup(t *testing.T) {
	t.Helper()
	rulesStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-rules.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiOntologySchema{}, "wiki-ontology-schema-rules"},
			{core.Document{}, "document-rules"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-rules-test", handler)
	})
}

// restoreFixtureSchema reinstalls the standard vocabulary so rules tests
// don't leave a rules-only tenant schema behind for other tests.
func restoreFixtureSchema(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { schemaFixture(t) })
	schemaFixture(t)
}

func TestSchemaRulesRoundTrip(t *testing.T) {
	rulesSetup(t)
	ctx := context.Background()

	doc := &OntologySchemaDoc{
		EntityTypes: rulesFixtureTypes,
		Rules: &core.WikiCompileRules{
			ConceptMinSources: 3,
			ConflictStrategy:  core.WikiConflictStrategyIsolate,
			SourcePriority:    []string{"ds-official", "ds-mirror"},
		},
	}
	require.NoError(t, saveOntologySchema(ctx, ontologyTenantScope, doc))

	loaded := loadOntologySchema(ctx, ontologyTenantScope)
	require.NotNil(t, loaded, "rules-only schema must load, not fall through")
	assert.Equal(t, 3, loaded.Rules.ConceptMinSources)
	assert.Equal(t, core.WikiConflictStrategyIsolate, loaded.Rules.ConflictStrategy)
	assert.Equal(t, []string{"ds-official", "ds-mirror"}, loaded.Rules.SourcePriority)
}

// rulesFixtureTypes keeps the vocabulary present while rules tests mutate
// the schema document (rules-only docs are valid, but the round-trip should
// prove entity types survive alongside).
var rulesFixtureTypes = []OntologyEntityTypeDef{{Name: "product"}, {Name: "organization"}}

func TestSchemaRulesNormalizeOnSave(t *testing.T) {
	rulesSetup(t)
	ctx := context.Background()

	doc := &OntologySchemaDoc{
		EntityTypes: rulesFixtureTypes,
		Rules:       &core.WikiCompileRules{ConceptMinSources: 99, ConflictStrategy: "bogus"},
	}
	require.NoError(t, saveOntologySchema(ctx, ontologyTenantScope, doc))

	loaded := loadOntologySchema(ctx, ontologyTenantScope)
	require.NotNil(t, loaded)
	// normalize clamps the floor and the strategy before the read side sees them
	assert.Equal(t, core.DefaultConceptMinSources, loaded.Rules.ConceptMinSources)
	assert.Equal(t, core.WikiConflictStrategyMark, loaded.Rules.ConflictStrategy)
}

func TestDuplicatePriorityHintPrefersHigherSource(t *testing.T) {
	rulesSetup(t)
	ctx := context.Background()

	// priority: official wins over mirror
	require.NoError(t, saveOntologySchema(ctx, ontologyTenantScope, &OntologySchemaDoc{
		EntityTypes: rulesFixtureTypes,
		Rules:       &core.WikiCompileRules{SourcePriority: []string{"ds-official", "ds-mirror"}},
	}))

	a := &core.WikiArticle{Title: "from-mirror", Sources: []core.WikiSourceReference{{DocID: "doc-mirror"}}}
	b := &core.WikiArticle{Title: "from-official", Sources: []core.WikiSourceReference{{DocID: "doc-official"}}}

	// seed two documents with different datasources
	octx := orm.NewContext()
	octx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.Document{})
	for _, seed := range []struct {
		id, datasource string
	}{{"doc-mirror", "ds-mirror"}, {"doc-official", "ds-official"}} {
		d := core.Document{Source: core.DataSourceReference{ID: seed.datasource}}
		d.SetID(seed.id)
		require.NoError(t, orm.Create(octx, &d))
	}

	hint := duplicatePriorityHint(ctx, "", a, b)
	assert.Contains(t, hint, "from-official")
	assert.Contains(t, hint, "ds-official")

	// same datasource on both sides: no recommendation
	b.Sources = []core.WikiSourceReference{{DocID: "doc-mirror"}}
	assert.Empty(t, duplicatePriorityHint(ctx, "", a, b))

	// no priority configured: never recommends
	require.NoError(t, saveOntologySchema(ctx, ontologyTenantScope, &OntologySchemaDoc{
		EntityTypes: rulesFixtureTypes,
		Rules:       &core.WikiCompileRules{},
	}))
	assert.Empty(t, duplicatePriorityHint(ctx, "", a, b))
}
