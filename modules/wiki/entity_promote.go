package wiki

import (
	"strings"

	"infini.sh/coco/core"
	"infini.sh/framework/core/log"
	"infini.sh/framework/core/orm"
)

/* entity promotion: the compile pipeline carries key entities as plain
 * strings on articles ("KeyEntities chips"); this bridges them into the
 * ontology pool at proposal-apply time so the graph / lookup / MCP surfaces
 * see people, organizations and events without a hand-create step. The
 * ontology vocabulary (declared types, lenient degrade) and the duplicate
 * guard (name/alias absorption) are the same machinery the manual create
 * endpoint uses. */

// promoteTopicEntities upserts an article's key entities into the KB's
// entity pool. Existing entities (name or alias match) are left alone —
// their wikilink binding already covers the mention; new ones land as
// proposed with the LLM-typed ontology type (concept fallback) so the
// governance queue can review and enrich them. Per-entity failures log and
// continue: promotion must never fail the governance apply.
func promoteTopicEntities(ctx *orm.Context, kbID string, names []string, types map[string]string) int {
	if kbID == "" || len(names) == 0 {
		return 0
	}
	created := 0
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if findKBEntityByNameOrAlias(ctx, name, kbID) != nil {
			continue
		}
		typ := strings.TrimSpace(types[name])
		if typ == "" {
			typ = "concept"
		}
		e := &core.WikiEntity{
			KbID:       kbID,
			Type:       typ,
			Name:       name,
			Status:     core.WikiEntityProposed,
			Confidence: 0.5,
		}
		if err := applyEntityCreateDefaults(e); err != nil {
			log.Warnf("wiki entity promotion: defaults [%s]: %v", name, err)
			continue
		}
		if err := validateEntityForKB(ctx, e, kbID); err != nil {
			// lenient repair flow: undeclared type degrades to concept
			if !degradeUndeclaredType(e, kbID) {
				log.Warnf("wiki entity promotion: vocabulary reject [%s/%s]: %v", name, typ, err)
				continue
			}
		}
		cctx := orm.NewContextWithParent(ctx)
		cctx.Set(orm.SharingEnabled, true)
		cctx.Set(orm.SharingResourceType, entityResource)
		cctx.Refresh = orm.WaitForRefresh
		if err := orm.Create(cctx, e); err != nil {
			log.Warnf("wiki entity promotion: create [%s/%s]: %v", name, e.Type, err)
			continue
		}
		created++
	}
	if created > 0 {
		log.Infof("wiki entity promotion: %d entity(ies) promoted into kb %s", created, kbID)
	}
	return created
}
