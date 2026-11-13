/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package skill

import (
	"bytes"
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
	"infini.sh/framework/core/api/crud"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
)

// Skill lifecycle over the sqlite backend, mirroring the wiki module's
// flow_test.go recipe: crud handlers driven directly via httptest, global
// ORM pointed at sqlite so seeding and prompt assembly hit the same store.
func call(t *testing.T, h crud.HandlerFunc, method, target, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	ps := httprouter.Params{}
	if rest := strings.TrimPrefix(target, "/skill/"); rest != "" && !strings.Contains(rest, "/") {
		if i := strings.IndexByte(rest, '?'); i >= 0 {
			rest = rest[:i]
		}
		ps = httprouter.Params{{Key: "id", Value: rest}}
	}
	w := httptest.NewRecorder()
	h(w, req, ps)
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

var storeOnce sync.Once

func setup(t *testing.T) crud.Handlers {
	t.Helper()
	storeOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "skill.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		if err := handler.RegisterSchemaWithName(core.Skill{}, "skill-test"); err != nil {
			panic(err)
		}
		orm.Register("sqlite-skill-test", handler)
	})
	return crud.NewHandlers(skillConfig())
}

func createSkill(t *testing.T, h crud.Handlers, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	return call(t, h.Create, "POST", "/skill/", body)
}

func TestSkillCreateValidation(t *testing.T) {
	h := setup(t)

	w, _ := createSkill(t, h, `{"title":"missing instructions"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// builtin flag is stripped from user input
	w, out := createSkill(t, h, `{"title":"My Skill","instructions":"do things","builtin":true,"name":"my-skill"}`)
	require.Equal(t, http.StatusOK, w.Code)
	id, _ := out["_id"].(string)
	require.NotEmpty(t, id)

	w, out = call(t, h.Get, "GET", "/skill/"+id, "")
	require.Equal(t, http.StatusOK, w.Code)
	src, _ := out["_source"].(map[string]interface{})
	assert.Equal(t, false, src["builtin"])
	assert.Equal(t, "my-skill", src["name"])
}

func TestSkillBuiltinImmutability(t *testing.T) {
	h := setup(t)

	// seed one builtin record directly (what seedBuiltinSkills does);
	// disabled so later tests that count enabled skills stay isolated
	ctx := orm.NewContext()
	ctx.DirectAccess()
	seed := core.Skill{Name: "test-builtin", Title: "Test", Instructions: "x", Builtin: true}
	require.NoError(t, orm.Create(ctx, &seed))

	// a patch cannot demote it (paired with a regular field: a delta made
	// only of protected fields is a framework edge case, see crud.go)
	w, _ := call(t, h.Update, "PUT", "/skill/"+seed.ID, `{"builtin":false,"title":"Renamed"}`)
	require.Equal(t, http.StatusOK, w.Code)

	w, out := call(t, h.Get, "GET", "/skill/"+seed.ID, "")
	require.Equal(t, http.StatusOK, w.Code)
	src, _ := out["_source"].(map[string]interface{})
	assert.Equal(t, true, src["builtin"])
	assert.Equal(t, "Renamed", src["title"])

	// builtin skills cannot be deleted
	w, _ = call(t, h.Delete, "DELETE", "/skill/"+seed.ID, "")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSkillSeedIdempotency(t *testing.T) {
	setup(t)

	seedBuiltinSkills()
	seedBuiltinSkills() // second run must not duplicate

	skills, err := GetEnabledSkills()
	require.NoError(t, err)
	assert.Len(t, skills, 1) // only retrieval-expert ships enabled
	assert.Equal(t, "retrieval-expert", skills[0].Name)

	// user edits survive reseeding
	ctx := orm.NewContext()
	ctx.DirectAccess()
	edited := skills[0]
	edited.Title = "检索专家（自定义）"
	edited.Enabled = false
	require.NoError(t, orm.Update(ctx, &edited))

	seedBuiltinSkills()

	skills, err = GetEnabledSkills()
	require.NoError(t, err)
	assert.Empty(t, skills, "disabled seed must stay disabled after reseeding")
}

func TestBuildSkillsSection(t *testing.T) {
	setup(t)

	// nothing enabled (previous test disabled the seed)
	assert.Empty(t, BuildSkillsSection())

	ctx := orm.NewContext()
	ctx.DirectAccess()
	s := core.Skill{Name: "s1", Title: "技能一", Instructions: "指令一", Enabled: true, SortOrder: 20}
	require.NoError(t, orm.Create(ctx, &s))
	s2 := core.Skill{Name: "s2", Title: "技能二", Instructions: "指令二", Enabled: true, SortOrder: 10}
	require.NoError(t, orm.Create(ctx, &s2))

	section := BuildSkillsSection()
	assert.Contains(t, section, "## 已激活的技能")
	// sorted by sort_order: 技能二 (10) before 技能一 (20)
	assert.Less(t, strings.Index(section, "技能二"), strings.Index(section, "技能一"))
	assert.Contains(t, section, "指令一")
	assert.Contains(t, section, "指令二")
}
