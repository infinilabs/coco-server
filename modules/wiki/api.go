/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// statusTransitions is the article review workflow (design doc §4.2):
// publishing always requires an explicit human action.
var statusTransitions = map[string][]string{
	core.WikiArticleDraft:     {core.WikiArticleReviewed, core.WikiArticleArchived},
	core.WikiArticleReviewed:  {core.WikiArticlePublished, core.WikiArticleDraft, core.WikiArticleArchived},
	core.WikiArticlePublished: {core.WikiArticleArchived, core.WikiArticleReviewed},
	core.WikiArticleArchived:  {core.WikiArticleDraft},
}

func validateArticleStatus(status string) error {
	if _, ok := statusTransitions[status]; ok {
		return nil
	}
	return fmt.Errorf("invalid article status: %s", status)
}

/* ---------------- TOC ---------------- */

func (h *APIHandler) getToc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.WikiToc{})

	toc, found, err := findToc(ctx, kbID)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		// an empty tree is a valid state for a new KB
		toc = &core.WikiToc{KbID: kbID, Nodes: []core.WikiTocNode{}}
	}
	h.WriteGetOKJSON(w, toc.ID, util.MapStr{"kb_id": kbID, "nodes": toc.Nodes})
}

func (h *APIHandler) updateToc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")

	var nodes []core.WikiTocNode
	if err := h.DecodeJSON(req, &nodes); err != nil {
		h.Error400(w, err.Error())
		return
	}
	if err := validateTocNodes(nodes); err != nil {
		h.Error400(w, err.Error())
		return
	}

	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiToc{})

	toc, found, err := findToc(ctx, kbID)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	toc.KbID = kbID
	toc.Nodes = nodes
	if found {
		err = orm.Update(ctx, toc)
	} else {
		err = orm.Create(ctx, toc)
	}
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteUpdatedOKJSON(w, toc.ID)
}

func validateTocNodes(nodes []core.WikiTocNode) error {
	ids := map[string]bool{}
	for i := range nodes {
		n := &nodes[i]
		if n.ID == "" || n.Title == "" {
			return fmt.Errorf("toc node %d: id and title are required", i)
		}
		if ids[n.ID] {
			return fmt.Errorf("toc node id duplicated: %s", n.ID)
		}
		ids[n.ID] = true
		switch n.Type {
		case "folder":
		case "article":
			if n.ArticleID == "" {
				return fmt.Errorf("article node %s: article_id is required", n.ID)
			}
		default:
			return fmt.Errorf("toc node %s: invalid type %q", n.ID, n.Type)
		}
		if err := validateTocNodes(n.Children); err != nil {
			return err
		}
	}
	return nil
}

// tocMutate loads a KB's toc, applies the mutation and persists the result.
// Creating the WikiToc record lazily keeps new KBs clean of empty shells.
func tocMutate(kbID string, mutate func(nodes []core.WikiTocNode) ([]core.WikiTocNode, bool)) error {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	// back-to-back mutations (create → rename → delete) must read their own
	// writes, otherwise near-real-time search can resurrect stale nodes
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.WikiToc{})

	toc, found, err := findToc(ctx, kbID)
	if err != nil {
		return err
	}
	nodes, changed := mutate(toc.Nodes)
	if !changed {
		return nil
	}
	if !found {
		toc = &core.WikiToc{KbID: kbID}
	}
	toc.Nodes = nodes
	if found {
		return orm.Update(ctx, toc)
	}
	return orm.Create(ctx, toc)
}

// addArticleToToc appends an article node at the tree root unless it is
// already placed somewhere (the user may have moved it).
func addArticleToToc(kbID, articleID, title string) error {
	if kbID == "" || articleID == "" {
		return nil
	}
	return tocMutate(kbID, func(nodes []core.WikiTocNode) ([]core.WikiTocNode, bool) {
		if nodeForArticle(nodes, articleID) != nil {
			return nodes, false
		}
		return append(nodes, core.WikiTocNode{ID: "toc-" + articleID, Title: title, Type: "article", ArticleID: articleID}), true
	})
}

