/* Copyright © INFINI LTD.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package pipeline

import (
	"context"
	"strings"
	"testing"

	"infini.sh/coco/core"
	"infini.sh/framework/core/config"
	fpipeline "infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

// stub processors registered per-test-suite: they operate on the document
// stream exactly like coco's enrichment processors do (read "messages",
// transform the document JSON, write "messages" back).

type studioStubProcessor struct {
	name   string
	mutate func(pctx *fpipeline.Context) error
}

func (p *studioStubProcessor) Name() string { return p.name }

func (p *studioStubProcessor) Process(pctx *fpipeline.Context) error {
	return p.mutate(pctx)
}

func readFirstDoc(t *testing.T, pctx *fpipeline.Context) util.MapStr {
	t.Helper()
	msgs, ok := pctx.Get(core.PipelineContextDocuments).([]queue.Message)
	if !ok || len(msgs) == 0 {
		t.Fatalf("no messages in context")
	}
	var doc util.MapStr
	if err := util.FromJSONBytes(msgs[0].Data, &doc); err != nil {
		t.Fatalf("message payload is not JSON: %v", err)
	}
	return doc
}

func writeMessages(pctx *fpipeline.Context, msgs []queue.Message) {
	pctx.Set(core.PipelineContextDocuments, msgs)
}

func registerStub(t *testing.T, name string, mutate func(pctx *fpipeline.Context) error) {
	t.Helper()
	fpipeline.RegisterProcessorPlugin(name, func(cfg *config.Config) (fpipeline.Processor, error) {
		return &studioStubProcessor{name: name, mutate: mutate}, nil
	})
	t.Cleanup(func() {
		// The registry is process-global; nothing to deregister with, so the
		// names are test-unique to avoid cross-test collisions.
	})
}

func TestBuildNamedProcessor(t *testing.T) {
	registerStub(t, "studio_upper_stub", func(pctx *fpipeline.Context) error { return nil })

	if _, err := buildNamedProcessor(map[string]interface{}{"no_such_processor": map[string]interface{}{}}); err == nil {
		t.Fatal("unknown processor should fail")
	}
	if _, err := buildNamedProcessor(map[string]interface{}{"a": map[string]interface{}{}, "b": map[string]interface{}{}}); err == nil {
		t.Fatal("multi-key entry should fail")
	}
	if _, err := buildNamedProcessor(map[string]interface{}{"studio_upper_stub": map[string]interface{}{}}); err != nil {
		t.Fatalf("registered processor should build: %v", err)
	}
	if _, err := buildNamedProcessor(map[string]interface{}{"if": map[string]interface{}{"equals": map[string]interface{}{"lang": "en"}}, "then": []interface{}{map[string]interface{}{"studio_upper_stub": map[string]interface{}{}}}}); err != nil {
		t.Fatalf("if/then/else entry should build: %v", err)
	}
}

func TestReplayChainTracedSnapshots(t *testing.T) {
	registerStub(t, "studio_set_lang", func(pctx *fpipeline.Context) error {
		doc := readFirstDoc(t, pctx)
		doc["lang"] = "en"
		writeMessages(pctx, []queue.Message{{Data: util.MustToJSONBytes(doc)}})
		return nil
	})
	registerStub(t, "studio_panic", func(pctx *fpipeline.Context) error {
		panic("boom")
	})
	registerStub(t, "studio_expand", func(pctx *fpipeline.Context) error {
		msgs, _ := pctx.Get(core.PipelineContextDocuments).([]queue.Message)
		var doc util.MapStr
		if err := util.FromJSONBytes(msgs[0].Data, &doc); err != nil {
			return err
		}
		out := []queue.Message{}
		for _, tag := range []string{"a", "b"} {
			clone := util.MapStr{}
			for k, v := range doc {
				clone[k] = v
			}
			clone["tag"] = tag
			out = append(out, queue.Message{Data: util.MustToJSONBytes(clone)})
		}
		writeMessages(pctx, out)
		return nil
	})

	chain := []map[string]interface{}{
		{"studio_set_lang": map[string]interface{}{}},
		{"studio_panic": map[string]interface{}{}},
		{"studio_expand": map[string]interface{}{}},
	}
	docs := []util.MapStr{{"title": "hello", "content": "world"}}

	steps, finals, err := replayChainTraced(context.Background(), chain, docs)
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	if len(steps) != 1 || len(steps[0]) != 3 {
		t.Fatalf("expected 3 steps for 1 document, got %v", steps)
	}

	s1, s2, s3 := steps[0][0], steps[0][1], steps[0][2]
	if s1.Name != "studio_set_lang" || s1.ErrorStr != "" {
		t.Fatalf("step1 unexpected: %+v", s1)
	}
	if got := s1.Fields["lang"]; got != "en" {
		t.Fatalf("step1 should snapshot the mutated field, got %v", got)
	}
	if s1.Count != 1 {
		t.Fatalf("step1 count = %d, want 1", s1.Count)
	}
	if !strings.Contains(s2.ErrorStr, "panic") {
		t.Fatalf("step2 should capture the panic as an error, got %+v", s2)
	}
	if s3.Count != 2 {
		t.Fatalf("step3 should see the expanded stream, count = %d", s3.Count)
	}

	if len(finals) != 1 {
		t.Fatalf("expected 1 final, got %d", len(finals))
	}
	if finals[0]["lang"] != "en" {
		t.Fatalf("final should carry the enriched field: %+v", finals[0])
	}

	// Snapshot isolation: the step-1 snapshot must not be mutated by the
	// expand step that ran afterwards (snapshots parse from bytes).
	if s1.Count != 1 || s1.Fields["tag"] != nil {
		t.Fatalf("step1 snapshot was aliased by a later processor: %+v", s1)
	}
}

func TestReplayChainInvalidEntry(t *testing.T) {
	_, _, err := replayChainTraced(context.Background(),
		[]map[string]interface{}{{"no_such_processor": map[string]interface{}{}}},
		[]util.MapStr{{"title": "x"}})
	if err == nil {
		t.Fatal("invalid chain should fail")
	}
}
