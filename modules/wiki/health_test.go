/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
)

var healthStoreOnce sync.Once

// healthSetup points the global ORM at a sqlite store carrying the
// models the health scan touches.
func healthSetup(t *testing.T) APIHandler {
	t.Helper()
	healthStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-health.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiKnowledgeBase{}, "wiki-kb-health"},
			{core.WikiArticle{}, "wiki-article-health"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-health-test", handler)
	})
	return APIHandler{}
}

var healthFixtureOnce sync.Once

// healthFixture: hub (title-linked from three pages), a chain a→b, a
// thin orphan with no sources, an archived page that must drop out of
// every finding, and a red link ("Ghost Topic") referenced twice — the
// next page worth writing. Seeded once: tests share the package store.
func healthFixture(t *testing.T) (kbID string) {
	t.Helper()
	healthSetup(t)
	healthFixtureOnce.Do(func() {
		kb := &core.WikiKnowledgeBase{}
		kb.SetID("health-kb")
		kb.Name = "Health KB"
		healthWrite(t, kb)

		rich := strings.Repeat("知识沉淀正文。", 40) // well past the readability floor
		source := []core.WikiSourceReference{{DocID: "doc-1"}}

		hub := &core.WikiArticle{KbID: "health-kb", Title: "Hub Page", Status: core.WikiArticlePublished, PageType: core.WikiPageTypeTopic, Content: rich, Sources: source}
		hub.SetID("health-hub")
		// a hub links back out to its cluster, so B and C stay reachable
		hub.LinkedPages = []core.WikiLinkedPage{{Name: "Page B"}, {Name: "Page C"}}
		healthWrite(t, hub)

		a := &core.WikiArticle{KbID: "health-kb", Title: "Page A", Status: core.WikiArticlePublished, PageType: core.WikiPageTypeConcept, Content: rich, Sources: source}
		a.SetID("health-a")
		a.LinkedPages = []core.WikiLinkedPage{{Name: "Hub Page"}}
		healthWrite(t, a)

		b := &core.WikiArticle{KbID: "health-kb", Title: "Page B", Status: core.WikiArticlePublished, PageType: core.WikiPageTypeConcept, Content: rich, Sources: source}
		b.SetID("health-b")
		b.LinkedPages = []core.WikiLinkedPage{{Name: "Page A"}, {Name: "Hub Page"}, {Name: "Ghost Topic"}}
		healthWrite(t, b)

		c := &core.WikiArticle{KbID: "health-kb", Title: "Page C", Status: core.WikiArticleDraft, PageType: core.WikiPageTypeConcept, Content: rich, Sources: source}
		c.SetID("health-c")
		c.LinkedPages = []core.WikiLinkedPage{{Name: "Hub Page"}, {Name: "Ghost Topic"}}
		healthWrite(t, c)

		orphan := &core.WikiArticle{KbID: "health-kb", Title: "Lonely Stub", Status: core.WikiArticleDraft, PageType: core.WikiPageTypeConcept}
		orphan.SetID("health-orphan")
		healthWrite(t, orphan)

		archived := &core.WikiArticle{KbID: "health-kb", Title: "Retired Page", Status: core.WikiArticleArchived, PageType: core.WikiPageTypeConcept, Content: rich}
		archived.SetID("health-archived")
		healthWrite(t, archived)
	})
	return "health-kb"
}

func healthWrite(t *testing.T, obj interface{}) {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	require.NoError(t, orm.Create(ctx, obj))
}

func callHealth(t *testing.T, h APIHandler, kbID string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/wiki/kb/"+kbID+"/_health", nil)
	w := httptest.NewRecorder()
	h.kbHealth(w, req, httprouter.Params{{Key: "id", Value: kbID}})
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func TestKbHealth_Report(t *testing.T) {
	h := healthSetup(t)
	kbID := healthFixture(t)

	w, out := callHealth(t, h, kbID)
	require.Equal(t, http.StatusOK, w.Code)

	counts, _ := out["counts"].(map[string]interface{})
	require.NotNil(t, counts)

	// 6 pages total, one archived and out of every finding
	assert.Equal(t, float64(6), out["total_pages"])
	assert.Equal(t, float64(1), out["archived_pages"])
	assert.Equal(t, float64(1), counts["orphan_pages"], "only Lonely Stub is orphaned")
	assert.Equal(t, float64(1), counts["thin_pages"], "only Lonely Stub is thin")
	assert.Equal(t, float64(1), counts["no_source_pages"], "Lonely Stub is the only unbacked concept page")
	assert.Equal(t, float64(1), counts["knowledge_gaps"], "Ghost Topic is the only red link")

	// gaps: the red link aggregated with demand count and citing pages
	gaps, _ := out["knowledge_gaps"].([]interface{})
	require.Len(t, gaps, 1)
	gap := gaps[0].(map[string]interface{})
	assert.Equal(t, "Ghost Topic", gap["name"])
	assert.Equal(t, float64(2), gap["count"])
	citedBy, _ := gap["cited_by"].([]interface{})
	assert.Len(t, citedBy, 2)

	// hubs: Hub Page anchors three inbound links and tops the list
	hubs, _ := out["hubs"].([]interface{})
	require.NotEmpty(t, hubs)
	top := hubs[0].(map[string]interface{})
	assert.Equal(t, "Hub Page", top["title"])
	assert.Equal(t, float64(3), top["inbound_links"])

	// flagged rows carry the codes; the archived page never appears
	flagged, _ := out["flagged"].([]interface{})
	flaggedTitles := map[string]interface{}{}
	for _, f := range flagged {
		row := f.(map[string]interface{})
		flaggedTitles[row["title"].(string)] = row["codes"]
	}
	require.Contains(t, flaggedTitles, "Lonely Stub")
	assert.Contains(t, flaggedTitles["Lonely Stub"], "orphan")
	assert.NotContains(t, flaggedTitles, "Retired Page")
}

func TestKbHealth_MissingKb(t *testing.T) {
	h := healthSetup(t)
	healthFixture(t)

	w, _ := callHealth(t, h, "nope")
	assert.Equal(t, http.StatusNotFound, w.Code)
}
