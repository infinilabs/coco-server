package wiki

import (
	"strings"

	"infini.sh/coco/core"
	"infini.sh/framework/core/log"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
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
// governance queue can review and enrich them. descriptions (may be nil)
// ground a new entity's Description in the topic's own claims — promoted
// entities used to land description-less and only healed when a later
// extraction happened to re-hit the name. Per-entity failures log and
// continue: promotion must never fail the governance apply.
func promoteTopicEntities(ctx *orm.Context, kbID string, names []string, types map[string]string, descriptions map[string]string) int {
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
			// grounded in the topic's own claims (first statement naming
			// the entity); extract_entities' merge backfill only writes
			// into an EMPTY description, so this first-write wins and is
			// never clobbered by later pipeline summaries
			Description: strings.TrimSpace(descriptions[name]),
		}
		// name-anchored id (same scheme as extract_entities): promotion and
		// extraction racing on the same name must land on ONE row — generated
		// ids here let a promote shell and an extraction row coexist as
		// twins that never self-merge (live data: Deepseek/DeepSeek ×3)
		e.SetID(core.AnchoredEntityID(kbID, name))
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
		// apply often runs from the background auto-approve sweep with no
		// session user: sharing hooks then reject the create ("invalid
		// user info"). Fall back to a direct write owned by the KB owner
		// so promoted entities stay visible to sharing-scoped reads.
		if sessionUser, uerr := security.GetUserFromContext(cctx.Context); uerr != nil || sessionUser == nil {
			cctx.DirectAccess()
			if ownerID := kbOwnerID(ctx, kbID); ownerID != "" {
				e.SetSystemValue(orm.OwnerIDKey, ownerID)
			}
		} else {
			cctx.Set(orm.SharingEnabled, true)
			cctx.Set(orm.SharingResourceType, entityResource)
		}
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

	// twin reconcile: promote shells and pre-bind extraction rows coexist
	// when a datasource was ingested before KB binding (resolveKB found no
	// KB then) — the shell gets cited while relations/sources stay on the
	// kb-less twin and the graph loses its typed edges. Fold the twins in
	// after every promotion pass so identity converges without a pipeline
	// re-run. Idempotent: a clean pool short-circuits on the twin scan.
	rctx := orm.NewContextWithParent(ctx)
	rctx.DirectAccess()
	if n, rerr := ReconcileEntityTwins(rctx, kbID); rerr != nil {
		log.Warnf("wiki entity promotion: twin reconcile kb %s: %v", kbID, rerr)
	} else if n > 0 {
		log.Infof("wiki entity promotion: reconciled %d kb-less twin(s) into kb %s", n, kbID)
	}
	return created
}

// entityDescriptionsFromStatements grounds a one-sentence description for
// each promoted entity in the topic's own claims — the same philosophy as
// extract_entities' per-document summary, minus the LLM call: the apply
// path often runs inside the auto-approve sweep and must stay cheap and
// deterministic. The first claim naming the entity (case-insensitive)
// wins; names no claim mentions fall back to the topic's own summary, so
// promotion never lands a bare shell (live data: 4 promoted entities with
// no description at all, invisible to the description-bearing search
// route until a later extraction happened to re-hit the name).
func entityDescriptionsFromStatements(names []string, statements []compileStatement, fallback string) map[string]string {
	if len(names) == 0 {
		return nil
	}
	fallback = strings.TrimSpace(fallback)
	if r := []rune(fallback); len(r) > 160 {
		fallback = string(r[:160]) + "…"
	}
	desc := make(map[string]string, len(names))
	for _, name := range names {
		n := strings.TrimSpace(name)
		if n == "" || desc[n] != "" {
			continue
		}
		lower := strings.ToLower(n)
		for _, st := range statements {
			claim := strings.TrimSpace(st.Claim)
			if claim == "" || !strings.Contains(strings.ToLower(claim), lower) {
				continue
			}
			if r := []rune(claim); len(r) > 160 {
				claim = string(r[:160]) + "…"
			}
			desc[n] = claim
			break
		}
		// no claim names this entity: the topic summary is still grounded
		// provenance (KeyEntities were extracted from this very page) and
		// merge-extract deliberately never overwrites a non-empty
		// description, so leaving it blank would freeze the shell forever
		if desc[n] == "" && fallback != "" {
			desc[n] = fallback
		}
	}
	return desc
}

// kbOwnerID reads the KB's system owner so background-promoted entities
// inherit it: sharing-scoped entity reads filter by owner, and an
// ownerless row would be invisible outside admin.
func kbOwnerID(ctx *orm.Context, kbID string) string {
	var kb core.WikiKnowledgeBase
	kb.SetID(kbID)
	rctx := orm.NewContextWithParent(ctx)
	rctx.DirectReadAccess()
	orm.WithModel(rctx, &core.WikiKnowledgeBase{})
	if exists, err := orm.GetV2(rctx, &kb); err != nil || !exists {
		return ""
	}
	if v, ok := kb.GetSystemValue(orm.OwnerIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
