/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package extract_tags

import (
	"context"
	"encoding/json"
	"testing"

	"infini.sh/coco/core/codecs"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
)

func codecsDocumentDecode(t *testing.T, payload string) (*event.Event, error) {
	t.Helper()
	return codecs.DocumentCodec{}.Decode([]byte(payload))
}

func testCtx(t *testing.T) *pipeline.Context {
	t.Helper()
	cfg := pipeline.PipelineConfigV2{Name: "dualmode-test"}
	ctx := pipeline.AcquireContext(cfg)
	ctx.Context = context.Background()
	ctx.Started()
	return ctx
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Dual-mode plumbing equivalence: records without ai_insights pass through
// BOTH conventions untouched (skip path — the LLM is never invoked), and
// the record convention leaves Fields authoritative.
func TestExtractTags_DualModeSkipEquivalence(t *testing.T) {
	payload := `{"id":"d1","title":"T","content":"C","ai_insights":{"text":""},"tags":["old"]}`
	rec, err := codecsDocumentDecode(t, payload)
	if err != nil {
		t.Fatal(err)
	}
	before := string(mustJSON(t, rec.Fields["document"]))

	p := &ExtractTagsProcessor{config: &Config{}}
	// skip path must not error and must not touch the record
	if err := p.ProcessBatch(testCtx(t), []*event.Event{rec}); err != nil {
		t.Fatalf("ProcessBatch skip path errored: %v", err)
	}
	if before != string(mustJSON(t, rec.Fields["document"])) {
		t.Fatal("skip path mutated the record")
	}
}

// record-scope fallback (nested batch_capable degradation)
func TestExtractTags_RecordScopeFallback(t *testing.T) {
	payload := `{"id":"d2","title":"T2","content":"C2"}`
	rec, _ := codecsDocumentDecode(t, payload)
	ctx := testCtx(t)
	ctx.Set(pipeline.RecordContextKey, rec)
	p := &ExtractTagsProcessor{config: &Config{}}
	if err := p.Process(ctx); err != nil {
		t.Fatalf("record fallback errored: %v", err)
	}
}
