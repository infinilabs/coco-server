/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package core

import (
	"strconv"
	"strings"

	"infini.sh/framework/core/util"
)

// Deterministic row ids for check-then-create write paths (TOCTOU): every
// writer that mints the same logical record derives the same id, so a race
// through the check window makes the loser overwrite byte-identical content
// instead of leaving a duplicate row. This is the locking discipline the
// stores do not give us — the id IS the lock.

// AnchoredProposalID mints the row id for a governance proposal from its
// idempotency anchor (article_id + type) and its generation. Generation >1
// suffixes the key: a refiled anchor — a correction reported again after the
// previous proposal was resolved, an orphan re-detected after a dismissal —
// opens a fresh row while the resolved rows stay as audit history.
func AnchoredProposalID(anchor, pType string, generation int) string {
	key := "proposal|" + pType + "|" + anchor
	if generation > 1 {
		key += "|#" + strconv.Itoa(generation)
	}
	return util.MD5digest(key)
}

// AnchoredEntityID mints the name-anchored id for pipeline-proposed
// entities. Extraction resolves identity by name (exact, then alias), so
// workers racing on the same name write the same row instead of leaving
// twin proposals that never self-merge; the loser's overwrite differs only
// in provenance, and the next extraction pass re-appends missing sources.
// Manual entity creation is NOT anchored — it keeps generated ids and its
// own review flow.
func AnchoredEntityID(name string) string {
	return util.MD5digest("entity|" + strings.ToLower(strings.TrimSpace(name)))
}
