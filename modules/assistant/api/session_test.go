/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"encoding/json"
	"testing"
)

func TestParseSearchResultPayloadKeepsQueryAndExtractsHits(t *testing.T) {
	message := `{"query":"这是用户的查询","result":{"took":22,"total":160,"hits":[{"id":"doc-1","title":"First"}]},"unused":"value"}`

	searchCtx, err := parseSearchResultPayload(message)
	if err != nil {
		t.Fatalf("parseSearchResultPayload returned error: %v", err)
	}
	if searchCtx.UserMessage != "这是用户的查询" {
		t.Fatalf("unexpected user message: got %q", searchCtx.UserMessage)
	}
	if searchCtx.ModelMessage != message {
		t.Fatalf("unexpected model message: got %q", searchCtx.ModelMessage)
	}

	hits, ok := searchCtx.FetchSource.([]interface{})
	if !ok {
		t.Fatalf("unexpected fetch source type: %T", searchCtx.FetchSource)
	}
	if len(hits) != 1 {
		t.Fatalf("unexpected hit count: got %d", len(hits))
	}
	first, ok := hits[0].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected hit type: %T", hits[0])
	}
	if first["id"] != "doc-1" {
		t.Fatalf("unexpected first hit id: got %v", first["id"])
	}
}

func TestParseSearchResultPayloadUsesArrayResultAsFetchSource(t *testing.T) {
	message := `{"query":"search text","result":[{"id":"doc-1"},{"id":"doc-2"}]}`

	searchCtx, err := parseSearchResultPayload(message)
	if err != nil {
		t.Fatalf("parseSearchResultPayload returned error: %v", err)
	}
	hits, ok := searchCtx.FetchSource.([]interface{})
	if !ok {
		t.Fatalf("unexpected fetch source type: %T", searchCtx.FetchSource)
	}
	if len(hits) != 2 {
		t.Fatalf("unexpected hit count: got %d", len(hits))
	}
}

func TestParseSearchResultPayloadAcceptsEncodedJSONString(t *testing.T) {
	message := `{"query":"搜索","result":{"hits":[{"id":"doc-1"}]}}`
	encodedMessage, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	searchCtx, err := parseSearchResultPayload(string(encodedMessage))
	if err != nil {
		t.Fatalf("parseSearchResultPayload returned error: %v", err)
	}
	if searchCtx.UserMessage != "搜索" {
		t.Fatalf("unexpected user message: got %q", searchCtx.UserMessage)
	}
	if searchCtx.ModelMessage != string(encodedMessage) {
		t.Fatalf("unexpected model message: got %q", searchCtx.ModelMessage)
	}

	hits, ok := searchCtx.FetchSource.([]interface{})
	if !ok {
		t.Fatalf("unexpected fetch source type: %T", searchCtx.FetchSource)
	}
	if len(hits) != 1 {
		t.Fatalf("unexpected hit count: got %d", len(hits))
	}
}

func TestParseSearchResultPayloadRequiresQuery(t *testing.T) {
	if _, err := parseSearchResultPayload(`{"result":{"total":160}}`); err == nil {
		t.Fatalf("expected error for search result payload without query")
	}
}
