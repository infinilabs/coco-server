/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package extract_entities

import (
	"testing"

	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

// Re-extraction is the self-heal window for dangling relation edges: the
// same-name dedup deletes superseded entity rows and the enabled:false
// relations mapping made retargeting a no-op, so stored edges keep
// pointing at ids no row answers to (189 of 224 live edges dangled at the
// time of writing). The prune must keep batch-resolved targets and alive
// lookups, drop only confirmed-dead ones, and never touch empty targets.
func TestStaleRelationTargets(t *testing.T) {
	batchAlive := map[string]bool{"ent-live-1": true, "ent-live-2": true}
	rels := []core.WikiEntityRelation{
		{TargetID: "ent-live-1", Relation: "serves"},
		{TargetID: "ent-live-2", Relation: "serves"},
		{TargetID: "ent-live-1", Relation: "bundled_with"}, // dedup, same target
		{TargetID: "ent-gone-a", Relation: "bundled_with"},
		{TargetID: "ent-gone-b", Relation: "serves"},
		{TargetID: "", Relation: "serves"}, // legacy junk: never looked up
	}
	stale := staleRelationTargets(rels, batchAlive)
	require.ElementsMatch(t, []string{"ent-gone-a", "ent-gone-b"}, stale)

	// no stale candidates → no lookup, no churn
	require.Empty(t, staleRelationTargets(nil, batchAlive))
	require.Empty(t, staleRelationTargets([]core.WikiEntityRelation{{TargetID: "ent-live-1"}}, batchAlive))
}

func TestDropDeadRelations(t *testing.T) {
	rels := []core.WikiEntityRelation{
		{TargetID: "ent-live", Relation: "serves", Provenance: "doc-1"},
		{TargetID: "ent-gone", Relation: "serves", Provenance: "doc-1"},
		{TargetID: "ent-gone", Relation: "bundled_with", Provenance: "doc-2"},
		{TargetID: "", Relation: "serves"},
	}
	kept, dropped := dropDeadRelations(rels, map[string]bool{"ent-gone": true})
	require.Equal(t, 2, dropped)
	require.Len(t, kept, 2)
	require.Equal(t, "ent-live", kept[0].TargetID)
	require.Equal(t, "", kept[1].TargetID) // empty target passes through untouched

	// no confirmed-dead ids → slice returned unchanged
	kept2, dropped2 := dropDeadRelations(rels, nil)
	require.Zero(t, dropped2)
	require.Equal(t, rels, kept2)

	// every resolvable target dead → only the legacy empty-target edge survives
	kept3, dropped3 := dropDeadRelations(rels, map[string]bool{"ent-live": true, "ent-gone": true})
	require.Equal(t, 3, dropped3)
	require.Len(t, kept3, 1)
	require.Equal(t, "", kept3[0].TargetID)
}
