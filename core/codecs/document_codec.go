/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// Package codecs carries the Coco domain payload codecs: they project
// queue-payload JSON onto the framework record convention (coco_document /
// coco_attachment) so generic processors can address records without
// knowing domain types.
//
// The projection works on JSON field names — the frozen public contract
// (document.*) — never on Go struct types; the typed view lives in coco's
// documentview.
package codecs

import (
	"encoding/json"
	"fmt"
	"time"

	"infini.sh/framework/core/event"
	pipeline "infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

const (
	// CodecNameDocument / NamespaceDocument identify the document codec.
	CodecNameDocument = "coco_document"
	NamespaceDocument = "document"

	// FieldDocument is the event Fields key carrying the whole projected
	// document tree; generic processors address leaves via dotted paths
	// ("document.content").
	FieldDocument = NamespaceDocument

	// codecLabel stamps the originating codec on decoded records so output
	// encoding can round-trip symmetrically (three-level resolution).
	codecLabel = "_codec"
)

// DocumentCodec projects a core.Document JSON payload onto *event.Event:
// the full tree rides under Fields["document"] (dotted-path addressable,
// lossless — new payload fields appear without mapping changes), routing
// metadata is lifted into Metadata.Labels.
type DocumentCodec struct{}

func (DocumentCodec) Name() string      { return CodecNameDocument }
func (DocumentCodec) Namespace() string { return NamespaceDocument }

// ColdSubtrees lists payload keys that ride as raw JSON bytes (L2
// cold/hot split): the heavy embedding vectors have no in-chain readers —
// only the embedding processor touches them — so they are never float-
// parsed on the hot path; Encode splices the original bytes back verbatim.
var ColdSubtrees = []string{"document_chunk"}

// hotJSON was jsoniter (D-6); measured on embedding-heavy payloads it
// LOSES to the stdlib on both directions — jsoniter's generic
// interface{} encode/decode allocates per float element (~122k allocs for
// a 1MB document_chunk subtree vs 19 for the stdlib with identical
// bytes; see for_each_bench_test.go). The stdlib stays; the hot-path
// wins come from the raw-map two-pass decode and the cold split below.

func (DocumentCodec) Decode(data []byte) (*event.Event, error) {
	// first pass: keep every top-level value raw so cold subtrees stay
	// unparsed bytes
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: decode payload: %w", CodecNameDocument, err)
	}
	cold := map[string]bool{}
	for _, k := range ColdSubtrees {
		cold[k] = true
	}
	m := util.MapStr{}
	for k, v := range raw {
		if cold[k] {
			m[k] = v // raw bytes ride through untouched
			continue
		}
		if len(v) == 0 || string(v) == "null" {
			m[k] = nil
			continue
		}
		var val interface{}
		if err := json.Unmarshal(v, &val); err != nil {
			return nil, fmt.Errorf("%s: decode field %q: %w", CodecNameDocument, k, err)
		}
		m[k] = val
	}
	rec := event.AcquireEvent()
	// reuse the pooled map shells (cleared in place by the pool); only the
	// inner tree is fresh — one less allocation per record at small sizes
	if rec.Fields == nil {
		rec.Fields = util.MapStr{}
	}
	rec.Fields[FieldDocument] = m
	rec.SetID(asString(m["id"]))
	labels := rec.Metadata.Labels
	if labels == nil {
		labels = util.MapStr{}
	}
	if src, ok := m["source"].(map[string]interface{}); ok {
		putLabel(labels, "datasource", src["id"])
		putLabel(labels, "connector_id", src["connector_id"])
	}
	putLabel(labels, "lang", m["lang"])
	putLabel(labels, "type", m["type"])
	if ts := asString(m["updated"]); ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			rec.Timestamp = t
		}
	}
	labels[codecLabel] = CodecNameDocument
	rec.Metadata.Labels = labels
	return rec, nil
}

func (DocumentCodec) Encode(rec *event.Event) ([]byte, error) {
	v, err := rec.Fields.GetValue(FieldDocument)
	if err != nil || v == nil {
		return nil, fmt.Errorf("%s: missing %q subtree in record fields", CodecNameDocument, FieldDocument)
	}
	// stdlib json.Marshal: jsoniter's interface{} encoder allocates per
	// element on the cold RawMessage subtree (see the hotJSON note above)
	return json.Marshal(v)
}

func init() {
	pipeline.MustRegisterRecordCodec(DocumentCodec{})
}

// putLabel sets a metadata label when the value is a non-empty string.
func putLabel(labels util.MapStr, key string, v interface{}) {
	if s := asString(v); s != "" {
		labels[key] = s
	}
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}
