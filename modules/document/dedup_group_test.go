/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDedupGroupKeyOrderIndependent(t *testing.T) {
	a := DedupGroupKey([]string{"doc-1", "doc-2", "doc-3"})
	b := DedupGroupKey([]string{"doc-3", "doc-1", "doc-2"})
	assert.Equal(t, a, b, "member order must not change the group key")

	c := DedupGroupKey([]string{"doc-1", "doc-2"})
	assert.NotEqual(t, a, c, "different member sets produce different keys")
}

func TestConfirmedGroupForDoc(t *testing.T) {
	groups := map[string][]string{
		"key-a": {"doc-1", "doc-2"},
	}

	key, members, ok := confirmedGroupForDoc(groups, "doc-2")
	require.True(t, ok)
	assert.Equal(t, "key-a", key)
	assert.Equal(t, []string{"doc-1", "doc-2"}, members)

	_, _, ok = confirmedGroupForDoc(groups, "doc-9")
	assert.False(t, ok)
}
