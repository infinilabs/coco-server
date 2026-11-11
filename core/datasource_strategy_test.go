/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncConfigEffectiveStrategy(t *testing.T) {
	// empty and unknown normalize to interval — the historical default —
	// so pre-existing records and API payloads keep their behavior
	assert.Equal(t, SyncStrategyInterval, SyncConfig{Strategy: ""}.EffectiveStrategy())
	assert.Equal(t, SyncStrategyInterval, SyncConfig{Strategy: "cron"}.EffectiveStrategy())

	assert.Equal(t, SyncStrategyInterval, SyncConfig{Strategy: SyncStrategyInterval}.EffectiveStrategy())
	assert.Equal(t, SyncStrategyManual, SyncConfig{Strategy: SyncStrategyManual}.EffectiveStrategy())
	assert.Equal(t, SyncStrategyRealtime, SyncConfig{Strategy: SyncStrategyRealtime}.EffectiveStrategy())
}