// syncArticleTitleInToc keeps the tree label aligned with the article.
func syncArticleTitleInToc(kbID, articleID, title string) error {
	if kbID == "" || articleID == "" || title == "" {
		return nil
	}
	return tocMutate(kbID, func(nodes []core.WikiTocNode) ([]core.WikiTocNode, bool) {
		changed := false
		var rename func(list []core.WikiTocNode) []core.WikiTocNode
		rename = func(list []core.WikiTocNode) []core.WikiTocNode {
			for i := range list {
				if list[i].ArticleID == articleID && list[i].Title != title {
					list[i].Title = title
					changed = true
				}
				if len(list[i].Children) > 0 {
					list[i].Children = rename(list[i].Children)
				}
			}
			return list
		}
		next := rename(nodes)
		return next, changed
	})
}

// removeArticleFromToc drops the article's node so deletes leave no
// dangling tree entry.
func removeArticleFromToc(kbID, articleID string) error {
	if kbID == "" || articleID == "" {
		return nil
	}
	return tocMutate(kbID, func(nodes []core.WikiTocNode) ([]core.WikiTocNode, bool) {
		removed := false
		var strip func(list []core.WikiTocNode) []core.WikiTocNode
		strip = func(list []core.WikiTocNode) []core.WikiTocNode {
			out := make([]core.WikiTocNode, 0, len(list))
			for _, n := range list {
				if n.ArticleID == articleID {
					removed = true
					continue
				}
				if len(n.Children) > 0 {
					n.Children = strip(n.Children)
				}
				out = append(out, n)
			}
			return out
		}
		next := strip(nodes)
		return next, removed
	})
}

func nodeForArticle(nodes []core.WikiTocNode, articleID string) *core.WikiTocNode {
	for i := range nodes {
		if nodes[i].ArticleID == articleID {
			return &nodes[i]
		}
		if len(nodes[i].Children) > 0 {
			if found := nodeForArticle(nodes[i].Children, articleID); found != nil {
				return found
			}
		}
	}
	return nil
}

func findToc(ctx *orm.Context, kbID string) (*core.WikiToc, bool, error) {
	builder := orm.NewQuery().
		Size(1).
		Filter(orm.TermQuery("kb_id", kbID))
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return nil, false, err
	}
	tocs, _, err := elastic.DecodeHits[core.WikiToc](res)
	if err != nil {
		return nil, false, err
	}
	if len(tocs) == 0 {
		return &core.WikiToc{}, false, nil
	}
	return &tocs[0], true, nil
}

/* ---------------- versions ---------------- */

func (h *APIHandler) articleVersions(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	articleID := ps.ByName("id")

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.WikiVersion{})

	builder := orm.NewQuery().
		Size(100).
		Filter(orm.TermQuery("article_id", articleID)).
		SortBy(orm.Sort{Field: "version", SortType: orm.DESC})
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	versions, _, err := elastic.DecodeHits[core.WikiVersion](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, util.MapStr{
		"data":  versions,
		"total": len(versions),
	})
}

func (h *APIHandler) articleVersionDetail(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	articleID := ps.ByName("id")
	versionNo, err := strconv.Atoi(ps.ByName("version"))
	if err != nil {
		h.Error400(w, "version must be a number")
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.WikiVersion{})

	builder := orm.NewQuery().
		Size(1).
		Filter(orm.TermQuery("article_id", articleID), orm.TermQuery("version", versionNo))
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	versions, _, err := elastic.DecodeHits[core.WikiVersion](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(versions) == 0 {
		h.WriteOpRecordNotFoundJSON(w, fmt.Sprintf("%s/v%d", articleID, versionNo))
		return
	}
	h.WriteGetOKJSON(w, versions[0].ID, versions[0])
}

/* ---------------- status machine ---------------- */

func (h *APIHandler) updateArticleStatus(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	articleID := ps.ByName("id")

	var body struct {
		Status string `json:"status"`
	}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.Error400(w, err.Error())
		return
	}
	if err := validateArticleStatus(body.Status); err != nil {
		h.Error400(w, err.Error())
		return
	}

	// permission is enforced at the route; direct write skips the ORM-level
	// owner check so shared-KB editors can transition status too
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})

	var article core.WikiArticle
	article.SetID(articleID)
	exists, err := orm.GetV2(ctx, &article)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, articleID)
		return
	}

	if article.Status == body.Status {
		h.WriteUpdatedOKJSON(w, article.ID)
		return
	}
	allowed := statusTransitions[article.Status]
	if !util.AnyInArrayEquals(allowed, body.Status) {
		h.Error400(w, fmt.Sprintf("cannot transition status from %s to %s", article.Status, body.Status))
		return
	}

	prev := article.Status
	article.Status = body.Status
	if err := orm.Update(ctx, &article); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// entity pages mirror their workflow onto the ontology (B4)
	syncEntityStatus(&article, body.Status)

	// audit trail: record the workflow transition alongside the content
	_ = writeVersionSnapshot(&article, nextVersionNumber(article.ID), core.WikiChangeHumanEdited,
		fmt.Sprintf("status: %s -> %s", prev, body.Status))

	h.WriteUpdatedOKJSON(w, article.ID)
}

