/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// the citation-marker guard: provenance must survive entity re-linking
func TestDocCiteMarkerForms(t *testing.T) {
	assert.True(t, docCiteMarker.MatchString("[Doc: Neo4j]"))
	assert.True(t, docCiteMarker.MatchString("[Doc: [[product:Neo4j]]]"))
	assert.True(t, docCiteMarker.MatchString("[Doc: How to integrate Coco AI]"))
	assert.False(t, docCiteMarker.MatchString("see [Doc] docs"))
	assert.False(t, docCiteMarker.MatchString("plain prose about Neo4j"))
}

// entity wrapping must never touch `[Doc: ...]` marker spans
func TestLinkWithCitationGuard(t *testing.T) {
	links := []mention{{full: "Neo4j", name: "Neo4j", typ: "product"}}
	out := linkWithCitationGuard("use [Doc: Neo4j] for graph queries with Neo4j", links)
	assert.Equal(t, "use [Doc: Neo4j] for graph queries with [[product:Neo4j]]", out)

	out = linkWithCitationGuard("[Doc: [[product:Neo4j]]]", links)
	assert.Equal(t, "[Doc: [[product:Neo4j]]]", out)

	// marker-free prose still wraps
	out = linkWithCitationGuard("Neo4j and Neo4j", links)
	assert.Equal(t, "[[product:Neo4j]] and [[product:Neo4j]]", out)

	// the real-world corruption: relink must keep the marker intact while
	// wrapping the same name in prose
	out = linkWithCitationGuard("cite [Doc: Neo4j] then discuss Neo4j graphs", links)
	assert.Equal(t, "cite [Doc: Neo4j] then discuss [[product:Neo4j]] graphs", out)
}

// citation name extraction: distinct names, namespace stripped
func TestCitedDocNames(t *testing.T) {
	body := "a [Doc: PostgreSQL] and [Doc: Jira], again [Doc: PostgreSQL] " +
		"wrapped [Doc: [[product:Neo4j]]] end"
	assert.Equal(t, []string{"PostgreSQL", "Jira", "Neo4j"}, citedDocNames(body))
	assert.Empty(t, citedDocNames("no citations here"))
}
