/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotifyPendingMemoriesGuards(t *testing.T) {
	// zero count and empty user are no-ops — both must not panic even with
	// no store registered (the recover guard catches store panics)
	NotifyPendingMemories("", 3)
	NotifyPendingMemories("u-1", 0)
	NotifyPendingMemories("u-1", -1)
	assert.True(t, true, "reached here = no panic")
}
