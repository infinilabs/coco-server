/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/sqlite"
)

var ontologyStoreOnce sync.Once

func ontologySetup(t *testing.T) APIHandler {
	t.Helper()
	ontologyStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-ontology.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiOntologySchema{}, "wiki-ontology-schema-test"},
			{core.WikiEntity{}, "wiki-entity-onto"},
			{core.WikiKnowledgeBase{}, "wiki-kb-onto"},
			{core.WikiGovernanceProposal{}, "wiki-governance-onto"},
			{core.WikiNotification{}, "wiki-notification-onto"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-ontology-test", handler)
	})
	return APIHandler{}
}

// schemaFixture installs a small vocabulary and returns its doc.
func schemaFixture(t *testing.T) *OntologySchemaDoc {
	t.Helper()
	doc := &OntologySchemaDoc{
		EntityTypes: []OntologyEntityTypeDef{
			{
				Name: "product",
				Properties: []OntologyPropertyDef{
					{Key: "version", Type: "string"},
					{Key: "status", Type: "enum", Enum: []string{"active", "beta"}},
					{Key: "rank", Type: "number"},
					{Key: "code", Type: "string", Required: true},
				},
				Relations: []OntologyRelationDef{
					{Name: "made_by", TargetType: "organization", Cardinality: "one"},
					{Name: "depends_on", TargetType: "product"},
				},
			},
			{Name: "organization"},
		},
	}
	require.NoError(t, doc.normalize())
	require.NoError(t, saveOntologySchema(context.Background(), ontologyTenantScope, doc))
	return doc
}

func TestOntologySchemaNormalize(t *testing.T) {
	doc := &OntologySchemaDoc{EntityTypes: []OntologyEntityTypeDef{
		{Name: "a", Properties: []OntologyPropertyDef{{Key: "p"}}},
		{Name: "a"},
	}}
	assert.Error(t, doc.normalize())

	doc = &OntologySchemaDoc{EntityTypes: []OntologyEntityTypeDef{
		{Name: "a", Properties: []OntologyPropertyDef{{Key: "p", Type: "enum"}}},
	}}
	assert.Error(t, doc.normalize(), "enum without allowed values")

	doc = &OntologySchemaDoc{EntityTypes: []OntologyEntityTypeDef{
		{Name: "  a ", Properties: []OntologyPropertyDef{{Key: " p "}}},
	}}
	require.NoError(t, doc.normalize())
	assert.Equal(t, "a", doc.EntityTypes[0].Name)
	assert.Equal(t, "string", doc.EntityTypes[0].Properties[0].Type, "type defaults to string")
}

func TestValidateEntityAgainstSchema(t *testing.T) {
	ontologySetup(t)
	ctx := context.Background()

	// lenient without a schema on file
	assert.NoError(t, validateEntityAgainstSchema(ctx, &core.WikiEntity{
		Type: "anything", Name: "x", Properties: map[string]interface{}{"free": "form"},
	}))

	schemaFixture(t)

	// undeclared type is rejected once a schema exists
	err := validateEntityAgainstSchema(ctx, &core.WikiEntity{Type: "alien", Name: "x"})
	assert.ErrorContains(t, err, "not declared")

	// missing required property
	err = validateEntityAgainstSchema(ctx, &core.WikiEntity{Type: "product", Name: "x"})
	assert.ErrorContains(t, err, "is required")

	// wrong value type / enum violation
	err = validateEntityAgainstSchema(ctx, &core.WikiEntity{Type: "product", Name: "x",
		Properties: map[string]interface{}{"code": "c1", "rank": "high"}})
	assert.ErrorContains(t, err, `"rank"`)

	err = validateEntityAgainstSchema(ctx, &core.WikiEntity{Type: "product", Name: "x",
		Properties: map[string]interface{}{"code": "c1", "status": "zombie"}})
	assert.ErrorContains(t, err, "enum")

	// undeclared relation
	err = validateEntityAgainstSchema(ctx, &core.WikiEntity{Type: "product", Name: "x",
		Properties: map[string]interface{}{"code": "c1"},
		Relations:  []core.WikiEntityRelation{{Relation: "hates", TargetID: "e1"}}})
	assert.ErrorContains(t, err, `"hates" is not declared`)

	// cardinality one violated
	err = validateEntityAgainstSchema(ctx, &core.WikiEntity{Type: "product", Name: "x",
		Properties: map[string]interface{}{"code": "c1"},
		Relations: []core.WikiEntityRelation{
			{Relation: "made_by", TargetID: "e1"},
			{Relation: "made_by", TargetID: "e2"},
		}})
	assert.ErrorContains(t, err, "at most one target")

	// happy path with a declared relation; unknown target skips the type check
	assert.NoError(t, validateEntityAgainstSchema(ctx, &core.WikiEntity{Type: "product", Name: "x",
		Properties: map[string]interface{}{"code": "c1", "status": "beta", "rank": 3.0},
		Relations:  []core.WikiEntityRelation{{Relation: "depends_on", TargetID: "missing"}}}))
}

