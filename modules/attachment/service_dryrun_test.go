/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package attachment

/* Dry-run guard for the blob-store choke: pipeline dry-runs must never
 * persist attachments (the cover/chunk processors used to write real
 * metadata rows + blobs during dry-runs — caught live during the P8
 * migration-equivalence A/B). The write is captured as a side effect and
 * the caller still gets an ID. */

import (
	"bytes"
	"context"
	"testing"

	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/pipeline"
)

// bytesFile adapts an in-memory buffer to multipart.File
// (Reader + ReaderAt + Seeker + Closer).
type bytesFile struct{ r *bytes.Reader }

func (b bytesFile) Read(p []byte) (int, error)            { return b.r.Read(p) }
func (b bytesFile) ReadAt(p []byte, o int64) (int, error) { return b.r.ReadAt(p, o) }
func (b bytesFile) Seek(o int64, w int) (int64, error)    { return b.r.Seek(o, w) }
func (b bytesFile) Close() error                          { return nil }

func TestUploadToBlobStore_DryRunCapturesWithoutPersisting(t *testing.T) {
	pc := &pipeline.Context{Context: t.Context()}
	sink := &[]pipeline.DryRunWrite{}
	pipeline.EnableDryRun(pc, sink, true)

	// the real-world shape: processors hand orm a context DERIVED from the
	// pipeline context (utils.EachDocumentRecord callbacks), never the
	// pipeline wrapper itself — the guard must see through the chain
	derived := context.WithValue(pc, struct{ k string }{"layer"}, "processor-callback")
	ctx := orm.NewContextWithParent(derived)

	payload := []byte("fake png payload for the dry-run guard test")
	id, err := UploadToBlobStore(ctx, "", bytesFile{bytes.NewReader(payload)}, nil,
		"cover.png", "owner-1", nil, "", true)
	if err != nil {
		t.Fatalf("UploadToBlobStore: %v", err)
	}
	if id == "" {
		t.Fatal("dry-run upload must still return an attachment ID")
	}

	if len(*sink) != 1 {
		t.Fatalf("expected exactly one captured side effect, got %d", len(*sink))
	}
	w := (*sink)[0]
	if w.Kind != "attachment" || w.Target != id || w.Records != 1 {
		t.Fatalf("unexpected capture: %+v", w)
	}
	if w.Preview == "" {
		t.Fatal("capture preview should describe the file")
	}
}

// non-dry-run callers (HTTP upload API, wiki) pass request contexts — the
// guard must not misfire and change their behavior; here we only assert the
// guard itself stays inactive for a plain parent (no orm/kv needed: an
// inactive guard with no persist backend would fail loudly below).
func TestUploadToBlobStore_PlainContextBypassesGuard(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("plain (non dry-run) path should proceed to the real persist calls")
		}
	}()
	// no kv/orm backend in unit tests: reaching the persist section panics
	// (orm.Upsert/kv.AddValue error paths) — which proves the guard did
	// NOT capture-and-return early
	ctx := orm.NewContextWithParent(t.Context())
	_, _ = UploadToBlobStore(ctx, "", bytesFile{bytes.NewReader([]byte("x"))}, nil,
		"a.txt", "", nil, "", true)
}
