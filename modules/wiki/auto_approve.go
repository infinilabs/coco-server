/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
)

// Auto-approve (自动审批, per-KB KB.AutoApprove): the pipeline updates the
// knowledge base without the human gate. Two hooks:
//   - entities: resolveOrCreateEntity publishes new extractions directly
//   - proposals: this sweep applies open governance proposals of
//     auto-approve KBs (compile drafts land published; the tree applies;
//     rebuilds run) — corrections and knowledge-gap signals stay manual
//     in every mode, they are work signals, not KB updates.

// KBAutoApprove reports the KB's auto-approve flag; false on any error —
// the gate stays closed unless the KB explicitly opened it.
func KBAutoApprove(ctx *orm.Context, kbID string) bool {
	if kbID == "" {
		return false
	}
	kctx := orm.NewContextWithParent(ctx)
	kctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(kctx, &core.WikiKnowledgeBase{})
	var kb core.WikiKnowledgeBase
	kb.SetID(kbID)
	exists, err := orm.GetV2(kctx, &kb)
	if err != nil || !exists {
		return false
	}
	return kb.AutoApprove
}

const autoApproveBatch = 10

func init() {
	interval := strings.TrimSpace(os.Getenv("WIKI_AUTO_APPROVE_INTERVAL"))
	if interval == "" {
		interval = autoApproveDefaultInterval
	}
	task.RegisterScheduleTask(task.ScheduleTask{
		ID:          "wiki-auto-approve",
		Group:       "wiki",
		Description: "Wiki 自动审批:应用开启自动审批知识库的治理提案(实体在抽取侧直接发布)",
		Interval:    interval,
		Singleton:   true,
		Task:        autoApproveSweep,
	})
}

const envAutoApproveInterval = "WIKI_AUTO_APPROVE_INTERVAL"

// default interval: compile applies with the writing pass take ~40s each,
// so a sweep processes at most autoApproveBatch items per tick
const autoApproveDefaultInterval = "2m"

var autoApproveDefaultIntervalVar = autoApproveDefaultInterval

func autoApproveSweep(ctx context.Context) {
	start := time.Now()

	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiGovernanceProposal{})

	// oldest first — the queue drains in filing order
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(autoApproveBatch).
		Filter(orm.TermQuery("status", core.WikiGovernanceOpen)).
		SortBy(orm.Sort{Field: "created", SortType: "asc"}))
	if err != nil || res == nil {
		return
	}
	proposals, _, derr := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	if derr != nil || len(proposals) == 0 {
		return
	}

	applied, skipped, failed := 0, 0, 0
	for i := range proposals {
		p := &proposals[i]
		// work signals stay human-gated in every mode
		if p.Type == core.WikiGovernanceCorrection || p.Type == core.WikiGovernanceKnowledgeGap {
			skipped++
			continue
		}
		if !KBAutoApprove(octx, p.KbID) {
			skipped++
			continue
		}
		actx := orm.NewContextWithParent(ctx)
		actx.Set(orm.DirectReadWithoutPermissionCheck, true)
		actx.Set(orm.DirectWriteWithoutPermissionCheck, true)
		actx.Refresh = orm.WaitForRefresh
		orm.WithModel(actx, &core.WikiGovernanceProposal{})
		// reload: the sweep context did not fetch the full body
		var full core.WikiGovernanceProposal
		full.SetID(p.ID)
		exists, gerr := orm.GetV2(actx, &full)
		if gerr != nil || !exists {
			failed++
			continue
		}
		if _, _, aerr := applyOneProposal(actx, &full, "auto-approve"); aerr != nil {
			failed++
			log.Warnf("wiki: auto-approve apply [%s/%s] failed: %v", p.Type, p.ID, aerr)
			continue
		}
		applied++
	}
	if applied > 0 || failed > 0 {
		log.Infof("wiki: auto-approve sweep — applied %d, skipped %d, failed %d (took %s)",
			applied, skipped, failed, time.Since(start).Round(time.Second))
	}
	return
}
