/* Copyright © INFINI LTD. All rights reserved. */

package generate_cover

import (
	"context"
	"encoding/json"
	"testing"

	"infini.sh/coco/core/codecs"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

// skip-path equivalence: documents without a supported file connector pass
// both conventions untouched (connector resolution fails → skip).
func TestGenerateDocumentCoverProcessorDualModeSkip(t *testing.T) {
	rec, err := codecs.DocumentCodec{}.Decode([]byte(`{"id":"d1","title":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	before := string(mustJSON(t, rec.Fields["document"]))

	c, err := config.NewConfigFrom(util.MapStr{})
	if err != nil {
		t.Fatal(err)
	}
	p, err := New(c)
	if err != nil {
		t.Fatal(err)
	}

	cfg := pipeline.PipelineConfigV2{Name: "t"}
	ctx := pipeline.AcquireContext(cfg)
	ctx.Context = context.Background()
	ctx.Started()
	defer pipeline.ReleaseContext(ctx)

	bp, ok := p.(pipeline.BatchProcessor)
	if !ok {
		t.Fatal("processor does not implement BatchProcessor")
	}
	if err := bp.ProcessBatch(ctx, []*event.Event{rec}); err != nil {
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
