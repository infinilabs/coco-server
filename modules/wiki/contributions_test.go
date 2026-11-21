/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://www.infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
)

// The contribution tests carry their own sqlite store: the global orm
// handler is last-registration-wins and the shared flow store must keep its
// lazy first-registration point inside flow_test.go (registering it here
// would let the entity tests' stores steal the handler before the flow
// tests run).
var contribStoreOnce sync.Once

func setupContribStore(t *testing.T) {
	t.Helper()
	contribStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-contrib.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		if err := handler.RegisterSchemaWithName(core.WikiContribution{}, "wiki-contribution-test"); err != nil {
			panic(err)
		}
		orm.Register("sqlite-wiki-contrib", handler)
	})
}

func TestSyncPageContributions(t *testing.T) {
	setupContribStore(t)

	syncPageContributions(nil, "kb-1", "page-1", []string{"doc-a", "doc-b"})
	rows := contributionsForPages(nil, "kb-1", []string{"page-1"})
	assert.Len(t, rows, 2, "one row per cited doc")

	// re-sync with a changed citation set: doc-b dropped, doc-c added,
	// doc-a must survive without duplicating
	syncPageContributions(nil, "kb-1", "page-1", []string{"doc-a", "doc-c"})
	rows = contributionsForPages(nil, "kb-1", []string{"page-1"})
	assert.Len(t, rows, 2)
	set := map[string]bool{}
	for _, r := range rows {
		set[r.DocID] = true
		assert.Equal(t, "kb-1", r.KbID)
		assert.Equal(t, "page-1", r.PageID)
	}
	assert.True(t, set["doc-a"] && set["doc-c"], "citations follow the latest sync: %v", set)

	// the reverse lookup finds the page from either cited doc
	for _, doc := range []string{"doc-a", "doc-c"} {
		hits := contributionsForDocs(nil, "kb-1", []string{doc})
		assert.Len(t, hits, 1, "doc %s resolves to its page", doc)
	}

	// page deletion drops the whole row set
	dropPageContributions(nil, "page-1")
	assert.Empty(t, contributionsForPages(nil, "kb-1", []string{"page-1"}))
}

func TestPlanIncrementalScope(t *testing.T) {
	ops := []CompileOp{
		{Op: "ingest", DocID: "d1"},
		{Op: "retract", DocID: "d2"},
	}
	byDoc := []core.WikiContribution{
		{KbID: "kb", DocID: "d1", PageID: "p1"},
		{KbID: "kb", DocID: "d1", PageID: "p2"},
		{KbID: "kb", DocID: "d2", PageID: "p1"},
		{KbID: "kb", DocID: "d3", PageID: "p9"}, // unrelated doc: noise
	}
	byPage := []core.WikiContribution{
		{KbID: "kb", DocID: "d1", PageID: "p1"},
		{KbID: "kb", DocID: "d4", PageID: "p1"}, // co-cited unchanged doc
		{KbID: "kb", DocID: "d2", PageID: "p2"},
		{KbID: "kb", DocID: "d3", PageID: "p9"}, // unaffected page: noise
	}

	scope := planIncrementalScope(ops, byDoc, byPage)
	assert.ElementsMatch(t, []string{"d1", "d2"}, scope.OpDocs)
	assert.ElementsMatch(t, []string{"p1", "p2"}, scope.AffectedPages, "only pages citing op docs")
	// map set = op docs ∪ docs cited by affected pages: d4 joins because p1
	// cites it; p9's d3 must NOT — its page is unaffected
	assert.ElementsMatch(t, []string{"d1", "d2", "d4"}, scope.MapDocs)

	// phase-1 call (no byPage rows yet) still yields the affected pages
	p1 := planIncrementalScope(ops, byDoc, nil)
	assert.ElementsMatch(t, []string{"p1", "p2"}, p1.AffectedPages)
	assert.ElementsMatch(t, []string{"d1", "d2"}, p1.MapDocs, "without byPage the map set is the op docs")

	// a whole-corpus op signals the full-pipeline fallback with an empty scope
	fallback := planIncrementalScope([]CompileOp{{Op: "ingest", DocID: ""}}, byDoc, byPage)
	assert.Empty(t, fallback.OpDocs)
	assert.Empty(t, fallback.AffectedPages)
	assert.Empty(t, fallback.MapDocs)

	// empty op doc lists never fan out into queries
	assert.Nil(t, contributionsForDocs(nil, "kb", nil))
	assert.Nil(t, contributionsForPages(nil, "kb", nil))
}
