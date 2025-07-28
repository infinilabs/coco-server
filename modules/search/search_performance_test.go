package search

import (
	"strings"
	"testing"
)

// Benchmark the old vs new string operations to demonstrate improvements

// BenchmarkOldStringOperations simulates the old approach with repeated string splits
func BenchmarkOldStringOperations(b *testing.B) {
	datasources := []string{
		"source1,source2,source3",
		"source4,source5",
		"source6",
		"source7,source8,source9,source10",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ds := range datasources {
			// Simulate the old repeated splits without caching
			arr1 := strings.Split(ds, ",")
			arr2 := strings.Split(ds, ",")
			arr3 := strings.Split(ds, ",")

			// Simulate query building with map allocations
			_ = map[string]interface{}{
				"terms": map[string]interface{}{
					"source.id": arr1,
				},
			}
			_ = map[string]interface{}{
				"terms": map[string]interface{}{
					"source.id": arr2,
				},
			}
			_ = map[string]interface{}{
				"terms": map[string]interface{}{
					"source.id": arr3,
				},
			}
		}
	}
}

// BenchmarkNewOptimizedOperations simulates the new optimized approach
func BenchmarkNewOptimizedOperations(b *testing.B) {
	datasources := []string{
		"source1,source2,source3",
		"source4,source5",
		"source6",
		"source7,source8,source9,source10",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ds := range datasources {
			// Use cached splits
			arr1 := splitCache.getCachedSplit(ds)
			arr2 := splitCache.getCachedSplit(ds)
			arr3 := splitCache.getCachedSplit(ds)

			// Use pooled query objects
			query1 := createTermsQuery("source.id", arr1)
			query2 := createTermsQuery("source.id", arr2)
			query3 := createTermsQuery("source.id", arr3)

			// Return to pool (simulate cleanup)
			putTermsQuery(query1)
			putTermsQuery(query2)
			putTermsQuery(query3)
		}
	}
}

// BenchmarkStringJoinOperations compares old vs new string joining
func BenchmarkOldStringJoin(b *testing.B) {
	arrays := [][]string{
		{"source1", "source2", "source3"},
		{"source4", "source5"},
		{"source6"},
		{"source7", "source8", "source9", "source10", "source11", "source12"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, arr := range arrays {
			_ = strings.Join(arr, ",")
		}
	}
}

func BenchmarkOptimizedStringJoin(b *testing.B) {
	arrays := [][]string{
		{"source1", "source2", "source3"},
		{"source4", "source5"},
		{"source6"},
		{"source7", "source8", "source9", "source10", "source11", "source12"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, arr := range arrays {
			_ = optimizedJoin(arr, ",")
		}
	}
}

// BenchmarkDisabledFilterCreation compares old vs new disabled filter creation
func BenchmarkOldDisabledFilterCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Old approach: create new complex map every time
		_ = map[string]interface{}{
			"bool": map[string]interface{}{
				"minimum_should_match": 1,
				"should": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"disabled": false,
						},
					},
					map[string]interface{}{
						"bool": map[string]interface{}{
							"must_not": map[string]interface{}{
								"exists": map[string]interface{}{
									"field": "disabled",
								},
							},
						},
					},
				},
			},
		}
	}
}

func BenchmarkNewDisabledFilterCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// New approach: get from pool
		filter := getDisabledFilter()
		putDisabledFilter(filter)
	}
}
