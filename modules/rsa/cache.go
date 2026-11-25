/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rsa

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"infini.sh/coco/core"
)

// Per-source result cache — the same minimal shape the RRF fusion cache
// uses (mutex map + FIFO eviction, DESIGN-RSA.md §5.5): low-frequency
// federation needs nothing fancier, and no new dependency enters the
// tree. Entries die with their TTL; a source config change (updated
// bump) invalidates the whole source section.
type cacheEntry struct {
	result  Result
	expires time.Time
}

const (
	defaultCacheSize   = 256
	defaultCacheTTL    = 60 * time.Second
	cacheSizeHardLimit = 4096
)

var (
	cacheMu    sync.Mutex
	cacheStore = map[string]cacheEntry{}
	cacheKeys  []string // FIFO order
)

func cacheCapacity() int {
	if n := effectiveSettings().RsaCacheSize; n > 0 {
		if n > cacheSizeHardLimit {
			return cacheSizeHardLimit
		}
		return n
	}
	return defaultCacheSize
}

// sourceCacheTTL resolves the per-source TTL with the config default.
func sourceCacheTTL(src *core.RemoteSource) time.Duration {
	secs := src.Controls.CacheTTLSeconds
	if secs == 0 {
		return 0 // explicitly off — zero is a meaningful setting here
	}
	if secs < 0 {
		secs = 60
	}
	return time.Duration(secs) * time.Second
}

func cacheKey(src *core.RemoteSource, q Query) string {
	h := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d|%s", src.ID, q.From, q.Size, q.Text)))
	return src.ID + "/" + hex.EncodeToString(h[:8])
}

func cacheGet(src *core.RemoteSource, q Query) *Result {
	ttl := sourceCacheTTL(src)
	if ttl <= 0 {
		return nil
	}
	key := cacheKey(src, q)
	cacheMu.Lock()
	defer cacheMu.Unlock()
	e, ok := cacheStore[key]
	if !ok {
		return nil
	}
	if time.Now().After(e.expires) {
		delete(cacheStore, key)
		return nil
	}
	r := e.result
	return &r
}

func cachePut(src *core.RemoteSource, q Query, r Result) {
	ttl := sourceCacheTTL(src)
	if ttl <= 0 {
		return
	}
	key := cacheKey(src, q)
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if _, exists := cacheStore[key]; !exists {
		cacheKeys = append(cacheKeys, key)
	}
	cacheStore[key] = cacheEntry{result: r, expires: time.Now().Add(ttl)}
	for len(cacheKeys) > cacheCapacity() {
		oldest := cacheKeys[0]
		cacheKeys = cacheKeys[1:]
		delete(cacheStore, oldest)
	}
}

// InvalidateSource drops every entry of one source (config changed /
// deleted).
func InvalidateSource(sourceID string) {
	prefix := sourceID + "/"
	cacheMu.Lock()
	defer cacheMu.Unlock()
	kept := cacheKeys[:0]
	for _, k := range cacheKeys {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			delete(cacheStore, k)
			continue
		}
		kept = append(kept, k)
	}
	cacheKeys = kept
}
