/* Copyright © INFINI LTD. All rights reserved. */

package face_extraction

import (
	"context"
	"encoding/json"
	"testing"

	codecs "infini.sh/coco-codecs"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
)

// skip-path equivalence: documents without a supported file connector pass
// both conventions untouched (connector resolution fails → skip).
func TestFaceExtractionProcessorDualModeSkip(t *testing.T) {
	rec, err := codecs.DocumentCodec{}.Decode([]byte(`{"id":"d1","title":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	before := string(mustJSON(t, rec.Fields["document"]))

	p := &FaceExtractionProcessor{config: &Config{}}

	cfg := pipeline.PipelineConfigV2{Name: "t"}
	ctx := pipeline.AcquireContext(cfg)
	ctx.Context = context.Background()
	ctx.Started()
	defer pipeline.ReleaseContext(ctx)

	if err := p.ProcessBatch(ctx, []*event.Event{rec}); err != nil {
		t.Fatalf("ProcessBatch skip path errored: %v", err)
	}
	ctx.Set(pipeline.RecordContextKey, rec)
	if err := p.Process(ctx); err != nil {
		t.Fatalf("record path errored: %v", err)
	}
	if before != string(mustJSON(t, rec.Fields["document"])) {
		t.Fatal("skip path mutated the record")
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
