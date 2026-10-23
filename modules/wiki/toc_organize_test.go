package wiki

import (
	"strings"
	"testing"

	"infini.sh/coco/core"
)

func allClassified(nodes []core.WikiTocNode, more ...string) map[string]bool {
	set := map[string]bool{}
	var walk func(ns []core.WikiTocNode)
	walk = func(ns []core.WikiTocNode) {
		for _, n := range ns {
			if n.ArticleID != "" {
				set[n.ArticleID] = true
			}
			if len(n.Children) > 0 {
				walk(n.Children)
			}
		}
	}
	walk(nodes)
	for _, id := range more {
		set[id] = true
	}
	return set
}

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

	out, stats := buildOrganizedToc([]core.WikiTocNode{looseA, manual, looseB, looseC}, assigns, categories, allClassified([]core.WikiTocNode{looseA, manual, looseB, looseC}), "kb1")

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
	again, _ := buildOrganizedToc([]core.WikiTocNode{looseA, looseB}, map[string]string{"art-a": "数据连接器", "art-b": "安全与认证"}, categories, allClassified([]core.WikiTocNode{looseA, looseB}), "kb1")
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
	out, stats := buildOrganizedToc(loose, map[string]string{}, []string{"Cat"}, allClassified(loose), "kb")
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

func TestBuildOrganizedTocMergesIntoExistingFolder(t *testing.T) {
	aiID := organizeFolderID("Connectors", "kb")
	existing := []core.WikiTocNode{
		node(aiID, "Connectors", "folder", "", node("t1", "ZZ Old Page", "article", "a1")),
		node("t2", "New Page", "article", "a2"),
		node("t3", "Another New", "article", "a3"),
	}
	out, stats := buildOrganizedToc(existing,
		map[string]string{"a2": "Connectors", "a3": "Connectors"},
		[]string{"Connectors"}, allClassified(existing), "kb")
	if len(out) != 1 || out[0].ID != aiID {
		t.Fatalf("must merge into the existing folder, got %d nodes, first id %s", len(out), out[0].ID)
	}
	titles := []string{}
	for _, c := range out[0].Children {
		titles = append(titles, c.Title)
	}
	// AI folder → merged children re-sort alphabetically
	if strings.Join(titles, "|") != "Another New|New Page|ZZ Old Page" {
		t.Fatalf("merged children out of order: %v", titles)
	}
	if stats.Moved != 2 || stats.Merged != 2 || stats.KeptFolders != 1 {
		t.Fatalf("stats: moved=%d merged=%d kept=%d", stats.Moved, stats.Merged, stats.KeptFolders)
	}
}

func TestBuildOrganizedTocMergeKeepsManualFolderOrder(t *testing.T) {
	existing := []core.WikiTocNode{
		node("folder-manual", "产品与服务", "folder", "",
			node("t1", "Zed", "article", "a1"),
			node("t2", "Alpha", "article", "a2")),
		node("t3", "Mid Product", "article", "a3"),
	}
	out, stats := buildOrganizedToc(existing,
		map[string]string{"a3": "产品与服务"},
		[]string{"产品与服务"}, allClassified(existing), "kb")
	if len(out) != 1 || out[0].ID != "folder-manual" {
		t.Fatalf("must merge into the manual folder, got %d nodes", len(out))
	}
	titles := []string{}
	for _, c := range out[0].Children {
		titles = append(titles, c.Title)
	}
	// hand-built folder: original order untouched, new pages appended
	if strings.Join(titles, "|") != "Zed|Alpha|Mid Product" {
		t.Fatalf("manual order must survive: %v", titles)
	}
	if stats.Merged != 1 || stats.Moved != 1 {
		t.Fatalf("stats: moved=%d merged=%d", stats.Moved, stats.Merged)
	}
}

func TestBuildProposalPrompt(t *testing.T) {
	small := buildProposalPrompt(6, []string{"专题与概念"})
	if !strings.Contains(small, "between 2 and 10") {
		t.Fatalf("small set must relax the lower bound: %s", small)
	}
	if !strings.Contains(small, "Existing folders: 专题与概念") {
		t.Fatalf("existing titles must be offered for reuse: %s", small)
	}
	large := buildProposalPrompt(120, nil)
	if !strings.Contains(large, "between 4 and 10") {
		t.Fatalf("large set keeps the full spread: %s", large)
	}
	if strings.Contains(large, "Existing folders") {
		t.Fatalf("no existing-titles line when the tree has none: %s", large)
	}
}

func TestBuildOrganizedTocLeavesMidPassPagesAtRoot(t *testing.T) {
	existing := []core.WikiTocNode{
		node("t1", "Classified Page", "article", "a1"),
		node("t2", "Created Mid-Pass", "article", "a2"), // not in the classify set
	}
	out, stats := buildOrganizedToc(existing,
		map[string]string{"a1": "Cat"},
		[]string{"Cat"}, map[string]bool{"a1": true}, "kb")
	// a1 filed into the new folder; a2 stays at the root for the next pass
	if len(out) != 2 || out[0].Type != "folder" || out[1].ArticleID != "a2" {
		t.Fatalf("mid-pass page must stay at root: %+v", out)
	}
	if stats.Other != 0 || stats.Moved != 1 {
		t.Fatalf("mid-pass page must not hit the fallback: moved=%d other=%d", stats.Moved, stats.Other)
	}
}

func TestEntriesBlockExcerpt(t *testing.T) {
	nodes := []core.WikiTocNode{node("t1", "Division", "article", "a1")}
	summaries := map[string]string{"a1": "香港美心集团 旗下\n分店网络 与业务分工"}
	block := entriesBlock(nodes, summaries)
	// excerpt flattened to one line, appended after the title
	if block != "a1. Division — 香港美心集团 旗下 分店网络 与业务分工\n" {
		t.Fatalf("unexpected block: %q", block)
	}
	// long summary is capped
	long := strings.Repeat("字", 200)
	if got := excerptLine(long, 120); len([]rune(got)) != 121 {
		t.Fatalf("excerpt not capped: %d runes", len([]rune(got)))
	}
	// no summary → bare title line, no dash
	if block2 := entriesBlock(nodes, nil); block2 != "a1. Division\n" {
		t.Fatalf("unexpected bare block: %q", block2)
	}
}
