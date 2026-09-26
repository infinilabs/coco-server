/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
	"infini.sh/framework/core/api/crud"
	"infini.sh/framework/core/orm"
)

/* ---------------- pure unit tests ---------------- */

func TestValidateCandidates(t *testing.T) {
	docs := make([]core.Document, 2)
	pages := []pageCandidate{
		{Title: "Coffee Sourcing", PageType: "concept", DocRefs: []int{1, 2}},
		{Title: "  "},                           // empty after trim
		{Title: "Bad Refs", DocRefs: []int{99}}, // no usable refs
		{Title: "Odd Type", PageType: "mystery", DocRefs: []int{1, 0, 2}},
		{Title: "coffee sourcing", DocRefs: []int{1}}, // dup, case-insensitive
	}
	got := validateCandidates(pages, docs, 5)
	require.Len(t, got, 2)
	assert.Equal(t, "Coffee Sourcing", got[0].Title)
	assert.Equal(t, []int{1, 2}, got[0].DocRefs)
	assert.Equal(t, core.WikiPageTypeConcept, got[1].PageType, "unknown type falls back to concept")
	assert.Equal(t, []int{1, 2}, got[1].DocRefs, "out-of-range refs are dropped, valid ones kept")

	// cap
	many := []pageCandidate{
		{Title: "a", DocRefs: []int{1}}, {Title: "b", DocRefs: []int{1}}, {Title: "c", DocRefs: []int{1}},
	}
	assert.Len(t, validateCandidates(many, docs, 2), 2)
}

func TestConfidenceFor(t *testing.T) {
	assert.Equal(t, "high", confidenceFor(8, 2, 2)) // saturated density + full diversity
	assert.Equal(t, "low", confidenceFor(0, 0, 2))
	assert.Contains(t, []string{"medium", "low"}, confidenceFor(2, 1, 2))
}

func TestMatchArticlesToChangedDocs(t *testing.T) {
	articles := []core.WikiArticle{
		{Status: core.WikiArticlePublished, Sources: []core.WikiSourceReference{{DocID: "doc-1"}}},
		{Status: core.WikiArticleDraft, Sources: []core.WikiSourceReference{{DocID: "doc-1"}}},    // draft: skipped
		{Status: core.WikiArticleReviewed, Sources: []core.WikiSourceReference{{DocID: "doc-9"}}}, // no hit
		{Status: core.WikiArticlePublished, Sources: []core.WikiSourceReference{{DocID: ""}}},     // empty doc_id
		{Status: core.WikiArticleArchived, Sources: []core.WikiSourceReference{{DocID: "doc-1"}}}, // archived: skipped
	}
	got := MatchArticlesToChangedDocs(articles, map[string]bool{"doc-1": true})
	require.Len(t, got, 1)
	assert.Equal(t, core.WikiArticlePublished, got[0].Status)
}

/* ---------------- fake LLM ---------------- */

// fakeKMModel answers the pipeline's prompt shapes with canned payloads:
// cluster plans, page outlines and chapter bodies. Callers can assert on
// call counts afterwards.
type fakeKMModel struct {
	calls int
}

func lastHumanText(messages []llms.MessageContent) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == llms.ChatMessageTypeHuman {
			var text strings.Builder
			for _, part := range messages[i].Parts {
				if part, ok := part.(llms.TextContent); ok {
					text.WriteString(part.Text)
				}
			}
			return text.String()
		}
	}
	return ""
}

func (m *fakeKMModel) GenerateContent(_ context.Context, messages []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
	m.calls++
	prompt := lastHumanText(messages)
	var reply string
	switch {
	case strings.Contains(prompt, "Plan wiki pages"):
		reply = `{"pages":[` +
			`{"title":"Coffee Sourcing","page_type":"concept","doc_refs":[1,2]},` +
			`{"title":"   "},` +
			`{"title":"Bad","doc_refs":[99]},` +
			`{"title":"coffee sourcing","doc_refs":[1]}` +
			`]}`
	case strings.Contains(prompt, "Design the chapter outline"):
		reply = `{"summary":"Sourcing overview in two acts.","chapters":[` +
			`{"title":"Overview","key_points":["origins"]},` +
			`{"title":"NoCite","key_points":["uncited"]}` +
			`]}`
	case strings.Contains(prompt, "Chapter: NoCite"):
		reply = "Coffee grows in the highlands." // no citation: chapter must be dropped
	case strings.Contains(prompt, "Write one chapter"):
		reply = "Arabica is documented in the sourcing report [1]. See also [[concept:Robusta]]."
	case strings.Contains(prompt, "Rewrite"):
		reply = "Arabica is sourced from new farms [1]."
	default:
		reply = `{}` // think-stripped empty object
	}
	return &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: reply}}}, nil
}

