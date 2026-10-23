/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://www.infinilabs.com
 * Email: hello#infini.ltd */

package wiki

// Document-level incremental compile op queue (DESIGN-incremental-compile.md
// P1, 2026-10-12). Replaces the in-memory KB dirty map + debounce timer:
// ops persist in badger so a restart can no longer lose a pending compile,
// document bursts fold per doc_id, and a per-KB lease keeps two triggers
// from compiling the same KB concurrently.
//
// P1 scope: the queue replaces the TRIGGER only — the pipeline still runs
// whole-KB, with MAP's batch-fingerprint cache already making unchanged
// documents free of LLM calls. Narrowing REDUCE/PLAN to the affected pages
// needs the contribution inverted index and stays P2.
//
// Ops are transient instructions, not audit facts: the kv keys are the only
// state, a lost queue is recoverable by a full rescan, and the audit trail
// lives where it already does (governance proposals + pipeline runs).

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"infini.sh/coco/core"
	llmmodule "infini.sh/coco/modules/llm"

	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

const (
	compileOpsBucket    = "wiki-compileops"
	compileLockBucket   = "wiki-compilelock"
	compileDeadBucket   = "wiki-opdead"
	compileOpsIndexKey  = "__index__" // kbIDs with pending ops (kv has no scan)
	compileLaneTaskID   = "wiki-compile-ops-lane"
	compileLaneInterval = "30s"
	// trailing edge: a KB is consumed only after its newest op has settled,
	// so ingestion bursts coalesce into one run — the persistent replacement
	// of the old 5-minute in-memory debounce (same cadence, crash-safe)
	compileLaneSettle = 5 * time.Minute
	// lease TTL: a crashed lane run can hold a KB at most this long before
	// the next sweep may retake it
	compileLeaseTTL = 15 * time.Minute
	// per-KB cooldown after a failed run — without it the lane would burn
	// the 5 allowed attempts in as many ticks
	compileFailCooldown = 10 * time.Minute
	compileOpMaxFails  = 5
	compileDeadCap     = 100
)

// CompileOp is one pending corpus change. DocID "" means "the whole corpus"
// (manual marks, KB-level events) — it folds with every other whole-corpus
// op but never with a specific-document op.
type CompileOp struct {
	Op         string     `json:"op"` // ingest | retract
	DocID      string     `json:"doc_id,omitempty"`
	Reason     string     `json:"reason,omitempty"` // content-changed | document-deleted | kb-file-uploaded | manual-mark
	FailCount  int        `json:"fail_count,omitempty"`
	EnqueuedAt time.Time `json:"enqueued_at"`
}

// compileDeadOp is a dead-lettered op with its last error.
type compileDeadOp struct {
	Op       CompileOp `json:"op"`
	Error    string    `json:"error"`
	FailedAt time.Time `json:"failed_at"`
}

// compileOpsMu guards the read-modify-write cycles on the ops index and the
// per-KB op arrays (bbolt has no CAS; all writers are in-process).
var compileOpsMu sync.Mutex

func init() {
	task.RegisterScheduleTask(task.ScheduleTask{
		ID:          compileLaneTaskID,
		Group:       "wiki",
		Description: "Wiki 编译 op 消费 lane:按文档级 op 触发受影响 KB 的编译(增量编译 P1)",
		Interval:    compileLaneInterval,
		Singleton:   true,
		Task:        compileOpsSweep,
	})
}

/* ---------------- queue ---------------- */

// EnqueueCompileOp records a corpus change for a KB. Same-doc ops fold into
// one entry (latest reason wins); kbID joins the pending index. The lane
// picks the KB up after the settle window.
func EnqueueCompileOp(kbID, op, docID, reason string) {
	if kbID == "" {
		return
	}
	compileOpsMu.Lock()
	defer compileOpsMu.Unlock()

	ops := readCompileOpsLocked(kbID)
	folded := false
	for i := range ops {
		if ops[i].DocID == docID {
			ops[i].Op = op
			ops[i].Reason = reason
			ops[i].EnqueuedAt = time.Now()
			folded = true
			break
		}
	}
	if !folded {
		ops = append(ops, CompileOp{Op: op, DocID: docID, Reason: reason, EnqueuedAt: time.Now()})
	}
	writeCompileOpsLocked(kbID, ops)
}

// pendingCompileKBs lists KB ids with at least one queued op.
func pendingCompileKBs() []string {
	raw, err := kv.GetValue(compileOpsBucket, []byte(compileOpsIndexKey))
	if err != nil || len(raw) == 0 {
		return nil
	}
	var ids []string
	if err := util.FromJSONBytes(raw, &ids); err != nil {
		return nil
	}
	return ids
}

// peekCompileOps returns the KB's queued ops without consuming them.
func peekCompileOps(kbID string) []CompileOp {
	compileOpsMu.Lock()
	defer compileOpsMu.Unlock()
	return readCompileOpsLocked(kbID)
}

