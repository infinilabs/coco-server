/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_metadata

import (
	"context"
	"encoding/json"
	"testing"

	"infini.sh/coco/core/codecs"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
)

// plumbing equivalence at the skip path: unsupported documents pass both
// conventions untouched (no connector resolution, no download).
func TestFileMetadata_RecordScopeSkip(t *testing.T) {
	rec, err := codecs.DocumentCodec{}.Decode([]byte(`{"id":"d1","title":"readme.md"}`))
	if err != nil {
		t.Fatal(err)
	}
	before := string(mustJSON(t, rec.Fields["document"]))

	cfg := pipeline.PipelineConfigV2{Name: "t"}
	ctx := pipeline.AcquireContext(cfg)
	ctx.Context = context.Background()
	ctx.Started()
	defer pipeline.ReleaseContext(ctx)
	ctx.Set(pipeline.RecordContextKey, rec)

	p := &FileMetadataProcessor{config: &Config{}}
	if err := p.Process(ctx); err != nil {
		t.Fatalf("record path errored: %v", err)
	}
	if before != string(mustJSON(t, rec.Fields["document"])) {
		t.Fatal("skip path mutated the record")
	}
	if err := p.ProcessBatch(ctx, []*event.Event{rec}); err != nil {
		t.Fatal(err)
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
