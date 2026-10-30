/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
)

// Usage-side impact tracking (the WikiSkill skill-impact.md idea, P0
// slice): when the answer pipeline pulls a wiki page into a model context,
// count it. The collector is in-memory; a flush task batches the counts
// into the articles' Impact field every few minutes so the chat hot path
// never writes to the store. Counts are programmatic telemetry — never
// written by the LLM or by hand.

const (
	impactTaskID          = "wiki-impact-flush"
	impactDefaultInterval = "5m"
)

// projectionIDPrefix is the deterministic documents-index id prefix of the
// published-article projection rows (projection.go); hits arrive under the
// projection id, the counter belongs to the article.
const projectionIDPrefix = "wikiproj_"

var (
	impactMu    sync.Mutex
	impactHits  = map[string]int{}
	impactSeen  = map[string]time.Time{}
	impactMaxIn = 10000
)

// RecordKnowledgeHits counts one answer-context usage per article. Called
// fire-and-forget from the assistant reference formatting; ids may be raw
// article ids or wikiproj_ projection ids. Entity pseudo-doc ids (entity
// rows surfaced in retrieval) do not resolve to articles and are dropped
// at flush time.
func RecordKnowledgeHits(articleIDs []string) {
	if len(articleIDs) == 0 {
		return
	}
	now := time.Now()
	impactMu.Lock()
	defer impactMu.Unlock()
	// session-level dedup is coarse (per-process recency window): the same
	// page hammered by retries within 10 minutes counts once
	for _, id := range articleIDs {
		id = strings.TrimPrefix(strings.TrimSpace(id), projectionIDPrefix)
		if id == "" {
			continue
		}
		if t, ok := impactSeen[id]; ok && now.Sub(t) < 10*time.Minute {
			continue
		}
		if len(impactHits) >= impactMaxIn {
			log.Debugf("wiki: impact collector full, dropping hit for [%s]", id)
			continue
		}
		impactHits[id]++
		impactSeen[id] = now
	}
	// prune the recency index when it drifts large
	if len(impactSeen) > impactMaxIn*2 {
		for k, t := range impactSeen {
			if now.Sub(t) > 10*time.Minute {
				delete(impactSeen, k)
			}
		}
	}
}

func init() {
	interval := strings.TrimSpace(os.Getenv("WIKI_IMPACT_INTERVAL"))
	if interval == "" {
		interval = impactDefaultInterval
	}
	switch strings.ToLower(interval) {
	case "off", "disabled", "0":
		return
	}
	task.RegisterScheduleTask(task.ScheduleTask{
		ID:          impactTaskID,
		Group:       "wiki",
		Description: "Wiki 影响计数:把回答管线对 wiki 页面的引用命中批量刷入文章 Impact 字段",
		Interval:    interval,
		Singleton:   true,
		Task:        flushImpactHits,
	})
}

// flushImpactHits drains the collector into the articles. One Get+Update
// per id is fine at answer-timeline volumes; failures drop the batch with
// a warning — telemetry must never disturb the wiki.
func flushImpactHits(ctx context.Context) {
	impactMu.Lock()
	if len(impactHits) == 0 {
		impactMu.Unlock()
		return
	}
	pending := impactHits
	impactHits = map[string]int{}
	impactMu.Unlock()

	start := time.Now()
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	octx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiArticle{})

	flushed := 0
	now := time.Now()
	for id, hits := range pending {
		var article core.WikiArticle
		article.SetID(id)
		exists, err := orm.GetV2(octx, &article)
		if err != nil || !exists {
			continue // entity pseudo-doc ids and deleted articles land here
		}
		if article.Impact == nil {
			article.Impact = &core.WikiImpact{}
		}
		article.Impact.AnswerHits += hits
		article.Impact.LastUsedAt = &now
		if err := orm.Update(octx, &article); err != nil {
			log.Warnf("wiki: impact flush failed for article [%s]: %v", id, err)
			continue
		}
		flushed++
	}
	if flushed > 0 {
		log.Debugf("wiki: impact flush wrote %d/%d article counters in %s", flushed, len(pending), time.Since(start))
	}
}
