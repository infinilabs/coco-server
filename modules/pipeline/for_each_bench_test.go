/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infini.ltd */

package pipeline

import (
	"encoding/json"
	"fmt"
	"testing"

	"infini.sh/coco/core"
	"infini.sh/coco/core/codecs"
	_ "infini.sh/coco/core/codecs" // register the coco_document record codec
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/param"
	pipeline "infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	pipmod "infini.sh/framework/modules/pipeline"
)

// P4 benchmark (#75), coco side: the real document codec with its cold
// document_chunk subtree versus the legacy messages path (full typed
// Document decode + re-encode per processor hop).
//
// The payload is a ~1MB document where the chunk embeddings make up most
// of the bytes — the shape the L2 cold/hot split exists for. The records
// bench never float-parses them; the messages bench pays for all of them
// on every hop.

// benchNoop touches one hot field and returns — the chain shape that
// isolates data movement.
type benchNoop struct{}

func (benchNoop) Name() string { return "bench_noop" }
func (benchNoop) Process(c *pipeline.Context) error {
	rec, _ := pipeline.CurrentRecord(c)
	if rec == nil {
		return nil
	}
	if doc, ok := rec.Fields["document"].(map[string]interface{}); ok {
		_ = doc["title"]
	}
	return nil
}

func init() {
	// module-setup registrations a test binary must replicate
	pipeline.RegisterDomainProcessorWithConfigMetadata("event", "for_each", pipmod.NewForEachProcessor, pipmod.ForEachConfig{})
	pipeline.RegisterProcessorPlugin("bench_noop", func(*config.Config) (pipeline.Processor, error) { return benchNoop{}, nil })
}

// benchDocPayload builds a ~1MB document JSON: hot text fields plus a
// document_chunk array whose embeddings dominate the byte count.
func benchDocPayload(target int) []byte {
	chunk := func(i, contentLen int) map[string]interface{} {
		// production shape: dimension-keyed float array (core.Embedding),
		// sized so embeddings dominate the payload as they do in real
		// embedded documents
		embedding := make([]float64, 1536)
		for j := range embedding {
			embedding[j] = float64(i+j%7) * 0.000001
		}
		content := make([]byte, contentLen)
		unit := []byte("chunk text ")
		for p := 0; p < contentLen; p += len(unit) {
			copy(content[p:], unit)
		}
		return map[string]interface{}{
			"id":            fmt.Sprintf("chunk-%d", i),
			"content":       string(content),
			"embedding1536": embedding,
		}
	}
	content := make([]byte, target/3)
	unit := []byte("document body text ")
	for p := 0; p < len(content); p += len(unit) {
		copy(content[p:], unit)
	}
	chunks := make([]interface{}, 0, 32)
	for i := 0; i < 32; i++ {
		chunks = append(chunks, chunk(i, 1024))
	}
	doc := map[string]interface{}{
		"id":             "bench-doc",
		"title":          "Benchmark Document",
		"content":        string(content),
		"status":         "completed",
		"document_chunk": chunks,
		"source":         map[string]interface{}{"id": "bench-src", "type": "test"},
	}
	b, _ := json.Marshal(doc)
	return b
}

func newBenchForEach(b *testing.B) *pipmod.ForEachProcessor {
	b.Helper()
	cfgMap := util.MapStr{
		"codec": "coco_document",
		"processor": []interface{}{
			util.MapStr{"bench_noop": util.MapStr{}},
			util.MapStr{"bench_noop": util.MapStr{}},
			util.MapStr{"bench_noop": util.MapStr{}},
			util.MapStr{"bench_noop": util.MapStr{}},
		},
	}
	c, err := config.NewConfigFrom(cfgMap)
	if err != nil {
		b.Fatal(err)
	}
	p, err := pipmod.NewForEachProcessor(c)
	if err != nil {
		b.Fatalf("NewForEachProcessor: %v", err)
	}
	return p.(*pipmod.ForEachProcessor)
}

// BenchmarkForEach_Records_1MB_K4: decode → 4 no-op processors → encode
// through the real codec. The document_chunk subtree rides as raw bytes.
func BenchmarkForEach_Records_1MB_K4(b *testing.B) {
	payload := benchDocPayload(1 << 20)
	p := newBenchForEach(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msgs := []queue.Message{{Data: payload, Size: len(payload), Offset: queue.AcquireOffset(0, int64(i))}}
		ctx := &pipeline.Context{Context: b.Context()}
		ctx.Set(param.ParaKey("messages"), msgs)
		if err := p.Process(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLegacy_Messages_1MB_FullTypedRoundTrip: the messages-path
// equivalent — every hop pays a full typed decode (embeddings float-parsed)
// and re-encode of the same document.
func BenchmarkLegacy_Messages_1MB_FullTypedRoundTrip(b *testing.B) {
	payload := benchDocPayload(1 << 20)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for hop := 0; hop < 4; hop++ {
			var doc core.Document
			if err := util.FromJSONBytes(payload, &doc); err != nil {
				b.Fatal(err)
			}
			_ = doc.Title
			payload2 := util.MustToJSONBytes(doc)
			payload = payload2
		}
	}
}

// BenchmarkForEach_Records_1MB_K4_NoColdSplit is the counterfactual for
// the L2 cold/hot split (#71): same chain, but document_chunk is parsed
// into the field tree like any hot field. The delta against
// BenchmarkForEach_Records_1MB_K4 is what the split saves on the data
// movement itself.
func BenchmarkForEach_Records_1MB_K4_NoColdSplit(b *testing.B) {
	saved := codecs.ColdSubtrees
	codecs.ColdSubtrees = nil
	defer func() { codecs.ColdSubtrees = saved }()
	payload := benchDocPayload(1 << 20)
	p := newBenchForEach(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msgs := []queue.Message{{Data: payload, Size: len(payload), Offset: queue.AcquireOffset(0, int64(i))}}
		ctx := &pipeline.Context{Context: b.Context()}
		ctx.Set(param.ParaKey("messages"), msgs)
		if err := p.Process(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
