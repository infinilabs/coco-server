/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"testing"

	"infini.sh/framework/core/elastic"
)

// esTopHitsBucket builds a source.id terms bucket shaped exactly like the
// engine returns it: the label the widget renders lives deep in the
// top_hits payload (bucket.top.hits.hits[0]._source.source.name)
func esTopHitsBucket(key, name string) elastic.BucketBase {
	return elastic.BucketBase{
		"key":       key,
		"doc_count": 3,
		"top": map[string]interface{}{
			"hits": map[string]interface{}{
				"hits": []interface{}{
					map[string]interface{}{
						"_source": map[string]interface{}{
							"source": map[string]interface{}{
								"id":   key,
								"name": name,
							},
						},
					},
				},
			},
		},
	}
}

func bucketLabel(t *testing.T, bucket elastic.BucketBase) string {
	t.Helper()
	top, ok := bucket["top"].(map[string]interface{})
	if !ok {
		t.Fatal("bucket lost its top_hits payload")
	}
	hitsObj := top["hits"].(map[string]interface{})
	hits := hitsObj["hits"].([]interface{})
	hit := hits[0].(map[string]interface{})
	source := hit["_source"].(map[string]interface{})
	docSource := source["source"].(map[string]interface{})
	name, _ := docSource["name"].(string)
	return name
}

func TestRewriteBucketDatasourceName(t *testing.T) {
	bucket := esTopHitsBucket("ds-1", "本地文件系统")
	rewriteBucketDatasourceName(bucket, "美心")
	if got := bucketLabel(t, bucket); got != "美心" {
		t.Fatalf("label should be refreshed, got %q", got)
	}

	// doc_count and key must survive the rewrite untouched
	if bucket["doc_count"] != 3 || bucket["key"] != "ds-1" {
		t.Fatalf("bucket metadata was clobbered: %+v", bucket)
	}
}

func TestRewriteBucketDatasourceNameSkipsMalformedBuckets(t *testing.T) {
	// none of these may panic — real aggregation payloads vary
	for _, bucket := range []elastic.BucketBase{
		{},
		{"key": "ds-1"},                     // no top_hits at all
		{"key": "ds-1", "top": "not-a-map"}, // wrong type
		{"key": "ds-1", "top": map[string]interface{}{"hits": "nope"}},
		{"key": "ds-1", "top": map[string]interface{}{"hits": map[string]interface{}{"hits": []interface{}{}}}},
		{"key": "ds-1", "top": map[string]interface{}{"hits": map[string]interface{}{"hits": []interface{}{"str-hit"}}}},
		{"key": "ds-1", "top": map[string]interface{}{"hits": map[string]interface{}{"hits": []interface{}{map[string]interface{}{"_source": 1}}}}},
	} {
		rewriteBucketDatasourceName(bucket, "美心")
	}
}
