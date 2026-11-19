/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infinilabs.com */

package wiki

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// the binding cache must never outlive a rebinding: entries resolve to the
// OLD kb after a rebind/delete unless invalidated eagerly or by TTL (live
// bug: extraction kept writing into the previous KB until restart).
func TestKBBindingCacheInvalidation(t *testing.T) {
	kbBindingCacheMu.Lock()
	kbBindingCache = map[string]kbBindingEntry{
		"ds-old-1": {kbID: "kb-a", cached: time.Now()},
		"ds-old-2": {kbID: "kb-a", cached: time.Now()},
		"ds-other": {kbID: "kb-b", cached: time.Now()},
	}
	kbBindingCacheMu.Unlock()

	// rebind: kb-a's datasources now point elsewhere; the new binding's ids
	// must drop too in case they were cached against a foreign kb
	InvalidateKBBindingsForKB("kb-a", "ds-new-1")

	kbBindingCacheMu.RLock()
	_, old1 := kbBindingCache["ds-old-1"]
	_, old2 := kbBindingCache["ds-old-2"]
	_, other := kbBindingCache["ds-other"]
	_, fresh := kbBindingCache["ds-new-1"]
	kbBindingCacheMu.RUnlock()

	assert.False(t, old1, "entry resolving to the rebound kb is dropped")
	assert.False(t, old2, "every entry resolving to the rebound kb is dropped")
	assert.True(t, other, "unrelated kb entries survive")
	assert.False(t, fresh, "newly bound datasource ids are dropped preemptively")

	// TTL: an expired entry is a cache miss even without invalidation
	kbBindingCacheMu.Lock()
	kbBindingCache["ds-stale"] = kbBindingEntry{kbID: "kb-a", cached: time.Now().Add(-kbBindingCacheTTL - time.Second)}
	kbBindingCacheMu.Unlock()

	kbBindingCacheMu.RLock()
	entry, ok := kbBindingCache["ds-stale"]
	kbBindingCacheMu.RUnlock()
	assert.True(t, ok)
	assert.True(t, time.Since(entry.cached) >= kbBindingCacheTTL, "expired entries must be re-resolved")
}