/* ---------------- ontology: lookup & neighbors ---------------- */

// entityLookup resolves a name (or alias) to an entity — the MCP-facing
// exact-match counterpart of entity search (design doc §4.3).
func (h *APIHandler) entityLookup(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	name := strings.TrimSpace(req.URL.Query().Get("name"))
	entityType := strings.TrimSpace(req.URL.Query().Get("type"))
	if name == "" {
		h.Error400(w, "name is required")
		return
	}

	ctx := orm.NewContextWithParent(req.Context())

	if entity := FindEntityByNameOrAlias(ctx, name, entityType); entity != nil {
		h.WriteGetOKJSON(w, entity.ID, entity)
		return
	}
	h.WriteGetMissingJSON(w, name)
}

// entityBacklinks lists the articles whose wikilinks resolved to this
// entity (ontology O4: the entity's "who references me" face). linked_pages
// is stored unmapped (enabled:false), so the scan runs in memory over a
// recency-bounded window — the same trade-off kbGraph makes.
func (h *APIHandler) entityBacklinks(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.WikiArticle{})

	builder := orm.NewQuery().Size(graphMaxArticles).
		SortBy(orm.Sort{Field: "updated", SortType: orm.DESC})
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	articles, _, err := elastic.DecodeHits[core.WikiArticle](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := []util.MapStr{}
	for _, article := range articles {
		for _, lp := range article.LinkedPages {
			if lp.EntityID != id {
				continue
			}
			items = append(items, util.MapStr{
				"article_id": article.ID,
				"title":      article.Title,
				"kb_id":      article.KbID,
				"status":     article.Status,
				"updated":    article.Updated,
				"link_type":  lp.Type,
				"link_name":  lp.Name,
				"snippet":    backlinkSnippet(article.Content, lp.Name),
			})
			break // one row per article, even when it links twice
		}
	}

	h.WriteOKJSON(w, util.MapStr{"items": items, "total": len(items)})
}

// backlinkSnippet extracts a short window around the first wikilink
// occurrence of the linked name, so the backlink row shows context.
func backlinkSnippet(content string, name string) string {
	idx := strings.Index(content, name)
	if idx < 0 {
		if len(content) > 120 {
			return content[:120] + "…"
		}
		return content
	}
	start := idx - 40
	if start < 0 {
		start = 0
	}
	end := idx + len(name) + 60
	if end > len(content) {
		end = len(content)
	}
	snippet := content[start:end]
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(content) {
		snippet += "…"
	}
	return snippet
}

