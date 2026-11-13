/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package extract_entities

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

// Live failure mode on 2026-11-08: ~14% of GLM extraction responses were
// slightly malformed JSON (trailing commas, missing commas between adjacent
// objects) and each failure cost a full doc re-extraction. The repair pass
// must recover all of these, and must NOT mangle quotes containing the
// same byte pairs.
func TestParseEntitiesFromResponseLooseJSON(t *testing.T) {
	// trailing comma after last array element
	result, err := parseEntitiesFromResponse(`{"entities":[{"name":"A","type":"concept"},]}`)
	if err != nil || len(result.Entities) != 1 {
		t.Fatalf("trailing comma not repaired: %v %v", result, err)
	}
	// trailing comma in object
	result, err = parseEntitiesFromResponse(`{"entities":[{"name":"A","type":"concept",}]}`)
	if err != nil || len(result.Entities) != 1 {
		t.Fatalf("object trailing comma not repaired: %v %v", result, err)
	}
	// missing comma between adjacent array objects (the observed error:
	// "invalid character '{' looking for beginning of object key string")
	result, err = parseEntitiesFromResponse(`{"entities":[{"name":"A","type":"concept"} {"name":"B","type":"product"}]}`)
	if err != nil || len(result.Entities) != 2 {
		t.Fatalf("missing comma between objects not repaired: %v %v", result, err)
	}
	if result.Entities[0].Name != "A" || result.Entities[1].Name != "B" {
		t.Fatalf("unexpected entities: %+v", result.Entities)
	}
	// missing comma between two string array elements
	result, err = parseEntitiesFromResponse(`{"entities":[{"name":"A","type":"concept","aliases":["x" "y"]}]}`)
	if err != nil || len(result.Entities[0].Aliases) != 2 {
		t.Fatalf("missing comma between strings not repaired: %v %v", result, err)
	}
	// multi-char literals must not get commas injected mid-token
	if _, err := parseEntitiesFromResponse(`{"entities":[{"name":"A","type":"concept","properties":{"n":123,"ok":true,"x":null}}]}`); err != nil {
		t.Fatalf("valid numeric/bool/null literals broken by repair: %v", err)
	}
	// quotes containing }{ byte pairs must be preserved untouched
	result, err = parseEntitiesFromResponse(`{"entities":[{"name":"A","type":"concept","evidence":"code }{ block"}]}`)
	if err != nil || result.Entities[0].Evidence != "code }{ block" {
		t.Fatalf("in-string braces mangled: %v %v", result, err)
	}
}

