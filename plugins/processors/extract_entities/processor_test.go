/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package extract_entities

import (
	"testing"

	"infini.sh/coco/core"
)

func TestParseEntitiesFromResponse(t *testing.T) {
	resp := "Here is the extraction:\n```json\n" +
		`{"entities":[{"name":"Maxim's HK Store 001","type":"store","aliases":["HK001"],"properties":{"region":"HK"},"relations":[{"relation":"promotes","target":"Autumn Promo"}],"evidence":"Store 001 promotes the autumn campaign."}]}` +
		"\n```"
	result, err := parseEntitiesFromResponse(resp)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(result.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(result.Entities))
	}
	e := result.Entities[0]
	if e.Name != "Maxim's HK Store 001" || e.Type != "store" {
		t.Errorf("unexpected entity: %+v", e)
	}
	if len(e.Aliases) != 1 || e.Aliases[0] != "HK001" {
		t.Errorf("unexpected aliases: %v", e.Aliases)
	}
	if len(e.Relations) != 1 || e.Relations[0].Target != "Autumn Promo" {
		t.Errorf("unexpected relations: %v", e.Relations)
	}
}

func TestParseEntitiesFromResponseEmpty(t *testing.T) {
	for _, resp := range []string{"", "no json at all", "[1,2,3]"} {
		if _, err := parseEntitiesFromResponse(resp); err == nil {
			t.Errorf("expected error for %q", resp)
		}
	}
	if result, err := parseEntitiesFromResponse(`{"entities":[]}`); err != nil || len(result.Entities) != 0 {
		t.Errorf("empty entities should parse cleanly: %v %v", result, err)
	}
}

func TestNormalizeAliases(t *testing.T) {
	got := normalizeAliases([]string{" A ", "", "A", "B"})
	if len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Errorf("unexpected aliases: %v", got)
	}
}

func TestBuildExtractionMaterial(t *testing.T) {
	doc := core.Document{}
	doc.Title = "T"
	doc.Summary = "S"
	if got := buildExtractionMaterial(&doc); got != "Title: T\n\nSummary: S" {
		t.Errorf("unexpected material: %q", got)
	}
	empty := core.Document{}
	if got := buildExtractionMaterial(&empty); got != "" {
		t.Errorf("expected empty material, got %q", got)
	}
}
