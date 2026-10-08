/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"strings"
	"testing"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

func TestEnsureDocumentFingerprint(t *testing.T) {
	doc := &core.Document{Content: "Payment service failed at 2026-09-30 03:14 UTC, retries exhausted"}
	ensureDocumentFingerprint(doc)
	if doc.ContentHash == "" || doc.ContentSimhash == 0 {
		t.Fatalf("fingerprint must be stamped, got %+v", doc)
	}

	// idempotent: same content, same values
	hash, sim := doc.ContentHash, doc.ContentSimhash
	doc.Content = "  Payment   service FAILED at 2026-09-30 03:14 UTC,\nretries exhausted  "
	ensureDocumentFingerprint(doc)
	if doc.ContentHash != hash || doc.ContentSimhash != sim {
		t.Fatalf("normalization must make fingerprints stable: %v/%v vs %v/%v", doc.ContentHash, doc.ContentSimhash, hash, sim)
	}

	// blank and too-short content never stamp
	for _, content := range []string{"", "   ", "short"} {
		doc := &core.Document{Content: content}
		ensureDocumentFingerprint(doc)
		if doc.ContentHash != "" || doc.ContentSimhash != 0 {
			t.Fatalf("content %q must not be fingerprinted", content)
		}
	}
}

func TestFingerprintHookPassesThroughNonDocuments(t *testing.T) {
	var hook orm.HookFunc = func(ctx *orm.Context, _ orm.Operation, model interface{}) (*orm.Context, interface{}, error) {
		if doc, ok := model.(*core.Document); ok {
			ensureDocumentFingerprint(doc)
		}
		return ctx, model, nil
	}
	article := &core.WikiArticle{Title: "untouched"}
	_, out, err := hook(orm.NewContext(), orm.OpCreate, article)
	if err != nil || out.(*core.WikiArticle) != article {
		t.Fatalf("non-document models must pass through untouched: %v %v", out, err)
	}
}

func docHitWithHash(id, hash string, score float32) elastic.DocumentWithMeta[core.Document] {
	hit := rrfHit(id, score)
	hit.Source.ContentHash = hash
	return hit
}

func TestFoldDuplicateHits(t *testing.T) {
	hits := []elastic.DocumentWithMeta[core.Document]{
		docHitWithHash("a", "h1", 9),
		rrfHit("no-hash", 8),         // legacy doc: passes through
		docHitWithHash("b", "h1", 7), // duplicate of a
		docHitWithHash("c", "h1", 6), // duplicate of a
		docHitWithHash("d", "h2", 5),
	}
	folded := foldDuplicateHits(hits, nil)
	if len(folded) != 3 {
		t.Fatalf("expected 3 hits after folding, got %d", len(folded))
	}
	if folded[0].ID != "a" || folded[1].ID != "no-hash" || folded[2].ID != "d" {
		t.Fatalf("folded order wrong: %v %v %v", folded[0].ID, folded[1].ID, folded[2].ID)
	}
	dupes, ok := folded[0].Source.Metadata["fingerprint_duplicates"].(util.MapStr)
	if !ok || dupes["count"] != 2 {
		t.Fatalf("representative must carry the copy count, got %+v", folded[0].Source.Metadata)
	}
	ids := strings.Join(dupes["ids"].([]string), ",")
	if ids != "b,c" {
		t.Fatalf("copy ids wrong: %v", ids)
	}
	// different hash: no annotation
	if _, has := folded[2].Source.Metadata["fingerprint_duplicates"]; has {
		t.Fatal("unique hits must not carry duplicate metadata")
	}
}

func TestFoldDuplicateHitsEmpty(t *testing.T) {
	if got := foldDuplicateHits(nil, nil); len(got) != 0 {
		t.Fatalf("empty input must give empty output, got %d", len(got))
	}
}

func TestFoldEnrichesMembers(t *testing.T) {
	mk := func(id, title, src string) elastic.DocumentWithMeta[core.Document] {
		hit := elastic.DocumentWithMeta[core.Document]{ID: id}
		hit.Source.Title = title
		hit.Source.ContentHash = "hash-x"
		hit.Source.Source.Name = src
		hit.Source.Size = 10
		return hit
	}
	hits := []elastic.DocumentWithMeta[core.Document]{
		mk("a", "原件", "hr"), mk("b", "改名副本", "wiki"), mk("c", "第三份", "fs"),
	}

	folded := foldDuplicateHits(hits, nil)
	if len(folded) != 1 {
		t.Fatalf("expected 1 representative, got %d", len(folded))
	}
	dupes, _ := folded[0].Source.Metadata["fingerprint_duplicates"].(util.MapStr)
	if dupes == nil {
		t.Fatal("missing fingerprint_duplicates")
	}
	if dupes["count"] != 2 || dupes["tier"] != "exact" || dupes["similarity"] != 100 {
		t.Fatalf("unexpected fold payload: %v", dupes)
	}
	ids, _ := dupes["ids"].([]string)
	if len(ids) != 2 || ids[0] != "b" || ids[1] != "c" {
		t.Fatalf("unexpected ids: %v", ids)
	}
	members, _ := dupes["members"].([]interface{})
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	m1 := members[0].(util.MapStr)
	if m1["id"] != "b" || m1["title"] != "改名副本" || m1["source"] != "wiki" {
		t.Fatalf("unexpected member: %v", m1)
	}
}

func TestFoldRespectsDismissedPairs(t *testing.T) {
	mk := func(id string) elastic.DocumentWithMeta[core.Document] {
		hit := elastic.DocumentWithMeta[core.Document]{ID: id}
		hit.Source.ContentHash = "hash-y"
		return hit
	}
	hits := []elastic.DocumentWithMeta[core.Document]{mk("first"), mk("second"), mk("third")}

	// the operator explicitly marked first+second "not duplicates"
	dismissed := map[string]bool{dedupPairKey("first", "second"): true}
	folded := foldDuplicateHits(hits, dismissed)
	if len(folded) != 2 {
		t.Fatalf("dismissed pair must not fold: expected 2 hits, got %d", len(folded))
	}
	// third still folds into first — dismissal is per pair, not per hash
	dupes, _ := folded[0].Source.Metadata["fingerprint_duplicates"].(util.MapStr)
	if dupes == nil || dupes["count"] != 1 {
		t.Fatalf("third copy should fold into first: %v", folded[0].Source.Metadata["fingerprint_duplicates"])
	}
}
