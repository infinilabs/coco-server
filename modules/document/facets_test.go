/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
)

type elasticHit = elastic.DocumentWithMeta[core.Document]

func facetHit(id, sourceName, docType string, tags []string, updated *time.Time) elasticHit {
	hit := elasticHit{ID: id}
	hit.Source.Source.Name = sourceName
	hit.Source.Type = docType
	hit.Source.Tags = tags
	if updated != nil {
		hit.Source.Updated = updated
	}
	return hit
}

func TestTopBucketsSortedAndCapped(t *testing.T) {
	counts := map[string]int{}
	for i := 0; i < aggFacetCap+5; i++ {
		counts[string(rune('a'+i%26))] = i // later = more
	}
	out := topBuckets(counts)
	require.Len(t, out, aggFacetCap)
	for i := 1; i < len(out); i++ {
		assert.GreaterOrEqual(t, out[i-1].Count, out[i].Count)
	}
}

func TestBucketByDatasourceAndType(t *testing.T) {
	hits := []elasticHit{
		facetHit("1", "HR", "", nil, nil),
		facetHit("2", "HR", "", nil, nil), // empty type reads as document
		facetHit("3", "Wiki", "wiki_article", nil, nil),
		facetHit("4", "Wiki", "entity", nil, nil),
	}

	ds := bucketBy(hits, func(hit elasticHit) (string, bool) {
		name := hit.Source.Source.Name
		if name == "" {
			name = hit.Source.Source.ID
		}
		return name, name != ""
	})
	require.Len(t, ds, 2)
	assert.Equal(t, "HR", ds[0].Value)
	assert.Equal(t, 2, ds[0].Count)
	assert.Equal(t, "Wiki", ds[1].Value)

	types := bucketBy(hits, func(hit elasticHit) (string, bool) {
		tp := hit.Source.Type
		if tp == "" {
			tp = "document"
		}
		return tp, true
	})
	require.Len(t, types, 3)
	assert.Equal(t, "document", types[0].Value)
	assert.Equal(t, 2, types[0].Count)
}

func TestBucketByManyTags(t *testing.T) {
	hits := []elasticHit{
		facetHit("1", "HR", "", []string{"支付", "合规"}, nil),
		facetHit("2", "HR", "", []string{"支付"}, nil),
	}
	tags := bucketByMany(hits, func(hit elasticHit) ([]string, bool) {
		return hit.Source.Tags, len(hit.Source.Tags) > 0
	})
	require.Len(t, tags, 2)
	assert.Equal(t, "支付", tags[0].Value)
	assert.Equal(t, 2, tags[0].Count)
	assert.Equal(t, 1, tags[1].Count)
}

func TestBucketByMonth(t *testing.T) {
	jan := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	hits := []elasticHit{
		facetHit("1", "a", "", nil, &jan),
		facetHit("2", "a", "", nil, &jan),
		facetHit("3", "a", "", nil, &feb),
		facetHit("4", "a", "", nil, nil), // no timestamp → skipped
	}
	months := bucketBy(hits, func(hit elasticHit) (string, bool) {
		if hit.Source.Updated == nil || hit.Source.Updated.IsZero() {
			return "", false
		}
		return hit.Source.Updated.Format("2006-01"), true
	})
	require.Len(t, months, 2)
	assert.Equal(t, "2026-01", months[0].Value)
	assert.Equal(t, 2, months[0].Count)
	assert.Equal(t, "2026-02", months[1].Value)
}
