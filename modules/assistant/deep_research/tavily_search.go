package deep_research

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

// TavilySearchResult represents a single search result from Tavily
type TavilySearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// TavilySearchResponse represents the response from Tavily API
type TavilySearchResponse struct {
	Results []TavilySearchResult `json:"results"`
}

// TavilySearchTool implements web search using Tavily API
type TavilySearchTool struct {
	APIKey string
}

// Name returns the tool name
func (t *TavilySearchTool) Name() string {
	return "tavily_search"
}

// Description returns the tool description
func (t *TavilySearchTool) Description() string {
	return "Search the web for information using Tavily search API. Input should be a search query string."
}

// Call executes the search
func (t *TavilySearchTool) Call(ctx context.Context, input string) (string, error) {
	log.Info("start call TavilySearchTool:", input)

	if t.APIKey == "" {
		return "", fmt.Errorf("TAVILY_API_KEY not set")
	}

	// Prepare request
	reqBody := map[string]interface{}{
		"api_key":     t.APIKey,
		"query":       input,
		"max_results": 5,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	resp, err := http.Post(
		"https://api.tavily.com/search",
		"application/json",
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	//log.Error("start call TavilySearchTool", string(body))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var searchResp TavilySearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Format results
	var results []string
	for i, result := range searchResp.Results {
		results = append(results, fmt.Sprintf(
			"[Result %d]\nTitle: %s\nURL: %s\nContent: %s\n",
			i+1, result.Title, result.URL, result.Content,
		))
	}

	if global.Env().IsDebug {
		log.Trace(util.MustToJSON(results))
	}

	return strings.Join(results, "\n---\n"), nil
}
