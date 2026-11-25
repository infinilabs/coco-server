/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"encoding/json"
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
//     auto-approve KBs (compile drafts land published; the tree applies)
//     — corrections, knowledge-gap signals AND page rebuilds stay manual
//     in every mode. A rebuild wholesale replaces a published page's body
//     (observed: an auto-approve KB lost 28 published articles' content in
//     one sweep, each swapped for a thin source-claims list); that is a
//     work signal needing judgment, not a routine KB update.

// KBAutoApprove reports the KB's auto-approve flag; false on any error —
// the gate stays closed unless the KB explicitly opened it.
func KBAutoApprove(ctx *orm.Context, kbID string) bool {
	if kbID == "" {
		return false
	}
	kctx := orm.NewContextWithParent(ctx)
	kctx.DirectReadAccess()
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

// autoApproveMinBodyChars keeps stub-thin compile drafts out of auto
// publishing: a page whose whole body is under this many runes stays open
// for the human gate. Auto-approving 100-char pages is how KBs filled up
// with thin published stubs (observed: 92-191 chars, 7 pages in one KB).
const autoApproveMinBodyChars = 300

// autoApproveBodyChars sums a compile proposal's section bodies (runes).
// Evidence shape varies by storage path (live struct vs ES round-trip), so
// every plausible encoding is handled; undecodable evidence counts as 0 —
// such proposals fall back to the gate, never bypass it.
func autoApproveBodyChars(p *core.WikiGovernanceProposal) int {
	var secs []core.WikiSection
	switch v := p.Evidence["sections"].(type) {
	case []core.WikiSection:
		secs = v
	case json.RawMessage:
		if err := json.Unmarshal(v, &secs); err != nil {
			return 0
		}
	case []byte:
		if err := json.Unmarshal(v, &secs); err != nil {
			return 0
		}
	case string:
		if err := json.Unmarshal([]byte(v), &secs); err != nil {
			return 0
		}
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return 0
		}
		if err := json.Unmarshal(raw, &secs); err != nil {
			return 0
		}
	}
	n := 0
	for i := range secs {
		n += len([]rune(strings.TrimSpace(secs[i].Body)))
	}
	return n
}

// currentArticleBodyChars returns the rune length of the article's live
// content, 0 when the article is missing or unreadable (no shrink guard).
func currentArticleBodyChars(parent context.Context, articleID string) int {
	acx := orm.NewContextWithParent(parent)
	acx.DirectReadAccess()
	orm.WithModel(acx, &core.WikiArticle{})
	var cur core.WikiArticle
	cur.SetID(articleID)
	ok, err := orm.GetV2(acx, &cur)
	if err != nil || !ok {
		return 0
	}
	return len([]rune(cur.Content))
}

// compilePatchBodyChars returns the rune length of a compile_patch's full
// replacement body (Evidence["content"]); 0 when absent — such proposals
// fall back to the gate, never bypass it.
func compilePatchBodyChars(p *core.WikiGovernanceProposal) int {
	if p == nil || p.Evidence == nil {
		return 0
	}
	s, _ := p.Evidence["content"].(string)
	return len([]rune(strings.TrimSpace(s)))
}

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
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.WikiGovernanceProposal{})

	// oldest first — the queue drains in filing order. The window is wider
	// than the batch so orphaned / manual-KB proposals at the head of the
	// queue are skipped over instead of starving auto-approve KBs behind
	// them (head-of-line blocking used to be silent: nothing applied, no
	// log, every tick).
	const windowSize = autoApproveBatch * 10
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(windowSize).
		Filter(orm.TermQuery("status", core.WikiGovernanceOpen)).
		Filter(orm.ExistsQuery("kb_id")).
		SortBy(orm.Sort{Field: "created", SortType: "asc"}))
	if err != nil || res == nil {
		log.Errorf("wiki: auto-approve sweep search failed: %v (res nil: %v)", err, res == nil)
		return
	}
	proposals, _, derr := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	if derr != nil {
		log.Errorf("wiki: auto-approve sweep decode failed: %v", derr)
		return
	}
	if len(proposals) == 0 {
		return
	}

	applied, skipped, failed := 0, 0, 0
	skipGated, skipManualKB, skipThin, skipShrink := 0, 0, 0, 0
	for i := range proposals {
		p := &proposals[i]
		// work signals stay human-gated in every mode: corrections and
		// knowledge gaps are work queues, and stale/refresh rebuilds
		// wholesale replace a published page's body — a mass auto-rebuild
		// once overwrote 28 published articles with thin claims lists
		if p.Type == core.WikiGovernanceCorrection || p.Type == core.WikiGovernanceKnowledgeGap ||
			p.Type == core.WikiGovernanceStale || p.Type == core.WikiGovernanceArticleRefresh {
			skipped++
			skipGated++
			continue
		}
		if !KBAutoApprove(octx, p.KbID) {
			skipped++
			skipManualKB++
			continue
		}
		if applied >= autoApproveBatch {
			break // cap the work per tick, the rest waits for the next one
		}
		actx := orm.NewContextWithParent(ctx)
		actx.DirectAccess()
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
		// thin-page floor: stub-thin compile drafts stay human-gated even
		// in auto-approve KBs (they stay open in the queue, nothing lost)
		if p.Type == compileProposalType && autoApproveBodyChars(&full) < autoApproveMinBodyChars {
			skipped++
			skipThin++
			continue
		}
		// shrink guard: an agent compile_patch that would replace a
		// substantially longer live body with a fraction of it is almost
		// always a mis-read (the agent perceived an empty page) — hand it
		// to a human instead of silently overwriting a full article with a
		// stub. compile_patch has no thin-page floor (evidence carries the
		// full replacement body in "content", not sections), so this is its
		// only automatic quality gate.
		if p.Type == core.WikiGovernanceCompilePatch && p.ArticleID != "" {
			newChars := compilePatchBodyChars(&full)
			if curChars := currentArticleBodyChars(ctx, p.ArticleID); curChars > 0 && newChars*2 < curChars {
				skipped++
				skipShrink++
				log.Warnf("wiki: auto-approve shrink guard [%s/%s]: new=%d current=%d chars — gated for manual review",
					p.Type, p.ID, newChars, curChars)
				continue
			}
		}
		if _, _, aerr := applyOneProposal(actx, &full, "auto-approve"); aerr != nil {
			failed++
			log.Warnf("wiki: auto-approve apply [%s/%s] failed: %v", p.Type, p.ID, aerr)
			continue
		}
		applied++
	}
	if applied > 0 || failed > 0 || skipShrink > 0 {
		log.Infof("wiki: auto-approve sweep — applied %d, skipped %d (gated %d, manual-kb %d, thin %d, shrink %d), failed %d (took %s)",
			applied, skipped, skipGated, skipManualKB, skipThin, skipShrink, failed, time.Since(start).Round(time.Second))
	} else if skipped > 0 {
		// full-skip ticks used to be completely silent, which made stuck
		// queues indistinguishable from a dead sweep — say why, once
		log.Infof("wiki: auto-approve sweep — nothing applied (%d queued): gated=%d manual-kb=%d thin=%d (took %s)",
			len(proposals), skipGated, skipManualKB, skipThin, time.Since(start).Round(time.Second))
	}
	return
}
