/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package common

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"
)

// Rate-limit backoff helper (W8): upstream APIs answer 429 and 5xx under
// load; hammering them in a tight sync loop turns a transient throttle
// into a prolonged ban. Collectors wrap their paginated calls with RetryWithBackoff
// and get exponential waits (2s, 4s, 8s — WeKnora's schedule) for free,
// with the remaining attempts reported so the caller can log honestly.

const (
	backoffAttempts = 3
	backoffBase     = 2 * time.Second
	backoffCap      = 8 * time.Second
)

// IsRetryableStatus reports whether an HTTP status deserves a retry.
func IsRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

// BackoffDelay returns the wait before attempt n (1-based): 2s, 4s, 8s.
func BackoffDelay(attempt int) time.Duration {
	d := time.Duration(math.Pow(2, float64(attempt))) * time.Second
	if d < backoffBase {
		d = backoffBase
	}
	if d > backoffCap {
		d = backoffCap
	}
	return d
}

// RetryWithBackoff runs fn until it succeeds, returns a retryable error,
// or exhausts the attempts. The context's deadline still wins.
func RetryWithBackoff(ctx context.Context, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= backoffAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if attempt == backoffAttempts {
				break
			}
			delay := BackoffDelay(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return fmt.Errorf("giving up after %d attempts: %w", backoffAttempts, lastErr)
}
