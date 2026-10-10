/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"sort"
	"strings"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
)

// LinkEntityMentions wraps known entity names occurring in content as
// `[[type:name]]` wikilinks, so generated pages become navigable: the save
// path resolves them into LinkedPages (capsules in the article, edges in
// the graph, mentions on the entity). Longest name wins so "Maxim's Cakes
// HK" is not cut into "Maxim's Cakes"; `exclude` (a page's own subject) is
// never self-linked. Callers pass freshly generated content without
// pre-existing wikilinks.
func LinkEntityMentions(ctx context.Context, content, exclude string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}
	readCtx := orm.NewContextWithParent(ctx)
	readCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(readCtx, &core.WikiEntity{})

	res, err := orm.SearchV2(readCtx, orm.NewQuery().Size(1000))
	if err != nil || res == nil {
		return content
	}
	entities, _, derr := elastic.DecodeHits[core.WikiEntity](res)
	if derr != nil || len(entities) == 0 {
		return content
	}

	type mention struct{ full, name, typ string }
	links := make([]mention, 0, len(entities))
	for i := range entities {
		e := &entities[i]
		name := strings.TrimSpace(e.Name)
		if len(name) < 4 || name == exclude {
			continue
		}
		links = append(links, mention{full: name, name: name, typ: e.Type})
		for _, alias := range e.Aliases {
			alias = strings.TrimSpace(alias)
			if len(alias) >= 5 {
				links = append(links, mention{full: alias, name: name, typ: e.Type})
			}
		}
	}
	if len(links) == 0 {
		return content
	}
	// longest name first — specific names before their substrings
	sort.Slice(links, func(i, j int) bool { return len(links[i].full) > len(links[j].full) })

	var sb strings.Builder
	rest := content
	for {
		bestIdx, best := -1, mention{}
		for _, l := range links {
			idx := strings.Index(rest, l.full)
			if idx >= 0 && (bestIdx == -1 || idx < bestIdx || (idx == bestIdx && len(l.full) > len(best.full))) {
				bestIdx, best = idx, l
			}
		}
		if bestIdx == -1 {
			sb.WriteString(rest)
			break
		}
		sb.WriteString(rest[:bestIdx])
		if best.typ != "" {
			sb.WriteString("[[" + best.typ + ":" + best.name + "]]")
		} else {
			sb.WriteString("[[" + best.name + "]]")
		}
		rest = rest[bestIdx+len(best.full):]
	}
	return sb.String()
}
