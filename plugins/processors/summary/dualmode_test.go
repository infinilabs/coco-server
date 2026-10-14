/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package summary

import (
	"context"
	"encoding/json"
	"testing"

	codecs "infini.sh/coco-codecs"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
)

// skip-path equivalence: short / empty-content documents pass both
// conventions untouched without requiring LLM configuration (lazy resolve).
func TestSummary_DualModeSkipEquivalence(t *testing.T) {
	for _, payload := range []string{
		`{"id":"d1","title":"short","content":"hi"}`,                // below min length
		`{"id":"d2","title":"folder","type":"folder","content":""}`, // empty-text guard
	} {
		rec, err := codecs.DocumentCodec{}.Decode([]byte(payload))
		if err != nil {
			t.Fatal(err)
		}
		before := string(mustJSON(t, rec.Fields["document"]))

		cfg := pipeline.PipelineConfigV2{Name: "t"}
		ctx := pipeline.AcquireContext(cfg)
		ctx.Context = context.Background()
		ctx.Started()

		p := &DocumentSummarizationProcessor{config: &Config{MinInputDocumentLength: 100}}
		if err := p.ProcessBatch(ctx, []*event.Event{rec}); err != nil {
			t.Fatalf("ProcessBatch skip path errored: %v", err)
		}
		ctx.Set(pipeline.RecordContextKey, rec)
		if err := p.Process(ctx); err != nil {
			t.Fatalf("record fallback errored: %v", err)
		}
		if before != string(mustJSON(t, rec.Fields["document"])) {
			t.Fatalf("skip path mutated the record: %s", payload)
		}
		pipeline.ReleaseContext(ctx)
	}
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