// deleteCompileOps drops the KB's queue and pending-index entry.
func deleteCompileOps(kbID string) {
	compileOpsMu.Lock()
	defer compileOpsMu.Unlock()
	if err := kv.DeleteKey(compileOpsBucket, []byte(kbID)); err != nil {
		log.Warnf("wiki compileops: failed to clear ops for kb [%s]: %v", kbID, err)
	}
	removePendingIndexLocked(kbID)
}

// bumpCompileOpFailures records one failed consume attempt on every op;
// ops that exhausted their attempts move to the dead letter and drop out.
// Returns the error to log.
func bumpCompileOpFailures(kbID string, runErr error) {
	compileOpsMu.Lock()
	defer compileOpsMu.Unlock()

	ops := readCompileOpsLocked(kbID)
	kept := ops[:0:0]
	dead := make([]compileDeadOp, 0, len(ops))
	for _, op := range ops {
		op.FailCount++
		if op.FailCount >= compileOpMaxFails {
			dead = append(dead, compileDeadOp{Op: op, Error: fmt.Sprintf("%v", runErr), FailedAt: time.Now()})
			continue
		}
		kept = append(kept, op)
	}
	if len(dead) > 0 {
		appendCompileDeadLocked(kbID, dead)
	}
	if len(kept) == 0 {
		_ = kv.DeleteKey(compileOpsBucket, []byte(kbID))
		removePendingIndexLocked(kbID)
		return
	}
	writeCompileOpsLocked(kbID, kept)
}

func readCompileOpsLocked(kbID string) []CompileOp {
	raw, err := kv.GetValue(compileOpsBucket, []byte(kbID))
	if err != nil || len(raw) == 0 {
		return nil
	}
	var ops []CompileOp
	if err := util.FromJSONBytes(raw, &ops); err != nil {
		return nil
	}
	return ops
}

func writeCompileOpsLocked(kbID string, ops []CompileOp) {
	if err := kv.AddValue(compileOpsBucket, []byte(kbID), util.MustToJSONBytes(ops)); err != nil {
		log.Warnf("wiki compileops: failed to persist ops for kb [%s]: %v", kbID, err)
		return
	}
	ids := pendingCompileKBs()
	for _, id := range ids {
		if id == kbID {
			return
		}
	}
	ids = append(ids, kbID)
	if err := kv.AddValue(compileOpsBucket, []byte(compileOpsIndexKey), util.MustToJSONBytes(ids)); err != nil {
		log.Warnf("wiki compileops: failed to update pending index: %v", err)
	}
}

func removePendingIndexLocked(kbID string) {
	ids := pendingCompileKBs()
	kept := ids[:0:0]
	for _, id := range ids {
		if id != kbID {
			kept = append(kept, id)
		}
	}
	if len(kept) == len(ids) {
		return
	}
	if err := kv.AddValue(compileOpsBucket, []byte(compileOpsIndexKey), util.MustToJSONBytes(kept)); err != nil {
		log.Warnf("wiki compileops: failed to update pending index: %v", err)
	}
}

// opsSettled reports whether the KB's newest op has aged past the settle
// window (pure function for tests).
func opsSettled(ops []CompileOp, now time.Time) bool {
	if len(ops) == 0 {
		return false
	}
	newest := ops[0].EnqueuedAt
	for _, op := range ops[1:] {
		if op.EnqueuedAt.After(newest) {
			newest = op.EnqueuedAt
		}
	}
	return now.Sub(newest) >= compileLaneSettle
}

/* ---------------- lease & failure cooldown ---------------- */

func tryAcquireCompileLease(kbID string) bool {
	if v, err := kv.GetValue(compileLockBucket, []byte(kbID)); err == nil && len(v) > 0 {
		if ts, perr := time.Parse(time.RFC3339Nano, string(v)); perr == nil && time.Since(ts) < compileLeaseTTL {
			return false
		}
	}
	return kv.AddValue(compileLockBucket, []byte(kbID), []byte(time.Now().Format(time.RFC3339Nano))) == nil
}

func releaseCompileLease(kbID string) {
	_ = kv.DeleteKey(compileLockBucket, []byte(kbID))
}

func noteCompileFailure(kbID string) {
	_ = kv.AddValue(compileLockBucket, []byte("fail:"+kbID), []byte(time.Now().Format(time.RFC3339Nano)))
}

func compileFailureCooldownActive(kbID string) bool {
	v, err := kv.GetValue(compileLockBucket, []byte("fail:"+kbID))
	if err != nil || len(v) == 0 {
		return false
	}
	ts, perr := time.Parse(time.RFC3339Nano, string(v))
	return perr == nil && time.Since(ts) < compileFailCooldown
}