func TestValidateEntityRelationTargetType(t *testing.T) {
	ontologySetup(t)
	ctx := context.Background()
	schemaFixture(t)

	// a product target resolves to the wrong type -> rejected
	org := &core.WikiEntity{Type: "organization", Name: "ACME"}
	prod := &core.WikiEntity{Type: "product", Name: "Widget",
		Properties: map[string]interface{}{"code": "c1"}}
	for _, obj := range []*core.WikiEntity{org, prod} {
		c := orm.NewContext()
		c.Set(orm.DirectWriteWithoutPermissionCheck, true)
		orm.WithModel(c, &core.WikiEntity{})
		require.NoError(t, orm.Create(c, obj))
	}

	err := validateEntityAgainstSchema(ctx, &core.WikiEntity{Type: "product", Name: "Bad",
		Properties: map[string]interface{}{"code": "c"},
		Relations:  []core.WikiEntityRelation{{Relation: "depends_on", TargetID: org.ID}}})
	assert.ErrorContains(t, err, `requires target type "product"`)

	assert.NoError(t, validateEntityAgainstSchema(ctx, &core.WikiEntity{Type: "product", Name: "OK",
		Properties: map[string]interface{}{"code": "c"},
		Relations:  []core.WikiEntityRelation{{Relation: "depends_on", TargetID: prod.ID}}}))
}

func TestOntologySchemaAPI(t *testing.T) {
	h := ontologySetup(t)

	// seed default vocabulary once, then idempotently again
	seedDefaultOntologySchema()
	seedDefaultOntologySchema()

	get := func(target string) (*httptest.ResponseRecorder, map[string]interface{}) {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		w := httptest.NewRecorder()
		h.getOntologySchema(w, req, httprouter.Params{})
		out := map[string]interface{}{}
		_ = jsonUnmarshal(w.Body.Bytes(), &out)
		return w, out
	}

	w, out := get("/wiki/ontology/schema")
	require.Equal(t, http.StatusOK, w.Code)
	src, _ := out["_source"].(map[string]interface{})
	types, _ := src["entity_types"].([]interface{})
	require.NotEmpty(t, types, "default vocabulary seeded")

	// PUT replaces the tenant schema
	body := `{"schema":{"entity_types":[{"name":"thing","properties":[{"key":"code","type":"string","required":true}]}]}}`
	req := httptest.NewRequest(http.MethodPut, "/wiki/ontology/schema", httptestBody(body))
	req.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.putOntologySchema(w2, req, httprouter.Params{})
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	_, out = get("/wiki/ontology/schema")
	src, _ = out["_source"].(map[string]interface{})
	types, _ = src["entity_types"].([]interface{})
	require.Len(t, types, 1)

	// invalid document is rejected with 400
	req = httptest.NewRequest(http.MethodPut, "/wiki/ontology/schema", httptestBody(`{"schema":{"entity_types":[{"name":"a"},{"name":"a"}]}}`))
	req.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	h.putOntologySchema(w3, req, httprouter.Params{})
	assert.Equal(t, http.StatusBadRequest, w3.Code)
}