func (m *fakeKMModel) Call(_ context.Context, _ string, _ ...llms.CallOption) (string, error) {
	return "", nil
}

func stubGenerationLLM(t *testing.T, model llms.Model) *fakeKMModel {
	t.Helper()
	fake, ok := model.(*fakeKMModel)
	require.True(t, ok)
	original := resolveLanguageLLM
	resolveLanguageLLM = func(string, string) (llms.Model, error) {
		return fake, nil
	}
	t.Cleanup(func() { resolveLanguageLLM = original })
	return fake
}

/* ---------------- SSE helpers ---------------- */

type sseEvent struct {
	Name string
	Data map[string]interface{}
}

func parseSSE(t *testing.T, body string) []sseEvent {
	t.Helper()
	events := []sseEvent{}
	for _, block := range strings.Split(strings.TrimSpace(body), "\n\n") {
		event := sseEvent{}
		for _, line := range strings.Split(block, "\n") {
			if name, ok := strings.CutPrefix(line, "event: "); ok {
				event.Name = name
			} else if data, ok := strings.CutPrefix(line, "data: "); ok {
				parsed := map[string]interface{}{}
				require.NoError(t, json.Unmarshal([]byte(data), &parsed), "SSE data must be JSON: %s", data)
				event.Data = parsed
			}
		}
		if event.Name != "" {
			events = append(events, event)
		}
	}
	return events
}

func callAI(t *testing.T, handler crud.HandlerFunc, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// /wiki/{kb|article}/<id>/ai/<action> — the router would bind :id
	parts := strings.Split(target, "/")
	w := httptest.NewRecorder()
	handler(w, req, httprouter.Params{{Key: "id", Value: parts[3]}})
	return w
}

func sseEventsByName(events []sseEvent, name string) []map[string]interface{} {
	out := []map[string]interface{}{}
	for _, event := range events {
		if event.Name == name {
			out = append(out, event.Data)
		}
	}
	return out
}

func seedDocument(t *testing.T, id, title, datasourceID string, updated time.Time) core.Document {
	t.Helper()
	doc := core.Document{
		Title:   title,
		Summary: title + " summary",
		Content: title + " full body with sourcing details.",
		Source:  core.DataSourceReference{ID: datasourceID, Name: "DS " + datasourceID, Type: "connector"},
	}
	doc.ID = id
	doc.Updated = &updated
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.Document{})
	require.NoError(t, orm.Create(ctx, &doc))
	return doc
}

/* ---------------- C1–C3: generation flow ---------------- */

