/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package extract_entities

import (
	"context"
	"encoding/json"
	"testing"

	"infini.sh/coco/core/codecs"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
)

// skip-path equivalence: no extractable text → both conventions leave the
// record untouched, LLM never resolved (lazy).
func TestExtractEntities_DualModeSkipEquivalence(t *testing.T) {
	rec, err := codecs.DocumentCodec{}.Decode([]byte(`{"id":"d1","title":"empty"}`))
	if err != nil {
		t.Fatal(err)
	}
	before := string(mustJSON(t, rec.Fields["document"]))
	cfg := pipeline.PipelineConfigV2{Name: "t"}
	ctx := pipeline.AcquireContext(cfg)
	ctx.Context = context.Background()
	ctx.Started()

	p := &ExtractEntitiesProcessor{config: &Config{}}
	if err := p.ProcessBatch(ctx, []*event.Event{rec}); err != nil {
		t.Fatalf("ProcessBatch skip path errored: %v", err)
	}
	ctx.Set(pipeline.RecordContextKey, rec)
	if err := p.Process(ctx); err != nil {
		t.Fatalf("record fallback errored: %v", err)
	}
	if before != string(mustJSON(t, rec.Fields["document"])) {
		t.Fatal("skip path mutated the record")
	}
	pipeline.ReleaseContext(ctx)
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
