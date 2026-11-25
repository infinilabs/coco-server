/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"fmt"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
)

// Dynamic entity scope (plan D): a KB's entity set is computed at query time
// instead of being pinned by a kb_id stamped at write time. An entity belongs
// to a KB when either
//
//   - it carries kb_id == kbID (explicit: promote / legacy / manual writes), or
//   - its sources reference a document whose datasource is bound to the KB.
//
// Binding or unbinding a datasource therefore needs zero data migration: the
// kb-less entities of a pre-bind sync show up in the KB graph the moment the
// binding exists, and disappear from it the moment it is removed — the asset
// itself stays untouched and globally searchable either way.
//
// Both directions of the resolution live behind a TTL cache with the same
// self-healing contract as the wiki module's datasource→KB binding cache;
// KB rebinding/delete invalidates eagerly via InvalidateKBDocScope. The
// shared lookup lives here (not in modules/wiki) because two consumers need
// it — the wiki graph scoping and the document search module's entity-hit
// KB grouping — and document cannot import wiki (crud-registration cycle).

const kbDocScopeTTL = 5 * time.Minute

const kbDocScopeMaxDocs = 10000

type kbDocScopeEntry struct {
	docIDs []string
	cached time.Time
}

var (
	kbDocScopeMu    sync.RWMutex
	kbDocScopeCache = map[string]kbDocScopeEntry{}
)

// KBDocIDsFor returns the ids of the documents visible to the KB — every
// document whose source datasource is bound to it. Nil when the KB does not
// exist or the lookup fails; empty (non-nil is not guaranteed) when the KB
// binds no datasource. Callers treat empty as "kb_id legacy rows only".
func KBDocIDsFor(parent *orm.Context, kbID string) []string {
	if kbID == "" {
		return nil
	}
	kbDocScopeMu.RLock()
	if e, ok := kbDocScopeCache[kbID]; ok && time.Since(e.cached) < kbDocScopeTTL {
		kbDocScopeMu.RUnlock()
		return e.docIDs
	}
	kbDocScopeMu.RUnlock()

	ctx := orm.NewContextWithParent(parent)
	ctx.DirectReadAccess()

	orm.WithModel(ctx, &WikiKnowledgeBase{})
	var kb WikiKnowledgeBase
	kb.SetID(kbID)
	exists, err := orm.GetV2(ctx, &kb)
	if err != nil || !exists {
		log.Warnf("wiki: doc scope for kb [%s] skipped: kb not found: %v", kbID, err)
		return nil
	}
	if len(kb.DatasourceIDs) == 0 {
		kbDocScopeMu.Lock()
		kbDocScopeCache[kbID] = kbDocScopeEntry{docIDs: []string{}, cached: time.Now()}
		kbDocScopeMu.Unlock()
		return []string{}
	}

	orm.WithModel(ctx, &Document{})
	builder := orm.NewQuery().From(0).Size(kbDocScopeMaxDocs)
	builder.Filter(orm.TermsQuery("source.id", kb.DatasourceIDs))
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		log.Warnf("wiki: doc scope query for kb [%s] failed: %v", kbID, err)
		return nil
	}
	docs, _, derr := elastic.DecodeHits[Document](res)
	if derr != nil {
		log.Warnf("wiki: doc scope decode for kb [%s] failed: %v", kbID, derr)
		return nil
	}
	docIDs := make([]string, 0, len(docs))
	for i := range docs {
		docIDs = append(docIDs, docs[i].ID)
	}

	kbDocScopeMu.Lock()
	kbDocScopeCache[kbID] = kbDocScopeEntry{docIDs: docIDs, cached: time.Now()}
	kbDocScopeMu.Unlock()
	log.Infof("wiki: doc scope for kb [%s]: %d docs over %d datasource(s)", kbID, len(docIDs), len(kb.DatasourceIDs))
	return docIDs
}

// InvalidateKBDocScope drops the cached document scope of a KB — called from
// the KB update/delete hooks next to the binding-cache invalidation.
func InvalidateKBDocScope(kbID string) {
	kbDocScopeMu.Lock()
	defer kbDocScopeMu.Unlock()
	delete(kbDocScopeCache, kbID)
}

// KBsForDocs returns the KB ids whose document scope contains any of docIDs —
// the document→KB reverse direction the search-side groupers need: entity
// hits carry source document ids, and the search page groups them under the
// KBs that can see those documents. Built by scanning every KB's document
// scope once per TTL window — KB counts are small (tens), so the scan is a
// handful of cached terms queries. On failure returns nil: callers fall back
// to the entity's own kb_id, so grouping degrades to legacy behavior rather
// than breaking.
func KBsForDocs(parent *orm.Context, docIDs []string) []string {
	if len(docIDs) == 0 {
		return nil
	}
	kbDocReverseMu.RLock()
	entry := kbDocReverse
	fresh := entry != nil && time.Since(entry.cached) < kbDocScopeTTL
	kbDocReverseMu.RUnlock()
	if !fresh {
		// recover-wrapped: the reverse index is a best-effort grouping aid —
		// a missing ES handler (unit tests) or engine hiccup must degrade to
		// legacy kb_id grouping, never panic the search path
		byDoc, err := func() (m map[string][]string, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("kb doc reverse build recovered: %v", r)
				}
			}()
			return buildKBDocReverse(parent)
		}()
		if err != nil {
			log.Warnf("wiki: kb doc reverse index build failed: %v", err)
			return nil
		}
		kbDocReverseMu.Lock()
		kbDocReverse = &kbDocReverseEntry{byDoc: byDoc, cached: time.Now()}
		kbDocReverseMu.Unlock()
		entry = kbDocReverse
	}

	seen := map[string]bool{}
	out := []string{}
	for _, docID := range docIDs {
		for _, kbID := range entry.byDoc[docID] {
			if !seen[kbID] {
				seen[kbID] = true
				out = append(out, kbID)
			}
		}
	}
	return out
}

type kbDocReverseEntry struct {
	byDoc  map[string][]string
	cached time.Time
}

var (
	kbDocReverseMu sync.RWMutex
	kbDocReverse   *kbDocReverseEntry
)

func buildKBDocReverse(parent *orm.Context) (map[string][]string, error) {
	ctx := orm.NewContextWithParent(parent)
	ctx.DirectReadAccess()
	orm.WithModel(ctx, &WikiKnowledgeBase{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().From(0).Size(100))
	if err != nil || res == nil {
		return nil, err
	}
	kbs, _, derr := elastic.DecodeHits[WikiKnowledgeBase](res)
	if derr != nil {
		return nil, derr
	}
	byDoc := map[string][]string{}
	for i := range kbs {
		for _, docID := range KBDocIDsFor(ctx, kbs[i].ID) {
			byDoc[docID] = append(byDoc[docID], kbs[i].ID)
		}
	}
	return byDoc, nil
}
