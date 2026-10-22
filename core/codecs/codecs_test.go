/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package codecs

import (
	"encoding/json"
	"reflect"
	"testing"

	"infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

// documentFixture mirrors a realistic core.Document JSON: ORM base keys,
// source reference, hot text fields and the heavy chunk subtree (as it
// would arrive from the queue).
func documentFixture() []byte {
	return []byte(`{
		"id": "doc-42",
		"created": "2026-10-01T00:00:00Z",
		"updated": "2026-10-02T03:04:05Z",
		"title": "Quarterly Report",
		"content": "1.2.3.4 - - [10/Oct/2026] \"GET /api HTTP/1.1\" 200",
		"lang": "zh",
		"type": "pdf",
		"tags": ["finance", "q3"],
		"size": 1024,
		"source": {"id": "ds-7", "name": "My Drive", "connector_id": "google_drive"},
		"document_chunk": [
			{"range": {"start": 1, "end": 2}, "text": "chunk one", "breadcrumb": "年报 > 财务"}
		]
	}`)
}

func TestDocumentCodec_RoundtripLossless(t *testing.T) {
	src := util.MapStr{}
	if err := util.FromJSONBytes(documentFixture(), &src); err != nil {
		t.Fatal(err)
	}
	c := DocumentCodec{}
	rec, err := c.Decode(documentFixture())
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.Encode(rec)
	if err != nil {
		t.Fatal(err)
	}
	got := util.MapStr{}
	if err := util.FromJSONBytes(out, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(src, got) {
		t.Fatalf("round-trip not lossless:\nsrc=%v\ngot=%v", src, got)
	}
}

func TestDocumentDecode_Projection(t *testing.T) {
	c := DocumentCodec{}
	rec, err := c.Decode(documentFixture())
	if err != nil {
		t.Fatal(err)
	}
	if rec.Fields[FieldDocument] == nil {
		t.Fatal("document subtree missing")
	}
	// dotted-path addressability for generic processors
	if v, err := rec.Fields.GetValue(FieldDocument + ".title"); err != nil || v != "Quarterly Report" {
		t.Fatalf("document.title not addressable: %v %v", v, err)
	}
	if rec.Meta["_id"] != "doc-42" {
		t.Fatalf("record id not lifted: %v", rec.Meta["_id"])
	}
	for key, want := range map[string]string{
		"datasource":   "ds-7",
		"connector_id": "google_drive",
		"lang":         "zh",
		"type":         "pdf",
		codecLabel:     CodecNameDocument,
	} {
		if got := asString(rec.Metadata.Labels[key]); got != want {
			t.Fatalf("label %s = %q, want %q", key, got, want)
		}
	}
	// mutate-then-encode reflects the edit (Fields is authoritative)
	doc, _ := rec.Fields.GetValue(FieldDocument)
	doc.(util.MapStr)["title"] = "Edited"
	out, _ := c.Encode(rec)
	check := util.MapStr{}
	util.FromJSONBytes(out, &check)
	if v, _ := check.GetValue("title"); v != "Edited" {
		t.Fatal("mutation not reflected in encode — Fields is not authoritative")
	}
}

func TestDocumentEncode_MissingSubtree(t *testing.T) {
	c := DocumentCodec{}
	if _, err := c.Encode(&event.Event{Fields: util.MapStr{}}); err == nil {
		t.Fatal("expected error for missing document subtree")
	}
}

func TestAttachmentCodec_Roundtrip(t *testing.T) {
	fixture := []byte(`{"id":"att-1","name":"logo.png","mime_type":"image/png","size":2048,"text":"ocr text"}`)
	rec, err := AttachmentCodec{}.Decode(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if got := asString(rec.Metadata.Labels["mime_type"]); got != "image/png" {
		t.Fatalf("mime label = %q", got)
	}
	out, err := AttachmentCodec{}.Encode(rec)
	if err != nil {
		t.Fatal(err)
	}
	got := util.MapStr{}
	util.FromJSONBytes(out, &got)
	want := util.MapStr{}
	util.FromJSONBytes(fixture, &want)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("attachment round-trip not lossless")
	}
}

func TestRegistry_BothCodecsDiscoverable(t *testing.T) {
	found := map[string]pipeline.CodecInfo{}
	for _, info := range pipeline.ListRecordCodecs() {
		found[info.Name] = info
	}
	for _, name := range []string{CodecNameDocument, CodecNameAttachment} {
		if _, ok := found[name]; !ok {
			t.Fatalf("codec %s not in registry", name)
		}
	}
	if found[CodecNameDocument].Namespace != NamespaceDocument {
		t.Fatal("document codec namespace mismatch")
	}
}

func TestDocumentDecode_ColdSubtreeStaysRaw(t *testing.T) {
	rec, err := DocumentCodec{}.Decode(documentFixture())
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := rec.Fields.GetValue(FieldDocument)
	chunks := doc.(util.MapStr)["document_chunk"]
	if _, isRaw := chunks.(json.RawMessage); !isRaw {
		t.Fatalf("cold subtree parsed as %T, want json.RawMessage (L2 cold/hot split)", chunks)
	}
	// and generic readers can still address hot paths
	if v, err := rec.Fields.GetValue(FieldDocument + ".title"); err != nil || v != "Quarterly Report" {
		t.Fatalf("hot path broken: %v", v)
	}
}

// hotJSON must behave like the stdlib on the edge cases where jsoniter
// defaults diverge — the compatibility gate for the hot-path swap.
func TestHotJSON_StdlibCompatibilityGate(t *testing.T) {
	fixtures := []string{
		`{"i":9007199254740993,"f":0.1,"big":1.7976931348623157e308}`, // big ints & floats
		`{"s":"<html>&\"quote\"</html>","u":"中文🚀","n":null}`,          // HTML escape, unicode, nil
		`{"a":[1,2,null,"x"],"o":{"k":{"deep":[[[]]]}}}`,              // nesting, empty containers
		`{"z":"","empty":{},"zero":0,"neg":-0.0000001}`,               // zero values
	}
	for _, fx := range fixtures {
		var want, got interface{}
		if err := json.Unmarshal([]byte(fx), &want); err != nil {
			t.Fatal(err)
		}
		if err := hotJSON.Unmarshal([]byte(fx), &got); err != nil {
			t.Fatalf("jsoniter decode %s: %v", fx, err)
		}
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("decode divergence on %s:\nstdlib=%v\njsoniter=%v", fx, want, got)
		}
		// encode divergence check via round-trip of the decoded value
		b1, _ := json.Marshal(want)
		b2, _ := hotJSON.Marshal(got)
		var r1, r2 interface{}
		json.Unmarshal(b1, &r1)
		hotJSON.Unmarshal(b2, &r2)
		if !reflect.DeepEqual(r1, r2) {
			t.Fatalf("encode divergence on %s:\nstdlib=%s\njsoniter=%s", fx, b1, b2)
		}
	}
}

// documents with sensitive scalar shapes must survive a codec round-trip
// byte-faithfully through the jsoniter path (cold subtree spliced verbatim).
func TestCodecRoundtrip_EdgeScalars(t *testing.T) {
	payload := `{"id":"e1","title":"a<b>&c","num":9007199254740993,"f":0.1,
		"tags":null,"document_chunk":[{"text":"中文🚀","v":[1.5,null]}]}`
	c := DocumentCodec{}
	rec, err := c.Decode([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.Encode(rec)
	if err != nil {
		t.Fatal(err)
	}
	// cold subtree must be byte-identical (RawMessage splice)
	if !contains(string(out), `"v":[1.5,null]`) || !contains(string(out), `"text":"中文🚀"`) {
		t.Fatalf("cold subtree not spliced verbatim: %s", out)
	}
	src, got := util.MapStr{}, util.MapStr{}
	if err := util.FromJSONBytes([]byte(payload), &src); err != nil {
		t.Fatal(err)
	}
	if err := util.FromJSONBytes(out, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(src, got) {
		t.Fatalf("round-trip lossy:\nsrc=%v\ngot=%v", src, got)
	}
}

func contains(s, sub string) bool { return len(sub) == 0 || len(s) >= len(sub) && indexOf(s, sub) >= 0 }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func BenchmarkDocumentCodec_DecodeEncode(b *testing.B) {
	payload := documentFixture()
	c := DocumentCodec{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec, err := c.Decode(payload)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := c.Encode(rec); err != nil {
			b.Fatal(err)
		}
	}
}
