/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"testing"
)

func TestNormalizeFingerprintText(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"  Hello   World  ", "hello world"},
		{"ＡＢＣ１２３", "abc123"},             // full-width folds to half-width
		{"ｆｏｏ　ｂａｒ", "foo bar"},           // full-width space
		{"CAFE\u0045", "cafee"},               // NFKC keeps it simple
		{"ｶﾀｶﾅ", "カタカナ"},                // half-width katakana folds to full-width
		{"\n\t\rMix\r\n", "mix"},             // all whitespace collapses
	}
	for _, c := range cases {
		if got := normalizeFingerprintText(c.in); got != c.want {
			t.Fatalf("normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestContentFingerprint(t *testing.T) {
	long := "the quick brown fox jumps over the lazy dog and keeps running"

	h1, s1, ok := contentFingerprint(long)
	if !ok || h1 == "" || s1 == 0 {
		t.Fatalf("fingerprint failed: %v %v %v", h1, s1, ok)
	}

	// identical text → identical fingerprint
	h2, s2, _ := contentFingerprint("  THE QUICK   BROWN fox jumps over the lazy dog\nand keeps running  ")
	if h2 != h1 || s2 != s1 {
		t.Fatal("case/whitespace variants must fingerprint identically")
	}

	// full-width variant normalizes to the same fingerprint
	h3, _, _ := contentFingerprint("ｔｈｅ quick brown fox jumps over the lazy dog and keeps running")
	if h3 != h1 {
		t.Fatal("full-width variant must fold to the same hash")
	}

	// different text → different hash and a distant simhash
	h4, s4, _ := contentFingerprint("a completely unrelated sentence about database sharding strategies")
	if h4 == h1 {
		t.Fatal("different text must not share the hash")
	}
	if hammingDistance64(s1, s4) < 10 {
		t.Fatalf("unrelated texts should be far apart, distance=%d", hammingDistance64(s1, s4))
	}

	// small edit → same hash impossible, but simhash stays close
	edited := long + "!"
	h5, s5, _ := contentFingerprint(edited)
	if h5 == h1 {
		t.Fatal("an edit must change the hash")
	}
	if d := hammingDistance64(s1, s5); d > 3 {
		t.Fatalf("a small edit should flip few simhash bits, distance=%d", d)
	}

	// too short to judge
	if _, _, ok := contentFingerprint("tiny"); ok {
		t.Fatal("short text should not be fingerprinted")
	}
}

func TestHammingDistance64(t *testing.T) {
	if hammingDistance64(0, 0) != 0 || hammingDistance64(0, 1) != 1 || hammingDistance64(0xFF, 0x00) != 8 {
		t.Fatal("hamming math broken")
	}
}

func TestDedupPairKey(t *testing.T) {
	if dedupPairKey("b", "a") != dedupPairKey("a", "b") {
		t.Fatal("pair key must be order-independent")
	}
}
