/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package extract_entities

import (
	"strings"
	"testing"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/wiki"
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

func TestBuildExtractionPromptWithSchema(t *testing.T) {
	schema := &wiki.OntologySchemaDoc{EntityTypes: []wiki.OntologyEntityTypeDef{
		{
			Name: "product", Label: "产品",
			Properties: []wiki.OntologyPropertyDef{
				{Key: "code", Type: "string", Required: true},
				{Key: "status", Type: "enum", Enum: []string{"active", "beta"}},
			},
			Relations: []wiki.OntologyRelationDef{
				{Name: "made_by", TargetType: "organization", Cardinality: "one"},
				{Name: "depends_on", TargetType: "product"},
			},
		},
	}}
	cfg := &Config{EntityTypes: []string{"legacy"}, MaxEntities: 5, LLMGenerationLang: "zh-CN"}
	prompt := buildExtractionPrompt("doc body", cfg, schema)

	if !strings.Contains(prompt, "product (产品)") {
		t.Errorf("prompt missing typed vocabulary: %s", prompt)
	}
	if !strings.Contains(prompt, "properties: code!, status") {
		t.Errorf("prompt missing declared properties: %s", prompt)
	}
	if !strings.Contains(prompt, "relations: made_by->organization (single), depends_on->product") {
		t.Errorf("prompt missing relation vocabulary: %s", prompt)
	}
	if !strings.Contains(prompt, "Use ONLY the listed relation names") {
		t.Errorf("prompt missing schema constraint line")
	}

	// without a schema the flat POC list stays
	prompt = buildExtractionPrompt("doc body", cfg, nil)
	if !strings.Contains(prompt, "[\"legacy\"]") {
		t.Errorf("flat fallback missing: %s", prompt)
	}
}
