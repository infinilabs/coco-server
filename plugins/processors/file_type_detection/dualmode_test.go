/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_type_detection

import (
	"context"
	"testing"

	codecs "infini.sh/coco-codecs"
	"infini.sh/framework/core/pipeline"
)

// file_type_detection needs connector metadata the test env cannot fetch,
// so exercise the skip path (no connector resolution) — plumbing only.
func TestFileTypeDetection_RecordScopeSkip(t *testing.T) {
	rec, err := codecs.DocumentCodec{}.Decode([]byte(`{"id":"d1","title":"notes.txt"}`))
	if err != nil {
		t.Fatal(err)
	}
	cfg := pipeline.PipelineConfigV2{Name: "t"}
	ctx := pipeline.AcquireContext(cfg)
	ctx.Context = context.Background()
	ctx.Started()
	defer pipeline.ReleaseContext(ctx)
	ctx.Set(pipeline.RecordContextKey, rec)

	p := &FileTypeDetectionProcessor{config: &Config{}}
	if err := p.Process(ctx); err != nil {
		t.Fatalf("record path errored: %v", err)
	}
	if err := p.ProcessBatch(ctx, nil); err != nil {
		t.Fatal(err)
	}
}
