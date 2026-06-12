/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"encoding/json"
	"testing"
)

func TestParseAskPayloadKeepsQueryAndExtractsHits(t *testing.T) {
	message := `{"query":"这是用户的查询","result":{"took":22,"total":160,"hits":[{"id":"doc-1","title":"First"}]},"unused":"value"}`

	askContext, err := parseAskPayload(message)
	if err != nil {
		t.Fatalf("parseAskPayload returned error: %v", err)
	}
	if askContext.UserMessage != "这是用户的查询" {
		t.Fatalf("unexpected user message: got %q", askContext.UserMessage)
	}
	if askContext.ModelMessage != message {
		t.Fatalf("unexpected model message: got %q", askContext.ModelMessage)
	}

	hits, ok := askContext.FetchSource.([]interface{})
	if !ok {
		t.Fatalf("unexpected fetch source type: %T", askContext.FetchSource)
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

func TestParseAskPayloadUsesArrayResultAsFetchSource(t *testing.T) {
	message := `{"query":"search text","result":[{"id":"doc-1"},{"id":"doc-2"}]}`

	askContext, err := parseAskPayload(message)
	if err != nil {
		t.Fatalf("parseAskPayload returned error: %v", err)
	}
	hits, ok := askContext.FetchSource.([]interface{})
	if !ok {
		t.Fatalf("unexpected fetch source type: %T", askContext.FetchSource)
	}
	if len(hits) != 2 {
		t.Fatalf("unexpected hit count: got %d", len(hits))
	}
}

func TestParseAskPayloadAcceptsEncodedJSONString(t *testing.T) {
	message := `{"query":"搜索","result":{"hits":[{"id":"doc-1"}]}}`
	encodedMessage, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	askContext, err := parseAskPayload(string(encodedMessage))
	if err != nil {
		t.Fatalf("parseAskPayload returned error: %v", err)
	}
	if askContext.UserMessage != "搜索" {
		t.Fatalf("unexpected user message: got %q", askContext.UserMessage)
	}
	if askContext.ModelMessage != string(encodedMessage) {
		t.Fatalf("unexpected model message: got %q", askContext.ModelMessage)
	}

	hits, ok := askContext.FetchSource.([]interface{})
	if !ok {
		t.Fatalf("unexpected fetch source type: %T", askContext.FetchSource)
	}
	if len(hits) != 1 {
		t.Fatalf("unexpected hit count: got %d", len(hits))
	}
}

func TestParseAskPayloadRequiresQuery(t *testing.T) {
	if _, err := parseAskPayload(`{"result":{"total":160}}`); err == nil {
		t.Fatalf("expected error for ask payload without query")
	}
}
