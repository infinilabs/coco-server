package deep_research

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/assistant/tools"
)

// SearchResult represents a single search result
type SearchResult struct {
	Source  string  `json:"source"` // "internal" or "external"
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

// SearchToolManager manages both internal and external search tools
type SearchToolManager struct {
	externalSearch *tools.TavilySearchTool
	internalSearch *tools.EnterpriseSearchTool
}

// NewSearchToolManager creates a new search tool manager
func NewSearchToolManager(tavilyAPIKey string) *SearchToolManager {
	return &SearchToolManager{
		externalSearch: &tools.TavilySearchTool{APIKey: tavilyAPIKey},
		internalSearch: &tools.EnterpriseSearchTool{},
	}
}

// SearchWithFeedback performs a comprehensive search with internal first, external as supplement
func (sm *SearchToolManager) SearchWithFeedback(ctx context.Context, query string, isInternalFirst bool) (*SearchResultCollection, error) {
	collection := &SearchResultCollection{
		Results: []SearchResult{},
		Query:   query,
	}

	log.Infof("Starting comprehensive search for query: %s", query)

	// Strategy 1: Internal search first (enterprise data priority)
	if isInternalFirst {
		internalResults, err := sm.performInternalSearch(ctx, query)
		if err != nil {
			log.Warnf("Internal search failed for query '%s': %v", query, err)
		} else {
			collection.Results = append(collection.Results, internalResults...)
			log.Infof("Internal search yielded %d results", len(internalResults))
		}

		// If internal search is insufficient, supplement with external search
		if !sm.isContentSufficient(collection) {
			log.Info("Internal search insufficient, proceeding with external search")
			externalResults, err := sm.performExternalSearch(ctx, query)
			if err != nil {
				log.Warnf("External search failed for query '%s': %v", query, err)
			} else {
				collection.Results = append(collection.Results, externalResults...)
				log.Infof("External search yielded %d results", len(externalResults))
			}
		}
	} else {
		// Strategy 2: External search first (for general topics)
		externalResults, err := sm.performExternalSearch(ctx, query)
		if err != nil {
			log.Warnf("External search failed for query '%s': %v", query, err)
		} else {
			collection.Results = append(collection.Results, externalResults...)
		}

		// Supplement with internal search regardless
		internalResults, err := sm.performInternalSearch(ctx, query)
		if err != nil {
			log.Warnf("Internal search failed for query '%s': %v", query, err)
		} else {
			collection.Results = append(collection.Results, internalResults...)
		}
	}

	// Assess search quality and sufficiency
	collection.evaluateSearchQuality()
	// Set IsSufficient based on content sufficiency check
	collection.IsSufficient = sm.isContentSufficient(collection)

	log.Infof("Search completed. Total results: %d, Sufficient: %t, Confidence: %.2f",
		len(collection.Results), collection.IsSufficient, collection.Confidence)

	return collection, nil
}

// performInternalSearch executes internal enterprise search
func (sm *SearchToolManager) performInternalSearch(ctx context.Context, query string) ([]SearchResult, error) {
	result, err := sm.internalSearch.Call(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("internal search failed: %w", err)
	}

	// Parse the result string into SearchResult structures
	return sm.parseInternalResults(result, query), nil
}

// performExternalSearch executes external Tavily search
func (sm *SearchToolManager) performExternalSearch(ctx context.Context, query string) ([]SearchResult, error) {
	result, err := sm.externalSearch.Call(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("external search failed: %w", err)
	}

	// Parse the result string into SearchResult structures
	return sm.parseExternalResults(result, query), nil
}

// parseInternalResults parses internal search result format
func (sm *SearchToolManager) parseInternalResults(result string, query string) []SearchResult {
	var results []SearchResult
	lines := strings.Split(result, "\n")

	var currentResult *SearchResult
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse result sections like "[Result 1]", "Title:", "URL:", "Content:"
		if strings.HasPrefix(line, "[Result") {
			if currentResult != nil && currentResult.Title != "" {
				results = append(results, *currentResult)
			}
			currentResult = &SearchResult{
				Source: "internal",
				Score:  0.8, // Internal results get higher base score
			}
		} else if strings.HasPrefix(line, "Title:") && currentResult != nil {
			currentResult.Title = strings.TrimPrefix(line, "Title:")
			currentResult.Title = strings.TrimSpace(currentResult.Title)
		} else if strings.HasPrefix(line, "URL:") && currentResult != nil {
			currentResult.URL = strings.TrimPrefix(line, "URL:")
			currentResult.URL = strings.TrimSpace(currentResult.URL)
		} else if strings.HasPrefix(line, "Content:") && currentResult != nil {
			currentResult.Content = strings.TrimPrefix(line, "Content:")
			currentResult.Content = strings.TrimSpace(currentResult.Content)
		}
	}

	// Add the last result
	if currentResult != nil && currentResult.Title != "" {
		results = append(results, *currentResult)
	}

	return results
}

