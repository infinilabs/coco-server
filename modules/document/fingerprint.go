/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"crypto/sha256"
	"encoding/hex"
	"hash/fnv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// The dedup fingerprints are deterministic local math — no model calls, no
// index writes. The same normalized text always yields the same pair, so the
// scan works over any existing corpus without reindexing.

// normalizeFingerprintText canonicalizes text for fingerprinting: NFKC folds
// full-width/half-width and compatibility forms together, case is dropped,
// and any whitespace run collapses to one space — so a PDF and its Word
// source, or a copy with a renamed file, land on the same fingerprint.
func normalizeFingerprintText(s string) string {
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

// contentFingerprint returns the sha256 of the normalized text plus a 64-bit
// simhash over its unicode bigrams. Bigrams work for both CJK (no spaces)
// and western text, and tolerate small edits that flip only a few bits.
// ok is false when the text is too short to judge.
func contentFingerprint(text string) (hashHex string, sim uint64, ok bool) {
	normalized := normalizeFingerprintText(text)
	runes := []rune(normalized)
	if len(runes) < 8 {
		return "", 0, false
	}

	sum := sha256.Sum256([]byte(normalized))
	hashHex = hex.EncodeToString(sum[:])

	var bits [64]int
	fnvHash := fnv.New64a()
	seen := 0
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
		seen++
	}
	for b := 0; b < 64; b++ {
		if bits[b] > 0 {
			sim |= 1 << b
		}
	}
	return hashHex, sim, true
}

// bigramJaccard is the verification stage: the Jaccard similarity of the
// two texts' unicode-bigram sets. Simhash buckets candidates cheaply, but
// short unrelated texts can still land within hamming 8 — this second,
// exact check kills those false positives before an edge is accepted.
func bigramJaccard(a, b string) float64 {
	setOf := func(s string) map[uint64]struct{} {
		runes := []rune(normalizeFingerprintText(s))
		out := make(map[uint64]struct{}, len(runes))
		h := fnv.New64a()
		for i := 0; i+1 < len(runes); i++ {
			h.Reset()
			h.Write([]byte(string(runes[i : i+2])))
			out[h.Sum64()] = struct{}{}
		}
		return out
	}
	sa, sb := setOf(a), setOf(b)
	if len(sa) == 0 || len(sb) == 0 {
		return 0
	}
	inter := 0
	for k := range sa {
		if _, ok := sb[k]; ok {
			inter++
		}
	}
	union := len(sa) + len(sb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// hammingDistance64 counts differing bits.
func hammingDistance64(a, b uint64) int {
	x := a ^ b
	dist := 0
	for x != 0 {
		x &= x - 1
		dist++
	}
	return dist
}

// dedupPairKey is the stable key of an unordered id pair.
func dedupPairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "|" + b
}