func TestOntologySchemaRoundTrip(t *testing.T) {
	ontologySetup(t)
	ctx := context.Background()

	doc := schemaFixture(t)
	loaded := loadOntologySchema(ctx, ontologyTenantScope)
	require.NotNil(t, loaded)
	require.Len(t, loaded.EntityTypes, 2)
	assert.Equal(t, doc.EntityTypes[0].Properties[1].Enum, loaded.EntityTypes[0].Properties[1].Enum)
	assert.Equal(t, "one", loaded.EntityTypes[0].Relations[0].Cardinality)
}

func jsonUnmarshal(b []byte, out interface{}) error {
	return json.Unmarshal(b, out)
}

func httptestBody(s string) *bytes.Reader {
	return bytes.NewReader([]byte(s))
}

func TestSanitizeEntityAgainstSchema(t *testing.T) {
	ontologySetup(t)
	doc := schemaFixture(t) // product: enum status, number rank, string code(required); relations made_by(one, org) / depends_on(product)

	entity := &core.WikiEntity{Type: "product", Name: "X",
		Properties: map[string]interface{}{"code": "c1", "status": "beta", "rank": 2.0, "note": "free-form kept"},
		Relations: []core.WikiEntityRelation{
			{Relation: "depends_on", TargetID: "e1"},
			{Relation: "made_by", TargetID: "e2"},
			{Relation: "made_by", TargetID: "e3"}, // cardinality overflow
			{Relation: "hates", TargetID: "e4"},   // undeclared
		}}
	dropped := SanitizeEntityAgainstSchema(entity, doc)
	assert.Len(t, entity.Relations, 2, "keeps depends_on + first made_by")
	assert.Equal(t, "e2", entity.Relations[1].TargetID)
	assert.Equal(t, "c1", entity.Properties["code"])
	assert.Equal(t, "free-form kept", entity.Properties["note"], "undeclared properties survive")
	assert.Len(t, dropped, 2)
	assert.Contains(t, dropped[0], "cardinality")
	assert.Contains(t, dropped[1], "hates")

	// value-type violations are stripped
	entity = &core.WikiEntity{Type: "product", Name: "Y",
		Properties: map[string]interface{}{"code": "c", "rank": "high"}}
	dropped = SanitizeEntityAgainstSchema(entity, doc)
	assert.NotContains(t, entity.Properties, "rank")
	assert.NotEmpty(t, dropped)

	// untyped / unknown-type entities pass untouched
	entity = &core.WikiEntity{Type: "legacy_free", Name: "Z",
		Relations: []core.WikiEntityRelation{{Relation: "whatever"}}}
	assert.Nil(t, SanitizeEntityAgainstSchema(entity, doc))
	assert.Len(t, entity.Relations, 1)
	assert.Nil(t, SanitizeEntityAgainstSchema(entity, nil))
}

func TestFileEntityGovernanceProposal(t *testing.T) {
	ontologySetup(t)

	entity := &core.WikiEntity{Type: "product", Name: "Coco Server"}
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.WikiEntity{})
	require.NoError(t, orm.Create(ctx, entity))

	assert.True(t, FileEntityGovernanceProposal("kb-1", entity, core.WikiGovernanceEntityConflict,
		"mentioned as person in doc-2", util.MapStr{"doc_id": "doc-2"}))

	// idempotent: same open (entity, type) is not filed twice
	assert.False(t, FileEntityGovernanceProposal("kb-1", entity, core.WikiGovernanceEntityConflict,
		"again", nil))

	listCtx := orm.NewContext()
	listCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(listCtx, &core.WikiGovernanceProposal{})
	res, err := orm.SearchV2(listCtx, orm.NewQuery().Size(10))
	require.NoError(t, err)
	proposals, _, err := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	require.NoError(t, err)
	require.Len(t, proposals, 1)
	assert.Equal(t, entity.ID, proposals[0].ArticleID)
	assert.Equal(t, entity.Name, proposals[0].ArticleTitle)
	assert.Equal(t, core.WikiGovernanceEntityConflict, proposals[0].Type)
	assert.Equal(t, core.WikiGovernanceOpen, proposals[0].Status)
}
