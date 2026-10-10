/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"errors"
	"math"
	"testing"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
)

func TestCosineSimilarityF32(t *testing.T) {
	cases := []struct {
		name string
		a, b []float32
		want float64
	}{
		{"identical", []float32{1, 2, 3}, []float32{1, 2, 3}, 1},
		{"scaled", []float32{1, 1}, []float32{2, 2}, 1},
		{"orthogonal", []float32{1, 0}, []float32{0, 1}, 0},
		{"opposite", []float32{1, 0}, []float32{-1, 0}, -1},
		{"zero vector", []float32{0, 0}, []float32{1, 1}, 0},
		{"length mismatch", []float32{1}, []float32{1, 2}, 0},
		{"empty", nil, nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := cosineSimilarityF32(c.a, c.b)
			if math.Abs(got-c.want) > 1e-9 {
				t.Fatalf("cosine = %v, want %v", got, c.want)
			}
		})
	}
}

func hit(id string, score float32) elastic.DocumentWithMeta[core.Document] {
	return elastic.DocumentWithMeta[core.Document]{ID: id, Score: score}
}

func TestRerankHitsBySemantic(t *testing.T) {
	queryVec := []float32{1, 0}
	vectors := map[string][]float32{
		"opposite": {0, 1}, // cosine 0 -> score 1
		"same":     {1, 0}, // cosine 1 -> score 2
		"partial":  {1, 1}, // cosine ~0.707 -> score ~1.707
	}
	hits := []elastic.DocumentWithMeta[core.Document]{hit("opposite", 3), hit("no-vector", 2), hit("same", 1), hit("partial", 0.5)}

	rerankHitsBySemantic(vectors, queryVec, hits)

	order := make([]string, len(hits))
	for i, h := range hits {
		order[i] = h.ID
	}
	want := []string{"same", "partial", "opposite", "no-vector"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
	// scored hits get cosine+1, the vectorless tail keeps its BM25 score
	if hits[0].Score != 2 {
		t.Fatalf("top score = %v, want 2", hits[0].Score)
	}
	if math.Abs(float64(hits[1].Score)-1.7071067811865475) > 1e-6 {
		t.Fatalf("partial score = %v, want ~1.7071", hits[1].Score)
	}
	if hits[3].Score != 2 { // original BM25 score of "no-vector"
		t.Fatalf("vectorless hit score = %v, want its BM25 score 2", hits[3].Score)
	}
}

type fakeEmbeddingClient struct {
	vectors [][]float32
	err     error
}

func (f fakeEmbeddingClient) CreateEmbedding(ctx context.Context, texts []string) ([][]float32, error) {
	return f.vectors, f.err
}

func TestEmbedTextWith(t *testing.T) {
	dims := core.RequiredEmbeddingDimension
	right := make([]float32, dims)
	short := make([]float32, dims-1)

	if _, err := embedTextWith(context.Background(), fakeEmbeddingClient{vectors: [][]float32{right}}, "q"); err != nil {
		t.Fatalf("valid embedding failed: %v", err)
	}
	if _, err := embedTextWith(context.Background(), fakeEmbeddingClient{vectors: [][]float32{short}}, "q"); err == nil {
		t.Fatal("dimension mismatch should fail")
	}
	if _, err := embedTextWith(context.Background(), fakeEmbeddingClient{vectors: nil}, "q"); err == nil {
		t.Fatal("missing vector should fail")
	}
	if _, err := embedTextWith(context.Background(), fakeEmbeddingClient{err: errors.New("boom")}, "q"); err == nil {
		t.Fatal("provider error should propagate")
	}
}

func TestPaginateHits(t *testing.T) {
	makeResult := func(n int) *elastic.SearchResponseWithMeta[core.Document] {
		out := &elastic.SearchResponseWithMeta[core.Document]{}
		for i := 0; i < n; i++ {
			out.Hits.Hits = append(out.Hits.Hits, hit(string(rune('a'+i)), float32(n-i)))
		}
		return out
	}

	r := makeResult(5)
	paginateHits(r, 1, 2)
	if len(r.Hits.Hits) != 2 || r.Hits.Hits[0].ID != "b" {
		t.Fatalf("page [1:3] of abcde = %v", r.Hits.Hits)
	}

	r = makeResult(5)
	paginateHits(r, 10, 3)
	if len(r.Hits.Hits) != 0 {
		t.Fatalf("page beyond the end should be empty, got %d hits", len(r.Hits.Hits))
	}
}

func TestRRFRecallWindow(t *testing.T) {
	cases := []struct {
		from, size, want int
	}{
		{0, 10, 10},
		{90, 10, 100},
		{190, 50, maxRRFWindow},
		{-5, 10, 10},
		{0, 0, 10},
	}
	for _, c := range cases {
		if got := rrfRecallWindow(c.from, c.size); got != c.want {
			t.Fatalf("rrfRecallWindow(%d, %d) = %d, want %d", c.from, c.size, got, c.want)
		}
	}
}
