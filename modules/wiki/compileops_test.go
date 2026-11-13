/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://www.infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/framework/core/kv"
)

// memKV is a hermetic in-memory KVStore for the op-queue tests — the real
// stores (badger/simple_kv) need a data dir and global env the unit suite
// doesn't bootstrap.
type memKV struct {
	mu      sync.RWMutex
	buckets map[string]map[string][]byte
}

func newMemKV() *memKV { return &memKV{buckets: map[string]map[string][]byte{}} }

func (m *memKV) Open() error  { return nil }
func (m *memKV) Close() error { return nil }

func (m *memKV) bucket(name string) map[string][]byte {
	if _, ok := m.buckets[name]; !ok {
		m.buckets[name] = map[string][]byte{}
	}
	return m.buckets[name]
}

func (m *memKV) GetValue(bucket string, key []byte) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.bucket(bucket)[string(key)]
	if !ok {
		return nil, fmt.Errorf("key not found: %s/%s", bucket, key)
	}
	out := make([]byte, len(v))
	copy(out, v)
	return out, nil
}

func (m *memKV) GetCompressedValue(bucket string, key []byte) ([]byte, error) {
	return m.GetValue(bucket, key)
}

func (m *memKV) AddValue(bucket string, key, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	v := make([]byte, len(value))
	copy(v, value)
	m.bucket(bucket)[string(key)] = v
	return nil
}

func (m *memKV) AddValueCompress(bucket string, key, value []byte) error {
	return m.AddValue(bucket, key, value)
}

func (m *memKV) ExistsKey(bucket string, key []byte) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.bucket(bucket)[string(key)]
	return ok, nil
}

func (m *memKV) DeleteKey(bucket string, key []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.bucket(bucket), string(key))
	return nil
}

var compileOpsTestKVOnce sync.Once

func setupCompileOpsKV(t *testing.T) {
	t.Helper()
	compileOpsTestKVOnce.Do(func() {
		kv.Register("wiki-compileops-test", newMemKV())
	})
}

func TestCompileOpsFoldAndConsume(t *testing.T) {
	setupCompileOpsKV(t)
	const kb = "kb-fold"

	// ten same-doc ops fold into one entry, latest reason wins
	for i := 0; i < 10; i++ {
		EnqueueCompileOp(kb, "ingest", "doc-1", fmt.Sprintf("reason-%d", i))
	}
	// a different doc and a whole-corpus mark stay separate
	EnqueueCompileOp(kb, "ingest", "doc-2", "content-changed")
	EnqueueCompileOp(kb, "ingest", "", "manual-mark")

	ops := peekCompileOps(kb)
	require.Len(t, ops, 3, "same-doc ops must fold; distinct docs and whole-corpus marks stay")
	folded := map[string]CompileOp{}
	for _, op := range ops {
		folded[op.DocID] = op
	}
	assert.Equal(t, "reason-9", folded["doc-1"].Reason, "latest write wins")
	assert.Equal(t, 0, folded["doc-1"].FailCount)
	assert.Contains(t, pendingCompileKBs(), kb)

	// consume drops the queue and the pending-index entry
	deleteCompileOps(kb)
	assert.Empty(t, peekCompileOps(kb))
	assert.NotContains(t, pendingCompileKBs(), kb)
}

func TestOpsSettledWindow(t *testing.T) {
	now := time.Now()
	assert.False(t, opsSettled(nil, now), "empty queue is never settled")
	assert.False(t, opsSettled([]CompileOp{{EnqueuedAt: now.Add(-time.Minute)}}, now),
		"fresh op must wait out the settle window")
	assert.True(t, opsSettled([]CompileOp{
		{EnqueuedAt: now.Add(-10 * time.Minute)},
		{EnqueuedAt: now.Add(-compileLaneSettle)}, // the newest op decides
		{EnqueuedAt: now.Add(-2 * time.Hour)},
	}, now), "settled once the newest op ages past the window")
}

func TestCompileLease(t *testing.T) {
	setupCompileOpsKV(t)
	const kb = "kb-lease"

	releaseCompileLease(kb) // clean slate from earlier test runs
	assert.True(t, tryAcquireCompileLease(kb), "first taker wins")
	assert.False(t, tryAcquireCompileLease(kb), "second taker is refused while held")

	// a stale lease (crashed holder past the TTL) may be retaken
	stale := time.Now().Add(-compileLeaseTTL - time.Minute).Format(time.RFC3339Nano)
	require.NoError(t, kv.AddValue(compileLockBucket, []byte(kb), []byte(stale)))
	assert.True(t, tryAcquireCompileLease(kb))

	releaseCompileLease(kb)
	assert.True(t, tryAcquireCompileLease(kb))
	releaseCompileLease(kb)
}

func TestCompileDeadLetterRoundtrip(t *testing.T) {
	setupCompileOpsKV(t)
	const kb = "kb-dead"

	EnqueueCompileOp(kb, "ingest", "doc-x", "content-changed")
	for i := 1; i < compileOpMaxFails; i++ {
		bumpCompileOpFailures(kb, fmt.Errorf("boom-%d", i))
	}
	// attempts left: the op is still queued, counting failures
	ops := peekCompileOps(kb)
	require.Len(t, ops, 1)
	assert.Equal(t, compileOpMaxFails-1, ops[0].FailCount)

	// the final failure dead-letters the op and empties the queue
	bumpCompileOpFailures(kb, fmt.Errorf("boom-final"))
	assert.Empty(t, peekCompileOps(kb))
	assert.NotContains(t, pendingCompileKBs(), kb)

	dead := listCompileDeadLetters(kb)
	require.Len(t, dead, 1)
	assert.Equal(t, "doc-x", dead[0].Op.DocID)
	assert.Contains(t, dead[0].Error, "boom-final")

	clearCompileDeadLetters(kb)
	assert.Empty(t, listCompileDeadLetters(kb))
}
