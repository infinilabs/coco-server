/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
)

// docCiteMarker matches a whole `[Doc: <title>]` citation marker — both the
// plain form and the wikilink-wrapped name (`[Doc: [[product:X]]]`). Names
// inside are provenance, not prose.
var docCiteMarker = regexp.MustCompile(`\[Doc:\s*(?:\[\[[^\[\]]+\]\]|[^\]]+)\]`)

// mention is one linkable entity name occurrence
type mention struct{ full, name, typ string }

// linkMentionSpans wraps known entity mentions inside one prose segment
// (the marker-free span-splitting lives in LinkEntityMentions).
func linkMentionSpans(content string, links []mention) string {
	if len(links) == 0 || strings.TrimSpace(content) == "" {
		return content
	}
	var sb strings.Builder
	rest := content
	for {
		bestIdx, best := -1, mention{}
		for _, l := range links {
			// first occurrence of this name that is not glued into a larger
			// ASCII word — "inventory" inside "inventoried" is a different
			// word, not a mention (CJK neighbours stay legal)
			from := 0
			for {
				idx := strings.Index(rest[from:], l.full)
				if idx < 0 {
					break
				}
				at := from + idx
				if !asciiWordFlanked(rest, at, at+len(l.full)) {
					if bestIdx == -1 || at < bestIdx || (at == bestIdx && len(l.full) > len(best.full)) {
						bestIdx, best = at, l
					}
					break
				}
				from = at + 1
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

	// citation markers `[Doc: <title>]` are provenance, not prose: entity
	// names inside them must NOT be capsule-wrapped, or the marker mangles
	// into `[Doc: [[product:Neo4j]]]` and the reading surface's citation
	// matching breaks (the hover chip loses its source link)
	return linkWithCitationGuard(content, links)
}

// linkWithCitationGuard splits the content around `[Doc: ...]` citation
// markers and entity-wraps only the prose segments between them.
func linkWithCitationGuard(content string, links []mention) string {
	if len(links) == 0 {
		return content
	}
	if !docCiteMarker.MatchString(content) {
		return linkMentionSpans(content, links)
	}
	var sb strings.Builder
	prev := 0
	for _, m := range docCiteMarker.FindAllStringIndex(content, -1) {
		sb.WriteString(linkMentionSpans(content[prev:m[0]], links))
		sb.WriteString(content[m[0]:m[1]])
		prev = m[1]
	}
	sb.WriteString(linkMentionSpans(content[prev:], links))
	return sb.String()
}

// asciiWordFlanked reports whether the match s[start:end] is glued into a
// larger ASCII word on either side (letter/digit on both the inner edge of
// the match and the neighbour) — such spans are parts of words, not
// mentions. CJK boundaries never block: 美心 inside 美心月餅 still links.
func asciiWordFlanked(s string, start, end int) bool {
	if start > 0 && isASCIILetter(s[start-1]) && isASCIILetter(s[start]) {
		return true
	}
	if end < len(s) && isASCIILetter(s[end]) && isASCIILetter(s[end-1]) {
		return true
	}
	return false
}

func isASCIILetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_'
}

// relinkEntityMentions normalizes a rewritten page's entity capsules: any
// existing wikilinks are unwrapped back to their plain names, then known
// entity mentions are re-linked from the entity store. The compile agent's
// whole-page rewrites drop, translate or mangle capsules unpredictably —
// re-linking at apply time is deterministic and matches what the other
// compile-apply paths do (LinkEntityMentions requires content without
// pre-existing wikilinks, hence the unwrap first).
func relinkEntityMentions(ctx context.Context, content, exclude string) string {
	plain := wikilinkPattern.ReplaceAllString(content, "$1")
	return LinkEntityMentions(ctx, plain, exclude)
}
