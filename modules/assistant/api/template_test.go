/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

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
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
)

var templateStoreOnce sync.Once

func templateSetup(t *testing.T) APIHandler {
	t.Helper()
	templateStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "assistant-template.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.AssistantTemplate{}, "assistant-template-test"},
			{core.Assistant{}, "assistant-test"},
			{core.WikiKnowledgeBase{}, "wiki-kb-template-test"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-assistant-template-test", handler)
	})
	return APIHandler{}
}

func templateCall(t *testing.T, h APIHandler, method, target, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
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
	if rest := strings.TrimPrefix(target, "/assistant-template/"); rest != "" {
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			rest = rest[:i]
		}
		if rest != "" {
			ps = httprouter.Params{{Key: "id", Value: rest}}
		}
	}
	w := httptest.NewRecorder()
	h.instantiateTemplate(w, req, ps)
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func listTemplates(t *testing.T) []core.AssistantTemplate {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.AssistantTemplate{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(100))
	require.NoError(t, err)
	hits, _, err := elastic.DecodeHits[core.AssistantTemplate](res)
	require.NoError(t, err)
	return hits
}

func listAssistants(t *testing.T) []core.Assistant {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.Assistant{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(100))
	require.NoError(t, err)
	hits, _, err := elastic.DecodeHits[core.Assistant](res)
	require.NoError(t, err)
	return hits
}

func TestTemplateSeedIdempotency(t *testing.T) {
	templateSetup(t)

	seedBuiltinTemplates()
	first := listTemplates(t)
	require.Len(t, first, len(builtinTemplateSeeds))

	// user edit survives: rename one, reseed must not duplicate or overwrite
	renamed := first[0]
	renamed.Title = "本地改名"
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.AssistantTemplate{})
	require.NoError(t, orm.Update(ctx, &renamed))

	seedBuiltinTemplates()
	again := listTemplates(t)
	require.Len(t, again, len(builtinTemplateSeeds))
	found := false
	for _, t2 := range again {
		if t2.Name == renamed.Name {
			found = true
			assert.Equal(t, "本地改名", t2.Title)
		}
	}
	assert.True(t, found)
}

func TestInstantiateTemplate(t *testing.T) {
	h := templateSetup(t)
	seedBuiltinTemplates()
	templates := listTemplates(t)
	require.NotEmpty(t, templates)
	support := templates[0]
	for _, tpl := range templates {
		if tpl.Name == "customer-support" {
			support = tpl
		}
	}

	// instantiate without a kb
	w, out := templateCall(t, h, "POST", "/assistant-template/"+support.ID+"/_instantiate", `{"name":"我的客服"}`)
	require.Equal(t, http.StatusOK, w.Code, out)
	assistantID, _ := out["_id"].(string)
	require.NotEmpty(t, assistantID)

	var created *core.Assistant
	for i, a := range listAssistants(t) {
		if a.ID == assistantID {
			created = &listAssistants(t)[i]
		}
	}
	require.NotNil(t, created)
	assert.Equal(t, "我的客服", created.Name)
	assert.False(t, created.Builtin, "instantiated assistants are never builtin")
	assert.True(t, created.Enabled)
	assert.Equal(t, support.RolePrompt, created.RolePrompt)

	// instantiate with a kb: datasource scope + reverse binding
	kb := &core.WikiKnowledgeBase{Name: "Support KB", DatasourceIDs: []string{"ds-1", "ds-2"}}
	kbCtx := orm.NewContext()
	kbCtx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	kbCtx.Refresh = orm.WaitForRefresh
	orm.WithModel(kbCtx, &core.WikiKnowledgeBase{})
	require.NoError(t, orm.Create(kbCtx, kb))

	w, out = templateCall(t, h, "POST", "/assistant-template/"+support.ID+"/_instantiate",
		`{"name":"客服·知识库版","kb_id":"`+kb.ID+`"}`)
	require.Equal(t, http.StatusOK, w.Code, out)
	boundID, _ := out["_id"].(string)
	require.NotEmpty(t, boundID)

	var bound *core.Assistant
	for _, a := range listAssistants(t) {
		if a.ID == boundID {
			bound = &a
		}
	}
	require.NotNil(t, bound)
	assert.True(t, bound.Datasource.Enabled)
	assert.ElementsMatch(t, []string{"ds-1", "ds-2"}, bound.Datasource.IDs)

	// unknown template -> 404
	w, _ = templateCall(t, h, "POST", "/assistant-template/nope/_instantiate", `{}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
