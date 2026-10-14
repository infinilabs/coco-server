/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"context"
	"sync"
	"time"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Compile rules for the retrieval noise gate (D6): the concept-page source
// floor is configured in the ontology schema (Settings → Ontology), not
// hardcoded here. The rules live in core so the wiki editor and this gate
// read the same document; this loader reads the tenant-scope schema with a
// short cache — the gate runs per article inside a search, it must not
// query the store per hit.

const compileRulesCacheTTL = 60 * time.Second

var (
	compileRulesMutex sync.RWMutex
	compileRulesCache *core.WikiCompileRules
	compileRulesAt    time.Time
)

// loadCompileRules returns the tenant-scope compile rules (defaults when
// nothing is on file).
func loadCompileRules(ctx context.Context) *core.WikiCompileRules {
	compileRulesMutex.RLock()
	if compileRulesCache != nil && time.Since(compileRulesAt) < compileRulesCacheTTL {
		rules := *compileRulesCache
		compileRulesMutex.RUnlock()
		return &rules
	}
	compileRulesMutex.RUnlock()

	rules := readCompileRules(ctx)

	compileRulesMutex.Lock()
	compileRulesCache = rules
	compileRulesAt = time.Now()
	compileRulesMutex.Unlock()
	return rules
}

// resetCompileRulesCache drops the cache (tests).
func resetCompileRulesCache() {
	compileRulesMutex.Lock()
	compileRulesCache = nil
	compileRulesAt = time.Time{}
	compileRulesMutex.Unlock()
}

func readCompileRules(ctx context.Context) *core.WikiCompileRules {
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiOntologySchema{})

	res, err := orm.SearchV2(octx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("scope", "tenant")))
	if err != nil {
		return (&core.WikiCompileRules{}).Normalized()
	}
	docs, _, err := elastic.DecodeHits[core.WikiOntologySchema](res)
	if err != nil || len(docs) == 0 {
		return (&core.WikiCompileRules{}).Normalized()
	}

	schema := docs[0].Schema
	raw, _ := schema["rules"]
	if raw == nil {
		return (&core.WikiCompileRules{}).Normalized()
	}
	bytes, err := util.ToJSONBytes(raw)
	if err != nil {
		return (&core.WikiCompileRules{}).Normalized()
	}
	rules := &core.WikiCompileRules{}
	if err := util.FromJSONBytes(bytes, rules); err != nil {
		return (&core.WikiCompileRules{}).Normalized()
	}
	return rules.Normalized()
}
