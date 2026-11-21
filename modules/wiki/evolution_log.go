/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"fmt"
	"net/http"
	"strconv"

	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

// The KB-level evolution log (the WikiSkill "evolution log" idea): an
// append-only record of what changed in a knowledge base and why —
// applications AND dismissals, humans AND the auto-approve sweep. Rejected
// entries are kept on purpose: downstream proposers read the log so the
// same fix is not re-proposed forever.
//
// Entries are written best-effort by the programs that act on proposals;
// a log failure never fails the action itself.

// AppendEvolutionEntry records one decision. Fire-and-forget by contract:
// callers must not branch on its error.
func AppendEvolutionEntry(parent *orm.Context, kbID, kind, actor, targetType, targetID, summary string, detail map[string]interface{}) {
	if kbID == "" || kind == "" {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Warnf("wiki: evolution log append recovered: %v", r)
		}
	}()
	entry := &core.WikiEvolutionEntry{
		KbID:       kbID,
		Kind:       kind,
		Actor:      actor,
		TargetType: targetType,
		TargetID:   targetID,
		Summary:    summary,
		Detail:     detail,
	}
	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	if err := orm.Create(ctx, entry); err != nil {
		log.Warnf("wiki: evolution log append failed for kb [%s]: %v", kbID, err)
	}
}

// recordProposalApplied hooks applyOneProposal's success path — the single
// choke point shared by the manual endpoint, the batch worker and the
// auto-approve sweep (actor distinguishes them).
func recordProposalApplied(ctx *orm.Context, proposal *core.WikiGovernanceProposal, actor, applied string) {
	if proposal == nil || applied == "" {
		return
	}
	AppendEvolutionEntry(ctx, proposal.KbID, core.WikiEvolutionProposalApplied, actor,
		"governance", proposal.ID,
		applied,
		map[string]interface{}{
			"proposal_type": proposal.Type,
			"article_id":    proposal.ArticleID,
			"article_title": proposal.ArticleTitle,
		})
}

// recordProposalDismissed hooks the status endpoint: a human retired the
// proposal without executing its fix.
func recordProposalDismissed(ctx *orm.Context, proposal *core.WikiGovernanceProposal, actor string) {
	if proposal == nil {
		return
	}
	AppendEvolutionEntry(ctx, proposal.KbID, core.WikiEvolutionProposalDismissed, actor,
		"governance", proposal.ID,
		proposal.Reason,
		map[string]interface{}{
			"proposal_type": proposal.Type,
			"article_id":    proposal.ArticleID,
			"article_title": proposal.ArticleTitle,
		})
}

// evolutionLog keeps the most recent lines — the log exists to be read
// back by humans and proposers, not to grow without bound.
const evolutionLogDefaultSize = 50
const evolutionLogMaxSize = 200

func (h *APIHandler) evolutionLog(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	from, _ := strconv.Atoi(req.FormValue("from"))
	size, _ := strconv.Atoi(req.FormValue("size"))
	if size <= 0 {
		size = evolutionLogDefaultSize
	}
	if size > evolutionLogMaxSize {
		size = evolutionLogMaxSize
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiEvolutionEntry{})

	builder := orm.NewQuery()
	builder.From(from)
	builder.Size(size)
	builder.Filter(orm.TermQuery("kb_id", kbID))
	builder.SortBy(orm.Sort{Field: "created", SortType: "desc"})

	res, err := orm.SearchV2(ctx, builder)
	if err != nil || res == nil {
		if err == nil {
			err = fmt.Errorf("empty search response")
		}
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	entries, total, derr := elastic.DecodeHits[core.WikiEvolutionEntry](res)
	if derr != nil {
		h.WriteError(w, derr.Error(), http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []core.WikiEvolutionEntry{}
	}
	h.WriteOKJSON(w, util.MapStr{
		"total":   total,
		"entries": entries,
	})
}