// entityNeighbors walks one hop over the entity's relations and expands the
// target entities (design doc B5).
func (h *APIHandler) entityNeighbors(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.WikiEntity{})

	var entity core.WikiEntity
	entity.SetID(id)
	exists, err := orm.GetV2(ctx, &entity)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	targetIDs := make([]string, 0, len(entity.Relations))
	for _, r := range entity.Relations {
		if r.TargetID != "" && r.TargetID != entity.ID {
			targetIDs = append(targetIDs, r.TargetID)
		}
	}

	neighbors := []core.WikiEntity{}
	if len(targetIDs) > 0 {
		builder := orm.NewQuery().
			Size(len(targetIDs)).
			Filter(orm.TermsQuery("id", targetIDs))
		res, err := orm.SearchV2(ctx, builder)
		if err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		neighbors, _, err = elastic.DecodeHits[core.WikiEntity](res)
		if err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// schema-resolved labels for both directions (O3): forward relations
	// carry the declared label, incoming ones the declared inverse
	schema := loadOntologySchemaForKB(req.Context(), "")
	typeDefs := map[string]*OntologyEntityTypeDef{}
	if schema != nil {
		for i := range schema.EntityTypes {
			typeDefs[schema.EntityTypes[i].Name] = &schema.EntityTypes[i]
		}
	}
	neighborByID := map[string]core.WikiEntity{}
	for _, n := range neighbors {
		neighborByID[n.ID] = n
	}
	relationsDetailed := make([]util.MapStr, 0, len(entity.Relations))
	for _, r := range entity.Relations {
		if r.TargetID == "" || r.TargetID == entity.ID || neighborByID[r.TargetID].ID == "" {
			continue
		}
		edge := GraphEdge{Label: r.Relation}
		decorateRelationEdge(&edge, typeDefs, entity.Type, r.Relation)
		entry := util.MapStr{
			"target_id":   r.TargetID,
			"target_name": neighborByID[r.TargetID].Name,
			"target_type": neighborByID[r.TargetID].Type,
			"relation":    r.Relation,
			"label":       edge.Label,
		}
		if edge.Inverse != "" {
			entry["inverse"] = edge.Inverse
			entry["inverse_label"] = edge.InverseLabel
		}
		relationsDetailed = append(relationsDetailed, entry)
	}

	h.WriteOKJSON(w, util.MapStr{
		"entity":             entity,
		"relations":          entity.Relations,
		"relations_detailed": relationsDetailed,
		"incoming":           entityInverseEdges(req.Context(), &entity),
		"neighbors":          neighbors,
	})
}

/* ---------------- version helpers ---------------- */

func changeTypeFor(article *core.WikiArticle) string {
	if article.AIGenerated {
		return core.WikiChangeAIGenerated
	}
	return core.WikiChangeHumanEdited
}

func writeVersionSnapshot(article *core.WikiArticle, version int, changeType, changeSummary string) error {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true) // internal snapshot; article owner recorded below
	orm.WithModel(ctx, &core.WikiVersion{})

	v := &core.WikiVersion{
		ArticleID:     article.ID,
		Version:       version,
		ChangeType:    changeType,
		ChangeSummary: changeSummary,
		Content:       article.Content,
		CreatedBy:     article.GetOwnerID(),
	}
	return orm.Create(ctx, v)
}

// writeVersionIfChanged snapshots only when the content differs from the
// latest version, so metadata-only updates (title, tags, or a field stripped
// by ProtectedFields) don't pollute the history.
func writeVersionIfChanged(article *core.WikiArticle) error {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiVersion{})

	builder := orm.NewQuery().
		Size(1).
		Filter(orm.TermQuery("article_id", article.ID)).
		SortBy(orm.Sort{Field: "version", SortType: orm.DESC})
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return err
	}
	versions, _, err := elastic.DecodeHits[core.WikiVersion](res)
	if err != nil {
		return err
	}
	if len(versions) > 0 && versions[0].Content == article.Content {
		return nil
	}
	return writeVersionSnapshot(article, nextVersionNumber(article.ID), core.WikiChangeHumanEdited, "")
}

func nextVersionNumber(articleID string) int {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiVersion{})

	builder := orm.NewQuery().
		Size(1).
		Filter(orm.TermQuery("article_id", articleID)).
		SortBy(orm.Sort{Field: "version", SortType: orm.DESC}).
		Include("version")
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return 1
	}
	versions, _, err := elastic.DecodeHits[core.WikiVersion](res)
	if err != nil || len(versions) == 0 {
		return 1
	}
	return versions[0].Version + 1
}

/* ---------------- misc helpers ---------------- */

// findIDs returns the ids of objects matching a single-term filter.
func findIDs(ctx *orm.Context, model interface{}, clause *orm.Clause) ([]string, error) {
	orm.WithModel(ctx, model)
	builder := orm.NewQuery().Size(10000).Filter(clause).Include("id")
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return nil, err
	}
	hits, _, err := elastic.DecodeHits[util.MapStr](res)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(hits))
	for _, hit := range hits {
		if id, ok := hit["id"].(string); ok && id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func deleteByID(ctx *orm.Context, model interface{}, id string) error {
	obj, ok := model.(orm.Object)
	if !ok {
		return fmt.Errorf("model does not implement orm.Object")
	}
	orm.WithModel(ctx, model)
	obj.SetID(id)
	return orm.Delete(ctx, obj)
}
