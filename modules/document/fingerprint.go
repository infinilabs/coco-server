/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"hash/fnv"

	"infini.sh/coco/modules/common/fingerprint"
)

// The dedup fingerprints are deterministic local math — no model calls, no
// index writes. The same normalized text always yields the same pair, so
// the scan works over any existing corpus without reindexing. The math
// lives in modules/common/fingerprint so the orm pre-hook here and the
// connector ingestion stamp (BatchCollect/webhooks, which bypass orm)
// produce identical values from one implementation.

// normalizeFingerprintText delegates to the shared fingerprint package.
func normalizeFingerprintText(s string) string {
	return fingerprint.Normalize(s)
}

// contentFingerprint delegates to the shared fingerprint package: sha256 of
// the normalized text plus a 64-bit simhash over its unicode bigrams.
// ok is false when the text is too short to judge.
func contentFingerprint(text string) (hashHex string, sim uint64, ok bool) {
	return fingerprint.Compute(text)
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
