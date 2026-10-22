/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package documentview

import (
	"testing"

	"infini.sh/coco/core/codecs"
	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

func TestMaterializeProject_Roundtrip(t *testing.T) {
	payload := []byte(`{"id":"doc-1","title":"原味标题","content":"hello","lang":"zh","tags":["a","b"],
		"source":{"id":"ds-1","connector_id":"local_fs"},
		"document_chunk":[{"range":{"start":1,"end":2},"text":"c1"}]}`)
	rec, err := codecs.DocumentCodec{}.Decode(payload)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := Materialize(rec)
	if err != nil {
		t.Fatal(err)
	}
	if doc.ID != "doc-1" || doc.Title != "原味标题" || doc.Source.ID != "ds-1" || len(doc.Chunks) != 1 {
		t.Fatalf("materialize mismatch: id=%q title=%q source=%v chunks=%d", doc.ID, doc.Title, doc.Source.ID, len(doc.Chunks))
	}

	// typed mutation → Project → visible at the dotted path
	doc.Title = "改后的标题"
	doc.Tags = append(doc.Tags, "new")
	if err := Project(doc, rec); err != nil {
		t.Fatal(err)
	}
	if v, err := rec.Fields.GetValue(codecs.FieldDocument + ".title"); err != nil || v != "改后的标题" {
		t.Fatalf("projected title = %v %v", v, err)
	}
	if v, err := rec.Fields.GetValue(codecs.FieldDocument + ".tags.2"); err != nil || v != "new" {
		t.Fatalf("projected tag = %v %v", v, err)
	}
}

func TestMaterialize_GenericMutationPreserved(t *testing.T) {
	// a generic processor mutates Fields directly; Materialize sees it
	payload := []byte(`{"id":"doc-2","title":"t","content":"c"}`)
	rec, _ := codecs.DocumentCodec{}.Decode(payload)
	rec.Fields[codecs.FieldDocument].(util.MapStr)["title"] = "由 grok 改的"
	doc, err := Materialize(rec)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "由 grok 改的" {
		t.Fatalf("generic mutation lost: %q", doc.Title)
	}
}

func TestMaterialize_MissingSubtree(t *testing.T) {
	rec, _ := codecs.DocumentCodec{}.Decode([]byte(`{}`))
	if _, err := Materialize(rec); err == nil {
		t.Fatal("expected error for empty document")
	}
	_ = core.Document{} // keep the core import honest for future assertions
}
