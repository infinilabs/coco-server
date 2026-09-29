/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"context"
	"net/http"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

// Index health check (P4): one read-only sweep over the knowledge hub's
// stores answering "is every layer actually there and how much is in it" —
// the operator's preflight before blaming search quality. A store whose
// backing index is missing or unreachable reports unavailable, never fails
// the sweep.

type indexHealthEntry struct {
	Name   string `json:"name"`
	Model  string `json:"model"`
	Status string `json:"status"` // ok | unavailable
	Docs   int64  `json:"docs"`
	Note   string `json:"note,omitempty"`
}

// indexHealthModels is the sweep list: every store the knowledge hub reads
// or writes, in pipeline order (raw documents → curated wiki → ontology →
// governance → telemetry). The model is an empty instance for orm.WithModel.
func indexHealthModels() []struct {
	name  string
	model interface{}
} {
	return []struct {
		name  string
		model interface{}
	}{
		{"documents", &core.Document{}},
		{"wiki_articles", &core.WikiArticle{}},
		{"wiki_entities", &core.WikiEntity{}},
		{"wiki_versions", &core.WikiVersion{}},
		{"governance_proposals", &core.WikiGovernanceProposal{}},
		{"search_logs", &core.SearchLog{}},
	}
}

// indexHealth checks one store: a size-0 search, so the cost is one count
// per store. Errors mean missing/unreachable backing index.
func indexHealth(ctx context.Context, name string, model interface{}) indexHealthEntry {
	entry := indexHealthEntry{Name: name}
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, model)
	err, res := elastic.SearchV2WithResultItemMapper(octx, nil, orm.NewQuery().Size(0), nil)
	if err != nil {
		entry.Status, entry.Docs, entry.Note = "unavailable", -1, err.Error()
		return entry
	}
	entry.Status, entry.Docs = "ok", res.Total
	return entry
}

// indexHealthHandler serves GET /search/ops/index-health.
func (h *APIHandler) indexHealthHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	ctx := req.Context()
	entries := make([]indexHealthEntry, 0, 6)
	for _, m := range indexHealthModels() {
		entries = append(entries, indexHealth(ctx, m.name, m.model))
	}
	healthy := 0
	for _, e := range entries {
		if e.Status == "ok" {
			healthy++
		}
	}
	h.WriteJSON(w, util.MapStr{
		"healthy":    healthy,
		"total":      len(entries),
		"indices":    entries,
		"checked_at": nowMilli(),
	}, http.StatusOK)
}
