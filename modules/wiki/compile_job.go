/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
)

// CompileJob is the observable state of one compile run: the console polls
// this while the worker goroutine drives the pipeline. One job per KB —
// a new run replaces a finished one and is refused while one is running.
type CompileJob struct {
	ID            string   `json:"id"`
	KbID          string   `json:"kb_id"`
	Status        string   `json:"status"` // running | done | failed | cancelled
	Stage         string   `json:"stage"`  // map | reduce | plan | propose
	Docs          int      `json:"docs"`
	Batches       int      `json:"batches"`
	BatchesDone   int      `json:"batches_done"`
	BatchesCached int      `json:"batches_cached"`
	BatchesFailed int      `json:"batches_failed"`
	Extracts      int      `json:"extracts"`
	Reduced       int      `json:"reduced"`
	Proposals     int      `json:"proposals"`
	Current       string   `json:"current,omitempty"`
	Logs          []string `json:"logs,omitempty"`

	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	cancel context.CancelFunc
	mu     sync.Mutex
}

var compileJobs sync.Map // kbID -> *CompileJob

func startCompileJob(kbID string) (*CompileJob, error) {
	var existing *CompileJob
	if v, ok := compileJobs.Load(kbID); ok {
		existing, _ = v.(*CompileJob)
	}
	if existing != nil && existing.Status == "running" {
		return nil, fmt.Errorf("compile already running for this knowledge base")
	}
	job := &CompileJob{
		ID:        util.GetUUID(),
		KbID:      kbID,
		Status:    "running",
		Stage:     "map",
		StartedAt: time.Now(),
	}
	job.cancel = func() {} // replaced by the caller's cancel
	compileJobs.Store(kbID, job)
	return job, nil
}

func getCompileJob(kbID string) *CompileJob {
	if v, ok := compileJobs.Load(kbID); ok {
		job, _ := v.(*CompileJob)
		return job
	}
	return nil
}

// log appends a timestamped line; the ring keeps the last 200 so a
// hours-long run cannot grow the payload unbounded.
func (j *CompileJob) log(format string, args ...any) {
	if j == nil {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	line := time.Now().Format("15:04:05") + " " + fmt.Sprintf(format, args...)
	j.Logs = append(j.Logs, line)
	if len(j.Logs) > 200 {
		j.Logs = j.Logs[len(j.Logs)-200:]
	}
}

func (j *CompileJob) cancelled() bool {
	return j != nil && j.Status == "cancelled"
}

// Cancel marks a running job cancelled and trips its context.
func (j *CompileJob) Cancel() {
	if j == nil || j.Status != "running" {
		return
	}
	j.mu.Lock()
	now := time.Now()
	j.Status = "cancelled"
	j.FinishedAt = &now
	j.mu.Unlock()
	if j.cancel != nil {
		j.cancel()
	}
	j.log("已取消")
}

func (j *CompileJob) finish(status string) {
	if j == nil {
		return
	}
	j.mu.Lock()
	now := time.Now()
	j.Status = status
	j.FinishedAt = &now
	j.Stage = "done"
	j.mu.Unlock()
}

/* ---------------- console endpoints ---------------- */

// GET /wiki/kb/:id/_compile — the current or last compile job.
func (h *APIHandler) compileJobStatus(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	job := getCompileJob(ps.ByName("id"))
	if job == nil {
		h.WriteOKJSON(w, util.MapStr{"status": "idle"})
		return
	}
	h.WriteOKJSON(w, job)
}

// POST /wiki/kb/:id/_compile/_cancel — trips a running job.
func (h *APIHandler) cancelCompile(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	job := getCompileJob(ps.ByName("id"))
	if job == nil || job.Status != "running" {
		h.WriteOKJSON(w, util.MapStr{"cancelled": false, "reason": "no running compile"})
		return
	}
	job.Cancel()
	h.WriteOKJSON(w, util.MapStr{"cancelled": true})
}
