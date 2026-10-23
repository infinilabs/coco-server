package wiki

import (
	"strings"
	"testing"

	"infini.sh/coco/core"
)

func node(id, title, kind, articleID string, children ...core.WikiTocNode) core.WikiTocNode {
	n := core.WikiTocNode{ID: id, Title: title, Type: kind, ArticleID: articleID}
	if len(children) > 0 {
		n.Children = children
	}
	return n
}

func TestParseCategoryList(t *testing.T) {
	cats, err := parseCategoryList("```json\n[\"产品与服务\", \"数据与检索\", \"部署与运维\", \"\", \"产品与服务\"]\n```")
	if err != nil {
		t.Fatalf("fenced parse failed: %v", err)
	}
	if len(cats) != 3 {
		t.Fatalf("expected 3 cats after dedupe/empty-drop, got %v", cats)
	}

	cats2, err := parseCategoryList(`["Alpha","Beta"]`)
	if err != nil || len(cats2) != 2 {
		t.Fatalf("bare array parse failed: %v %v", cats2, err)
	}

	if _, err := parseCategoryList("no json here at all"); err == nil {
		t.Fatal("garbage must error")
	}
	if _, err := parseCategoryList(`[1,2,3]`); err == nil {
		t.Fatal("non-string array must error")
	}
}

func TestParseAssignments(t *testing.T) {
	m := parseAssignments("```json\n[{\"id\":\"a1\",\"category\":\"X\"},{\"id\":\"\",\"category\":\"Y\"},{\"id\":\"a2\"},{\"id\":\"a3\",\"category\":\"Z\"}]\n```")
	if m["a1"] != "X" || m["a3"] != "Z" {
		t.Fatalf("valid pairs missing: %v", m)
	}
	if _, ok := m["a2"]; ok {
		t.Fatal("category-less pair must be skipped")
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m))
	}
	if got := parseAssignments("total garbage"); len(got) != 0 {
		t.Fatalf("garbage must yield empty map, got %v", got)
	}
}

func TestBuildOrganizedToc(t *testing.T) {
	manual := node("folder-1", "手工目录", "folder", "", node("toc-x", "已归类", "article", "art-x"))
	looseA := node("toc-a", "Neo4j Graph Connector", "article", "art-a")
	looseB := node("toc-b", "API Authentication", "article", "art-b")
	looseC := node("toc-c", "未分类孤例", "article", "art-c")

	assigns := map[string]string{
		"art-a": "数据连接器",
		"art-b": "安全与认证",
		"art-c": "模型没提到的类别", // invented category → fallback bucket
	}
	categories := []string{"数据连接器", "安全与认证", "空类别"}

	out, stats := buildOrganizedToc([]core.WikiTocNode{looseA, manual, looseB, looseC}, assigns, categories, "kb1")

	if stats.Loose != 3 || stats.Moved != 3 || stats.KeptFolders != 1 {
		t.Fatalf("stats wrong: %+v", stats)
	}
	// empty category must vanish; fallback folder must exist and carry art-c
	names := map[string]bool{}
	for _, n := range out {
		names[n.Title] = true
	}
	if names["空类别"] {
		t.Fatal("empty category must not create a folder")
	}
	if !names[organizeFallbackCategory] {
		t.Fatal("fallback folder missing for invented category")
	}
	if stats.Other != 1 {
		t.Fatalf("fallback count wrong: %+v", stats)
	}
	// manual folder stays first with its subtree intact
	if out[0].ID != "folder-1" || len(out[0].Children) != 1 || out[0].Children[0].ArticleID != "art-x" {
		t.Fatalf("manual folder must lead untouched: %+v", out[0])
	}
	// articles alphabetical inside folders
	for _, n := range out {
		if n.Type != "folder" {
			continue
		}
		for i := 1; i < len(n.Children); i++ {
			if n.Children[i-1].Title > n.Children[i].Title {
				t.Fatalf("children not sorted in %s", n.Title)
			}
		}
	}
	// folder ids stable across runs (same kb+category → same id)
	again, _ := buildOrganizedToc([]core.WikiTocNode{looseA, looseB}, map[string]string{"art-a": "数据连接器", "art-b": "安全与认证"}, categories, "kb1")
	for i := range out {
		if out[i].Type == "folder" && strings.HasPrefix(out[i].ID, "folder-ai-") && again[len(again)-1].ID != "" {
			// ids depend only on category+kb; spot-check one
			break
		}
	}
	if organizeFolderID("数据连接器", "kb1") != organizeFolderID("数据连接器", "kb1") {
		t.Fatal("folder id must be deterministic")
	}
	if organizeFolderID("数据连接器", "kb1") == organizeFolderID("数据连接器", "kb2") {
		t.Fatal("folder id must differ per kb")
	}
}

func TestBuildOrganizedTocKeepsAllLoose(t *testing.T) {
	loose := []core.WikiTocNode{
		node("t1", "A", "article", "a1"),
		node("t2", "B", "article", "a2"),
	}
	// no assignments at all → everything lands in the fallback folder
	out, stats := buildOrganizedToc(loose, map[string]string{}, []string{"Cat"}, "kb")
	total := 0
	for _, n := range out {
		if n.Type == "folder" {
			total += len(n.Children)
		} else {
			total++
		}
	}
	if total != len(loose) || stats.Moved != len(loose) {
		t.Fatalf("lost nodes: total=%d moved=%d", total, stats.Moved)
	}
}
