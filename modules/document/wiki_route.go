/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"fmt"
	"net/http"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

// errWikiRouteSkipped is returned when the wiki route cannot run (no
// permission); callers treat it like any failing route — fusion continues
// with the routes that can.
var errWikiRouteSkipped = fmt.Errorf("wiki route skipped: no wiki article search permission")

// appConfigFn is swapped out in tests (AppConfig touches the kv store).
var appConfigFn = common.AppConfig

// wikiSearchPermission reuses the wiki module's article-search permission
// key (coco#wiki/article/search); the literals are duplicated here because
// importing the wiki package from the document module would create an
// import cycle through the crud registrations.
func wikiSearchPermission(req *http.Request) bool {
	per := security.GetSimplePermission("coco", "wiki_article", string(security.Search))
	perID := security.GetOrInitPermissionKey(per)
	reqUser, err := security.GetUserFromRequest(req)
	if err != nil || reqUser == nil {
		return false
	}
	if reqUser.Roles != nil && util.AnyInArrayEquals(reqUser.Roles, security.RoleAdmin) {
		return true
	}
	return reqUser.UserAssignedPermission.ValidateFor(perID)
}

// passesWikiNoiseGate is the curated-layer noise gate: only published pages
// enter retrieval, and concept pages need the configured minimum of
// independent sources — a single-source concept is a hypothesis, not
// knowledge (the same rule the governance queue enforces on the write
// side). conceptMinSources comes from the compile rules (D6).
func passesWikiNoiseGate(article *core.WikiArticle, conceptMinSources int) bool {
	if article.Status != core.WikiArticlePublished {
		return false
	}
	if article.Confidence == "low" {
		return false
	}
	if article.PageType == core.WikiPageTypeConcept && len(article.Sources) < conceptMinSources {
		return false
	}
	return true
}

// wikiArticleToHit wraps a curated wiki page as a search hit: the wiki layer
// is the "read the compiled knowledge first" tier, and its provenance (the
// source documents with locators) travels along in metadata so downstream
// consumers can cite page/section/clause.
func wikiArticleToHit(article *core.WikiArticle, score float32) elastic.DocumentWithMeta[core.Document] {
	doc := core.Document{
		Title:   article.Title,
		Summary: article.Summary,
		URL:     fmt.Sprintf("%s/#/wiki/article/%s", appConfigFn().ServerInfo.Endpoint, article.ID),
		Source: core.DataSourceReference{
			ID:   "wiki",
			Name: "Wiki",
			Icon: "font_book",
		},
	}
	sources := make([]interface{}, 0, len(article.Sources))
	for _, s := range article.Sources {
		sources = append(sources, util.MapStr{
			"doc_id":  s.DocID,
			"title":   s.Title,
			"url":     s.URL,
			"excerpt": s.Excerpt,
			"locator": s.Locator,
		})
	}
	doc.Metadata = util.MapStr{
		"wiki_article": true,
		"kb_id":        article.KbID,
		"page_type":    article.PageType,
		"wiki_sources": sources,
	}
	return elastic.DocumentWithMeta[core.Document]{
		ID:     article.ID,
		Index:  "wiki_article",
		Source: doc,
		Score:  score,
	}
}

// wikiRoute runs the wiki recall leg: BM25 over the curated pages the user
// is allowed to search. Hits become pseudo-documents (wikiArticleToHit) so
// the fusion math treats them like any other route.
func (h *APIHandler) wikiRoute(req *http.Request, query string, window int) (*elastic.SearchResponseWithMeta[core.Document], error) {
	if !wikiSearchPermission(req) {
		return nil, errWikiRouteSkipped
	}

	octx := orm.NewContextWithParent(req.Context())
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.WikiArticle{})

	builder := orm.NewQuery().From(0).Size(window)
	builder.Query(query)
	builder.DefaultQueryField("title^20", "title.pinyin^8", "summary^4", "tags^2", "combined_fulltext")
	builder.Filter(
		orm.TermQuery("status", core.WikiArticlePublished),
		orm.MustNotQuery(orm.TermQuery("confidence", "low")),
	)
	builder.Exclude("content", "linked_pages")

	var articles []core.WikiArticle
	err, _ := elastic.SearchV2WithResultItemMapper(octx, &articles, builder, nil)
	if err != nil {
		return nil, err
	}

	out := &elastic.SearchResponseWithMeta[core.Document]{}
	rules := loadCompileRules(req.Context())
	for i := range articles {
		if !passesWikiNoiseGate(&articles[i], rules.ConceptMinSources) {
			continue
		}
		out.Hits.Hits = append(out.Hits.Hits, wikiArticleToHit(&articles[i], float32(window-i)))
	}
	// a plain map, not util.MapStr: GetTotal's type switch matches
	// map[string]interface{} only, a named type would read as zero and the
	// route's count would vanish from the fused total
	out.Hits.Total = map[string]interface{}{"value": int64(len(out.Hits.Hits)), "relation": "eq"}
	return out, nil
}
