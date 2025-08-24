/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/util"
)

const (
	APIPathSearch   = "/rest/api/2/search"
	APIPathComments = "/rest/api/2/issue/%s/comment"
	DefaultTimeout  = 30 * time.Second
)

type JiraClient struct {
	baseURL    string
	httpClient *http.Client
	config     *Config
	retryCount int
}

func NewJiraClient(config *Config) (*JiraClient, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	// 确保 URL 格式正确
	baseURL := strings.TrimSuffix(config.BaseURL, "/")

	client := &JiraClient{
		baseURL: baseURL,
		config:  config,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		retryCount: MaxRetries,
	}

	return client, nil
}

func (c *JiraClient) SearchIssues(ctx context.Context, jql string, startAt, maxResults int) (*SearchResult, error) {
	params := url.Values{}
	params.Set("jql", jql)
	params.Set("startAt", fmt.Sprintf("%d", startAt))
	params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	params.Set("expand", "names,schema")

	// 添加字段过滤
	if len(c.config.Fields) > 0 {
		fields := strings.Join(c.config.Fields, ",")
		params.Set("fields", fields)
	}

	apiURL := fmt.Sprintf("%s%s?%s", c.baseURL, APIPathSearch, params.Encode())

	var result SearchResult
	err := c.makeRequestWithRetry(ctx, "GET", apiURL, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to search issues: %w", err)
	}

	return &result, nil
}

func (c *JiraClient) GetIssueComments(ctx context.Context, issueKey string) (*CommentsResponse, error) {
	apiURL := fmt.Sprintf("%s"+APIPathComments, c.baseURL, issueKey)

	var result CommentsResponse
	err := c.makeRequestWithRetry(ctx, "GET", apiURL, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments for issue %s: %w", issueKey, err)
	}

	// 为每个评论设置关联的问题键
	for i := range result.Comments {
		result.Comments[i].IssueKey = issueKey
	}

	return &result, nil
}

func (c *JiraClient) makeRequestWithRetry(ctx context.Context, method, url string, body io.Reader, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.retryCount; attempt++ {
		if attempt > 0 {
			// 指数退避
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}

			log.Debugf("[jira client] retrying request after %v (attempt %d/%d)", backoff, attempt, c.retryCount)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := c.makeRequest(ctx, method, url, body, result)
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否应该重试
		if !c.shouldRetry(err) {
			break
		}

		log.Warnf("[jira client] request failed, will retry: %v", err)
	}

	return fmt.Errorf("request failed after %d attempts: %w", c.retryCount+1, lastErr)
}

func (c *JiraClient) makeRequest(ctx context.Context, method, url string, body io.Reader, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置认证
	c.setAuth(req)

	// 设置请求头
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Coco-Server-Jira-Connector/1.0")

	log.Debugf("[jira client] making request: %s %s", method, url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Debugf("[jira client] response status: %d", resp.StatusCode)

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查 HTTP 状态码
	if err := c.checkHTTPStatus(resp.StatusCode, respBody); err != nil {
		return err
	}

	// 解析 JSON 响应
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse JSON response: %w", err)
		}
	}

	return nil
}

func (c *JiraClient) setAuth(req *http.Request) {
	switch strings.ToLower(c.config.AuthType) {
	case "api_token":
		if c.config.Username != "" && c.config.APIToken != "" {
			req.SetBasicAuth(c.config.Username, c.config.APIToken)
		}
	case "basic_auth", "":
		if c.config.Username != "" && c.config.Password != "" {
			req.SetBasicAuth(c.config.Username, c.config.Password)
		}
	default:
		log.Warnf("[jira client] unsupported auth type: %s", c.config.AuthType)
	}
}

func (c *JiraClient) checkHTTPStatus(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return &JiraError{
			StatusCode: statusCode,
			Message:    "authentication failed - check your credentials",
			Retryable:  false,
		}
	case http.StatusForbidden:
		return &JiraError{
			StatusCode: statusCode,
			Message:    "access forbidden - check your permissions",
			Retryable:  false,
		}
	case http.StatusNotFound:
		return &JiraError{
			StatusCode: statusCode,
			Message:    "resource not found",
			Retryable:  false,
		}
	case http.StatusTooManyRequests:
		// 解析 Retry-After 头部
		return &JiraError{
			StatusCode: statusCode,
			Message:    "rate limit exceeded",
			Retryable:  true,
		}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return &JiraError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("server error: %d", statusCode),
			Retryable:  true,
		}
	default:
		message := fmt.Sprintf("unexpected status code: %d", statusCode)
		if len(body) > 0 && len(body) < 1000 {
			message += fmt.Sprintf(", body: %s", string(body))
		}
		return &JiraError{
			StatusCode: statusCode,
			Message:    message,
			Retryable:  statusCode >= 500,
		}
	}
}

func (c *JiraClient) shouldRetry(err error) bool {
	if jiraErr, ok := err.(*JiraError); ok {
		return jiraErr.Retryable
	}

	// 网络错误通常可以重试
	if util.IsNetworkError(err) {
		return true
	}

	return false
}

// JiraError 自定义错误类型
type JiraError struct {
	StatusCode int
	Message    string
	Retryable  bool
}

func (e *JiraError) Error() string {
	return fmt.Sprintf("jira api error (status %d): %s", e.StatusCode, e.Message)
}
