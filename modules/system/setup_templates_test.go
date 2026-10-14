/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	"encoding/json"
	"os"
	"path"
	"sort"
	"strings"
	"testing"
)

// The setup templates are replay files: verb lines (GET/PUT/POST/DELETE) start
// a request, "#" and "//" lines are comments, every other non-blank line is
// JSON body, and each buffered body is flushed when the next verb line (or EOF)
// arrives — the same rules as framework/plugins/replay.ReplayLines, which
// initializeTemplate feeds the rendered lines into.
//
// These tests keep both language packs installable: every template must render
// and parse, and the two locales must define the same set of documents so one
// language can't silently drift behind the other.

var setupLocales = []string{"en-US", "zh-CN"}

type replayRequest struct {
	file   string
	method string
	uri    string
	body   string
}

func renderSetupTemplate(t *testing.T, content string) string {
	t.Helper()
	replacements := map[string]string{
		"$[[SETUP_OWNER_ID]]":        "setup-test-user",
		"$[[SETUP_INDEX_PREFIX]]":    "coco_",
		"$[[SETUP_SCHEMA_VER]]":      "",
		"$[[SETUP_DOC_TYPE]]":        "_doc",
		"$[[SETUP_SERVER_ENDPOINT]]": "http://localhost:9000",
	}
	for from, to := range replacements {
		content = strings.ReplaceAll(content, from, to)
	}
	return content
}

// parseReplay mirrors ReplayLines' line handling and returns the requests in
// file order.
func parseReplay(t *testing.T, file, content string) []replayRequest {
	t.Helper()

	verbs := []string{"GET", "PUT", "POST", "DELETE"}
	isVerb := func(line string) (string, bool) {
		for _, verb := range verbs {
			if strings.HasPrefix(line, verb+" ") {
				return verb, true
			}
		}
		return "", false
	}

	var requests []replayRequest
	var current *replayRequest
	var bodyLines []string

	flush := func() {
		if current != nil {
			current.body = strings.Join(bodyLines, "\n")
			requests = append(requests, *current)
			current = nil
			bodyLines = nil
		}
	}

	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		if verb, ok := isVerb(line); ok {
			flush()
			fields := strings.Fields(line)
			if len(fields) < 2 {
				t.Fatalf("%s: invalid request line %q", file, line)
			}
			current = &replayRequest{file: file, method: verb, uri: fields[1]}
			continue
		}
		if current == nil {
			t.Fatalf("%s: body line without a request: %q", file, line)
		}
		bodyLines = append(bodyLines, line)
	}
	flush()
	return requests
}

func setupTemplateDir(t *testing.T, locale string) string {
	t.Helper()
	// go test runs in the package directory
	dir := path.Join("..", "..", "config", "setup", locale)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("setup template dir not found: %s", dir)
	}
	return dir
}

func TestSetupTemplatesRenderAndParse(t *testing.T) {
	for _, locale := range setupLocales {
		dir := setupTemplateDir(t, locale)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		if len(entries) == 0 {
			t.Fatalf("no templates in %s", dir)
		}
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasSuffix(name, ".tpl") {
				continue
			}
			raw, err := os.ReadFile(path.Join(dir, name))
			if err != nil {
				t.Fatalf("read %s/%s: %v", locale, name, err)
			}
			content := renderSetupTemplate(t, string(raw))

			if strings.Contains(content, "$[[SETUP_") {
				t.Errorf("%s/%s: unresolved SETUP_ variable after rendering", locale, name)
			}

			for _, req := range parseReplay(t, locale+"/"+name, content) {
				if req.body == "" {
					continue
				}
				if !json.Valid([]byte(req.body)) {
					snippet := req.body
					if len(snippet) > 200 {
						snippet = snippet[:200] + "..."
					}
					t.Errorf("%s: %s %s has invalid JSON body:\n%s", req.file, req.method, req.uri, snippet)
				}
			}
		}
	}
}

// TestSetupTemplatesNoDuplicateDocs asserts a locale never ships the same
// document twice — replay would index both and the later one silently wins,
// which makes the shipped default hard to reason about.
func TestSetupTemplatesNoDuplicateDocs(t *testing.T) {
	for _, locale := range setupLocales {
		dir := setupTemplateDir(t, locale)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		seen := map[string]string{}
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasSuffix(name, ".tpl") {
				continue
			}
			raw, err := os.ReadFile(path.Join(dir, name))
			if err != nil {
				t.Fatalf("read %s/%s: %v", locale, name, err)
			}
			for _, req := range parseReplay(t, locale+"/"+name, renderSetupTemplate(t, string(raw))) {
				key := req.method + "|" + req.uri
				if first, dup := seen[key]; dup {
					t.Errorf("%s: %s %s is defined twice (also in %s)", locale, req.method, req.uri, first)
				}
				seen[key] = req.file
			}
		}
	}
}

// TestSetupTemplatesLocaleParity asserts both language packs define the same
// documents (same file, method and URI). Display names may differ freely; IDs
// may not, because code and cross-references (roles, pipelines, assistants)
// depend on them.
func TestSetupTemplatesLocaleParity(t *testing.T) {
	localeKeys := map[string]map[string]bool{}

	for _, locale := range setupLocales {
		dir := setupTemplateDir(t, locale)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		keys := map[string]bool{}
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasSuffix(name, ".tpl") {
				continue
			}
			raw, err := os.ReadFile(path.Join(dir, name))
			if err != nil {
				t.Fatalf("read %s/%s: %v", locale, name, err)
			}
			for _, req := range parseReplay(t, locale+"/"+name, renderSetupTemplate(t, string(raw))) {
				keys[name+"|"+req.method+"|"+req.uri] = true
			}
		}
		localeKeys[locale] = keys
	}

	en, zh := localeKeys["en-US"], localeKeys["zh-CN"]
	missingIn := func(from, other map[string]bool) []string {
		var missing []string
		for key := range from {
			if !other[key] {
				missing = append(missing, key)
			}
		}
		sort.Strings(missing)
		return missing
	}

	if missing := missingIn(en, zh); len(missing) > 0 {
		t.Errorf("documents missing in zh-CN: %v", missing)
	}
	if missing := missingIn(zh, en); len(missing) > 0 {
		t.Errorf("documents missing in en-US: %v", missing)
	}
}

// TestSetupTemplatesNoHardcodedTika keeps the Tika address out of the shipped
// pipelines: processors resolve it per-call from the operator's settings
// (document_processing.tika_endpoint), and a pinned value here would silently
// override that.
func TestSetupTemplatesNoHardcodedTika(t *testing.T) {
	for _, locale := range setupLocales {
		dir := setupTemplateDir(t, locale)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".tpl") {
				continue
			}
			raw, err := os.ReadFile(path.Join(dir, entry.Name()))
			if err != nil {
				t.Fatalf("read %s/%s: %v", locale, entry.Name(), err)
			}
			if strings.Contains(string(raw), "tika_endpoint") {
				t.Errorf("%s/%s: hardcoded tika_endpoint found; the global document_processing setting governs it now", locale, entry.Name())
			}
		}
	}
}
