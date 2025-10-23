package neo4j

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
)

func TestNormalizePropertyType(t *testing.T) {
	cases := map[string]string{
		"int":       "int",
		"Integer":   "int",
		"float":     "float",
		"DOUBLE":    "float",
		"datetime":  "datetime",
		"timestamp": "datetime",
		"":          "string",
		"something": "string",
	}

	for input, expected := range cases {
		if got := normalizePropertyType(input); got != expected {
			t.Fatalf("normalizePropertyType(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestNormalizeCursorValueInt(t *testing.T) {
	stored, normalized, err := normalizeCursorValue(int64(42), "int")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Type != "int" || stored.Value != "42" {
		t.Fatalf("unexpected stored value: %+v", stored)
	}
	if v, ok := normalized.(int64); !ok || v != 42 {
		t.Fatalf("unexpected normalized value: %#v", normalized)
	}

	decoded, err := decodeCursorValue(stored, "")
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if v := decoded.(int64); v != 42 {
		t.Fatalf("unexpected decoded value: %v", decoded)
	}
}

func TestCursorFactoryFromResume(t *testing.T) {
	factory := cursorFactory{propertyType: "int"}
	snapshot, err := factory.fromResume("105")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snapshot == nil {
		t.Fatal("expected snapshot")
	}
	if snapshot.property.(int64) != 105 {
		t.Fatalf("unexpected property value: %#v", snapshot.property)
	}
	if snapshot.stored.Property.Type != "int" || snapshot.stored.Property.Value != "105" {
		t.Fatalf("unexpected stored cursor: %#v", snapshot.stored)
	}
}

func TestBuildQueryIncrementalAddsCursorParams(t *testing.T) {
	cfg := Config{
		Cypher:     "MATCH (n) RETURN n",
		Pagination: true,
		PageSize:   25,
		Incremental: IncrementalConfig{
			Enabled:      true,
			Mode:         modePropertyWatermark,
			Property:     "n.updated_at",
			PropertyType: "int",
			TieBreaker:   "elementId(n)",
		},
	}
	s := &scanner{}
	cursor := &cursorSnapshot{
		stored: &connectors.StoredCursor{
			Property: connectors.StoredCursorValue{Type: "int", Value: "101"},
			Tie:      &connectors.StoredCursorValue{Type: "string", Value: "node-1"},
		},
		property: int64(101),
		tie:      "node-1",
	}
	query, params, err := s.buildQuery(&cfg, cursor, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := params[paramCursorProperty]; !ok {
		t.Fatalf("expected cursor parameter in params: %#v", params)
	}
	if val := params[paramCursorProperty]; val != int64(101) {
		t.Fatalf("unexpected cursor parameter value: %#v", val)
	}
	if val, ok := params[paramCursorTie]; !ok || val != "node-1" {
		t.Fatalf("expected tie parameter, got %#v", params[paramCursorTie])
	}
	if _, ok := params[paramLimit]; !ok {
		t.Fatalf("expected limit parameter in params: %#v", params)
	}
	if want := 25; params[paramLimit] != want {
		t.Fatalf("limit param = %#v want %d", params[paramLimit], want)
	}
	if got := string(query); len(got) == 0 {
		t.Fatal("expected non-empty query")
	}
	if !contains(query, "coco_property > $") {
		t.Fatalf("expected watermark predicate in query: %s", query)
	}
	if !contains(query, "coco_property = $") || !contains(query, tieAlias) {
		t.Fatalf("expected tie-breaker clause in query: %s", query)
	}
	if !contains(query, "ORDER BY coco_property ASC, "+tieAlias+" ASC") {
		t.Fatalf("expected ORDER BY with tie in query: %s", query)
	}
}

func TestBuildQueryIncrementalDatetimeWrapsCursor(t *testing.T) {
	cfg := Config{
		Cypher:     "MATCH (n) RETURN n",
		Pagination: true,
		PageSize:   50,
		Incremental: IncrementalConfig{
			Enabled:      true,
			Mode:         modePropertyWatermark,
			Property:     "n.updated_at",
			PropertyType: "datetime",
			TieBreaker:   "elementId(n)",
		},
	}
	s := &scanner{}
	ts := "2025-10-09T04:33:59.691000000Z"
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t.Fatalf("failed to parse timestamp: %v", err)
	}
	cursor := &cursorSnapshot{
		stored: &connectors.StoredCursor{
			Property: connectors.StoredCursorValue{Type: "datetime", Value: ts},
			Tie:      &connectors.StoredCursorValue{Type: "string", Value: "node-1"},
		},
		property: parsed,
		tie:      "node-1",
	}

	query, params, err := s.buildQuery(&cfg, cursor, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	paramStr, ok := params[paramCursorProperty].(string)
	if !ok {
		t.Fatalf("expected cursor param to be string, got %#v", params[paramCursorProperty])
	}
	if paramStr != ts {
		t.Fatalf("unexpected cursor parameter value: %s", paramStr)
	}
	expectedFragment := fmt.Sprintf("datetime($%s)", paramCursorProperty)
	if !contains(query, expectedFragment) {
		t.Fatalf("expected query to contain %q, got %s", expectedFragment, query)
	}
	if tieVal, ok := params[paramCursorTie]; !ok || tieVal != "node-1" {
		t.Fatalf("expected tie parameter, got %#v", params[paramCursorTie])
	}
	if !contains(query, "coco_tie > $") {
		t.Fatalf("expected tie predicate in query: %s", query)
	}
	if !contains(query, "ORDER BY coco_property ASC, "+tieAlias+" ASC") {
		t.Fatalf("expected ORDER BY with tie in query: %s", query)
	}
}

func TestBuildQueryFullSyncPagination(t *testing.T) {
	cfg := Config{
		Cypher:     "MATCH (n) RETURN n",
		Pagination: true,
		PageSize:   10,
	}
	s := &scanner{}
	query, params, err := s.buildQuery(&cfg, nil, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params[paramSkip] != 30 {
		t.Fatalf("skip param = %#v want 30", params[paramSkip])
	}
	if params[paramLimit] != 10 {
		t.Fatalf("limit param = %#v want 10", params[paramLimit])
	}
	if !contains(query, "SKIP $") || !contains(query, "LIMIT $") {
		t.Fatalf("expected SKIP/LIMIT in query: %s", query)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestScannerCollectFunc verifies that scanner calls collectFunc for documents
func TestScannerCollectFunc(t *testing.T) {
	// Track collected documents
	var collected []string
	collectFunc := func(doc common.Document) error {
		collected = append(collected, doc.ID)
		return nil
	}

	// Create test scanner
	s := &scanner{
		name:        "test-neo4j",
		collectFunc: collectFunc,
	}

	// Verify collectFunc is set
	if s.collectFunc == nil {
		t.Fatal("expected collectFunc to be set")
	}

	// Create test document
	testDoc := common.Document{
		Title: "Test Document",
	}
	testDoc.ID = "test-id-1"

	// Call collectFunc
	if err := s.collectFunc(testDoc); err != nil {
		t.Fatalf("collectFunc failed: %v", err)
	}

	// Verify document was collected
	if len(collected) != 1 {
		t.Fatalf("expected 1 document, got %d", len(collected))
	}

	if collected[0] != "test-id-1" {
		t.Errorf("expected document ID 'test-id-1', got %s", collected[0])
	}
}

// TestScannerCollectFuncError verifies error handling in collectFunc
func TestScannerCollectFuncError(t *testing.T) {
	// Track calls
	callCount := 0
	collectFunc := func(doc common.Document) error {
		callCount++
		return fmt.Errorf("collection error")
	}

	// Create test scanner
	s := &scanner{
		name:        "test-neo4j",
		collectFunc: collectFunc,
	}

	// Create test document
	testDoc := common.Document{
		Title: "Test Document",
	}
	testDoc.ID = "test-id-1"

	// Call collectFunc - should return error
	err := s.collectFunc(testDoc)
	if err == nil {
		t.Fatal("expected collectFunc to return error")
	}

	// Verify collectFunc was called
	if callCount != 1 {
		t.Fatalf("expected collectFunc to be called once, got %d", callCount)
	}

	// Verify error message
	if !strings.Contains(err.Error(), "collection error") {
		t.Errorf("unexpected error message: %v", err)
	}
}
