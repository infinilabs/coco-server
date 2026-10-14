/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/sqlite"
)

var graphStoreOnce sync.Once

// graphSetup points the global ORM at a sqlite store carrying the models
// the knowledge-graph query touches.
func graphSetup(t *testing.T) APIHandler {
	t.Helper()
	graphStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-graph.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiKnowledgeBase{}, "wiki-kb-graph"},
			{core.WikiArticle{}, "wiki-article-graph"},
			{core.WikiEntity{}, "wiki-entity-graph"},
			{core.WikiOntologySchema{}, "wiki-ontology-schema-graph"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-graph-test", handler)
	})
	return APIHandler{}
}

func graphWrite(t *testing.T, obj interface{}) {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	require.NoError(t, orm.Create(ctx, obj))
}

var graphFixtureOnce sync.Once

// graphFixture: tenant schema declaring person/organization with an
// inverse pair; alice (cited) works_for acme (only reachable via relation
// expansion); latte cited directly; zoe cited nowhere — only visible from
// the reverse direction. Seeded once: tests share the package store.
func graphFixture(t *testing.T) (kbID, aliceID, acmeID string) {
	t.Helper()
	graphSetup(t)
	graphFixtureOnce.Do(func() {
		graphSeed(t)
	})
	return "graph-kb", "graph-alice", "graph-acme"
}

func graphSeed(t *testing.T) {
	t.Helper()
	schema := &core.WikiOntologySchema{}
	schema.SetID("graph-schema")
	schema.Scope = "tenant"
	schema.Schema = util.MapStr{
		"entity_types": []util.MapStr{
			{
				"name":  "person",
				"label": "人员",
				"relations": []util.MapStr{
					{"name": "works_for", "label": "任职于", "target_type": "organization", "inverse": "has_employee"},
				},
			},
			{
				"name":  "organization",
				"label": "组织",
				"relations": []util.MapStr{
					{"name": "has_employee", "label": "雇员", "target_type": "person"},
				},
			},
			{"name": "drink", "label": "饮品"},
		},
	}
	graphWrite(t, schema)

	kb := &core.WikiKnowledgeBase{}
	kb.SetID("graph-kb")
	kb.Name = "Graph KB"
	graphWrite(t, kb)

	alice := &core.WikiEntity{Type: "person", Name: "Alice", Status: core.WikiEntityPublished}
	alice.SetID("graph-alice")
	alice.Relations = []core.WikiEntityRelation{{TargetID: "graph-acme", TargetType: "organization", Relation: "works_for"}}
	graphWrite(t, alice)

	zoe := &core.WikiEntity{Type: "person", Name: "Zoe", Status: core.WikiEntityPublished}
	zoe.SetID("graph-zoe")
	zoe.Relations = []core.WikiEntityRelation{{TargetID: "graph-acme", TargetType: "organization", Relation: "works_for"}}
	graphWrite(t, zoe)

	acme := &core.WikiEntity{Type: "organization", Name: "Acme", Status: core.WikiEntityPublished}
	acme.SetID("graph-acme")
	graphWrite(t, acme)

	latte := &core.WikiEntity{Type: "drink", Name: "Latte", Status: core.WikiEntityProposed, Confidence: 0.7}
	latte.SetID("graph-latte")
	latte.Properties = map[string]interface{}{"origin": "Italy"}
	graphWrite(t, latte)

	article := &core.WikiArticle{KbID: "graph-kb", Title: "People and Drinks", Status: core.WikiArticlePublished, PageType: core.WikiPageTypeConcept}
	article.SetID("graph-article")
	article.LinkedPages = []core.WikiLinkedPage{
		{Type: "person", Name: "Alice", EntityID: "graph-alice"},
		{Type: "drink", Name: "Latte", EntityID: "graph-latte"},
		{Type: "topic", Name: "Missing Thing"},
	}
	graphWrite(t, article)
}

