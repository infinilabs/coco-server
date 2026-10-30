/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

// repair-flow lenient mode (O2): case-variant declared types canonicalize,
// undeclared types degrade to concept instead of 400.
func TestValidateEntityCanonicalizesCasing(t *testing.T) {
	doc := defaultOntologySchema() // declares lowercase "organization"
	e := &core.WikiEntity{Type: "Organization", Name: "Alibaba Cloud"}
	require.NoError(t, validateEntityWithDoc(context.Background(), e, doc))
	assert.Equal(t, "organization", e.Type, "declared casing wins on case-insensitive hit")
}

func TestDegradeUndeclaredType(t *testing.T) {
	// self-contained store: other wiki tests seed schemas that may declare
	// different vocabularies, and the last-registered store wins globally —
	// a unique store keeps this fixture's expectations stable
	once := &sync.Once{}
	var handler *sqlite.SQLiteORM
	once.Do(func() {
		handler = &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-lenient.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		if err := handler.RegisterSchemaWithName(core.WikiEntity{}, "wiki-entity-lenient"); err != nil {
			panic(err)
		}
		if err := handler.RegisterSchemaWithName(core.WikiOntologySchema{}, "wiki-ontology-lenient"); err != nil {
			panic(err)
		}
		orm.Register("sqlite-wiki-lenient-test", handler)

		// seed a minimal tenant vocabulary (person/organization — NOT tool)
		ctx := orm.NewContext()
		ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
		ctx.Refresh = orm.WaitForRefresh
		orm.WithModel(ctx, &core.WikiOntologySchema{})
		schema := core.WikiOntologySchema{
			Scope: ontologyTenantScope,
			Schema: map[string]interface{}{
				"entity_types": []map[string]interface{}{
					{"name": "person"},
					{"name": "organization"},
				},
			},
		}
		schema.ID = "lenient-tenant-schema"
		if err := orm.Create(ctx, &schema); err != nil {
			panic(err)
		}
	})
	doc := defaultOntologySchema()
	_ = doc

	// undeclared → concept
	e := &core.WikiEntity{Type: "Tool", Name: "Claude Code"}
	degraded := degradeUndeclaredType(e, "")
	schemaDoc := loadOntologySchemaForKB(context.Background(), "")
	if schemaDoc != nil {
		names := make([]string, 0, len(schemaDoc.EntityTypes))
		for i := range schemaDoc.EntityTypes {
			names = append(names, schemaDoc.EntityTypes[i].Name)
		}
		t.Logf("loaded types: %v", names)
	} else {
		t.Logf("loaded doc: nil")
	}
	if !degraded {
		t.Logf("degrade returned false; type now %q; typeDef(Tool) = %v", e.Type, schemaDoc != nil && schemaDoc.typeDef("Tool") != nil)
	}
	assert.True(t, degraded)
	assert.Equal(t, "concept", e.Type)

	// declared (case-insensitive) → NOT degraded, casing canonicalized
	e2 := &core.WikiEntity{Type: "Organization", Name: "Alibaba"}
	assert.False(t, degradeUndeclaredType(e2, ""))
	assert.Equal(t, "Organization", e2.Type, "degrade is a no-op for declared types — canonicalization is validateEntityWithDoc's job")

	// untyped → no-op
	e3 := &core.WikiEntity{}
	assert.False(t, degradeUndeclaredType(e3, ""))
}
