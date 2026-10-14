/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"fmt"
	"net/http"
	"strings"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

// Correction capture (D7): when a user corrects an answer at work, the
// correction becomes a governance proposal — reviewed by a human before
// anything changes, the same gate every other proposal passes (never turn
// one user's correction directly into company knowledge). Any logged-in
// user may report; the routing hint tells the reviewer which layer the
// fix belongs to (fact, preference, technique, source conflict).

const (
	correctionMaxComment = 4000
	correctionMaxAnswer  = 4000
)

// correctionRouteHints maps the client-supplied route hint to the layer the
// fix likely belongs to (the J.B. routing table, review-side).
var correctionRouteHints = map[string]string{
	"fact_missing":    "missing fact — likely needs a new/updated wiki page",
	"fact_outdated":   "outdated fact — the page needs a new version",
	"preference":      "recurring preference — likely a compile rule/policy",
	"technique":       "validated technique — likely a skill",
	"source_conflict": "sources disagree — needs a conflict decision",
}

type correctionRequest struct {
	Query       string   `json:"query"`
	Answer      string   `json:"answer"`
	MessageID   string   `json:"message_id"`
	CitedDocIDs []string `json:"cited_doc_ids"`
	RouteHint   string   `json:"route_hint"`
	Comment     string   `json:"comment"`
	KbID        string   `json:"kb_id"`
}

// createCorrection lands a work-side correction as a governance proposal.
// POST /wiki/governance/_correction.
func (h *APIHandler) createCorrection(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := correctionRequest{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	body.Comment = strings.TrimSpace(body.Comment)
	body.Query = strings.TrimSpace(body.Query)
	if body.Comment == "" {
		h.WriteError(w, "comment is required", http.StatusBadRequest)
		return
	}
	if len(body.Comment) > correctionMaxComment {
		h.WriteError(w, fmt.Sprintf("comment too long (max %d chars)", correctionMaxComment), http.StatusBadRequest)
		return
	}
	hint, ok := correctionRouteHints[body.RouteHint]
	if !ok {
		h.WriteError(w, fmt.Sprintf("invalid route_hint, one of: fact_missing, fact_outdated, preference, technique, source_conflict"), http.StatusBadRequest)
		return
	}

	user, err := security.GetUserFromRequest(req)
	if err != nil || user == nil {
		h.WriteError(w, "login required", http.StatusUnauthorized)
		return
	}

	anchor := correctionAnchor(body.MessageID, body.Query, body.Comment)

	answer := strings.TrimSpace(body.Answer)
	if len(answer) > correctionMaxAnswer {
		answer = answer[:correctionMaxAnswer]
	}
	evidence := util.MapStr{
		"query":         body.Query,
		"answer":        answer,
		"cited_doc_ids": body.CitedDocIDs,
		"route_hint":    body.RouteHint,
		"comment":       body.Comment,
		"reported_by":   user.UserID,
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	// operator-paced read-after-write: the reporter's next view of the
	// governance queue must show the proposal
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})

	// the anchor's row id is deterministic: two reports racing through the
	// check-then-create window land on the same id, the loser overwrites
	// byte-identical content instead of spawning a queue duplicate
	existing, generation := proposalAnchorState(ctx, anchor, core.WikiGovernanceCorrection)
	if existing != nil {
		// a second report on the same answer strengthens the existing
		// proposal instead of spamming the queue
		count := 1
		if v, ok := existing.Evidence["report_count"].(int); ok {
			count = v + 1
		} else if v, ok := existing.Evidence["report_count"].(float64); ok {
			count = int(v) + 1
		}
		evidence["report_count"] = count
		existing.Evidence = evidence
		if err := orm.Update(ctx, existing); err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.WriteOKJSON(w, util.MapStr{"id": existing.ID, "report_count": evidence["report_count"]})
		return
	}

	// first report — or a reopen after the previous proposal was resolved
	// (the generation suffix gives the refile its own row, the resolved one
	// stays as audit history)
	evidence["report_count"] = 1
	proposal := &core.WikiGovernanceProposal{
		KbID:         body.KbID,
		ArticleID:    anchor,
		ArticleTitle: truncateForTitle(body.Query),
		Type:         core.WikiGovernanceCorrection,
		Status:       core.WikiGovernanceOpen,
		Reason:       hint,
		Evidence:     evidence,
	}
	proposal.ID = core.AnchoredProposalID(anchor, core.WikiGovernanceCorrection, generation+1)
	if err := orm.Create(ctx, proposal); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, util.MapStr{"id": proposal.ID, "report_count": 1})
}

// correctionAnchor is the proposal idempotency anchor: the same correction
// submitted twice is one proposal (counter bumps), but a *different*
// correction of the same message/query is its own proposal — hashing in the
// comment keeps the first report's evidence from being clobbered by the
// second.
func correctionAnchor(messageID, query, comment string) string {
	base := messageID
	if base == "" {
		base = query
	}
	return "correction:" + util.MD5digest(base+"|"+comment)
}

func truncateForTitle(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 120 {
		return s[:120]
	}
	if s == "" {
		return "correction"
	}
	return s
}
