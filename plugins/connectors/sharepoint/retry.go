package sharepoint

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	log "github.com/cihub/seelog"
)

type RetryClient struct {
	config RetryConfig
}

func NewRetryClient(config RetryConfig) *RetryClient {
	// 设置默认值
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.InitialDelay == 0 {
		config.InitialDelay = time.Second
	}
	if config.MaxDelay == 0 {
		config.MaxDelay = time.Minute
	}
	if config.BackoffFactor == 0 {
		config.BackoffFactor = 2.0
	}

	return &RetryClient{config: config}
}

func (r *RetryClient) DoWithRetry(ctx context.Context, fn func() (*http.Response, error)) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := r.calculateDelay(attempt)
			log.Debugf("[sharepoint connector] retrying request after %v (attempt %d/%d)", delay, attempt, r.config.MaxRetries)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// 继续重试
			}
		}

		resp, err := fn()
		if err != nil {
			lastErr = err
			if !r.isRetryableError(err) {
				return nil, err
			}
			continue
		}

		// 检查HTTP状态码
		if r.isRetryableStatusCode(resp.StatusCode) {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (r *RetryClient) calculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(r.config.InitialDelay) * math.Pow(r.config.BackoffFactor, float64(attempt-1)))
	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}
	return delay
}

func (r *RetryClient) isRetryableError(err error) bool {
	// 网络相关错误通常可以重试
	return true
}

func (r *RetryClient) isRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}
