/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"testing"

	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

func TestRetargetEdgeList(t *testing.T) {
	canonicalID := "ent-canonical"

	t.Run("edges to absorbed ids are rewritten to canonical", func(t *testing.T) {
		rels := []core.WikiEntityRelation{
			{TargetID: "ent-gone", TargetType: "software", Relation: "integrates_with", Provenance: "doc1"},
			{TargetID: "ent-live", TargetType: "software", Relation: "mentions", Provenance: "doc2"},
		}
		out, changed := retargetEdgeList(rels, map[string]bool{"ent-gone": true}, canonicalID, "platform")
		require.True(t, changed)
		require.Len(t, out, 2)
		require.Equal(t, canonicalID, out[0].TargetID)
		require.Equal(t, "platform", out[0].TargetType) // canonical type wins
		require.Equal(t, "integrates_with", out[0].Relation)
		require.Equal(t, "doc1", out[0].Provenance) // provenance preserved
		require.Equal(t, "ent-live", out[1].TargetID)
	})

	t.Run("retargeted edge duplicates an existing canonical edge", func(t *testing.T) {
		rels := []core.WikiEntityRelation{
			{TargetID: canonicalID, TargetType: "platform", Relation: "integrates_with", Provenance: "doc0"},
			{TargetID: "ent-gone", TargetType: "software", Relation: "integrates_with", Provenance: "doc1"},
		}
		out, changed := retargetEdgeList(rels, map[string]bool{"ent-gone": true}, canonicalID, "platform")
		require.True(t, changed)
		require.Len(t, out, 1)
		require.Equal(t, "doc0", out[0].Provenance) // pre-existing edge wins
	})

	t.Run("edges already pointing at canonical are kept", func(t *testing.T) {
		rels := []core.WikiEntityRelation{
			{TargetID: canonicalID, TargetType: "platform", Relation: "mentions", Provenance: "doc1"},
		}
		out, changed := retargetEdgeList(rels, map[string]bool{"ent-gone": true}, canonicalID, "platform")
		require.False(t, changed)
		require.Len(t, out, 1)
		require.Equal(t, canonicalID, out[0].TargetID)
	})

	t.Run("no absorbed hits leaves the list untouched", func(t *testing.T) {
		rels := []core.WikiEntityRelation{
			{TargetID: "ent-live", Relation: "mentions"},
			{TargetID: "", Relation: "unresolved"}, // placeholder, pass through
		}
		out, changed := retargetEdgeList(rels, map[string]bool{"ent-gone": true}, canonicalID, "platform")
		require.False(t, changed)
		require.Len(t, out, 2)
		require.Equal(t, "ent-live", out[0].TargetID)
		require.Equal(t, "", out[1].TargetID)
	})

	t.Run("nil and empty absorbed sets are no-ops", func(t *testing.T) {
		out, changed := retargetEdgeList(nil, map[string]bool{"x": true}, canonicalID, "")
		require.False(t, changed)
		require.Empty(t, out)

		rels := []core.WikiEntityRelation{{TargetID: "ent-live", Relation: "mentions"}}
		out2, changed2 := retargetEdgeList(rels, map[string]bool{}, canonicalID, "")
		require.False(t, changed2)
		require.Len(t, out2, 1)
	})

	t.Run("non-destructive on the input slice", func(t *testing.T) {
		rels := []core.WikiEntityRelation{
			{TargetID: "ent-gone", Relation: "mentions"},
			{TargetID: "ent-live", Relation: "mentions"},
		}
		_, _ = retargetEdgeList(rels, map[string]bool{"ent-gone": true, "ent-live": true}, canonicalID, "")
		// input must be unchanged (no rels[:0] aliasing)
		require.Equal(t, "ent-gone", rels[0].TargetID)
		require.Equal(t, "ent-live", rels[1].TargetID)
	})
}