/* ---------------- dead letter ---------------- */

func appendCompileDeadLocked(kbID string, entries []compileDeadOp) {
	raw, _ := kv.GetValue(compileDeadBucket, []byte(kbID))
	var existing []compileDeadOp
	if len(raw) > 0 {
		_ = util.FromJSONBytes(raw, &existing)
	}
	existing = append(existing, entries...)
	if overflow := len(existing) - compileDeadCap; overflow > 0 {
		existing = existing[overflow:]
	}
	if err := kv.AddValue(compileDeadBucket, []byte(kbID), util.MustToJSONBytes(existing)); err != nil {
		log.Warnf("wiki compileops: failed to persist dead letters for kb [%s]: %v", kbID, err)
	}
}

// listCompileDeadLetters returns the KB's dead-lettered ops (oldest first).
func listCompileDeadLetters(kbID string) []compileDeadOp {
	raw, err := kv.GetValue(compileDeadBucket, []byte(kbID))
	if err != nil || len(raw) == 0 {
		return nil
	}
	var entries []compileDeadOp
	if err := util.FromJSONBytes(raw, &entries); err != nil {
		return nil
	}
	return entries
}

// clearCompileDeadLetters empties the KB's dead letter.
func clearCompileDeadLetters(kbID string) {
	_ = kv.DeleteKey(compileDeadBucket, []byte(kbID))
}

/* ---------------- consume lane ---------------- */

// compileOpsSweep is the lane tick: for every KB with settled, off-cooldown
// ops and no compile already in flight, take the lease and run the pipeline.
// Ops survive restarts (badger), so startup recovery is inherent — the first
// tick after boot resumes whatever the crash interrupted.
func compileOpsSweep(ctx context.Context) {
	for _, kbID := range pendingCompileKBs() {
		ops := peekCompileOps(kbID)
		if len(ops) == 0 {
			deleteCompileOps(kbID)
			continue
		}
		if !opsSettled(ops, time.Now()) {
			continue
		}
		if compileFailureCooldownActive(kbID) {
			continue
		}
		if job := getCompileJob(kbID); job != nil && job.Status == "running" {
			continue
		}
		if !tryAcquireCompileLease(kbID) {
			continue
		}
		runLaneCompile(ctx, kbID, ops)
	}
}

// runLaneCompile runs the (still whole-KB) pipeline for one KB and settles
// its ops: consumed on success, fail-counted on error. The lease is always
// released; a crash leaves it to expire (compileLeaseTTL).
func runLaneCompile(ctx context.Context, kbID string, ops []CompileOp) {
	defer releaseCompileLease(kbID)
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("wiki compileops: lane run panicked for kb [%s]: %v", kbID, r)
		}
	}()

	kb := core.WikiKnowledgeBase{}
	kb.ID = kbID
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	exists, err := orm.GetV2(octx, &kb)
	if err != nil || !exists {
		deleteCompileOps(kbID) // KB gone: nothing to compile, drop the ops
		return
	}
	model := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if model == nil {
		log.Debugf("wiki compileops: lane run skipped for [%s], no language model", kbID)
		return // ops stay queued; retried once a model is configured
	}
	job := &CompileJob{ID: util.GetUUID(), KbID: kbID, Status: "running", Stage: "map", StartedAt: time.Now()}
	job.cancel = func() {}
	compileJobs.Store(kbID, job)

	summary, err := runCompilePipeline(octx, &kb, model, job)
	if err != nil {
		job.finish("failed")
		noteCompileFailure(kbID)
		bumpCompileOpFailures(kbID, err)
		log.Warnf("wiki compileops: lane run failed for [%s] (%d ops): %v", kbID, len(ops), err)
		return
	}
	deleteCompileOps(kbID)
	job.finish("done")
	log.Infof("wiki compileops: lane run for kb [%s] — %d ops consumed, %d extracts, %d reduced, %d proposals",
		kb.Name, len(ops), summary.Extracts, summary.Reduced, summary.Proposals)
}

/* ---------------- HTTP: dead letter ---------------- */

// compileOpsDeadHandler: GET /wiki/kb/:id/_opdead — the KB's dead-lettered
// compile ops (v1 has no standalone panel; the KB overview shows the count).
func (h *APIHandler) compileOpsDeadHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	entries := listCompileDeadLetters(kbID)
	if entries == nil {
		entries = []compileDeadOp{}
	}
	h.WriteOKJSON(w, util.MapStr{"total": len(entries), "entries": entries})
}

// compileOpsDeadClearHandler: DELETE /wiki/kb/:id/_opdead — acknowledge and
// clear the dead letter.
func (h *APIHandler) compileOpsDeadClearHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	clearCompileDeadLetters(ps.ByName("id"))
	h.WriteOKJSON(w, util.MapStr{"result": "cleared"})
}
