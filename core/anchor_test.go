/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package core

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnchoredProposalIDDeterministic(t *testing.T) {
	first := AnchoredProposalID("art-1", WikiGovernanceCorrection, 1)
	assert.Equal(t, first, AnchoredProposalID("art-1", WikiGovernanceCorrection, 1),
		"same anchor/type/generation must mint the same id — the id is the lock")
	// generations 0 and 1 claim the same unsuffixed row
	assert.Equal(t, first, AnchoredProposalID("art-1", WikiGovernanceCorrection, 0))
}

func TestAnchoredProposalIDSeparatesDimensions(t *testing.T) {
	first := AnchoredProposalID("art-1", WikiGovernanceCorrection, 1)
	assert.NotEqual(t, first, AnchoredProposalID("art-2", WikiGovernanceCorrection, 1), "different anchors are different rows")
	assert.NotEqual(t, first, AnchoredProposalID("art-1", WikiGovernanceKnowledgeGap, 1), "different types are different rows")
	// every generation past the first re-keys: the refile gets its own row
	// and the resolved ancestors keep theirs
	second := AnchoredProposalID("art-1", WikiGovernanceCorrection, 2)
	assert.NotEqual(t, first, second)
	assert.NotEqual(t, second, AnchoredProposalID("art-1", WikiGovernanceCorrection, 3))
	// the id is a plain md5 hex digest — it must stay one (index ids, logs)
	assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{32}$`), first)
}

func TestAnchoredEntityIDNormalizesName(t *testing.T) {
	// extraction resolves identity by exact name: case and edge whitespace
	// must not fork one entity into twins
	assert.Equal(t, AnchoredEntityID("Gateway"), AnchoredEntityID("gateway"))
	assert.Equal(t, AnchoredEntityID("  Gateway  "), AnchoredEntityID("gateway"))
	// interior spacing and different names stay different identities
	assert.NotEqual(t, AnchoredEntityID("api gateway"), AnchoredEntityID("apigateway"))
	assert.NotEqual(t, AnchoredEntityID("gateway"), AnchoredEntityID("gateway v2"))
	// entity ids never collide with proposal ids (disjoint key namespaces)
	assert.NotEqual(t, AnchoredEntityID("anchor"), AnchoredProposalID("anchor", WikiGovernanceCorrection, 1))
}