func TestWikiFlow_AIGeneration(t *testing.T) {
	kbH, artH, h := setupFlow(t)
	_ = artH

	fake := stubGenerationLLM(t, &fakeKMModel{})

	w, out := call(t, kbH.Create, "POST", "/wiki/kb/", `{"name":"Gen KB","datasource_ids":["ds-gen"]}`)
	require.Equal(t, http.StatusOK, w.Code)
	kbID, _ := out["_id"].(string)
	require.NotEmpty(t, kbID)

	now := time.Now()
	seedDocument(t, "doc-gen-1", "Sourcing Report", "ds-gen", now)
	seedDocument(t, "doc-gen-2", "Audit 2026", "ds-gen", now)

	rec := callAI(t, h.aiGenerate, "/wiki/kb/"+kbID+"/ai/generate", `{}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))

	events := parseSSE(t, rec.Body.String())
	require.NotEmpty(t, events)

	// phase contract: scope -> cluster -> draft -> deliver -> done
	progress := sseEventsByName(events, "progress")
	phases := []string{}
	for _, p := range progress {
		phases = append(phases, p["phase"].(string))
	}
	for _, want := range []string{"scope", "cluster", "draft", "deliver"} {
		assert.Contains(t, phases, want)
	}

	// one article event: the only valid candidate, as a draft with sources
	articles := sseEventsByName(events, "article")
	require.Len(t, articles, 1)
	article := articles[0]
	assert.Equal(t, "Coffee Sourcing", article["title"])
	assert.Equal(t, core.WikiArticleDraft, article["status"])
	assert.Equal(t, true, article["ai_generated"])
	assert.Contains(t, []string{"high", "medium", "low"}, article["confidence"])
	assert.Contains(t, article["content"], "## Overview")
	assert.NotContains(t, article["content"], "NoCite", "uncited chapter must be dropped")

	sources, _ := article["sources"].([]interface{})
	require.Len(t, sources, 1, "only the cited document backlinks")
	source, _ := sources[0].(map[string]interface{})
	assert.Equal(t, "doc-gen-1", source["doc_id"])

	done := sseEventsByName(events, "done")
	require.Len(t, done, 1)
	assert.Equal(t, float64(1), done[0]["generated"])

	// delivered through the real path: persisted, versioned, counted
	artID, _ := article["id"].(string)
	require.NotEmpty(t, artID)
	w, out = call(t, artH.Get, "GET", "/wiki/article/"+artID, "")
	require.Equal(t, http.StatusOK, w.Code)
	src, _ := out["_source"].(map[string]interface{})
	assert.Equal(t, core.WikiArticleDraft, src["status"])
	linked, _ := src["linked_pages"].([]interface{})
	require.Len(t, linked, 1)
	link, _ := linked[0].(map[string]interface{})
	assert.Equal(t, "Robusta", link["name"])

	versions := listVersions(t, h, artID)
	require.Len(t, versions, 1)
	assert.Equal(t, core.WikiChangeAIGenerated, versions[0]["change_type"])

	w, out = call(t, kbH.Get, "GET", "/wiki/kb/"+kbID, "")
	src, _ = out["_source"].(map[string]interface{})
	assert.Equal(t, float64(1), src["article_count"])

	// re-running finds nothing new (title dedup) — done carries the reason
	rec = callAI(t, h.aiGenerate, "/wiki/kb/"+kbID+"/ai/generate", `{}`)
	require.Equal(t, http.StatusOK, rec.Code)
	done = sseEventsByName(parseSSE(t, rec.Body.String()), "done")
	require.Len(t, done, 1)
	assert.Equal(t, float64(0), done[0]["generated"])
	assert.NotEmpty(t, done[0]["reason"])

	assert.Greater(t, fake.calls, 0)
}

// the generation gate: no KB datasources refuses before any LLM call
func TestWikiFlow_AIGenerationNoDatasources(t *testing.T) {
	kbH, _, h := setupFlow(t)
	stubGenerationLLM(t, &fakeKMModel{})

	w, out := call(t, kbH.Create, "POST", "/wiki/kb/", `{"name":"Empty KB"}`)
	require.Equal(t, http.StatusOK, w.Code)
	kbID, _ := out["_id"].(string)

	req := httptest.NewRequest(http.MethodPost, "/wiki/kb/"+kbID+"/ai/generate", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	h.aiGenerate(rec, req, httprouter.Params{{Key: "id", Value: kbID}})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

/* ---------------- C4: instruction-based edit ---------------- */

func TestWikiFlow_AIEdit(t *testing.T) {
	_, artH, h := setupFlow(t)
	stubGenerationLLM(t, &fakeKMModel{})

	w, out := call(t, artH.Create, "POST", "/wiki/article/", `{"kb_id":"kb-edit","title":"Origins","content":`+jsonString("## Overview\n\nArabica is sourced [1]. Old sentence.")+`}`)
	require.Equal(t, http.StatusOK, w.Code)
	artID, _ := out["_id"].(string)

	// selection must exist in the content
	req := httptest.NewRequest(http.MethodPost, "/wiki/article/"+artID+"/ai/edit", strings.NewReader(`{"instruction":"rewrite","selection":"Missing"}`))
	rec := httptest.NewRecorder()
	h.aiEdit(rec, req, httprouter.Params{{Key: "id", Value: artID}})
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = callAI(t, h.aiEdit, "/wiki/article/"+artID+"/ai/edit", `{"instruction":"tighten","selection":"Old sentence."}`)
	require.Equal(t, http.StatusOK, rec.Code)
	events := parseSSE(t, rec.Body.String())
	done := sseEventsByName(events, "done")
	require.Len(t, done, 1)
	assert.Equal(t, artID, done[0]["_id"])
	assert.Equal(t, float64(2), done[0]["version"])

	// selection replaced in place, status untouched, version recorded
	w, out = call(t, artH.Get, "GET", "/wiki/article/"+artID, "")
	require.Equal(t, http.StatusOK, w.Code)
	src, _ := out["_source"].(map[string]interface{})
	assert.Equal(t, "## Overview\n\nArabica is sourced [1]. Arabica is sourced from new farms [1].", src["content"])
	assert.Equal(t, core.WikiArticleDraft, src["status"])

	versions := listVersions(t, h, artID)
	require.Len(t, versions, 2)
	assert.Equal(t, core.WikiChangeAIGenerated, versions[0]["change_type"])
	assert.Contains(t, versions[0]["change_summary"], "AI edit")
}

/* ---------------- C5: freshness sweep ---------------- */

func TestWikiFlow_Freshness(t *testing.T) {
	kbH, artH, h := setupFlow(t)
	_ = h

	// isolate this test's datasources from the other flow tests (shared store)
	w, out := call(t, kbH.Create, "POST", "/wiki/kb/", `{"name":"Fresh KB","datasource_ids":["ds-fresh"]}`)
	require.Equal(t, http.StatusOK, w.Code)
	kbID, _ := out["_id"].(string)

	now := time.Now()
	seedDocument(t, "doc-fresh-1", "Fresh Report", "ds-fresh", now)
	seedDocument(t, "doc-fresh-2", "Stale Report", "ds-fresh", now.Add(-48*time.Hour))

	// published article citing the changed document
	w, out = call(t, artH.Create, "POST", "/wiki/article/", `{"kb_id":"`+kbID+`","title":"Freshness Page","content":`+jsonString("## Overview\n\noriginal content")+
		`,"sources":[{"doc_id":"doc-fresh-1"}]}`)
	require.Equal(t, http.StatusOK, w.Code)
	artID, _ := out["_id"].(string)
	w, _ = callStatus(t, h, artID, `{"status":"reviewed"}`)
	require.Equal(t, http.StatusOK, w.Code)
	w, _ = callStatus(t, h, artID, `{"status":"published"}`)
	require.Equal(t, http.StatusOK, w.Code)

	originalFreshnessLLM := freshnessLLM
	fake := &fakeKMModel{}
	t.Cleanup(func() { freshnessLLM = originalFreshnessLLM })

	stats := freshnessSweep(context.Background(), fake, now.Add(-time.Hour))
	assert.Greater(t, stats.changedDocs, 0)
	require.Equal(t, 1, stats.regenerated)

	// live content untouched; an auto-updated version awaits review
	w, out = call(t, artH.Get, "GET", "/wiki/article/"+artID, "")
	require.Equal(t, http.StatusOK, w.Code)
	src, _ := out["_source"].(map[string]interface{})
	assert.Equal(t, "## Overview\n\noriginal content", src["content"], "auto-updated output must not overwrite live content")
	assert.Equal(t, core.WikiArticlePublished, src["status"])

	// v1 create + two status-transition audits + the auto-updated snapshot
	versions := listVersions(t, h, artID)
	require.Len(t, versions, 4)
	latest := versions[0]
	assert.Equal(t, core.WikiChangeAutoUpdated, latest["change_type"])
	assert.Contains(t, latest["content"], "## Overview")
	assert.Contains(t, latest["change_summary"], "freshness")
}

func TestSanitizeAIArtifactDebris(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"undefined\n\n## 相关实体\n\n- [[origin:巴西]]", "## 相关实体\n\n- [[origin:巴西]]"},
		{"  null\nNaN\n\n## 正文\n内容", "## 正文\n内容"},
		{"正常内容开头", "正常内容开头"},
		{"the value is undefined here", "the value is undefined here"},
		{" undefined ", "undefined"},  // standalone artifact only -> returned trimmed, never emptied
		{"undefined\n\nnull", "null"}, // strips leading, keeps the rest even if artifact-y
		{"\n\n直接正文", "直接正文"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, sanitizeAIArtifactDebris(c.in), "input: %q", c.in)
	}
}
