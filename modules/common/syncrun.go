/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

// (This journal lives in modules/common — the neutral floor both the
// connectors-common package and the document module can import without
// cycles.)

// Sync run journal (W8 ingestion observability): a per-datasource record
// of the CURRENT/MOST RECENT collect run — state, timestamps, progress
// counters. Written only from the two shared choke points every connector
// passes through (the dispatcher start and BatchCollect), so it works for
// all 25 connectors without touching any of them.
//
// Completion semantics rely on the dispatcher's singleton pipeline: two
// runs of one datasource never overlap, so a "running" record at the next
// start MUST be a run that ended without recording a finish (process
// crash, pipeline kill) — it is marked "superseded", and its progress
// counters say how far it got before dying.

const syncRunBucket = "/datasource/syncruns"

const (
	SyncRunRunning    = "running"
	SyncRunSuperseded = "superseded" // ended unrecorded; a newer run took over
	SyncRunCompleted  = "completed"  // dispatch pipeline ended (fetch done or killed by its timeout)
)

// SyncRun is the journal record.
type SyncRun struct {
	RunID      string `json:"run_id"`
	State      string `json:"state"`
	StartedAt  int64  `json:"started_at"`
	UpdatedAt  int64  `json:"updated_at"`
	Batches    int    `json:"batches"`
	Documents  int    `json:"documents"`
	FinishedAt int64  `json:"finished_at,omitempty"`
}

// MarkSyncRunStarted supersedes any unrecorded previous run and opens a
// new running record. Returns the previous record when one was superseded
// (nil when this is the first run or the previous one recorded).
func MarkSyncRunStarted(datasourceID string) (*SyncRun, error) {
	prev, _ := ReadSyncRun(datasourceID)
	if prev != nil && prev.State == SyncRunRunning {
		prev.State = SyncRunSuperseded
		prev.FinishedAt = time.Now().UnixMilli()
		_ = writeSyncRun(datasourceID, prev)
	}
	run := &SyncRun{
		RunID:     util.GetUUID(),
		State:     SyncRunRunning,
		StartedAt: time.Now().UnixMilli(),
		UpdatedAt: time.Now().UnixMilli(),
	}
	return prev, writeSyncRun(datasourceID, run)
}

// RecordSyncBatch bumps the progress counters of the running record.
// Batch-collect fires this once per connector page batch; a few
// read-modify-writes per second at most, kv handles that happily.
func RecordSyncBatch(datasourceID string, docs int) {
	run, err := ReadSyncRun(datasourceID)
	if err != nil || run == nil || run.State != SyncRunRunning {
		return
	}
	run.Batches++
	run.Documents += docs
	run.UpdatedAt = time.Now().UnixMilli()
	_ = writeSyncRun(datasourceID, run)
}

// ReadSyncRun loads the journal record (nil when never run).
func ReadSyncRun(datasourceID string) (*SyncRun, error) {
	v, err := kv.GetValue(syncRunBucket, []byte(datasourceID))
	if err != nil || len(v) == 0 {
		return nil, err
	}
	run := &SyncRun{}
	if err := json.Unmarshal(v, run); err != nil {
		return nil, err
	}
	return run, nil
}

// SyncRunStale reports whether a running record is older than maxAge —
// read-only crashed detection for the status surface (the record stays
// untouched until the next run supersedes it, keeping the evidence).
func SyncRunStale(run *SyncRun, maxAge time.Duration) bool {
	return run != nil && run.State == SyncRunRunning &&
		time.Since(time.UnixMilli(run.UpdatedAt)) > maxAge
}

// CloseSyncRunIfIdle closes a "running" journal whose dispatch pipeline is no
// longer registered in the transient-task table. Without this, the record
// stays "running" forever after the fetch phase ends (the journal is only
// written at dispatch start and per BatchCollect), and the status surface
// reads every datasource as stale ("疑似僵死") during the — often hours-long —
// processing window that follows a successful fetch. Called from the
// dispatcher's keep-running loop, right before it decides whether a new
// dispatch is due.
func CloseSyncRunIfIdle(datasourceID string) {
	run, err := ReadSyncRun(datasourceID)
	if err != nil || run == nil || run.State != SyncRunRunning {
		return
	}
	if dispatchPipelineActive(datasourceID) {
		return
	}
	now := time.Now().UnixMilli()
	run.State = SyncRunCompleted
	run.UpdatedAt = now
	run.FinishedAt = now
	_ = writeSyncRun(datasourceID, run)
}

// dispatchPipelineActive reports whether the transient task backing this
// datasource's dispatch pipeline ("dynamic-datasource-task-<id>") is still
// registered. RegisterTransientTask stores the task synchronously before the
// goroutine starts and the goroutine deletes it on exit, so "absent" reliably
// means the pipeline has ended (completed, failed or killed by its timeout).
func dispatchPipelineActive(datasourceID string) bool {
	prefix := fmt.Sprintf("[dynamic-datasource-task-%v]", datasourceID)
	active := false
	task.Tasks.Range(func(_, v interface{}) bool {
		if t, ok := v.(*task.ScheduleTask); ok && t != nil &&
			strings.Contains(t.Description, prefix) &&
			t.State != task.Finished {
			active = true
			return false
		}
		return true
	})
	return active
}

func writeSyncRun(datasourceID string, run *SyncRun) error {
	b, err := json.Marshal(run)
	if err != nil {
		return err
	}
	return kv.AddValue(syncRunBucket, []byte(datasourceID), b)
}
