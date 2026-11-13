/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSyncRunStale(t *testing.T) {
	fresh := &SyncRun{State: SyncRunRunning, UpdatedAt: time.Now().UnixMilli()}
	assert.False(t, SyncRunStale(fresh, 30*time.Minute))

	old := &SyncRun{State: SyncRunRunning, UpdatedAt: time.Now().Add(-time.Hour).UnixMilli()}
	assert.True(t, SyncRunStale(old, 30*time.Minute))

	// non-running records never read as stale — the evidence keeps
	finished := &SyncRun{State: SyncRunSuperseded, UpdatedAt: time.Now().Add(-time.Hour).UnixMilli()}
	assert.False(t, SyncRunStale(finished, 30*time.Minute))
	assert.False(t, SyncRunStale(nil, time.Second))
}