// parseExternalResults parses external search result format
func (sm *SearchToolManager) parseExternalResults(result string, query string) []SearchResult {
	var results []SearchResult
	lines := strings.Split(result, "\n")

	var currentResult *SearchResult
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse result sections like "[Result 1]", "Title:", "URL:", "Content:"
		if strings.HasPrefix(line, "[Result") {
			if currentResult != nil && currentResult.Title != "" {
				results = append(results, *currentResult)
			}
			currentResult = &SearchResult{
				Source: "external",
				Score:  0.6, // External results get lower base score
			}
		} else if strings.HasPrefix(line, "Title:") && currentResult != nil {
			currentResult.Title = strings.TrimPrefix(line, "Title:")
			currentResult.Title = strings.TrimSpace(currentResult.Title)
		} else if strings.HasPrefix(line, "URL:") && currentResult != nil {
			currentResult.URL = strings.TrimPrefix(line, "URL:")
			currentResult.URL = strings.TrimSpace(currentResult.URL)
		} else if strings.HasPrefix(line, "Content:") && currentResult != nil {
			currentResult.Content = strings.TrimPrefix(line, "Content:")
			currentResult.Content = strings.TrimSpace(currentResult.Content)
		}
	}

	// Add the last result
	if currentResult != nil && currentResult.Title != "" {
		results = append(results, *currentResult)
	}

	return results
}

// isContentSufficient checks if we have enough quality content
func (sm *SearchToolManager) isContentSufficient(collection *SearchResultCollection) bool {
	if len(collection.Results) < 2 {
		return false
	}

	// Check total content length
	totalContentLen := 0
	for _, result := range collection.Results {
		totalContentLen += len(result.Content)
	}

	// We need at least 1000 characters of meaningful content
	if totalContentLen < 1000 {
		return false
	}

	// Check if we have diverse sources
	hasInternal := false
	hasExternal := false
	for _, result := range collection.Results {
		if result.Source == "internal" {
			hasInternal = true
		} else {
			hasExternal = true
		}
	}

	return hasInternal || hasExternal
}

// evaluateSearchQuality assesses the overall quality of search results
func (collection *SearchResultCollection) evaluateSearchQuality() {
	if len(collection.Results) == 0 {
		collection.Confidence = 0.0
		return
	}

	var totalScore float64
	var totalLength int
	var internalCount, externalCount int

	for _, result := range collection.Results {
		totalScore += result.Score
		totalLength += len(result.Content)

		if result.Source == "internal" {
			internalCount++
		} else {
			externalCount++
		}
	}

	avgScore := totalScore / float64(len(collection.Results))

	// Calculate confidence based on multiple factors
	confidence := avgScore

	// Boost confidence if we have diverse sources
	if internalCount > 0 && externalCount > 0 {
		confidence += 0.1
	}

	// Boost confidence if we have substantial content
	if totalLength > 2000 {
		confidence += 0.05
	}

	// Ensure confidence is between 0 and 1
	if confidence > 1.0 {
		confidence = 1.0
	}

	collection.Confidence = confidence
}

// FormatResultsForLLM formats search results for LLM consumption
func (collection *SearchResultCollection) FormatResultsForLLM() string {
	if len(collection.Results) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("关于'%s'的搜索结果（共%d条）:\n\n", collection.Query, len(collection.Results)))

	for i, result := range collection.Results {
		builder.WriteString(fmt.Sprintf("[%d] %s\n", i+1, result.Title))
		builder.WriteString(fmt.Sprintf("来源: %s\n", result.Source))
		if result.URL != "" {
			builder.WriteString(fmt.Sprintf("链接: %s\n", result.URL))
		}
		builder.WriteString(fmt.Sprintf("内容: %s\n\n", result.Content))
	}

	builder.WriteString(fmt.Sprintf("搜索质量评估: 置信度 %.1f%%，内容%s",
		collection.Confidence*100,
		map[bool]string{true: "充足", false: "不足"}[collection.IsSufficient]))

	return builder.String()
}
