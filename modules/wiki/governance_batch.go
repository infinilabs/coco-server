/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

// Batch review runs the real fixes (page generation, rebuilds, merges) —
// work that can take ~40s per compile proposal with the writing pass. The
// batch therefore executes in a background worker and reports progress
// through kv; the UI polls instead of holding one HTTP request open.
// One batch at a time, server-wide: parallel applies would race the TOC
// and the dedup keys.

const batchApplyKVBucket = "/wiki/governance/batchapply"
const batchApplyKVKey = "current"
const batchApplyCap = 200

var batchApplyMu sync.Mutex

// BatchApplyProgress is the polled state of the current (or last) batch.
type BatchApplyProgress struct {
	Status  string   `json:"status"` // running | done
	Total   int      `json:"total"`
	Done    int      `json:"done"`
	Failed  int      `json:"failed"`
	Skipped int      `json:"skipped"`
	Current string   `json:"current,omitempty"` // title of the item in flight
	Errors  []string `json:"errors,omitempty"`  // per-item failures, capped
}

func writeBatchApplyProgress(p *BatchApplyProgress) {
	buf, _ := json.Marshal(p)
	_ = kv.AddValue(batchApplyKVBucket, []byte(batchApplyKVKey), buf)
}

func readBatchApplyProgress() *BatchApplyProgress {
	buf, err := kv.GetValue(batchApplyKVBucket, []byte(batchApplyKVKey))
	if err != nil || len(buf) == 0 {
		return &BatchApplyProgress{Status: "idle"}
	}
	p := &BatchApplyProgress{}
	if json.Unmarshal(buf, p) != nil {
		return &BatchApplyProgress{Status: "idle"}
	}
	return p
}

// POST /wiki/governance/_batch_apply {ids} — apply every open proposal in
// the background, sequentially (a compile apply with the writing pass is
// an LLM call per page). tree_organize is refused: applying a tree in bulk
// is not a review action.
func (h *APIHandler) startBatchApply(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.Error400(w, err.Error())
		return
	}
	if len(body.IDs) == 0 {
		h.Error400(w, "ids is empty")
		return
	}
	if len(body.IDs) > batchApplyCap {
		h.Error400(w, fmt.Sprintf("too many ids (cap %d per batch)", batchApplyCap))
		return
	}
	if !batchApplyMu.TryLock() {
		h.WriteError(w, "another batch apply is already running", http.StatusConflict)
		return
	}

	userID := requestUserID(req)
	progress := &BatchApplyProgress{Status: "running", Total: len(body.IDs)}
	writeBatchApplyProgress(progress)

	go func() {
		defer batchApplyMu.Unlock()
		for _, id := range body.IDs {
			if done := progress.Done + progress.Failed + progress.Skipped; done >= progress.Total {
				break
			}
			// a fresh context per item — the HTTP request that started this
			// worker is long gone
			ctx := orm.NewContext()
			ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
			ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
			ctx.Refresh = orm.WaitForRefresh
			orm.WithModel(ctx, &core.WikiGovernanceProposal{})

			var proposal core.WikiGovernanceProposal
			proposal.SetID(id)
			exists, err := orm.GetV2(ctx, &proposal)
			if err != nil || !exists {
				progress.Failed++
				progress.Errors = appendCapped(progress.Errors, fmt.Sprintf("%s: 提议不存在", id))
				progress.Done++
				writeBatchApplyProgress(progress)
				continue
			}
			if proposal.Status != core.WikiGovernanceOpen {
				progress.Skipped++
				writeBatchApplyProgress(progress)
				continue
			}
			if proposal.Type == core.WikiGovernanceTreeOrganize {
				progress.Skipped++
				progress.Errors = appendCapped(progress.Errors, fmt.Sprintf("%s: 目录提案请单独确认（会直接应用目录树）", proposal.ArticleTitle))
				writeBatchApplyProgress(progress)
				continue
			}

			progress.Current = proposal.ArticleTitle
			writeBatchApplyProgress(progress)
			applied, _, applyErr := applyOneProposal(ctx, &proposal, userID)
			if applyErr != nil || applied == "" {
				progress.Failed++
				reason := "证据不完整"
				if applyErr != nil {
					reason = applyErr.Error()
				}
				progress.Errors = appendCapped(progress.Errors, fmt.Sprintf("%s: %s", proposal.ArticleTitle, reason))
				log.Warnf("wiki: batch apply [%s/%s] failed: %v", proposal.Type, proposal.ID, applyErr)
			} else {
				progress.Done++
			}
			writeBatchApplyProgress(progress)
		}
		progress.Status = "done"
		progress.Current = ""
		writeBatchApplyProgress(progress)
		log.Infof("wiki: batch apply finished — total %d, done %d, failed %d, skipped %d",
			progress.Total, progress.Done, progress.Failed, progress.Skipped)
	}()

	h.WriteOKJSON(w, util.MapStr{"started": true, "total": len(body.IDs)})
}

func appendCapped(errs []string, msg string) []string {
	if len(errs) < 20 {
		errs = append(errs, msg)
	}
	return errs
}

// GET /wiki/governance/_batch_apply — progress of the current/last batch.
func (h *APIHandler) batchApplyProgressHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	if user, err := security.GetUserFromRequest(req); err != nil || user == nil {
		h.WriteError(w, "login required", http.StatusUnauthorized)
		return
	}
	h.WriteOKJSON(w, readBatchApplyProgress())
}