func callGraph(t *testing.T, h APIHandler, target, id string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	h.kbGraph(w, req, httprouter.Params{{Key: "id", Value: id}})
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func TestKbGraph_AssemblesSubgraph(t *testing.T) {
	h := graphSetup(t)
	kbID, _, acmeID := graphFixture(t)

	w, out := callGraph(t, h, "/wiki/kb/"+kbID+"/_graph", kbID)
	require.Equal(t, http.StatusOK, w.Code)

	nodes, _ := out["nodes"].([]interface{})
	require.NotNil(t, nodes)
	nodeByID := map[string]map[string]interface{}{}
	for _, n := range nodes {
		node, _ := n.(map[string]interface{})
		nodeByID[node["id"].(string)] = node
	}

	// the article, both cited entities, the relation-expanded organization
	// and the unresolved wikilink placeholder are all on the canvas
	require.Contains(t, nodeByID, "article:graph-article")
	require.Contains(t, nodeByID, "entity:graph-alice")
	require.Contains(t, nodeByID, "entity:graph-latte")
	require.Contains(t, nodeByID, "lp:topic:Missing Thing")
	acmeNode, hasAcme := nodeByID["entity:"+acmeID]
	require.True(t, hasAcme, "relation target should be expanded onto the canvas")
	assert.Equal(t, true, acmeNode["via_relation"], "expanded entity is not cited by any article")

	assert.Equal(t, "Alice", nodeByID["entity:graph-alice"]["label"])
	assert.Equal(t, "person", nodeByID["entity:graph-alice"]["type"])
	assert.Equal(t, "人员", nodeByID["entity:graph-alice"]["type_label"])
	assert.Equal(t, "proposed", nodeByID["entity:graph-latte"]["status"])
	assert.InEpsilon(t, 0.7, nodeByID["entity:graph-latte"]["confidence"], 0.001)
	assert.Equal(t, map[string]interface{}{"origin": "Italy"}, nodeByID["entity:graph-latte"]["properties"])

	// zoe is neither cited nor a relation target: absent from the canvas
	assert.NotContains(t, nodeByID, "entity:graph-zoe")

	edges, _ := out["edges"].([]interface{})
	edgeByKey := map[string]map[string]interface{}{}
	for _, e := range edges {
		edge, _ := e.(map[string]interface{})
		edgeByKey[edge["source"].(string)+"->"+edge["target"].(string)] = edge
	}

	wikilink := edgeByKey["article:graph-article->entity:graph-alice"]
	require.NotNil(t, wikilink, "wikilink edge missing")
	assert.Equal(t, "wikilink", wikilink["kind"])

	relation := edgeByKey["entity:graph-alice->entity:"+acmeID]
	require.NotNil(t, relation, "typed relation edge missing")
	assert.Equal(t, "relation", relation["kind"])
	assert.Equal(t, "works_for", relation["relation"])
	assert.Equal(t, "任职于", relation["label"], "schema label should override the raw name")
	assert.Equal(t, "has_employee", relation["inverse"])
	assert.Equal(t, "雇员", relation["inverse_label"], "inverse label resolved from the target type")

	require.NotNil(t, edgeByKey["article:graph-article->lp:topic:Missing Thing"], "unresolved wikilink edge missing")

	// entity type metadata for the frontend
	types, _ := out["entity_types"].(map[string]interface{})
	require.Contains(t, types, "person")
}

func TestKbGraph_MissingKb(t *testing.T) {
	h := graphSetup(t)
	graphFixture(t)

	w, _ := callGraph(t, h, "/wiki/kb/nope/_graph", "nope")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestEntityNeighbors_InverseRelations(t *testing.T) {
	h := graphSetup(t)
	_, _, acmeID := graphFixture(t)

	req := httptest.NewRequest(http.MethodGet, "/wiki/entity/"+acmeID+"/neighbors", nil)
	w := httptest.NewRecorder()
	h.entityNeighbors(w, req, httprouter.Params{{Key: "id", Value: acmeID}})
	require.Equal(t, http.StatusOK, w.Code)

	out := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))

	// acme has no forward relations, so detailed stays empty
	detailed, _ := out["relations_detailed"].([]interface{})
	assert.Empty(t, detailed)

	// incoming scan finds both alice and zoe with schema-resolved labels
	incoming, _ := out["incoming"].([]interface{})
	require.Len(t, incoming, 2)
	byName := map[string]map[string]interface{}{}
	for _, e := range incoming {
		entry, _ := e.(map[string]interface{})
		byName[entry["source_name"].(string)] = entry
	}
	aliceIn := byName["Alice"]
	require.NotNil(t, aliceIn)
	assert.Equal(t, "works_for", aliceIn["relation"])
	assert.Equal(t, "任职于", aliceIn["label"])
	assert.Equal(t, "has_employee", aliceIn["inverse"])
	assert.Equal(t, "雇员", aliceIn["inverse_label"])
	require.NotNil(t, byName["Zoe"])
}

func TestEntityNeighbors_UndeclaredRelationFallsBackToRawName(t *testing.T) {
	graphSetup(t)
	graphFixture(t)

	// "drives" is not in the vocabulary: the label falls back to the raw
	// relation name and no inverse is reported
	a := &core.WikiEntity{Type: "person", Name: "Odd Driver", Status: core.WikiEntityPublished}
	a.SetID("ns-alice")
	a.Relations = []core.WikiEntityRelation{{TargetID: "ns-acme", TargetType: "organization", Relation: "drives"}}
	graphWrite(t, a)
	b := &core.WikiEntity{Type: "organization", Name: "Odd Garage", Status: core.WikiEntityPublished}
	b.SetID("ns-acme")
	graphWrite(t, b)

	h := APIHandler{}
	req := httptest.NewRequest(http.MethodGet, "/wiki/entity/ns-alice/neighbors", nil)
	w := httptest.NewRecorder()
	h.entityNeighbors(w, req, httprouter.Params{{Key: "id", Value: "ns-alice"}})
	require.Equal(t, http.StatusOK, w.Code)

	out := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	detailed, _ := out["relations_detailed"].([]interface{})
	require.Len(t, detailed, 1)
	entry, _ := detailed[0].(map[string]interface{})
	assert.Equal(t, "drives", entry["relation"])
	assert.Equal(t, "drives", entry["label"], "undeclared relation: raw name as label")
	assert.NotContains(t, entry, "inverse")
	assert.Equal(t, "Odd Garage", entry["target_name"])
}
