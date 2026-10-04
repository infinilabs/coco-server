/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

// Package fingerprint holds the dedup content fingerprints (D1.5): sha256
// of the normalized text plus a 64-bit simhash over its unicode bigrams.
// The math is deterministic and local — no model calls, no index writes —
// and it is stamped at two write fronts: the document module's orm pre-hook
// (API/datasource/MCP writes) and the connector ingestion queue-push
// (BatchCollect/webhooks), because the ingestion pipelines index straight
// to the engine and never touch orm. One home for the math keeps both
// fronts producing identical values.
package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"hash/fnv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Normalize canonicalizes text for fingerprinting: NFKC folds
// full-width/half-width and compatibility forms together, case is dropped,
// and any whitespace run collapses to one space — so a PDF and its Word
// source, or a copy with a renamed file, land on the same fingerprint.
func Normalize(s string) string {
	s = norm.NFKC.String(s)
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// Compute returns the sha256 of the normalized text plus the simhash over
// its unicode bigrams. Bigrams work for both CJK (no spaces) and western
// text, and tolerate small edits that flip only a few bits. ok is false
// when the text is too short to judge — callers must clear any previously
// stamped fields in that case, a shrunken document must not keep the old
// write's fingerprint.
func Compute(text string) (hashHex string, sim uint64, ok bool) {
	normalized := Normalize(text)
	runes := []rune(normalized)
	if len(runes) < 8 {
		return "", 0, false
	}

	sum := sha256.Sum256([]byte(normalized))
	hashHex = hex.EncodeToString(sum[:])

	var bits [64]int
	fnvHash := fnv.New64a()
	for i := 0; i+1 < len(runes); i++ {
		fnvHash.Reset()
		fnvHash.Write([]byte(string(runes[i : i+2])))
		h := fnvHash.Sum64()
		for b := 0; b < 64; b++ {
			if h&(1<<b) != 0 {
				bits[b]++
			} else {
				bits[b]--
			}
		}
	}
	for b := 0; b < 64; b++ {
		if bits[b] > 0 {
			sim |= 1 << b
		}
	}
	return hashHex, sim, true
}
