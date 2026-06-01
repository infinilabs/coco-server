package deep_research

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	duckduckgotool "github.com/tmc/langchaingo/tools/duckduckgo"
	wikitool "github.com/tmc/langchaingo/tools/wikipedia"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/tools"
)

// SearchResult represents a single search result
type SearchResult struct {
	Source  string  `json:"source"` // "internal", "external", "duckduckgo", "wikipedia"
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// SearchResultCollection manages search results with quality assessment
type SearchResultCollection struct {
	Results      []SearchResult
	Query        string
	IsSufficient bool    // Whether we have enough quality content
	Confidence   float64 // Overall confidence score (0-1)
}

// SearchWithConfig performs a comprehensive search using DeepResearchConfig directly.
func SearchWithConfig(ctx context.Context, query string, cfg *core.DeepResearchConfig, isInternalFirst bool) (*SearchResultCollection, error) {
	collection := &SearchResultCollection{Results: []SearchResult{}, Query: query}

	log.Infof("Starting search for query: %s", query)

	internalTool := &tools.EnterpriseSearchTool{DatasourceIDs: cfg.InternalSearch.DatasourceIDs}

	doInternal := func() {
		result, err := internalTool.Call(ctx, query)
		if err != nil {
			log.Warnf("Internal search failed for query '%s': %v", query, err)
			return
		}
		r := parseResults(result, "internal", 0.8)
		collection.Results = append(collection.Results, r...)
		log.Infof("Internal search yielded %d results", len(r))
	}

	doExternal := func() {
		r, err := runExternalEngine(ctx, cfg, query)
		if err != nil {
			log.Warnf("External search (%s) failed for query '%s': %v", cfg.ExternalSearch.Engine, query, err)
			return
		}
		collection.Results = append(collection.Results, r...)
		log.Infof("External search (%s) yielded %d results", cfg.ExternalSearch.Engine, len(r))
	}

	if isInternalFirst {
		doInternal()
		doExternal()
	} else {
		doExternal()
		doInternal()
	}

	collection.evaluateSearchQuality()

	collection.IsSufficient = isContentSufficient(collection, cfg.ResearchDepth)
	log.Infof("Search completed. Total results: %d, Sufficient: %t, Confidence: %.2f",
		len(collection.Results), collection.IsSufficient, collection.Confidence)
	return collection, nil
}

// runExternalEngine dispatches to the configured external search engine.
func runExternalEngine(ctx context.Context, cfg *core.DeepResearchConfig, query string) ([]SearchResult, error) {
	maxResults := cfg.MaxResults
	if maxResults <= 0 {
		maxResults = 5
	}
	const webAgent = "Mozilla/5.0 (compatible; CocoResearch/1.0)"

	switch cfg.ExternalSearch.Engine {
	case "tavily":
		result, err := (&tools.TavilySearchTool{APIKey: cfg.ExternalSearch.APIKey, MaxResults: maxResults}).Call(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("tavily search failed: %w", err)
		}
		return parseResults(result, "external", 0.6), nil

	case "duckduckgo":
		ddg, err := duckduckgotool.New(maxResults, webAgent)
		if err != nil {
			return nil, fmt.Errorf("duckduckgo init failed: %w", err)
		}
		result, err := ddg.Call(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("duckduckgo search failed: %w", err)
		}
		return []SearchResult{{Source: "duckduckgo", Title: query, Content: result, Score: 0.6}}, nil

	case "wikipedia":
		wt := wikitool.New(webAgent)
		result, err := wt.Call(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("wikipedia search failed: %w", err)
		}
		wikiURL := fmt.Sprintf("https://en.wikipedia.org/wiki/%s", strings.ReplaceAll(query, " ", "_"))
		return []SearchResult{{Source: "wikipedia", Title: query, URL: wikiURL, Content: result, Score: 0.75}}, nil

	default:
		return nil, fmt.Errorf("unknown external search engine: %q", cfg.ExternalSearch.Engine)
	}
}

// parseResults parses the "Title:/URL:/Content:" line format returned by internal/Tavily tools.
func parseResults(raw, source string, baseScore float64) []SearchResult {
	var results []SearchResult
	var cur *SearchResult
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[Result") {
			if cur != nil && cur.Title != "" {
				results = append(results, *cur)
			}
			cur = &SearchResult{Source: source, Score: baseScore}
		} else if cur != nil {
			switch {
			case strings.HasPrefix(line, "Title:"):
				cur.Title = strings.TrimSpace(strings.TrimPrefix(line, "Title:"))
			case strings.HasPrefix(line, "URL:"):
				cur.URL = strings.TrimSpace(strings.TrimPrefix(line, "URL:"))
			case strings.HasPrefix(line, "Content:"):
				cur.Content = strings.TrimSpace(strings.TrimPrefix(line, "Content:"))
			}
		}
	}
	if cur != nil && cur.Title != "" {
		results = append(results, *cur)
	}
	return results
}

// isContentSufficient checks if we have enough quality content for the given research depth.
func isContentSufficient(collection *SearchResultCollection, researchDepth string) bool {
	var minResults, minContentLen int
	switch researchDepth {
	case "basic":
		minResults, minContentLen = 1, 500
	case "exhaustive":
		minResults, minContentLen = 4, 2000
	default: // "comprehensive"
		minResults, minContentLen = 2, 1000
	}
	if len(collection.Results) < minResults {
		return false
	}
	total := 0
	for _, r := range collection.Results {
		total += len(r.Content)
	}
	return total >= minContentLen
}

// evaluateSearchQuality assesses the overall quality of search results.
func (collection *SearchResultCollection) evaluateSearchQuality() {
	if len(collection.Results) == 0 {
		collection.Confidence = 0.0
		return
	}
	var totalScore float64
	var totalLength, internalCount, externalCount int
	for _, r := range collection.Results {
		totalScore += r.Score
		totalLength += len(r.Content)
		if r.Source == "internal" {
			internalCount++
		} else {
			externalCount++
		}
	}
	confidence := totalScore / float64(len(collection.Results))
	if internalCount > 0 && externalCount > 0 {
		confidence += 0.1
	}
	if totalLength > 2000 {
		confidence += 0.05
	}
	if confidence > 1.0 {
		confidence = 1.0
	}
	collection.Confidence = confidence
}

// FormatResultsForLLM formats search results for LLM consumption.
func (collection *SearchResultCollection) FormatResultsForLLM() string {
	if len(collection.Results) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Search results for '%s' (total %d items):\n\n", collection.Query, len(collection.Results))
	for i, r := range collection.Results {
		fmt.Fprintf(&b, "[%d] %s\nSource: %s\n", i+1, r.Title, r.Source)
		if r.URL != "" {
			fmt.Fprintf(&b, "Link: %s\n", r.URL)
		}
		fmt.Fprintf(&b, "Content: %s\n\n", r.Content)
	}
	sufficient := "insufficient"
	if collection.IsSufficient {
		sufficient = "sufficient"
	}
	fmt.Fprintf(&b, "Search quality: confidence=%.1f%%, content=%s", collection.Confidence*100, sufficient)
	return b.String()
}
