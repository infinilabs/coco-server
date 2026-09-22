/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"testing"

	"infini.sh/coco/core"
)

func TestValidateArticleStatus(t *testing.T) {
	for _, s := range []string{core.WikiArticleDraft, core.WikiArticleReviewed, core.WikiArticlePublished, core.WikiArticleArchived} {
		if err := validateArticleStatus(s); err != nil {
			t.Errorf("expected %q to be valid, got %v", s, err)
		}
	}
	for _, s := range []string{"", "live", "DRAFT"} {
		if err := validateArticleStatus(s); err == nil {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestStatusTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		ok       bool
	}{
		{core.WikiArticleDraft, core.WikiArticleReviewed, true},
		{core.WikiArticleDraft, core.WikiArticlePublished, false}, // publish requires review first
		{core.WikiArticleReviewed, core.WikiArticlePublished, true},
		{core.WikiArticlePublished, core.WikiArticleArchived, true},
		{core.WikiArticlePublished, core.WikiArticleDraft, false},
		{core.WikiArticleArchived, core.WikiArticleDraft, true}, // restore
	}
	for _, c := range cases {
		got := false
		for _, next := range statusTransitions[c.from] {
			if next == c.to {
				got = true
				break
			}
		}
		if got != c.ok {
			t.Errorf("transition %s -> %s: expected ok=%v", c.from, c.to, c.ok)
		}
	}
}

func TestValidateTocNodes(t *testing.T) {
	valid := []core.WikiTocNode{
		{ID: "n1", Title: "Getting Started", Type: "folder", Children: []core.WikiTocNode{
			{ID: "n2", Title: "Overview", Type: "article", ArticleID: "art-1"},
		}},
	}
	if err := validateTocNodes(valid); err != nil {
		t.Errorf("expected valid toc, got %v", err)
	}

	invalid := []struct {
		name  string
		nodes []core.WikiTocNode
	}{
		{"missing id", []core.WikiTocNode{{Title: "x", Type: "folder"}}},
		{"missing title", []core.WikiTocNode{{ID: "n1", Type: "folder"}}},
		{"dup id", []core.WikiTocNode{{ID: "n1", Title: "a", Type: "folder"}, {ID: "n1", Title: "b", Type: "folder"}}},
		{"bad type", []core.WikiTocNode{{ID: "n1", Title: "a", Type: "chapter"}}},
		{"article without article_id", []core.WikiTocNode{{ID: "n1", Title: "a", Type: "article"}}},
		{"nested invalid", []core.WikiTocNode{{ID: "n1", Title: "a", Type: "folder", Children: []core.WikiTocNode{{ID: "n2", Title: "b", Type: "article"}}}}},
	}
	for _, c := range invalid {
		if err := validateTocNodes(c.nodes); err == nil {
			t.Errorf("%s: expected error", c.name)
		}
	}
}

func TestChangeTypeFor(t *testing.T) {
	ai := &core.WikiArticle{AIGenerated: true}
	if got := changeTypeFor(ai); got != core.WikiChangeAIGenerated {
		t.Errorf("expected %s, got %s", core.WikiChangeAIGenerated, got)
	}
	human := &core.WikiArticle{}
	if got := changeTypeFor(human); got != core.WikiChangeHumanEdited {
		t.Errorf("expected %s, got %s", core.WikiChangeHumanEdited, got)
	}
}
