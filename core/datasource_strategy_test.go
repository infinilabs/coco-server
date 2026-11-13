/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSyncConfigEffectiveStrategy(t *testing.T) {
	// empty and unknown normalize to interval — the historical default —
	// so pre-existing records and API payloads keep their behavior
	assert.Equal(t, SyncStrategyInterval, SyncConfig{Strategy: ""}.EffectiveStrategy())
	assert.Equal(t, SyncStrategyInterval, SyncConfig{Strategy: "bogus"}.EffectiveStrategy())

	assert.Equal(t, SyncStrategyInterval, SyncConfig{Strategy: SyncStrategyInterval}.EffectiveStrategy())
	assert.Equal(t, SyncStrategyManual, SyncConfig{Strategy: SyncStrategyManual}.EffectiveStrategy())
	assert.Equal(t, SyncStrategyRealtime, SyncConfig{Strategy: SyncStrategyRealtime}.EffectiveStrategy())
	assert.Equal(t, SyncStrategyCron, SyncConfig{Strategy: SyncStrategyCron}.EffectiveStrategy())
}

// stamp formats a local time the way FormatTimeWithLocalTZ writes the
// last-access kv stamp — the format CronDue must parse back.
func stamp(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05")
}

func TestSyncConfigCronDue(t *testing.T) {
	dailyMidnight := "0 0 0 * * *"
	now := time.Now()

	t.Run("invalid or empty crontab is never due", func(t *testing.T) {
		assert.False(t, SyncConfig{Strategy: SyncStrategyCron, Crontab: ""}.CronDue("", now))
		assert.False(t, SyncConfig{Strategy: SyncStrategyCron, Crontab: "not a cron"}.CronDue("", now))
		assert.False(t, SyncConfig{Strategy: SyncStrategyCron, Crontab: "0 0 25 * *"}.CronDue("", now))
	})

	t.Run("never ran syncs immediately", func(t *testing.T) {
		assert.True(t, SyncConfig{Strategy: SyncStrategyCron, Crontab: dailyMidnight}.CronDue("", now))
		assert.True(t, SyncConfig{Strategy: SyncStrategyCron, Crontab: dailyMidnight}.CronDue("   ", now))
	})

	t.Run("unparsable stamp falls back to never-ran", func(t *testing.T) {
		// must not panic (util.ParseTimeWithLocalTZ panics on bad input)
		assert.True(t, SyncConfig{Strategy: SyncStrategyCron, Crontab: dailyMidnight}.CronDue("garbage", now))
	})

	t.Run("due after crossing a slot", func(t *testing.T) {
		// last ran yesterday 23:00, now is today 00:30: midnight slot passed
		assert.True(t, SyncConfig{Strategy: SyncStrategyCron, Crontab: dailyMidnight}.
			CronDue(stamp(now.AddDate(0, 0, -1).Add(-time.Hour)), now))
	})

	t.Run("not due before the next slot", func(t *testing.T) {
		// last ran yesterday 23:00, now is yesterday 23:30: next slot (midnight) is in the future
		assert.False(t, SyncConfig{Strategy: SyncStrategyCron, Crontab: dailyMidnight}.
			CronDue(stamp(now.AddDate(0, 0, -1).Add(-time.Hour)), now.AddDate(0, 0, -1).Add(30*time.Minute)))
	})

	t.Run("exact slot moment does not double-fire", func(t *testing.T) {
		// ran exactly at midnight: NextTime(last) is tomorrow — not due now
		assert.False(t, SyncConfig{Strategy: SyncStrategyCron, Crontab: dailyMidnight}.
			CronDue(stamp(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)), now))
	})

	t.Run("missed slot catches up once", func(t *testing.T) {
		// server down for 3 days: last stamp predates several slots, still one catch-up
		assert.True(t, SyncConfig{Strategy: SyncStrategyCron, Crontab: dailyMidnight}.
			CronDue(stamp(now.AddDate(0, 0, -3)), now))
	})

	t.Run("weekly expression respects day-of-week", func(t *testing.T) {
		weeklyMonday2am := "0 0 2 * * 1"
		// find last Monday 02:00 relative to now, then probe around it
		last := now.AddDate(0, 0, -7)
		for last.Weekday() != time.Monday {
			last = last.AddDate(0, 0, 1)
		}
		last = time.Date(last.Year(), last.Month(), last.Day(), 2, 0, 0, 0, time.Local)
		cfg := SyncConfig{Strategy: SyncStrategyCron, Crontab: weeklyMonday2am}
		assert.True(t, cfg.CronDue(stamp(last.Add(-time.Hour)), last.Add(30*time.Minute)))
		assert.False(t, cfg.CronDue(stamp(last.Add(time.Hour)), last.Add(2*time.Hour)))
	})
}
