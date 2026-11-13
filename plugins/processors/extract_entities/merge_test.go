/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package extract_entities

import (
	"testing"

	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

// The update path used to only append sources: an entity first created
// without properties or a summary stayed frozen that way no matter how
// many later documents mentioned it (live data: 12 mentions, 0
// properties). mergeExtractedInto must backfill property keys, absorb
// aliases and take the first summary — without clobbering what's there.
func TestMergeExtractedInto(t *testing.T) {
	existing := &core.WikiEntity{
		Name:        "東海堂",
		Aliases:     []string{"Arome Bakery"},
		Properties:  util.MapStr{"founded": "1985"},
		Description: "existing summary stays",
	}

	mergeExtractedInto(existing, &extractedEntity{
		Name:       "東海堂",
		Aliases:    []string{"東海堂", "Arome", "東海堂餅屋"},
		Properties: util.MapStr{"founded": "OVERRIDDEN?", "hq": "Hong Kong", "skipme": "", "void": nil},
		Summary:    "new summary ignored",
	})

	// existing values win — provenance stability over churn
	require.Equal(t, "1985", existing.Properties["founded"])
	// new keys land; empty/nil values never do
	require.Equal(t, "Hong Kong", existing.Properties["hq"])
	require.NotContains(t, existing.Properties, "skipme")
	require.NotContains(t, existing.Properties, "void")
	// missing aliases absorb, dupes don't stack
	require.Equal(t, []string{"Arome Bakery", "Arome", "東海堂餅屋"}, existing.Aliases)
	// description only fills when empty
	require.Equal(t, "existing summary stays", existing.Description)

	empty := &core.WikiEntity{Name: "x"}
	mergeExtractedInto(empty, &extractedEntity{Aliases: []string{"y"}, Properties: util.MapStr{"k": "v"}, Summary: "  first summary  "})
	require.Equal(t, "first summary", empty.Description)
	require.Equal(t, "v", empty.Properties["k"])
	require.Equal(t, []string{"y"}, empty.Aliases)
}
