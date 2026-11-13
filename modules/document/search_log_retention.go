/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
)

// Search-log retention (W6): the telemetry that feeds the ops overview and
// the knowledge-gap loop is a log, not an archive — without a cutoff it
// grows forever. A daily singleton sweep deletes entries older than the
// retention window; the numbers the overview shows are recent behavior
// anyway.

const (
	searchLogRetentionTaskID      = "search-log-retention"
	searchLogDefaultInterval      = "24h"
	searchLogDefaultRetentionDays = 30
)

func init() {
	registerSearchLogRetentionTask()
}

func registerSearchLogRetentionTask() {
	interval := strings.TrimSpace(os.Getenv("SEARCH_LOG_RETENTION_INTERVAL"))
	switch strings.ToLower(interval) {
	case "":
		interval = searchLogDefaultInterval
	case "off", "disabled", "0":
		return
	}
	task.RegisterScheduleTask(task.ScheduleTask{
		ID:          searchLogRetentionTaskID,
		Group:       "document",
		Description: "Search log retention: delete telemetry entries past the retention window (default 30 days, SEARCH_LOG_RETENTION_DAYS)",
		Interval:    interval,
		Singleton:   true,
		Task:        sweepSearchLogs,
	})
}

// searchLogRetentionCutoff returns the delete-before boundary. Pure so the
// retention math is testable.
func searchLogRetentionCutoff(now time.Time, days int) time.Time {
	if days < 1 {
		days = searchLogDefaultRetentionDays
	}
	return now.AddDate(0, 0, -days)
}

// sweepSearchLogs deletes every search log older than the retention window.
// One delete-by-query per day, best-effort: a failed sweep leaves the logs
// in place and tries again on the next tick.
func sweepSearchLogs(ctx context.Context) {
	days := searchLogDefaultRetentionDays
	if v := strings.TrimSpace(os.Getenv("SEARCH_LOG_RETENTION_DAYS")); v != "" {
		if parsed, err := parsePositiveInt(v); err == nil {
			days = parsed
		}
	}
	cutoff := searchLogRetentionCutoff(time.Now(), days)

	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	octx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.SearchLog{})

	resp, err := orm.DeleteByQuery(octx, orm.NewQuery().Size(searchLogScanLimit).
		Filter(orm.Range("created").Lt(cutoff)))
	if err != nil {
		log.Warnf("search log retention sweep failed: %v", err)
		return
	}
	if resp != nil && resp.Deleted > 0 {
		log.Infof("search log retention: deleted %d entries older than %s", resp.Deleted, cutoff.Format(time.RFC3339))
	}
}

func parsePositiveInt(s string) (int, error) {
	out, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if out < 1 {
		return 0, strconv.ErrRange
	}
	return out, nil
}