func TestBuildExtractionMaterial(t *testing.T) {
	// no real text — folder nodes / extraction-dead docs must not extract
	empty := core.Document{}
	empty.Title = "T"
	empty.Summary = "S"
	if got := buildExtractionMaterial(&empty); got != "" {
		t.Errorf("expected empty material without content, got %q", got)
	}
	folder := core.Document{}
	folder.Type = "folder"
	folder.Content = "some text"
	if got := buildExtractionMaterial(&folder); got != "" {
		t.Errorf("folder docs must not extract, got %q", got)
	}
	// content-grounded: content is the body, title/summary are context
	doc := core.Document{}
	doc.Title = "T"
	doc.Summary = "S"
	doc.Content = "C"
	got := buildExtractionMaterial(&doc)
	if !strings.Contains(got, "Title: T") || !strings.Contains(got, "Summary: S") || !strings.Contains(got, "Content:\nC") {
		t.Errorf("unexpected material: %q", got)
	}
	// over-limit content is truncated, not rejected
	long := core.Document{}
	long.Content = strings.Repeat("x", maxContentForExtraction+100)
	if got := buildExtractionMaterial(&long); len([]rune(got)) == 0 {
		t.Errorf("long content should truncate to a material, got empty")
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

func TestEntityTypeCanonicalization(t *testing.T) {
	// LLM emits "Organization" (capitalized), schema declares "organization"
	// (lowercase) — the stored type must use the DECLARED casing
	schema := &wiki.OntologySchemaDoc{
		EntityTypes: []wiki.OntologyEntityTypeDef{
			{Name: "organization"},
			{Name: "person"},
		},
	}
	allowed := make([]string, 0, len(schema.EntityTypes))
	for i := range schema.EntityTypes {
		allowed = append(allowed, schema.EntityTypes[i].Name)
	}

	// LLM emits "Organization" (capitalized) — stored as the DECLARED casing
	assert.Equal(t, "organization", canonicalizeEntityType("Organization", allowed, schema))
	assert.Equal(t, "person", canonicalizeEntityType("PERSON", allowed, schema))

	// unknown types degrade to concept
	assert.Equal(t, "concept", canonicalizeEntityType("Spaceship", allowed, schema))
	assert.Equal(t, "concept", canonicalizeEntityType("", allowed, schema))
}

func TestChunkLocator(t *testing.T) {
	doc := core.Document{}
	doc.Chunks = []core.DocumentChunk{
		{Text: "品牌与营销 首页 介绍"}, {Text: "美心 流心奶黄 月饼 产品目录, Dedicated vouchers for Maxim's Cake"}, {Text: "门店地址"},
	}
	if got := chunkLocator(&doc, "Dedicated vouchers for Maxim's Cake and Tohato"); got != "块 2/3" {
		t.Errorf("expected ordinal locator, got %q", got)
	}
	doc.Chunks[0].Breadcrumb = "年报 > 财务"
	if got := chunkLocator(&doc, "介绍"); got != "年报 > 财务" {
		t.Errorf("expected breadcrumb locator, got %q", got)
	}
	// paraphrased evidence: head fragment still matches
	if got := chunkLocator(&doc, "Dedicated vouchers for Maxim's"); got != "块 2/3" {
		t.Errorf("expected head-fragment match, got %q", got)
	}
	// no chunks / no evidence → empty
	if got := chunkLocator(&core.Document{}, "x"); got != "" {
		t.Errorf("expected empty locator without chunks, got %q", got)
	}
}

// verbatim-evidence quotes/newlines inside string values (the second live
// failure mode) must be escaped, not lost
func TestParseEntitiesFromResponseRawStrings(t *testing.T) {
	// unescaped inner quotes + a raw newline inside evidence
	resp := `{"entities":[{"name":"Assistant API","type":"document","evidence":"Assistant API Reference #  Below "the line" continues
with text"}]}`
	result, err := parseEntitiesFromResponse(resp)
	if err != nil {
		t.Fatalf("raw-string response not repaired: %v", err)
	}
	if len(result.Entities) != 1 || result.Entities[0].Name != "Assistant API" {
		t.Fatalf("unexpected entities: %+v", result.Entities)
	}
	// a genuine close followed by the next key still parses normally
	result, err = parseEntitiesFromResponse(`{"entities":[{"name":"A","type":"concept","evidence":"clean evidence"}]}`)
	if err != nil || result.Entities[0].Evidence != "clean evidence" {
		t.Fatalf("clean response mangled: %v %v", result, err)
	}
}

// the dominant structural failure observed live (2026-11-08): relation
// objects dropped at entity-object key positions instead of inside a
// relations array
func TestParseEntitiesFromResponseStrayRelations(t *testing.T) {
	for _, resp := range []string{
		// well-formed except the stray relation object
		`{"entities":[{"name":"Coco Server","type":"product","aliases":["Coco-Server"],"properties":{"status":"active","description":"Retrieval-augmented chat platform"},{"relation":"makes","target":"Assistant"}}]}`,
		// ...and the variant that also omits the entity object's close
		`{"entities":[{"name":"Coco Server","type":"product","aliases":["Coco-Server"],"properties":{"status":"active","description":"Retrieval-augmented chat platform"},{"relation":"makes","target":"Assistant"}]}`,
	} {
		result, err := parseEntitiesFromResponse(resp)
		if err != nil {
			t.Fatalf("stray relation object not hoisted: %v", err)
		}
		e := result.Entities[0]
		if len(e.Properties) != 2 {
			t.Fatalf("expected 2 properties, got %v", e.Properties)
		}
		if len(e.Relations) != 1 || e.Relations[0].Relation != "makes" || e.Relations[0].Target != "Assistant" {
			t.Fatalf("expected hoisted relation, got %+v", e.Relations)
		}
	}
	// a real relations array must survive untouched
	resp := `{"entities":[{"name":"A","type":"concept","properties":{"k":"v"},"relations":[{"relation":"uses","target":"B"}]}]}`
	result, err := parseEntitiesFromResponse(resp)
	if err != nil || len(result.Entities[0].Relations) != 1 || result.Entities[0].Relations[0].Relation != "uses" {
		t.Fatalf("declared relations damaged: %v %+v", err, result)
	}
}
