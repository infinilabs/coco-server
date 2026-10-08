/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Entity edit propagation (W10): editing an entity (rename, aliases, type,
// relations) invalidates the articles that reference it — its own article
// and every article whose LinkedPages point at it. Those articles get an
// article_refresh proposal in the governance queue; merging the edit into
// the article body stays a human action, like every proposal here.
//
// Mechanics: the crud PrepareUpdate hook records what changed (the store
// still holds the old entity there), the PostUpdate hook — which only runs
// after a successful write — files the proposals. A failed save leaves a
// stale pending entry behind; entries are TTL-scavenged on next use.

// propagationArticleCap bounds how many articles one entity edit can ask
// to refresh — a hub entity linked from everywhere must not flood the queue.
const propagationArticleCap = 20

// pendingEntityEdits carries entity id → recorded change facets between the
// prepare and post hooks of one update request.
var pendingEntityEdits sync.Map

type entityEditFacets struct {
	EntityID string
	OldName  string
	NewName  string
	Changed  []string
	At       time.Time
}

// recordEntityEditForPropagation loads the pre-update entity and records
// which propagation-relevant facets the incoming edit changes. Called from
// the crud PrepareUpdate hook (before the store is overwritten).
func recordEntityEditForPropagation(obj *core.WikiEntity) {
	if obj == nil || obj.ID == "" {
		return
	}
	octx := orm.NewContext()
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiEntity{})

	old := core.WikiEntity{}
	old.ID = obj.ID
	exists, err := orm.GetV2(octx, &old)
	if err != nil || !exists {
		return // new entity or unreadable: nothing to propagate
	}

	var changed []string
	if old.Name != obj.Name {
		changed = append(changed, "renamed")
	}
	if !reflect.DeepEqual(old.Aliases, obj.Aliases) {
		changed = append(changed, "aliases")
	}
	if old.Type != obj.Type {
		changed = append(changed, "type")
	}
	if old.Status != obj.Status {
		changed = append(changed, "status")
	}
	if !reflect.DeepEqual(old.Relations, obj.Relations) {
		changed = append(changed, "relations")
	}

	if len(changed) == 0 {
		pendingEntityEdits.Delete(obj.ID)
		return
	}
	pendingEntityEdits.Store(obj.ID, entityEditFacets{
		EntityID: obj.ID,
		OldName:  old.Name,
		NewName:  obj.Name,
		Changed:  changed,
		At:       time.Now(),
	})
}

// propagateEntityEditAfterSave files article_refresh proposals for every
// article affected by the entity edit recorded in the prepare hook. Called
// from the crud PostUpdate hook — proposals only exist for saves that
// actually happened.
func propagateEntityEditAfterSave(obj *core.WikiEntity) {
	if obj == nil {
		return
	}
	v, ok := pendingEntityEdits.LoadAndDelete(obj.ID)
	if !ok {
		return
	}
	facets := v.(entityEditFacets)

	affected := affectedArticlesForEntity(obj)
	if len(affected) == 0 {
		return
	}
	if len(affected) > propagationArticleCap {
		log.Warnf("wiki: entity [%s] edit affects %d articles, capping refresh proposals at %d", obj.Name, len(affected), propagationArticleCap)
		affected = affected[:propagationArticleCap]
	}

	reason := fmt.Sprintf("entity updated: %s (%v)", facets.NewName, facets.Changed)
	evidence := util.MapStr{
		"entity_id":   facets.EntityID,
		"entity_name": facets.NewName,
		"old_name":    facets.OldName,
		"changed":     facets.Changed,
		"cascade":     "entity-edit",
	}

	kbCache := map[string]*core.WikiKnowledgeBase{}
	for _, article := range affected {
		kb := kbCache[article.KbID]
		if kb == nil {
			kbObj := core.WikiKnowledgeBase{}
			kbObj.ID = article.KbID
			kctx := orm.NewContext()
			kctx.Set(orm.DirectReadWithoutPermissionCheck, true)
			orm.WithModel(kctx, &core.WikiKnowledgeBase{})
			exists, err := orm.GetV2(kctx, &kbObj)
			if err != nil || !exists {
				continue
			}
			kb = &kbObj
			kbCache[article.KbID] = kb
		}
		// open-twin folding is fileProposal's own idempotency: two entity
		// edits hitting the same article collapse into one open proposal
		fileProposal(kb, &article, core.WikiGovernanceArticleRefresh, reason, evidence)
	}
}

// affectedArticlesForEntity returns the entity's own article plus every
// article whose LinkedPages reference it (the same client-side scan the
// backlinks endpoint runs — LinkedPages is not an indexed field).
func affectedArticlesForEntity(entity *core.WikiEntity) []core.WikiArticle {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})

	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(graphMaxArticles).
		Filter(orm.MustNotQuery(orm.TermQuery("status", "archived"))))
	if err != nil {
		return nil
	}
	articles, _, err := elastic.DecodeHits[core.WikiArticle](res)
	if err != nil {
		return nil
	}

	out := make([]core.WikiArticle, 0, 4)
	for i := range articles {
		a := &articles[i]
		if entity.ArticleID != "" && a.ID == entity.ArticleID {
			out = append(out, *a)
			continue
		}
		for _, lp := range a.LinkedPages {
			if lp.EntityID == entity.ID {
				out = append(out, *a)
				break
			}
		}
	}
	return out
}
